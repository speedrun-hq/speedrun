package services

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/speedrun-hq/speedrun/api/config"
	"github.com/speedrun-hq/speedrun/api/db"
	"github.com/speedrun-hq/speedrun/api/logging"
)

// Constants for timeouts and monitoring
const (
	// CatchupOperationTimeout is the maximum time allowed for a single catchup operation
	CatchupOperationTimeout = 10 * time.Minute

	// BlockRangeProcessTimeout is the maximum time allowed for processing a range of blocks
	BlockRangeProcessTimeout = 5 * time.Minute

	// LogBatchProcessTimeout is the maximum time allowed for processing a batch of logs
	LogBatchProcessTimeout = 5 * time.Minute

	// FilterLogsTimeout is the maximum time allowed for filtering logs
	FilterLogsTimeout = 3 * time.Minute

	// MonitoringInterval is how often to log the status of ongoing operations
	MonitoringInterval = 30 * time.Second

	// DefaultMaxBlockRange is the default maximum block range for catchup operations
	// add these constants for block range optimization
	// default max range for most chains
	DefaultMaxBlockRange = uint64(5000)

	// EthereumMaxBlockRange is the maximum block range for Ethereum chains
	// NOTE: Smaller range for Ethereum mainnet (chain ID 1) since it has higher transaction density
	EthereumMaxBlockRange = uint64(1000)
)

// EventCatchupService coordinates the catch-up process between intent and fulfillment services
type EventCatchupService struct {
	intentServices      map[uint64]*IntentService
	fulfillmentServices map[uint64]*FulfillmentService
	settlementServices  map[uint64]*SettlementService
	db                  db.Database
	mu                  sync.Mutex
	intentProgress      map[uint64]uint64 // chainID -> last processed block
	fulfillmentProgress map[uint64]uint64 // chainID -> last processed block
	settlementProgress  map[uint64]uint64 // chainID -> last processed block
	activeCatchups      map[string]bool   // Track active catchup operations
	catchupMu           sync.Mutex        // Mutex for the activeCatchups map
	logger              zerolog.Logger

	// Goroutine tracking
	activeGoroutines int32 // Counter for active goroutines

	// Goroutine cleanup management
	cleanupCtx    context.Context    // Context for cleanup operations
	cleanupCancel context.CancelFunc // Cancel function for cleanup context
	goroutineWg   sync.WaitGroup     // WaitGroup to track all goroutines
	isShutdown    bool               // Flag to prevent new goroutines after shutdown
	shutdownMu    sync.RWMutex       // Mutex for shutdown operations
}

// NewEventCatchupService creates a new EventCatchupService instance
func NewEventCatchupService(
	intentServices map[uint64]*IntentService,
	fulfillmentServices map[uint64]*FulfillmentService,
	settlementServices map[uint64]*SettlementService,
	db db.Database,
	logger zerolog.Logger,
) *EventCatchupService {
	// Create cleanup context
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())

	return &EventCatchupService{
		intentServices:      intentServices,
		fulfillmentServices: fulfillmentServices,
		settlementServices:  settlementServices,
		db:                  db,
		intentProgress:      make(map[uint64]uint64),
		fulfillmentProgress: make(map[uint64]uint64),
		settlementProgress:  make(map[uint64]uint64),
		activeCatchups:      make(map[string]bool),
		logger:              logger,
		cleanupCtx:          cleanupCtx,
		cleanupCancel:       cleanupCancel,
	}
}

// StartListening starts the coordinated event listening process
func (s *EventCatchupService) StartListening(ctx context.Context) error {
	// Check if service is shutdown
	if s.IsShutdown() {
		return fmt.Errorf("cannot start listening: service is shutdown")
	}

	s.logger.Info().Msg("Starting event catchup service")

	// Start a monitoring goroutine to periodically log the status
	s.StartGoroutine("catchup-monitor", func() {
		s.monitorCatchupProgress(s.cleanupCtx)
	})

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// Print information about active chains for debugging
	for chainID, chainConfig := range cfg.ChainConfigs {
		s.logger.Debug().
			Uint64(logging.FieldChain, chainID).
			Str("contract", chainConfig.ContractAddr).
			Msg("Configured chain")
	}

	// Initialize progress tracking for all chains
	s.mu.Lock()
	for chainID := range s.intentServices {
		s.logger.Info().
			Uint64(logging.FieldChain, chainID).
			Msg("Initializing intent progress tracking")
		lastBlock, err := s.db.GetLastProcessedBlock(ctx, chainID)
		if err != nil {
			s.mu.Unlock()
			return fmt.Errorf("failed to get last processed block for chain %d: %v", chainID, err)
		}
		if lastBlock < cfg.ChainConfigs[chainID].DefaultBlock {
			s.logger.Info().
				Uint64(logging.FieldChain, chainID).
				Uint64(logging.FieldBlock, lastBlock).
				Uint64("default_block", cfg.ChainConfigs[chainID].DefaultBlock).
				Msg("Last processed block is less than default block, using default")
			lastBlock = cfg.ChainConfigs[chainID].DefaultBlock
		}
		s.intentProgress[chainID] = lastBlock
		s.logger.Info().
			Uint64(logging.FieldChain, chainID).
			Uint64(logging.FieldBlock, lastBlock).
			Msg("Setting intent progress block")
	}
	s.mu.Unlock()

	// Get current block numbers for all chains
	currentBlocks := make(map[uint64]uint64)
	for chainID, intentService := range s.intentServices {
		// Add timeout for RPC call
		blockCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		currentBlock, err := intentService.client.BlockNumber(blockCtx)
		cancel()

		if err != nil {
			return fmt.Errorf("failed to get current block number for chain %d: %v", chainID, err)
		}
		currentBlocks[chainID] = currentBlock
		s.logger.Info().
			Uint64(logging.FieldChain, chainID).
			Uint64(logging.FieldBlock, currentBlock).
			Msg("Got current block number")
	}

	// Track any errors that occur during catchup
	var catchupErrors []error

	// INTENT CATCHUP
	s.logger.Info().Msg("Starting intent event catchup")
	if err := s.runIntentCatchup(ctx, cfg, currentBlocks); err != nil {
		// Store the error but continue with fulfillment and settlement catchup
		catchupErrors = append(catchupErrors, fmt.Errorf("intent catchup failed: %v", err))
		// TODO: consider throwing an error here
		s.logger.Warn().Err(err).Msg("Intent catchup encountered errors, continuing with fulfillment catchup")
	} else {
		s.logger.Info().Msg("All intent services have completed catchup successfully")
	}

	// FULFILLMENT CATCHUP
	log.Printf("Starting fulfillment catchup")
	if err := s.runFulfillmentCatchup(ctx, cfg, currentBlocks); err != nil {
		// Store the error but continue with settlement catchup
		catchupErrors = append(catchupErrors, fmt.Errorf("fulfillment catchup failed: %v", err))
		// TODO: consider throwing an error here
		s.logger.Warn().Err(err).Msg("Fulfillment catchup encountered errors, continuing with settlement catchup")
	} else {
		s.logger.Info().Msg("All fulfillment services have completed catchup successfully")
	}

	// SETTLEMENT CATCHUP
	s.logger.Info().Msg("Starting settlement catchup")
	if err := s.runSettlementCatchup(ctx, cfg, currentBlocks); err != nil {
		// Store the error
		catchupErrors = append(catchupErrors, fmt.Errorf("settlement catchup failed: %v", err))
		// TODO: consider throwing an error here
		s.logger.Warn().Err(err).Msg("Settlement catchup encountered errors")
	} else {
		s.logger.Info().Msg("All settlement services have completed catchup successfully")
	}

	// Only attempt to update processed blocks for chains that completed successfully
	for chainID, currentBlock := range currentBlocks {
		updateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		if err := s.db.UpdateLastProcessedBlock(updateCtx, chainID, currentBlock); err != nil {
			s.logger.Warn().
				Uint64(logging.FieldChain, chainID).
				Err(err).
				Msg("Failed to update last processed block")
			// Don't return an error here, just log the warning
		} else {
			s.logger.Info().
				Uint64(logging.FieldChain, chainID).
				Uint64(logging.FieldBlock, currentBlock).
				Msg("Updated last processed block")
		}
		cancel()
	}

	// Start live subscriptions for all services
	if err := s.StartLiveEventListeners(ctx, cfg); err != nil {
		catchupErrors = append(catchupErrors, fmt.Errorf("failed to start live subscriptions: %v", err))
		s.logger.Warn().Err(err).Msg("Failed to start some live subscriptions")
	}

	// If there were any errors during the catchup process, log them but don't fail
	if len(catchupErrors) > 0 {
		s.logger.Debug().
			Int("error_count", len(catchupErrors)).
			Msg("Catchup process completed with errors")
		for i, err := range catchupErrors {
			s.logger.Error().
				Int("error_index", i+1).
				Err(err).
				Msg("Catchup error")
		}
	}

	return nil
}

// monitorCatchupProgress periodically logs the status of active catchup operations
func (s *EventCatchupService) monitorCatchupProgress(ctx context.Context) {
	ticker := time.NewTicker(MonitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.catchupMu.Lock()
			activeOps := len(s.activeCatchups)
			var activeList []string
			for op := range s.activeCatchups {
				activeList = append(activeList, op)
			}
			s.catchupMu.Unlock()

			s.logger.Debug().
				Int("active_operations", activeOps).
				Msg("CATCHUP STATUS")
			if activeOps > 0 {
				s.logger.Debug().
					Strs("operations", activeList).
					Msg("Active operations")
			}

			// Log intent service goroutines if available
			for chainID, service := range s.intentServices {
				if service != nil {
					activeGoroutines := service.ActiveGoroutines()
					s.logger.Debug().
						Uint64(logging.FieldChain, chainID).
						Int32("active_goroutines", activeGoroutines).
						Msg("Intent service")
				}
			}
		case <-ctx.Done():
			s.logger.Debug().Msg("Stopping catchup monitoring")
			return
		}
	}
}

// trackCatchupOperation adds an operation to the active operations map
func (s *EventCatchupService) trackCatchupOperation(operation string) {
	s.catchupMu.Lock()
	defer s.catchupMu.Unlock()
	s.activeCatchups[operation] = true
	s.logger.Debug().Str("operation", operation).Msg("Starting catchup operation")
}

// untrackCatchupOperation removes an operation from the active operations map
func (s *EventCatchupService) untrackCatchupOperation(operation string) {
	s.catchupMu.Lock()
	defer s.catchupMu.Unlock()
	delete(s.activeCatchups, operation)
	s.logger.Debug().Str("operation", operation).Msg("Completed catchup operation")
}

// runIntentCatchup handles the intent catchup process with proper error handling and timeouts
func (s *EventCatchupService) runIntentCatchup(ctx context.Context, cfg *config.Config, currentBlocks map[uint64]uint64) error {
	// Create a context with a global timeout
	catchupCtx, catchupCancel := context.WithTimeout(ctx, CatchupOperationTimeout)
	defer catchupCancel()

	var intentWg sync.WaitGroup
	intentErrors := make(chan error, len(s.intentServices))

	// Track number of chains that need catchup
	chainsToProcess := 0

	// Start intent catch-up for all chains in parallel
	for chainID, intentService := range s.intentServices {
		lastBlock := s.intentProgress[chainID]
		currentBlock := currentBlocks[chainID]
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		if lastBlock >= currentBlock {
			s.logger.Info().
				Uint64(logging.FieldChain, chainID).
				Msg("No missed events to process")
			continue
		}

		chainsToProcess++
		intentWg.Add(1)

		// Use a descriptive operation name
		opName := fmt.Sprintf("intent_catchup_chain_%d", chainID)
		s.trackCatchupOperation(opName)

		go func(chainID uint64, intentService *IntentService, lastBlock, currentBlock uint64, opName string) {
			defer intentWg.Done()
			defer s.untrackCatchupOperation(opName)

			s.logger.Info().
				Uint64(logging.FieldChain, chainID).
				Uint64("from_block", lastBlock+1).
				Uint64("to_block", currentBlock).
				Msg("Starting intent event catch-up")

			// Create a timeout context for this specific chain's catchup
			chainCtx, chainCancel := context.WithTimeout(catchupCtx, CatchupOperationTimeout)
			defer chainCancel()

			if err := s.catchUpOnIntentEvents(chainCtx, intentService, contractAddress, lastBlock, currentBlock, opName); err != nil {
				intentErrors <- fmt.Errorf("failed to catch up on intent events for chain %d: %v", chainID, err)
				s.logger.Error().
					Uint64(logging.FieldChain, chainID).
					Err(err).
					Msg("Intent catchup failed")
				return
			}

			// Update progress
			s.UpdateIntentProgress(chainID, currentBlock)
			s.logger.Info().
				Uint64(logging.FieldChain, chainID).
				Msg("Completed intent event catch-up")
		}(chainID, intentService, lastBlock, currentBlock, opName)
	}

	// If there are no chains to process, we can return early
	if chainsToProcess == 0 {
		s.logger.Debug().Msg("No intent catchup needed for any chain")
		return nil
	}

	// Create a separate goroutine to wait for all work to complete and close the error channel
	done := make(chan struct{})
	go func() {
		intentWg.Wait()
		close(intentErrors)
		close(done)
	}()

	// Wait for either completion or timeout
	var errs []error
	select {
	case <-done:
		// Process any errors that were collected
		for err := range intentErrors {
			if err != nil {
				errs = append(errs, err)
				s.logger.Error().Err(err).Msg("Intent catchup error")
			}
		}
	case <-catchupCtx.Done():
		return fmt.Errorf("intent catchup timed out after %v", CatchupOperationTimeout)
	}

	// Return combined errors if any
	if len(errs) > 0 {
		return fmt.Errorf("intent catchup completed with %d errors", len(errs))
	}

	return nil
}

// runFulfillmentCatchup handles the fulfillment catchup process with proper error handling and timeouts
func (s *EventCatchupService) runFulfillmentCatchup(ctx context.Context, cfg *config.Config, currentBlocks map[uint64]uint64) error {
	// Create a context with a global timeout
	catchupCtx, catchupCancel := context.WithTimeout(ctx, CatchupOperationTimeout)
	defer catchupCancel()

	// Initialize progress tracking for all chains
	s.mu.Lock()
	for chainID := range s.fulfillmentServices {
		blockCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		lastBlock, err := s.db.GetLastProcessedBlock(blockCtx, chainID)
		cancel()

		if err != nil {
			s.mu.Unlock()
			return fmt.Errorf("failed to get last processed block for chain %d: %v", chainID, err)
		}
		if lastBlock < cfg.ChainConfigs[chainID].DefaultBlock {
			lastBlock = cfg.ChainConfigs[chainID].DefaultBlock
		}
		s.fulfillmentProgress[chainID] = lastBlock
	}
	s.mu.Unlock()

	var fulfillmentWg sync.WaitGroup
	fulfillmentErrors := make(chan error, len(s.fulfillmentServices))

	// Track number of chains that need catchup
	chainsToProcess := 0

	for chainID, fulfillmentService := range s.fulfillmentServices {
		lastBlock := s.fulfillmentProgress[chainID]
		currentBlock := currentBlocks[chainID]
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		if lastBlock >= currentBlock {
			s.logger.Debug().
				Uint64(logging.FieldChain, chainID).
				Msg("No missed fulfillment events to process")
			continue
		}

		chainsToProcess++
		fulfillmentWg.Add(1)

		// Use a descriptive operation name
		opName := fmt.Sprintf("fulfillment_catchup_chain_%d", chainID)
		s.trackCatchupOperation(opName)

		go func(chainID uint64, fulfillmentService *FulfillmentService, lastBlock, currentBlock uint64, opName string) {
			defer fulfillmentWg.Done()
			defer s.untrackCatchupOperation(opName)

			s.logger.Info().
				Uint64(logging.FieldChain, chainID).
				Uint64("from_block", lastBlock+1).
				Uint64("to_block", currentBlock).
				Msg("Starting fulfillment event catch-up")

			// Create a timeout context for this specific chain's catchup
			chainCtx, chainCancel := context.WithTimeout(catchupCtx, CatchupOperationTimeout)
			defer chainCancel()

			if err := s.catchUpOnFulfillmentEvents(chainCtx, fulfillmentService, contractAddress, lastBlock, currentBlock, opName); err != nil {
				fulfillmentErrors <- fmt.Errorf("failed to catch up on fulfillment events for chain %d: %v", chainID, err)
				s.logger.Error().
					Uint64(logging.FieldChain, chainID).
					Err(err).
					Msg("Fulfillment catchup failed")
				return
			}

			// Update progress
			s.UpdateFulfillmentProgress(chainID, currentBlock)
			s.logger.Info().
				Uint64(logging.FieldChain, chainID).
				Msg("Completed fulfillment event catch-up")
		}(chainID, fulfillmentService, lastBlock, currentBlock, opName)
	}

	// If there are no chains to process, we can return early
	if chainsToProcess == 0 {
		s.logger.Debug().Msg("No fulfillment catchup needed for any chain")
		return nil
	}

	// Create a separate goroutine to wait for all work to complete and close the error channel
	done := make(chan struct{})
	go func() {
		fulfillmentWg.Wait()
		close(fulfillmentErrors)
		close(done)
	}()

	// Wait for either completion or timeout
	var errs []error
	select {
	case <-done:
		// Process any errors that were collected
		for err := range fulfillmentErrors {
			if err != nil {
				errs = append(errs, err)
				s.logger.Error().Err(err).Msg("Fulfillment catchup error")
			}
		}
	case <-catchupCtx.Done():
		return fmt.Errorf("fulfillment catchup timed out after %v", CatchupOperationTimeout)
	}

	// Return combined errors if any
	if len(errs) > 0 {
		return fmt.Errorf("fulfillment catchup completed with %d errors", len(errs))
	}

	return nil
}

// runSettlementCatchup handles the settlement catchup process with proper error handling and timeouts
// TODO: lot of duplicated logic among these catchup functions, check for factorization
func (s *EventCatchupService) runSettlementCatchup(ctx context.Context, cfg *config.Config, currentBlocks map[uint64]uint64) error {
	// Create a context with a global timeout
	catchupCtx, catchupCancel := context.WithTimeout(ctx, CatchupOperationTimeout)
	defer catchupCancel()

	// Initialize progress tracking for all chains
	s.mu.Lock()
	for chainID := range s.settlementServices {
		blockCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		lastBlock, err := s.db.GetLastProcessedBlock(blockCtx, chainID)
		cancel()

		if err != nil {
			s.mu.Unlock()
			return fmt.Errorf("failed to get last processed block for chain %d: %v", chainID, err)
		}
		if lastBlock < cfg.ChainConfigs[chainID].DefaultBlock {
			lastBlock = cfg.ChainConfigs[chainID].DefaultBlock
		}
		s.settlementProgress[chainID] = lastBlock
	}
	s.mu.Unlock()

	var settlementWg sync.WaitGroup
	settlementErrors := make(chan error, len(s.settlementServices))

	// Track number of chains that need catchup
	chainsToProcess := 0

	for chainID, settlementService := range s.settlementServices {
		lastBlock := s.settlementProgress[chainID]
		currentBlock := currentBlocks[chainID]
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		if lastBlock >= currentBlock {
			s.logger.Debug().
				Uint64(logging.FieldChain, chainID).
				Msg("No missed settlement events to process")
			continue
		}

		chainsToProcess++
		settlementWg.Add(1)

		// Use a descriptive operation name
		opName := fmt.Sprintf("settlement_catchup_chain_%d", chainID)
		s.trackCatchupOperation(opName)

		go func(chainID uint64, settlementService *SettlementService, lastBlock, currentBlock uint64, opName string) {
			defer settlementWg.Done()
			defer s.untrackCatchupOperation(opName)

			s.logger.Info().
				Uint64(logging.FieldChain, chainID).
				Uint64("from_block", lastBlock+1).
				Uint64("to_block", currentBlock).
				Msg("Starting settlement event catch-up")

			// Create a timeout context for this specific chain's catchup
			chainCtx, chainCancel := context.WithTimeout(catchupCtx, CatchupOperationTimeout)
			defer chainCancel()

			if err := s.catchUpOnSettlementEvents(chainCtx, settlementService, contractAddress, lastBlock, currentBlock, opName); err != nil {
				settlementErrors <- fmt.Errorf("failed to catch up on settlement events for chain %d: %v", chainID, err)
				s.logger.Error().
					Uint64(logging.FieldChain, chainID).
					Err(err).
					Msg("Settlement catchup failed")
				return
			}

			// Update progress
			s.UpdateSettlementProgress(chainID, currentBlock)
			s.logger.Info().
				Uint64(logging.FieldChain, chainID).
				Msg("Completed settlement event catch-up")
		}(chainID, settlementService, lastBlock, currentBlock, opName)
	}

	// If there are no chains to process, we can return early
	if chainsToProcess == 0 {
		s.logger.Debug().Msg("No settlement catchup needed for any chain")
		return nil
	}

	// Create a separate goroutine to wait for all work to complete and close the error channel
	done := make(chan struct{})
	go func() {
		settlementWg.Wait()
		close(settlementErrors)
		close(done)
	}()

	// Wait for either completion or timeout
	var errs []error
	select {
	case <-done:
		// Process any errors that were collected
		for err := range settlementErrors {
			if err != nil {
				errs = append(errs, err)
				s.logger.Error().Err(err).Msg("Settlement catchup error")
			}
		}
	case <-catchupCtx.Done():
		return fmt.Errorf("settlement catchup timed out after %v", CatchupOperationTimeout)
	}

	// Return combined errors if any
	if len(errs) > 0 {
		return fmt.Errorf("settlement catchup completed with %d errors", len(errs))
	}

	return nil
}

// StartLiveEventListeners starts the live event listeners for all services
func (s *EventCatchupService) StartLiveEventListeners(ctx context.Context, cfg *config.Config) error {
	// Start intent listeners with block tracking
	s.logger.Info().Msg("Starting live intent event listeners")
	for chainID, intentService := range s.intentServices {
		chainID := chainID // Create a copy of the loop variable for the closure
		intentService := intentService

		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)
		s.logger.Info().
			Uint64(logging.FieldChain, chainID).
			Str("contract", contractAddress.Hex()).
			Msg("Starting intent event listener")

		// For live subscriptions, use the last processed block + 1 as the starting point
		// This ensures we don't miss events and don't process duplicates
		var fromBlock uint64
		s.mu.Lock()
		if lastBlock, exists := s.intentProgress[chainID]; exists && lastBlock > 0 {
			fromBlock = lastBlock + 1
			log.Printf("Setting intent listener for chain %d to start from block %d", chainID, fromBlock)
		} else {
			// If we don't have a stored last block, get the current one
			blockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			currentBlock, err := intentService.client.BlockNumber(blockCtx)
			cancel()
			if err != nil {
				s.logger.Warn().
					Uint64(logging.FieldChain, chainID).
					Err(err).
					Msg("Unable to get current block")
			} else {
				fromBlock = currentBlock
				s.logger.Debug().
					Uint64(logging.FieldChain, chainID).
					Uint64("from_block", fromBlock).
					Msg("No stored progress found, setting intent listener to start from current block")
			}
		}
		s.mu.Unlock()

		// Special handling for ZetaChain - use polling instead of subscription
		if chainID == 7000 {
			// Store the initial block to start polling from
			s.mu.Lock()
			s.intentProgress[chainID] = fromBlock
			s.mu.Unlock()

			s.logger.Info().
				Uint64(logging.FieldChain, chainID).
				Uint64("from_block", fromBlock).
				Msg("Setting up polling-based event monitoring for ZetaChain")

			// Start polling goroutine
			go s.pollZetachainEvents(ctx, intentService, contractAddress, cfg.ChainConfigs[chainID].BlockInterval)
			continue
		}

		// Start the intent service's own subscription management
		s.logger.Info().
			Uint64(logging.FieldChain, chainID).
			Msg("Starting intent service subscription through StartListening")
		if err := intentService.StartListening(ctx, contractAddress); err != nil {
			s.logger.Error().
				Uint64(logging.FieldChain, chainID).
				Err(err).
				Msg("Failed to start intent service")
			return fmt.Errorf("failed to start intent service for chain %d: %v", chainID, err)
		}
		s.logger.Info().
			Uint64(logging.FieldChain, chainID).
			Msg("Successfully started intent service subscription")
	}

	// Start fulfillment listeners with similar block tracking
	s.logger.Debug().Msg("Starting live fulfillment event listeners")
	for chainID, fulfillmentService := range s.fulfillmentServices {
		chainID := chainID // Create a copy of the loop variable for the closure
		fulfillmentService := fulfillmentService

		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		// For live subscriptions, use the last processed block + 1 as the starting point
		var fromBlock uint64
		s.mu.Lock()
		if lastBlock, exists := s.fulfillmentProgress[chainID]; exists && lastBlock > 0 {
			fromBlock = lastBlock + 1
			s.logger.Debug().
				Uint64(logging.FieldChain, chainID).
				Uint64("from_block", fromBlock).
				Msg("Setting fulfillment listener to start from block")
		} else {
			// If we don't have a stored last block, get the current one
			blockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			currentBlock, err := fulfillmentService.client.BlockNumber(blockCtx)
			cancel()
			if err != nil {
				s.logger.Warn().
					Uint64(logging.FieldChain, chainID).
					Err(err).
					Msg("Unable to get current block")
			} else {
				fromBlock = currentBlock
				s.logger.Debug().
					Uint64(logging.FieldChain, chainID).
					Uint64("from_block", fromBlock).
					Msg("No stored progress found, setting fulfillment listener to start from current block")
			}
		}
		s.mu.Unlock()

		// Special handling for ZetaChain - use polling instead of subscription
		if chainID == 7000 {
			// Store the initial block to start polling from
			s.mu.Lock()
			s.fulfillmentProgress[chainID] = fromBlock
			s.mu.Unlock()

			s.logger.Debug().
				Uint64(logging.FieldChain, chainID).
				Uint64("from_block", fromBlock).
				Msg("Setting up polling-based fulfillment monitoring for ZetaChain")

			// Start polling goroutine
			go s.pollZetachainFulfillmentEvents(ctx, fulfillmentService, contractAddress, cfg.ChainConfigs[chainID].BlockInterval)
			continue
		}

		fulfillmentQuery := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{fulfillmentService.abi.Events[IntentFulfilledEventName].ID},
			},
		}

		// Set FromBlock explicitly
		if fromBlock > 0 {
			fulfillmentQuery.FromBlock = big.NewInt(int64(fromBlock))
		}

		s.logger.Debug().
			Uint64(logging.FieldChain, chainID).
			Interface("from_block", fulfillmentQuery.FromBlock).
			Str("address", contractAddress.Hex()).
			Str("topic", fulfillmentQuery.Topics[0][0].Hex()).
			Msg("Fulfillment subscription filter")

		fulfillmentLogs := make(chan types.Log)
		// Use the parent context for the subscription
		fulfillmentSub, err := fulfillmentService.client.SubscribeFilterLogs(ctx, fulfillmentQuery, fulfillmentLogs)
		if err != nil {
			return fmt.Errorf("failed to subscribe to fulfillment logs for chain %d: %v", chainID, err)
		}

		// Add monitoring goroutine for subscription errors
		go func() {
			for {
				select {
				case err := <-fulfillmentSub.Err():
					if err != nil {
						s.logger.Error().
							Uint64(logging.FieldChain, chainID).
							Err(err).
							Msg("Fulfillment subscription encountered an error")
						// Try to resubscribe
						resubCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
						newSub, resubErr := fulfillmentService.client.SubscribeFilterLogs(resubCtx, fulfillmentQuery, fulfillmentLogs)
						cancel()

						if resubErr != nil {
							s.logger.Error().
								Uint64(logging.FieldChain, chainID).
								Err(resubErr).
								Msg("CRITICAL: Failed to resubscribe fulfillment listener")
						} else {
							fulfillmentSub = newSub
							s.logger.Info().
								Uint64(logging.FieldChain, chainID).
								Msg("Successfully resubscribed fulfillment listener")
						}
					}
				case <-ctx.Done():
					s.logger.Debug().
						Uint64(logging.FieldChain, chainID).
						Msg("Fulfillment subscription monitor shutting down")
					return
				}
			}
		}()

		go fulfillmentService.processEventLogs(ctx, fulfillmentSub, fulfillmentLogs, contractAddress.Hex())
	}

	// Start settlement listeners with similar block tracking
	s.logger.Info().Msg("Starting live settlement event listeners")
	for chainID, settlementService := range s.settlementServices {
		chainID := chainID // Create a copy of the loop variable for the closure
		settlementService := settlementService

		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		// For live subscriptions, use the last processed block + 1 as the starting point
		var fromBlock uint64
		s.mu.Lock()
		if lastBlock, exists := s.settlementProgress[chainID]; exists && lastBlock > 0 {
			fromBlock = lastBlock + 1
			s.logger.Debug().
				Uint64(logging.FieldChain, chainID).
				Uint64("from_block", fromBlock).
				Msg("Setting settlement listener to start from block")
		} else {
			// If we don't have a stored last block, get the current one
			blockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			currentBlock, err := settlementService.client.BlockNumber(blockCtx)
			cancel()
			if err != nil {
				s.logger.Warn().
					Uint64(logging.FieldChain, chainID).
					Err(err).
					Msg("Unable to get current block")
			} else {
				fromBlock = currentBlock
				s.logger.Debug().
					Uint64(logging.FieldChain, chainID).
					Uint64("from_block", fromBlock).
					Msg("No stored progress found, setting settlement listener to start from current block")
			}
		}
		s.mu.Unlock()

		// Special handling for ZetaChain - use polling instead of subscription
		if chainID == 7000 {
			// Store the initial block to start polling from
			s.mu.Lock()
			s.settlementProgress[chainID] = fromBlock
			s.mu.Unlock()

			s.logger.Info().
				Uint64(logging.FieldChain, chainID).
				Uint64("from_block", fromBlock).
				Msg("Setting up polling-based settlement monitoring for ZetaChain starting from block")

			// Start polling goroutine
			go s.pollZetachainSettlementEvents(ctx, settlementService, contractAddress, cfg.ChainConfigs[chainID].BlockInterval)
			continue
		}

		settlementQuery := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{settlementService.abi.Events[IntentSettledEventName].ID},
			},
		}

		// Set FromBlock explicitly
		if fromBlock > 0 {
			settlementQuery.FromBlock = big.NewInt(int64(fromBlock))
		}

		s.logger.Info().
			Uint64(logging.FieldChain, chainID).
			Interface("from_block", settlementQuery.FromBlock).
			Str("contract", contractAddress.Hex()).
			Str("topic", settlementQuery.Topics[0][0].Hex()).
			Msg("Settlement subscription filter")

		settlementLogs := make(chan types.Log)
		// Use the parent context for the subscription
		settlementSub, err := settlementService.client.SubscribeFilterLogs(ctx, settlementQuery, settlementLogs)
		if err != nil {
			return fmt.Errorf("failed to subscribe to settlement logs for chain %d: %v", chainID, err)
		}

		// Add monitoring goroutine for subscription errors
		go func() {
			for {
				select {
				case err := <-settlementSub.Err():
					if err != nil {
						s.logger.Error().
							Uint64(logging.FieldChain, chainID).
							Err(err).
							Msg("Settlement subscription encountered an error")
						// Try to resubscribe
						resubCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
						newSub, resubErr := settlementService.client.SubscribeFilterLogs(resubCtx, settlementQuery, settlementLogs)
						cancel()

						if resubErr != nil {
							s.logger.Error().
								Uint64(logging.FieldChain, chainID).
								Err(resubErr).
								Msg("CRITICAL: Failed to resubscribe settlement listener")
						} else {
							settlementSub = newSub
							s.logger.Debug().
								Uint64(logging.FieldChain, chainID).
								Msg("Successfully resubscribed settlement listener")
						}
					}
				case <-ctx.Done():
					s.logger.Debug().
						Uint64(logging.FieldChain, chainID).
						Msg("Settlement subscription monitor shutting down")
					return
				}
			}
		}()

		go settlementService.processEventLogs(ctx, settlementSub, settlementLogs, contractAddress.Hex(), contractAddress)
	}

	s.logger.Info().Msg("All live event listeners started successfully")
	return nil
}

// UpdateIntentProgress updates the progress of an intent service
func (s *EventCatchupService) UpdateIntentProgress(chainID, blockNumber uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.intentProgress[chainID] = blockNumber
}

// UpdateFulfillmentProgress updates the progress of a fulfillment service
func (s *EventCatchupService) UpdateFulfillmentProgress(chainID, blockNumber uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fulfillmentProgress[chainID] = blockNumber
}

// UpdateSettlementProgress updates the progress of a settlement service
func (s *EventCatchupService) UpdateSettlementProgress(chainID, blockNumber uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settlementProgress[chainID] = blockNumber
}

// hasEventsInBlockRange needs to check for both standard and call event signatures
func hasEventsInBlockRange(ctx context.Context, client *ethclient.Client, contractAddress common.Address, eventSigs []common.Hash, startBlock, endBlock uint64) (bool, error) {
	// Skip if range is too small
	if endBlock <= startBlock {
		return false, nil
	}

	// Sample a few blocks to check if they have events
	// The strategy is to check blocks at regular intervals in the range
	sampleCount := 3
	if endBlock-startBlock <= uint64(sampleCount) {
		sampleCount = 1 // If range is small, just check one block
	}

	step := (endBlock - startBlock) / uint64(sampleCount)
	if step == 0 {
		step = 1
	}

	// Use a timeout for each sample block check
	scanCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for i := uint64(0); i < uint64(sampleCount); i++ {
		blockNum := startBlock + (i * step)
		if blockNum > endBlock {
			blockNum = endBlock
		}

		// For each event signature, check if there are any logs
		for _, eventSig := range eventSigs {
			// Create a query for just this block
			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(int64(blockNum)),
				ToBlock:   big.NewInt(int64(blockNum)),
				Addresses: []common.Address{contractAddress},
				Topics: [][]common.Hash{
					{eventSig},
				},
			}

			// Check for logs in this block
			logs, err := client.FilterLogs(scanCtx, query)
			if err != nil {
				return true, nil // Error getting logs, so process the range to be safe
			}

			// If we found any logs in the sample block, there might be more in the range
			if len(logs) > 0 {
				return true, nil
			}
		}
	}

	return false, nil
}

// Modify the intent events function to use bloom filtering on Ethereum
func (s *EventCatchupService) catchUpOnIntentEvents(ctx context.Context, intentService *IntentService, contractAddress common.Address, fromBlock, toBlock uint64, opName string) error {
	// Use a chain-specific block range - smaller for Ethereum mainnet
	var maxBlockRange uint64
	if intentService.chainID == 1 { // Ethereum mainnet
		maxBlockRange = EthereumMaxBlockRange
		s.logger.Debug().
			Str("operation", opName).
			Uint64("max_block_range", maxBlockRange).
			Msg("Using smaller block range for Ethereum mainnet")
	} else {
		maxBlockRange = DefaultMaxBlockRange
	}

	// Prepare event signatures for bloom filtering
	eventSigs := []common.Hash{
		intentService.abi.Events[IntentInitiatedEventName].ID,
		intentService.abi.Events[IntentInitiatedWithCallEventName].ID,
	}

	// Process in chunks to avoid RPC provider limitations
	for chunkStart := fromBlock; chunkStart < toBlock; chunkStart += maxBlockRange {
		// Check for context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		chunkEnd := chunkStart + maxBlockRange
		if chunkEnd > toBlock {
			chunkEnd = toBlock
		}

		// For Ethereum, do a quick check if this range might have events
		// This can dramatically speed up scanning large empty ranges
		if intentService.chainID == 1 { // Ethereum mainnet
			hasEvents, err := hasEventsInBlockRange(ctx, intentService.client, contractAddress,
				eventSigs, chunkStart+1, chunkEnd)
			if err != nil {
				s.logger.Debug().
					Str("operation", opName).
					Err(err).
					Msg("Error in bloom check, will process range")
			} else if !hasEvents {
				// Skip this chunk as it likely has no events
				s.logger.Debug().
					Str("operation", opName).
					Uint64("from_block", chunkStart+1).
					Uint64("to_block", chunkEnd).
					Msg("Fast-forwarding through block range (no events detected)")

				// Update progress even though we're skipping
				s.UpdateIntentProgress(intentService.chainID, chunkEnd)
				continue
			}
		}

		// Track this chunk processing
		chunkOpName := fmt.Sprintf("%s_chunk_%d_%d", opName, chunkStart, chunkStart+maxBlockRange)
		s.trackCatchupOperation(chunkOpName)

		// Create a context with timeout for this chunk
		chunkCtx, chunkCancel := context.WithTimeout(ctx, BlockRangeProcessTimeout)

		err := func() error {
			defer chunkCancel()
			defer s.untrackCatchupOperation(chunkOpName)

			s.logger.Debug().
				Str("operation", opName).
				Uint64("from_block", chunkStart+1).
				Uint64("to_block", chunkEnd).
				Msg("Fetching intent logs for blocks")

			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(int64(chunkStart + 1)),
				ToBlock:   big.NewInt(int64(chunkEnd)),
				Addresses: []common.Address{contractAddress},
				Topics: [][]common.Hash{
					{
						intentService.abi.Events[IntentInitiatedEventName].ID,
						intentService.abi.Events[IntentInitiatedWithCallEventName].ID,
					},
				},
			}

			// Add timeout for FilterLogs
			filterCtx, filterCancel := context.WithTimeout(chunkCtx, FilterLogsTimeout)
			logs, err := intentService.client.FilterLogs(filterCtx, query)
			filterCancel()

			if err != nil {
				return fmt.Errorf("failed to fetch intent logs for range %d-%d: %v", chunkStart+1, chunkEnd, err)
			}

			s.logger.Debug().
				Str("operation", opName).
				Int("log_count", len(logs)).
				Uint64("from_block", chunkStart+1).
				Uint64("to_block", chunkEnd).
				Msg("Processing logs from blocks")

			// Process logs in batches to report progress
			batchSize := 100
			for i := 0; i < len(logs); i += batchSize {
				// Check for context cancellation
				if chunkCtx.Err() != nil {
					return chunkCtx.Err()
				}

				end := i + batchSize
				if end > len(logs) {
					end = len(logs)
				}

				batch := logs[i:end]

				// Create a context with timeout for this batch
				batchCtx, batchCancel := context.WithTimeout(chunkCtx, LogBatchProcessTimeout)

				batchErr := func() error {
					defer batchCancel()

					for j, txlog := range batch {
						// Check for context cancellation
						if batchCtx.Err() != nil {
							return batchCtx.Err()
						}

						s.logger.Debug().
							Str("operation", opName).
							Int("log_index", i+j+1).
							Int("total_logs", len(logs)).
							Uint64("block_number", txlog.BlockNumber).
							Str("tx_hash", txlog.TxHash.Hex()).
							Msg("Processing intent log")

						// Extract intent ID from the log
						intentID := txlog.Topics[1].Hex()

						// Check if intent already exists
						getIntentCtx, cancel := context.WithTimeout(batchCtx, 10*time.Second)
						existingIntent, err := s.db.GetIntent(getIntentCtx, intentID)
						cancel()

						if err != nil && !strings.Contains(err.Error(), "not found") {
							return fmt.Errorf("failed to check for existing intent: %v", err)
						}

						// Skip if intent already exists
						if existingIntent != nil {
							s.logger.Debug().
								Str("operation", opName).
								Str("intent_id", intentID).
								Msg("Skipping existing intent")
							continue
						}

						// Process log with timeout
						processCtx, processCancel := context.WithTimeout(batchCtx, 20*time.Second)
						err = intentService.processLog(processCtx, txlog)
						processCancel()

						if err != nil {
							// Skip if intent already exists
							if strings.Contains(err.Error(), "duplicate key") {
								s.logger.Debug().
									Str("operation", opName).
									Str("intent_id", intentID).
									Msg("Skipping duplicate intent")
								continue
							}
							return fmt.Errorf("failed to process intent log: %v", err)
						}
					}
					return nil
				}()

				if batchErr != nil {
					return batchErr
				}

				// Update progress after each batch
				if len(batch) > 0 {
					lastBlock := batch[len(batch)-1].BlockNumber
					s.UpdateIntentProgress(intentService.chainID, lastBlock)
					s.logger.Debug().
						Str("operation", opName).
						Uint64(logging.FieldChain, intentService.chainID).
						Uint64("block_number", lastBlock).
						Msg("Updated progress")
				}
			}

			// Update progress after processing each chunk
			s.UpdateIntentProgress(intentService.chainID, chunkEnd)

			// Persist progress to the database after each chunk
			dbUpdateCtx, dbUpdateCancel := context.WithTimeout(ctx, 10*time.Second)
			dbErr := s.db.UpdateLastProcessedBlock(dbUpdateCtx, intentService.chainID, chunkEnd)
			dbUpdateCancel()
			if dbErr != nil {
				s.logger.Debug().
					Str("operation", opName).
					Err(dbErr).
					Msg("Warning: Failed to persist progress to DB")
				// Continue processing even if DB update fails
			} else {
				s.logger.Debug().
					Str("operation", opName).
					Uint64(logging.FieldChain, intentService.chainID).
					Uint64("block_number", chunkEnd).
					Msg("Persisted progress to DB")
			}

			s.logger.Debug().
				Str("operation", opName).
				Uint64("from_block", chunkStart+1).
				Uint64("to_block", chunkEnd).
				Msg("Completed processing intent logs for blocks")

			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}

// catchUpOnFulfillmentEvents processes missed fulfillment events for a specific chain
func (s *EventCatchupService) catchUpOnFulfillmentEvents(ctx context.Context, fulfillmentService *FulfillmentService, contractAddress common.Address, fromBlock, toBlock uint64, opName string) error {
	// Use a chain-specific block range - smaller for Ethereum mainnet
	var maxBlockRange uint64
	if fulfillmentService.chainID == 1 { // Ethereum mainnet
		maxBlockRange = EthereumMaxBlockRange
		s.logger.Debug().
			Str("operation", opName).
			Uint64("max_block_range", maxBlockRange).
			Msg("Using smaller block range for Ethereum mainnet")
	} else {
		maxBlockRange = DefaultMaxBlockRange
	}

	// Process in chunks to avoid RPC provider limitations
	for chunkStart := fromBlock; chunkStart < toBlock; chunkStart += maxBlockRange {
		// Check for context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Track this chunk processing
		chunkOpName := fmt.Sprintf("%s_chunk_%d_%d", opName, chunkStart, chunkStart+maxBlockRange)
		s.trackCatchupOperation(chunkOpName)

		// Create a context with timeout for this chunk
		chunkCtx, chunkCancel := context.WithTimeout(ctx, BlockRangeProcessTimeout)

		err := func() error {
			defer chunkCancel()
			defer s.untrackCatchupOperation(chunkOpName)

			chunkEnd := chunkStart + maxBlockRange
			if chunkEnd > toBlock {
				chunkEnd = toBlock
			}

			s.logger.Debug().
				Str("operation", opName).
				Uint64("from_block", chunkStart+1).
				Uint64("to_block", chunkEnd).
				Msg("Fetching fulfillment logs for blocks")

			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(int64(chunkStart + 1)),
				ToBlock:   big.NewInt(int64(chunkEnd)),
				Addresses: []common.Address{contractAddress},
				Topics: [][]common.Hash{
					{
						fulfillmentService.abi.Events[IntentFulfilledEventName].ID,
						fulfillmentService.abi.Events[IntentFulfilledWithCallEventName].ID,
					},
				},
			}

			// Add timeout for FilterLogs
			filterCtx, filterCancel := context.WithTimeout(chunkCtx, FilterLogsTimeout)
			logs, err := fulfillmentService.client.FilterLogs(filterCtx, query)
			filterCancel()

			if err != nil {
				return fmt.Errorf("failed to fetch fulfillment logs for range %d-%d: %v", chunkStart+1, chunkEnd, err)
			}

			s.logger.Debug().
				Str("operation", opName).
				Int("log_count", len(logs)).
				Uint64("from_block", chunkStart+1).
				Uint64("to_block", chunkEnd).
				Msg("Processing logs from blocks")

			// Process logs in batches to report progress
			batchSize := 100
			for i := 0; i < len(logs); i += batchSize {
				// Check for context cancellation
				if chunkCtx.Err() != nil {
					return chunkCtx.Err()
				}

				end := i + batchSize
				if end > len(logs) {
					end = len(logs)
				}

				batch := logs[i:end]

				// Create a context with timeout for this batch
				batchCtx, batchCancel := context.WithTimeout(chunkCtx, LogBatchProcessTimeout)

				batchErr := func() error {
					defer batchCancel()

					for j, txlog := range batch {
						// Check for context cancellation
						if batchCtx.Err() != nil {
							return batchCtx.Err()
						}

						s.logger.Debug().
							Str("operation", opName).
							Int("log_index", i+j+1).
							Int("total_logs", len(logs)).
							Uint64("block_number", txlog.BlockNumber).
							Str("tx_hash", txlog.TxHash.Hex()).
							Msg("Processing fulfillment log")

						// Extract intent ID from the log
						intentID := txlog.Topics[1].Hex()

						// Check if intent exists (fulfillments need an intent)
						getIntentCtx, cancel := context.WithTimeout(batchCtx, 10*time.Second)
						_, err := s.db.GetIntent(getIntentCtx, intentID)
						cancel()

						if err != nil {
							if strings.Contains(err.Error(), "not found") {
								s.logger.Debug().
									Str("operation", opName).
									Str("intent_id", intentID).
									Msg("Skipping fulfillment for non-existent intent")
								continue
							}
							s.logger.Debug().
								Str("operation", opName).
								Err(err).
								Msg("Failed to check for existing intent")
							continue
						}

						// Process log with timeout
						processCtx, processCancel := context.WithTimeout(batchCtx, 20*time.Second)
						err = fulfillmentService.processLog(processCtx, txlog)
						processCancel()

						if err != nil {
							// Skip if fulfillment already exists
							if strings.Contains(err.Error(), "duplicate key") {
								s.logger.Debug().
									Str("operation", opName).
									Str("intent_id", intentID).
									Msg("Skipping duplicate fulfillment")
								continue
							}
							return fmt.Errorf("failed to process fulfillment log: %v", err)
						}
					}
					return nil
				}()

				if batchErr != nil {
					return batchErr
				}

				// Update progress after each batch
				if len(batch) > 0 {
					lastBlock := batch[len(batch)-1].BlockNumber
					s.UpdateFulfillmentProgress(fulfillmentService.chainID, lastBlock)
					s.logger.Debug().
						Str("operation", opName).
						Uint64(logging.FieldChain, fulfillmentService.chainID).
						Uint64("block_number", lastBlock).
						Msg("Updated progress")
				}
			}

			// Update progress after processing each chunk
			s.UpdateFulfillmentProgress(fulfillmentService.chainID, chunkEnd)

			// Persist progress to the database after each chunk
			dbUpdateCtx, dbUpdateCancel := context.WithTimeout(ctx, 10*time.Second)
			dbErr := s.db.UpdateLastProcessedBlock(dbUpdateCtx, fulfillmentService.chainID, chunkEnd)
			dbUpdateCancel()
			if dbErr != nil {
				s.logger.Debug().
					Str("operation", opName).
					Err(dbErr).
					Msg("Warning: Failed to persist progress to DB")
				// Continue processing even if DB update fails
			} else {
				s.logger.Debug().
					Str("operation", opName).
					Uint64(logging.FieldChain, fulfillmentService.chainID).
					Uint64("block_number", chunkEnd).
					Msg("Persisted progress to DB")
			}

			s.logger.Debug().
				Str("operation", opName).
				Uint64("from_block", chunkStart+1).
				Uint64("to_block", chunkEnd).
				Msg("Completed processing fulfillment logs for blocks")

			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}

// catchUpOnSettlementEvents processes missed settlement events for a specific chain
func (s *EventCatchupService) catchUpOnSettlementEvents(ctx context.Context, settlementService *SettlementService, contractAddress common.Address, fromBlock, toBlock uint64, opName string) error {
	// Use a chain-specific block range - smaller for Ethereum mainnet
	var maxBlockRange uint64
	if settlementService.chainID == 1 { // Ethereum mainnet
		maxBlockRange = EthereumMaxBlockRange
		s.logger.Debug().
			Str("operation", opName).
			Uint64("max_block_range", maxBlockRange).
			Msg("Using smaller block range for Ethereum mainnet")
	} else {
		maxBlockRange = DefaultMaxBlockRange
	}

	// Process in chunks to avoid RPC provider limitations
	for chunkStart := fromBlock; chunkStart < toBlock; chunkStart += maxBlockRange {
		// Check for context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Track this chunk processing
		chunkOpName := fmt.Sprintf("%s_chunk_%d_%d", opName, chunkStart, chunkStart+maxBlockRange)
		s.trackCatchupOperation(chunkOpName)

		// Create a context with timeout for this chunk
		chunkCtx, chunkCancel := context.WithTimeout(ctx, BlockRangeProcessTimeout)

		err := func() error {
			defer chunkCancel()
			defer s.untrackCatchupOperation(chunkOpName)

			chunkEnd := chunkStart + maxBlockRange
			if chunkEnd > toBlock {
				chunkEnd = toBlock
			}

			s.logger.Debug().
				Str("operation", opName).
				Uint64("from_block", chunkStart+1).
				Uint64("to_block", chunkEnd).
				Msg("Fetching settlement logs for blocks")

			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(int64(chunkStart + 1)),
				ToBlock:   big.NewInt(int64(chunkEnd)),
				Addresses: []common.Address{contractAddress},
				Topics: [][]common.Hash{
					{
						settlementService.abi.Events[IntentSettledEventName].ID,
						settlementService.abi.Events[IntentSettledWithCallEventName].ID,
					},
				},
			}

			// Add timeout for FilterLogs
			filterCtx, filterCancel := context.WithTimeout(chunkCtx, FilterLogsTimeout)
			logs, err := settlementService.client.FilterLogs(filterCtx, query)
			filterCancel()

			if err != nil {
				return fmt.Errorf("failed to fetch settlement logs for range %d-%d: %v", chunkStart+1, chunkEnd, err)
			}

			s.logger.Debug().
				Str("operation", opName).
				Int("log_count", len(logs)).
				Uint64("from_block", chunkStart+1).
				Uint64("to_block", chunkEnd).
				Msg("Processing logs from blocks")

			// Process logs in batches to report progress
			batchSize := 100
			for i := 0; i < len(logs); i += batchSize {
				// Check for context cancellation
				if chunkCtx.Err() != nil {
					return chunkCtx.Err()
				}

				end := i + batchSize
				if end > len(logs) {
					end = len(logs)
				}

				batch := logs[i:end]

				// Create a context with timeout for this batch
				batchCtx, batchCancel := context.WithTimeout(chunkCtx, LogBatchProcessTimeout)

				batchErr := func() error {
					defer batchCancel()

					for j, txlog := range batch {
						// Check for context cancellation
						if batchCtx.Err() != nil {
							return batchCtx.Err()
						}

						s.logger.Debug().
							Str("operation", opName).
							Int("log_index", i+j+1).
							Int("total_logs", len(logs)).
							Uint64("block_number", txlog.BlockNumber).
							Str("tx_hash", txlog.TxHash.Hex()).
							Msg("Processing settlement log")

						// Extract intent ID from the log
						intentID := txlog.Topics[1].Hex()

						// Check if intent exists (settlements need an intent)
						getIntentCtx, cancel := context.WithTimeout(batchCtx, 10*time.Second)
						_, err := s.db.GetIntent(getIntentCtx, intentID)
						cancel()

						if err != nil {
							if strings.Contains(err.Error(), "not found") {
								s.logger.Debug().
									Str("operation", opName).
									Str("intent_id", intentID).
									Msg("Skipping settlement for non-existent intent")
								continue
							}
							return fmt.Errorf("failed to check for existing intent: %v", err)
						}

						// Process log with timeout
						processCtx, processCancel := context.WithTimeout(batchCtx, 20*time.Second)
						err = settlementService.processLog(processCtx, txlog)
						processCancel()

						if err != nil {
							// Skip if settlement already exists
							if strings.Contains(err.Error(), "duplicate key") {
								s.logger.Debug().
									Str("operation", opName).
									Str("intent_id", intentID).
									Msg("Skipping duplicate settlement")
								continue
							}
							return fmt.Errorf("failed to process settlement log: %v", err)
						}
					}
					return nil
				}()

				if batchErr != nil {
					return batchErr
				}

				// Update progress after each batch
				if len(batch) > 0 {
					lastBlock := batch[len(batch)-1].BlockNumber
					s.UpdateSettlementProgress(settlementService.chainID, lastBlock)
					s.logger.Debug().
						Str("operation", opName).
						Uint64(logging.FieldChain, settlementService.chainID).
						Uint64("block_number", lastBlock).
						Msg("Updated progress")
				}
			}

			// Update progress after processing each chunk
			s.UpdateSettlementProgress(settlementService.chainID, chunkEnd)

			// Persist progress to the database after each chunk
			dbUpdateCtx, dbUpdateCancel := context.WithTimeout(ctx, 10*time.Second)
			dbErr := s.db.UpdateLastProcessedBlock(dbUpdateCtx, settlementService.chainID, chunkEnd)
			dbUpdateCancel()
			if dbErr != nil {
				s.logger.Debug().
					Str("operation", opName).
					Err(dbErr).
					Msg("Warning: Failed to persist progress to DB")
				// Continue processing even if DB update fails
			} else {
				s.logger.Debug().
					Str("operation", opName).
					Uint64(logging.FieldChain, settlementService.chainID).
					Uint64("block_number", chunkEnd).
					Msg("Persisted progress to DB")
			}

			s.logger.Debug().
				Str("operation", opName).
				Uint64("from_block", chunkStart+1).
				Uint64("to_block", chunkEnd).
				Msg("Completed processing settlement logs for blocks")

			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}

// pollZetachainEvents polls for events on ZetaChain at regular intervals instead of using WebSocket subscription
func (s *EventCatchupService) pollZetachainEvents(ctx context.Context, intentService *IntentService, contractAddress common.Address, blockInterval int64) {
	// Use the generic polling function with health reporting
	s.pollChainEvents(ctx, "intent", 7000, intentService.client, contractAddress,
		[]common.Hash{intentService.abi.Events[IntentInitiatedEventName].ID, intentService.abi.Events[IntentInitiatedWithCallEventName].ID},
		intentService.processLog,
		func(blockNum uint64) { s.UpdateIntentProgress(7000, blockNum) },
		blockInterval,
		intentService) // Pass intent service for health reporting
}

// pollZetachainFulfillmentEvents polls for events on ZetaChain at regular intervals instead of using WebSocket subscription
func (s *EventCatchupService) pollZetachainFulfillmentEvents(ctx context.Context, fulfillmentService *FulfillmentService, contractAddress common.Address, blockInterval int64) {
	// Use the generic polling function
	s.pollChainEvents(ctx, "fulfillment", 7000, fulfillmentService.client, contractAddress,
		[]common.Hash{fulfillmentService.abi.Events[IntentFulfilledEventName].ID, fulfillmentService.abi.Events[IntentFulfilledWithCallEventName].ID},
		fulfillmentService.processLog,
		func(blockNum uint64) { s.UpdateFulfillmentProgress(7000, blockNum) },
		blockInterval,
		nil) // No health reporting for fulfillment services yet
}

// pollZetachainSettlementEvents polls for events on ZetaChain at regular intervals instead of using WebSocket subscription
func (s *EventCatchupService) pollZetachainSettlementEvents(ctx context.Context, settlementService *SettlementService, contractAddress common.Address, blockInterval int64) {
	// Use the generic polling function
	s.pollChainEvents(ctx, "settlement", 7000, settlementService.client, contractAddress,
		[]common.Hash{settlementService.abi.Events[IntentSettledEventName].ID, settlementService.abi.Events[IntentSettledWithCallEventName].ID},
		settlementService.processLog,
		func(blockNum uint64) { s.UpdateSettlementProgress(7000, blockNum) },
		blockInterval,
		nil) // No health reporting for settlement services yet
}

// pollChainEvents is a generic function to poll for blockchain events
// It handles all the common logic for different event types
func (s *EventCatchupService) pollChainEvents(
	ctx context.Context,
	eventType string,
	chainID uint64,
	client *ethclient.Client,
	contractAddress common.Address,
	eventSignatures []common.Hash,
	processLogFunc func(context.Context, types.Log) error,
	updateProgressFunc func(uint64),
	blockInterval int64,
	intentService *IntentService, // Optional: for health reporting (ZetaChain only)
) {
	// Default to checking every 15 seconds if not specified in config
	interval := time.Duration(blockInterval) * time.Second
	if interval < 5*time.Second {
		interval = 15 * time.Second
	}

	maxRetries := 3
	baseRetryDelay := 5 * time.Second

	s.logger.Info().
		Str("event_type", eventType).
		Dur("interval", interval).
		Msg("Starting ZetaChain polling for events")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Track database persistence to avoid doing it every poll
	lastDbUpdateTime := time.Now()
	dbUpdateInterval := 5 * time.Minute

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().
				Str("event_type", eventType).
				Msg("Context cancelled, stopping ZetaChain event polling")
			return
		case <-ticker.C:
			// Get the last processed block
			s.mu.Lock()
			var lastProcessedBlock uint64
			switch eventType {
			case "intent":
				lastProcessedBlock = s.intentProgress[chainID]
			case "fulfillment":
				lastProcessedBlock = s.fulfillmentProgress[chainID]
			case "settlement":
				lastProcessedBlock = s.settlementProgress[chainID]
			}
			s.mu.Unlock()

			// Get current block with retry logic
			var currentBlock uint64
			var err error
			for retry := 0; retry < maxRetries; retry++ {
				blockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				currentBlock, err = client.BlockNumber(blockCtx)
				cancel()

				if err == nil {
					break
				}

				retryDelay := baseRetryDelay * time.Duration(1<<retry)
				s.logger.Error().
					Int("attempt", retry+1).
					Int("max_attempts", maxRetries).
					Err(err).
					Dur("retry_delay", retryDelay).
					Msg("Failed to get current block for ZetaChain")

				select {
				case <-time.After(retryDelay):
					continue
				case <-ctx.Done():
					return
				}
			}

			if err != nil {
				s.logger.Error().
					Int("max_attempts", maxRetries).
					Msg("CRITICAL: Failed to get current block for ZetaChain after retries. Skipping this polling cycle.")
				// Report unhealthy polling if we have an intent service to report to
				if intentService != nil && eventType == "intent" {
					intentService.UpdatePollingHealth(false)
				}
				continue
			}

			// Report healthy polling if we successfully got the current block
			if intentService != nil && eventType == "intent" {
				intentService.UpdatePollingHealth(true)
			}

			// Skip if no new blocks
			if currentBlock <= lastProcessedBlock {
				if time.Since(lastDbUpdateTime) >= dbUpdateInterval {
					// Even if no new blocks, periodically update the DB to ensure we don't lose progress
					dbUpdateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
					if err := s.db.UpdateLastProcessedBlock(dbUpdateCtx, chainID, lastProcessedBlock); err != nil {
						s.logger.Warn().
							Str("event_type", eventType).
							Err(err).
							Msg("Failed to persist progress to DB")
					} else {
						s.logger.Debug().
							Str("event_type", eventType).
							Uint64(logging.FieldChain, chainID).
							Uint64("block_number", lastProcessedBlock).
							Msg("Persisted progress to DB")
					}
					cancel()
					lastDbUpdateTime = time.Now()
				}
				continue
			}

			// Limit the number of blocks we process at once to avoid timeouts
			endBlock := lastProcessedBlock + 5000
			if endBlock > currentBlock {
				endBlock = currentBlock
			}

			s.logger.Debug().
				Str("event_type", eventType).
				Uint64("from_block", lastProcessedBlock+1).
				Uint64("to_block", endBlock).
				Msg("Polling ZetaChain for events")

			// Create query for the block range
			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(int64(lastProcessedBlock + 1)),
				ToBlock:   big.NewInt(int64(endBlock)),
				Addresses: []common.Address{contractAddress},
				Topics: [][]common.Hash{
					eventSignatures,
				},
			}

			// Filter logs with retry logic
			var logs []types.Log
			for retry := 0; retry < maxRetries; retry++ {
				filterCtx, filterCancel := context.WithTimeout(ctx, FilterLogsTimeout)
				logs, err = client.FilterLogs(filterCtx, query)
				filterCancel()

				if err == nil {
					break
				}

				retryDelay := baseRetryDelay * time.Duration(1<<retry)
				s.logger.Error().
					Str("event_type", eventType).
					Int("attempt", retry+1).
					Int("max_attempts", maxRetries).
					Err(err).
					Dur("retry_delay", retryDelay).
					Msg("Failed to filter logs for ZetaChain events")

				select {
				case <-time.After(retryDelay):
					continue
				case <-ctx.Done():
					return
				}
			}

			if err != nil {
				s.logger.Error().
					Str("event_type", eventType).
					Int("max_attempts", maxRetries).
					Msg("CRITICAL: Failed to filter logs for ZetaChain events after retries. Skipping this block range.")
				continue
			}

			// Process logs if any found
			processedCount := 0
			errorCount := 0
			if len(logs) > 0 {
				s.logger.Debug().
					Int("log_count", len(logs)).
					Str("event_type", eventType).
					Uint64("from_block", lastProcessedBlock+1).
					Uint64("to_block", endBlock).
					Msg("Found new events in ZetaChain blocks")

				// Process the logs with individual timeouts
				for _, logEntry := range logs {
					processCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
					err := processLogFunc(processCtx, logEntry)
					cancel()

					if err != nil {
						errorCount++
						if strings.Contains(err.Error(), "duplicate key") {
							// This is expected for duplicates, just log at debug level
							s.logger.Debug().
								Str("event_type", eventType).
								Str("tx_hash", logEntry.TxHash.Hex()).
								Msg("Skipping duplicate event")
						} else {
							s.logger.Error().
								Str("event_type", eventType).
								Err(err).
								Msg("Failed to process ZetaChain log")
						}
					} else {
						processedCount++
					}
				}
				s.logger.Info().
					Int("processed_count", processedCount).
					Int("total_logs", len(logs)).
					Str("event_type", eventType).
					Int("error_count", errorCount).
					Msg("Successfully processed events")
			} else {
				s.logger.Info().
					Str("event_type", eventType).
					Uint64("from_block", lastProcessedBlock+1).
					Uint64("to_block", endBlock).
					Msg("No new events found in ZetaChain blocks")
			}

			// Update the last processed block
			updateProgressFunc(endBlock)

			// Persist progress to the database
			dbUpdateCtx, dbUpdateCancel := context.WithTimeout(ctx, 10*time.Second)
			if err := s.db.UpdateLastProcessedBlock(dbUpdateCtx, chainID, endBlock); err != nil {
				s.logger.Warn().
					Str("event_type", eventType).
					Err(err).
					Msg("Failed to persist progress to DB")
			} else {
				s.logger.Debug().
					Str("event_type", eventType).
					Uint64(logging.FieldChain, chainID).
					Uint64("block_number", endBlock).
					Msg("Persisted progress to DB")
				lastDbUpdateTime = time.Now()
			}
			dbUpdateCancel()
		}
	}
}

// StartSubscriptionSupervisor starts a background goroutine that periodically checks
// if services are still running and restarts them if needed
func (s *EventCatchupService) StartSubscriptionSupervisor(ctx context.Context, cfg *config.Config) {
	s.logger.Info().Msg("Starting subscription supervisor to monitor service health")

	// Run health check every 5 minutes
	healthCheckTicker := time.NewTicker(5 * time.Minute)
	defer healthCheckTicker.Stop()

	// Run full reconnection every 2 hours
	reconnectTicker := time.NewTicker(2 * time.Hour)
	defer reconnectTicker.Stop()

	// Track last full reconnect time
	lastFullReconnect := time.Now()

	// ZetaChain ID constant
	const zetaChainID = uint64(7000)

	for {
		select {
		case <-healthCheckTicker.C:
			s.logger.Info().Msg("Subscription supervisor checking service health...")

			// Check intent services
			for chainID, intentService := range s.intentServices {
				// Skip health check for ZetaChain as it's using polling
				if chainID == zetaChainID {
					s.logger.Info().Msg("ZetaChain intent service using polling mechanism - skipping subscription check")
					continue
				}

				activeGoroutines := intentService.ActiveGoroutines()
				s.logger.Debug().
					Uint64(logging.FieldChain, chainID).
					Int32("active_goroutines", activeGoroutines).
					Msg("Intent service")

				if activeGoroutines == 0 {
					s.logger.Warn().
						Uint64(logging.FieldChain, chainID).
						Msg("Intent service has 0 active goroutines, restarting")
					contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

					// Create a context with timeout for restart
					restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					err := intentService.Restart(restartCtx, contractAddress)
					if err != nil {
						s.logger.Error().
							Uint64(logging.FieldChain, chainID).
							Err(err).
							Msg("Failed to restart intent service")
					} else {
						s.logger.Info().
							Uint64(logging.FieldChain, chainID).
							Msg("RECOVERY: Successfully restarted intent service")
					}
					cancel()
				}
			}

			// Check fulfillment services
			for chainID, fulfillmentService := range s.fulfillmentServices {
				// Skip health check for ZetaChain as it's using polling
				if chainID == zetaChainID {
					s.logger.Info().Msg("ZetaChain fulfillment service using polling mechanism - skipping subscription check")
					continue
				}

				count := fulfillmentService.GetSubscriptionCount()
				s.logger.Info().
					Uint64(logging.FieldChain, chainID).
					Int("active_subscriptions", count).
					Msg("Fulfillment service")

				if count == 0 {
					s.logger.Warn().
						Uint64(logging.FieldChain, chainID).
						Msg("Fulfillment service has no active subscriptions, restarting")
					contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

					// Create a context with timeout for restart
					restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					err := fulfillmentService.Restart(restartCtx, contractAddress)
					if err != nil {
						s.logger.Error().
							Uint64(logging.FieldChain, chainID).
							Err(err).
							Msg("Failed to restart fulfillment service")
					} else {
						s.logger.Info().
							Uint64(logging.FieldChain, chainID).
							Msg("Successfully restarted fulfillment service")
					}
					cancel()
				}
			}

			// Check settlement services
			for chainID, settlementService := range s.settlementServices {
				// Skip health check for ZetaChain as it's using polling
				if chainID == zetaChainID {
					s.logger.Info().Msg("ZetaChain settlement service using polling mechanism - skipping subscription check")
					continue
				}

				count := settlementService.GetSubscriptionCount()
				s.logger.Info().
					Uint64(logging.FieldChain, chainID).
					Int("active_subscriptions", count).
					Msg("Settlement service")

				if count == 0 {
					s.logger.Warn().
						Uint64(logging.FieldChain, chainID).
						Msg("Settlement service has no active subscriptions, restarting")
					contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

					// Create a context with timeout for restart
					restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					err := settlementService.Restart(restartCtx, contractAddress)
					if err != nil {
						s.logger.Error().
							Uint64(logging.FieldChain, chainID).
							Err(err).
							Msg("Failed to restart settlement service")
					} else {
						s.logger.Info().
							Uint64(logging.FieldChain, chainID).
							Msg("Successfully restarted settlement service")
					}
					cancel()
				}
			}

			// Check ZetaChain health by getting block number
			if client, ok := s.intentServices[zetaChainID]; ok && client != nil {
				s.logger.Info().Msg("Checking ZetaChain polling health...")
				blockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				_, err := client.client.BlockNumber(blockCtx)
				cancel()

				if err != nil {
					s.logger.Warn().Err(err).Msg("ZetaChain polling health check failed")
					client.UpdatePollingHealth(false)
				} else {
					s.logger.Info().Msg("ZetaChain polling health check passed")
					client.UpdatePollingHealth(true)
				}
			}

		case <-reconnectTicker.C:
			// Perform a complete refresh of all WebSocket connections every 2 hours
			timeSinceLastReconnect := time.Since(lastFullReconnect)
			s.logger.Info().
				Dur("time_since_last", timeSinceLastReconnect).
				Msg("Performing scheduled full reconnection of all services")
			lastFullReconnect = time.Now()

			// Force reconnect all intent services (except ZetaChain)
			for chainID, intentService := range s.intentServices {
				// Skip ZetaChain as it's using polling
				if chainID == zetaChainID {
					s.logger.Info().Msg("Skipping ZetaChain intent service reconnection (using polling)")
					continue
				}

				s.logger.Info().
					Uint64(logging.FieldChain, chainID).
					Msg("Scheduled reconnect: Restarting intent service")
				contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

				// First, unsubscribe from all existing subscriptions
				intentService.UnsubscribeAll()

				// Create a context with timeout for restart
				restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				err := intentService.Restart(restartCtx, contractAddress)
				if err != nil {
					s.logger.Error().
						Uint64(logging.FieldChain, chainID).
						Err(err).
						Msg("Failed to reconnect intent service")
				} else {
					s.logger.Info().
						Uint64(logging.FieldChain, chainID).
						Msg("Scheduled reconnect: Successfully reconnected intent service")
				}
				cancel()
			}

			// Force reconnect all fulfillment services (except ZetaChain)
			for chainID, fulfillmentService := range s.fulfillmentServices {
				// Skip ZetaChain as it's using polling
				if chainID == zetaChainID {
					s.logger.Info().Msg("Skipping ZetaChain fulfillment service reconnection (using polling)")
					continue
				}

				s.logger.Info().
					Uint64(logging.FieldChain, chainID).
					Msg("Scheduled reconnect: Restarting fulfillment service")
				contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

				// First, unsubscribe from all existing subscriptions
				fulfillmentService.UnsubscribeAll()

				// Create a context with timeout for restart
				restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				err := fulfillmentService.Restart(restartCtx, contractAddress)
				if err != nil {
					s.logger.Error().
						Uint64(logging.FieldChain, chainID).
						Err(err).
						Msg("Failed to reconnect fulfillment service")
				} else {
					s.logger.Info().
						Uint64(logging.FieldChain, chainID).
						Msg("Scheduled reconnect: Successfully reconnected fulfillment service")
				}
				cancel()
			}

			// Force reconnect all settlement services (except ZetaChain)
			for chainID, settlementService := range s.settlementServices {
				// Skip ZetaChain as it's using polling
				if chainID == zetaChainID {
					s.logger.Info().Msg("Skipping ZetaChain settlement service reconnection (using polling)")
					continue
				}

				s.logger.Info().
					Uint64(logging.FieldChain, chainID).
					Msg("Scheduled reconnect: Restarting settlement service")
				contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

				// First, unsubscribe from all existing subscriptions
				settlementService.UnsubscribeAll()

				// Create a context with timeout for restart
				restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				err := settlementService.Restart(restartCtx, contractAddress)
				if err != nil {
					s.logger.Error().
						Uint64(logging.FieldChain, chainID).
						Err(err).
						Msg("Failed to reconnect settlement service")
				} else {
					s.logger.Info().
						Uint64(logging.FieldChain, chainID).
						Msg("Scheduled reconnect: Successfully reconnected settlement service")
				}
				cancel()
			}

		case <-ctx.Done():
			s.logger.Debug().Msg("Subscription supervisor shutting down")
			return
		}
	}
}

// Shutdown gracefully shuts down the service and waits for all goroutines to complete
func (s *EventCatchupService) Shutdown(timeout time.Duration) error {
	s.shutdownMu.Lock()
	if s.isShutdown {
		s.shutdownMu.Unlock()
		return nil // Already shutdown
	}
	s.isShutdown = true
	s.shutdownMu.Unlock()

	s.logger.Info().Msg("Shutting down EventCatchupService...")

	// Cancel the cleanup context to signal all goroutines to stop
	s.cleanupCancel()

	// Wait for all goroutines to complete with timeout
	done := make(chan struct{})
	go func() {
		s.goroutineWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info().Msg("EventCatchupService shutdown completed successfully")
		return nil
	case <-time.After(timeout):
		s.logger.Error().
			Dur("timeout", timeout).
			Msg("EventCatchupService shutdown timed out")
		return fmt.Errorf("shutdown timed out after %v", timeout)
	}
}

// IsShutdown returns whether the service is in shutdown state
func (s *EventCatchupService) IsShutdown() bool {
	s.shutdownMu.RLock()
	defer s.shutdownMu.RUnlock()
	return s.isShutdown
}

// StartGoroutine safely starts a goroutine with proper cleanup tracking
func (s *EventCatchupService) StartGoroutine(name string, fn func()) {
	s.shutdownMu.RLock()
	if s.isShutdown {
		s.shutdownMu.RUnlock()
		s.logger.Debug().
			Str("goroutine_name", name).
			Msg("Cannot start goroutine: service is shutdown")
		return
	}

	s.shutdownMu.RUnlock()

	s.goroutineWg.Add(1)
	atomic.AddInt32(&s.activeGoroutines, 1)

	go func() {
		defer func() {
			s.goroutineWg.Done()
			atomic.AddInt32(&s.activeGoroutines, -1)

			// Recover from panics
			if r := recover(); r != nil {
				s.logger.Error().
					Str("goroutine_name", name).
					Any("panic", r).
					Msg("Panic in goroutine")
			}
		}()

		fn()
	}()
}

// ActiveGoroutines returns the current count of active goroutines
func (s *EventCatchupService) ActiveGoroutines() int32 {
	return atomic.LoadInt32(&s.activeGoroutines)
}
