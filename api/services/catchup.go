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

	"github.com/speedrun-hq/speedrun/api/logger"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/speedrun-hq/speedrun/api/config"
	"github.com/speedrun-hq/speedrun/api/db"
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

	// PeriodicCatchupInterval is how often to run periodic catchup operations
	PeriodicCatchupInterval = 30 * time.Minute

	// PeriodicCatchupTimeout is the maximum time allowed for a periodic catchup operation
	PeriodicCatchupTimeout = 15 * time.Minute

	// PeriodicCatchupLookbackBlocks is how many blocks to look back for missed events
	PeriodicCatchupLookbackBlocks = uint64(1000)
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
	logger              logger.Logger

	// Periodic catchup tracking - tracks the last block processed by periodic catchup
	periodicCatchupProgress map[uint64]uint64 // chainID -> last block processed by periodic catchup
	periodicCatchupMu       sync.Mutex        // Mutex for periodic catchup progress

	// Failure tracking for exponential backoff
	periodicCatchupFailures map[uint64]int // chainID -> consecutive failure count
	periodicCatchupMu2      sync.Mutex     // Mutex for failure tracking

	// Metrics service for monitoring
	metricsService *MetricsService

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
	logger logger.Logger,
	metricsService *MetricsService,
) *EventCatchupService {
	// Create cleanup context
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())

	return &EventCatchupService{
		intentServices:          intentServices,
		fulfillmentServices:     fulfillmentServices,
		settlementServices:      settlementServices,
		db:                      db,
		intentProgress:          make(map[uint64]uint64),
		fulfillmentProgress:     make(map[uint64]uint64),
		settlementProgress:      make(map[uint64]uint64),
		activeCatchups:          make(map[string]bool),
		periodicCatchupProgress: make(map[uint64]uint64),
		periodicCatchupFailures: make(map[uint64]int),
		metricsService:          metricsService,
		logger:                  logger,
		cleanupCtx:              cleanupCtx,
		cleanupCancel:           cleanupCancel,
	}
}

// StartListening starts the coordinated event listening process
func (s *EventCatchupService) StartListening(ctx context.Context) error {
	// Check if service is shutdown
	if s.IsShutdown() {
		return fmt.Errorf("cannot start listening: service is shutdown")
	}

	s.logger.Notice("Starting event catchup service")

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
		s.logger.Debug("Configured chain %d with contract address %s",
			chainID, chainConfig.ContractAddr)
	}

	// Initialize progress tracking for all chains
	s.mu.Lock()
	for chainID := range s.intentServices {
		s.logger.InfoWithChain(chainID, "Initializing intent progress tracking")
		lastBlock, err := s.db.GetLastProcessedBlock(ctx, chainID)
		if err != nil {
			s.mu.Unlock()
			return fmt.Errorf("failed to get last processed block for chain %d: %v", chainID, err)
		}
		if lastBlock < cfg.ChainConfigs[chainID].DefaultBlock {
			s.logger.InfoWithChain(chainID, "Last processed block %d is less than default block %d, using default",
				lastBlock, cfg.ChainConfigs[chainID].DefaultBlock,
			)
			lastBlock = cfg.ChainConfigs[chainID].DefaultBlock
		}
		s.intentProgress[chainID] = lastBlock
		s.logger.InfoWithChain(chainID, "Setting intent progress to block %d", lastBlock)
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
		s.logger.InfoWithChain(chainID, "Current block for chain: %d", currentBlock)
	}

	// Track any errors that occur during catchup
	var catchupErrors []error

	// INTENT CATCHUP
	s.logger.Info("Starting intent event catchup")
	if err := s.runIntentCatchup(ctx, cfg, currentBlocks); err != nil {
		// Store the error but continue with fulfillment and settlement catchup
		catchupErrors = append(catchupErrors, fmt.Errorf("intent catchup failed: %v", err))
		// TODO: consider throwing an error here
		s.logger.Info("WARNING: Intent catchup encountered errors: %v, continuing with fulfillment catchup", err)
	} else {
		s.logger.Info("All intent services have completed catchup successfully")
	}

	// FULFILLMENT CATCHUP
	log.Printf("Starting fulfillment catchup")
	if err := s.runFulfillmentCatchup(ctx, cfg, currentBlocks); err != nil {
		// Store the error but continue with settlement catchup
		catchupErrors = append(catchupErrors, fmt.Errorf("fulfillment catchup failed: %v", err))
		// TODO: consider throwing an error here
		s.logger.Info("WARNING: Fulfillment catchup encountered errors: %v, continuing with settlement catchup", err)
	} else {
		s.logger.Info("All fulfillment services have completed catchup successfully")
	}

	// SETTLEMENT CATCHUP
	s.logger.Info("Starting settlement catchup")
	if err := s.runSettlementCatchup(ctx, cfg, currentBlocks); err != nil {
		// Store the error
		catchupErrors = append(catchupErrors, fmt.Errorf("settlement catchup failed: %v", err))
		// TODO: consider throwing an error here
		s.logger.Info("WARNING: Settlement catchup encountered errors: %v", err)
	} else {
		s.logger.Info("All settlement services have completed catchup successfully")
	}

	// Only attempt to update processed blocks for chains that completed successfully
	for chainID, currentBlock := range currentBlocks {
		updateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		if err := s.db.UpdateLastProcessedBlock(updateCtx, chainID, currentBlock); err != nil {
			s.logger.InfoWithChain(chainID, "WARNING: Failed to update last processed block: %v", err)
			// Don't return an error here, just log the warning
		} else {
			s.logger.InfoWithChain(chainID, "Updated last processed block to %d", currentBlock)
		}
		cancel()
	}

	// Start live subscriptions for all services
	if err := s.StartLiveEventListeners(ctx, cfg); err != nil {
		catchupErrors = append(catchupErrors, fmt.Errorf("failed to start live subscriptions: %v", err))
		s.logger.Info("WARNING: Failed to start some live subscriptions: %v", err)
	}

	// If there were any errors during the catchup process, log them but don't fail
	if len(catchupErrors) > 0 {
		s.logger.Debug("Catchup process completed with %d errors:", len(catchupErrors))
		for i, err := range catchupErrors {
			s.logger.Info("Catchup error %d: %v", i+1, err)
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

			s.logger.Debug("CATCHUP STATUS: %d active operations", activeOps)
			if activeOps > 0 {
				s.logger.Debug("Active operations: %v", activeList)
			}

			// Log intent service goroutines if available
			for chainID, service := range s.intentServices {
				if service != nil {
					activeGoroutines := service.ActiveGoroutines()
					s.logger.DebugWithChain(chainID, "Intent service: %d active goroutines", activeGoroutines)
				}
			}
		case <-ctx.Done():
			s.logger.Debug("Stopping catchup monitoring")
			return
		}
	}
}

// trackCatchupOperation adds an operation to the active operations map
func (s *EventCatchupService) trackCatchupOperation(operation string) {
	s.catchupMu.Lock()
	defer s.catchupMu.Unlock()
	s.activeCatchups[operation] = true
	s.logger.Debug("Starting catchup operation: %s", operation)
}

// untrackCatchupOperation removes an operation from the active operations map
func (s *EventCatchupService) untrackCatchupOperation(operation string) {
	s.catchupMu.Lock()
	defer s.catchupMu.Unlock()
	delete(s.activeCatchups, operation)
	s.logger.Debug("Completed catchup operation: %s", operation)
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
			s.logger.InfoWithChain(chainID, "No missed events to process")
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

			s.logger.InfoWithChain(chainID, "Starting intent event catch-up (blocks %d to %d)",
				lastBlock+1, currentBlock)

			// Create a timeout context for this specific chain's catchup
			chainCtx, chainCancel := context.WithTimeout(catchupCtx, CatchupOperationTimeout)
			defer chainCancel()

			if err := s.catchUpOnIntentEvents(chainCtx, intentService, contractAddress, lastBlock, currentBlock, opName); err != nil {
				intentErrors <- fmt.Errorf("failed to catch up on intent events for chain %d: %v", chainID, err)
				s.logger.ErrorWithChain(chainID, "Intent catchup for failed: %v", err)
				return
			}

			// Update progress
			s.UpdateIntentProgress(chainID, currentBlock)
			s.logger.InfoWithChain(chainID, "Completed intent event catch-up")
		}(chainID, intentService, lastBlock, currentBlock, opName)
	}

	// If there are no chains to process, we can return early
	if chainsToProcess == 0 {
		s.logger.Debug("No intent catchup needed for any chain")
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
				s.logger.Error("Intent catchup error: %v", err)
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
			s.logger.DebugWithChain(chainID, "No missed fulfillment events to process")
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

			s.logger.InfoWithChain(chainID, "Starting fulfillment event catch-up (blocks %d to %d)",
				lastBlock+1, currentBlock)

			// Create a timeout context for this specific chain's catchup
			chainCtx, chainCancel := context.WithTimeout(catchupCtx, CatchupOperationTimeout)
			defer chainCancel()

			if err := s.catchUpOnFulfillmentEvents(chainCtx, fulfillmentService, contractAddress, lastBlock, currentBlock, opName); err != nil {
				fulfillmentErrors <- fmt.Errorf("failed to catch up on fulfillment events for chain %d: %v", chainID, err)
				s.logger.ErrorWithChain(chainID, "ERROR: Fulfillment catchup failed: %v", err)
				return
			}

			// Update progress
			s.UpdateFulfillmentProgress(chainID, currentBlock)
			s.logger.InfoWithChain(chainID, "Completed fulfillment event catch-up for chain %d", chainID)
		}(chainID, fulfillmentService, lastBlock, currentBlock, opName)
	}

	// If there are no chains to process, we can return early
	if chainsToProcess == 0 {
		s.logger.Debug("No fulfillment catchup needed for any chain")
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
				s.logger.Error("Fulfillment catchup error: %v", err)
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
			s.logger.DebugWithChain(chainID, "No missed settlement events to process")
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

			s.logger.InfoWithChain(chainID, "Starting settlement event catch-up (blocks %d to %d)", lastBlock+1, currentBlock)

			// Create a timeout context for this specific chain's catchup
			chainCtx, chainCancel := context.WithTimeout(catchupCtx, CatchupOperationTimeout)
			defer chainCancel()

			if err := s.catchUpOnSettlementEvents(chainCtx, settlementService, contractAddress, lastBlock, currentBlock, opName); err != nil {
				settlementErrors <- fmt.Errorf("failed to catch up on settlement events for chain %d: %v", chainID, err)
				s.logger.ErrorWithChain(chainID, "Settlement catchup failed: %v", err)
				return
			}

			// Update progress
			s.UpdateSettlementProgress(chainID, currentBlock)
			s.logger.InfoWithChain(chainID, "Completed settlement event catch-up")
		}(chainID, settlementService, lastBlock, currentBlock, opName)
	}

	// If there are no chains to process, we can return early
	if chainsToProcess == 0 {
		s.logger.Debug("No settlement catchup needed for any chain")
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
				s.logger.Error("Settlement catchup error: %v", err)
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
	s.logger.Notice("Starting live intent event listeners")
	for chainID, intentService := range s.intentServices {
		chainID := chainID // Create a copy of the loop variable for the closure
		intentService := intentService

		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)
		s.logger.InfoWithChain(chainID, "Starting intent event listener at contract %s", contractAddress.Hex())

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
				s.logger.InfoWithChain(chainID, "WARNING: Unable to get current block: %v", err)
			} else {
				fromBlock = currentBlock
				s.logger.DebugWithChain(chainID, "No stored progress found, setting intent listener for chain %d to start from current block %d", fromBlock)
			}
		}
		s.mu.Unlock()

		// Special handling for ZetaChain - use polling instead of subscription
		if chainID == 7000 {
			// Store the initial block to start polling from
			s.mu.Lock()
			s.intentProgress[chainID] = fromBlock
			s.mu.Unlock()

			s.logger.InfoWithChain(chainID, "Setting up polling-based event monitoring for ZetaChain starting from block %d", fromBlock)

			// Start polling goroutine
			go s.pollZetachainEvents(ctx, intentService, contractAddress, cfg.ChainConfigs[chainID].BlockInterval)
			continue
		}

		// Start the intent service's own subscription management
		s.logger.InfoWithChain(chainID, "Starting intent service subscription through StartListening")
		if err := intentService.StartListening(ctx, contractAddress); err != nil {
			s.logger.ErrorWithChain(chainID, "Failed to start intent service: %v", err)
			return fmt.Errorf("failed to start intent service for chain %d: %v", chainID, err)
		}
		s.logger.InfoWithChain(chainID, "Successfully started intent service subscription")
	}

	// Start fulfillment listeners with similar block tracking
	s.logger.Debug("Starting live fulfillment event listeners")
	for chainID, fulfillmentService := range s.fulfillmentServices {
		chainID := chainID // Create a copy of the loop variable for the closure
		fulfillmentService := fulfillmentService

		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		// For live subscriptions, use the last processed block + 1 as the starting point
		var fromBlock uint64
		s.mu.Lock()
		if lastBlock, exists := s.fulfillmentProgress[chainID]; exists && lastBlock > 0 {
			fromBlock = lastBlock + 1
			s.logger.DebugWithChain(chainID, "Setting fulfillment listener to start from block %d", fromBlock)
		} else {
			// If we don't have a stored last block, get the current one
			blockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			currentBlock, err := fulfillmentService.client.BlockNumber(blockCtx)
			cancel()
			if err != nil {
				s.logger.InfoWithChain(chainID, "WARNING: Unable to get current block: %v", err)
			} else {
				fromBlock = currentBlock
				s.logger.DebugWithChain(chainID, "No stored progress found, setting fulfillment listener to start from current block %d", fromBlock)
			}
		}
		s.mu.Unlock()

		// Special handling for ZetaChain - use polling instead of subscription
		if chainID == 7000 {
			// Store the initial block to start polling from
			s.mu.Lock()
			s.fulfillmentProgress[chainID] = fromBlock
			s.mu.Unlock()

			s.logger.DebugWithChain(chainID, "Setting up polling-based fulfillment monitoring for ZetaChain starting from block %d", fromBlock)

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

		s.logger.DebugWithChain(chainID, "Fulfillment subscription filter: FromBlock=%v, Addresses=%s, Topics=%v", fulfillmentQuery.FromBlock, contractAddress.Hex(), fulfillmentQuery.Topics[0][0].Hex())

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
						s.logger.ErrorWithChain(chainID, "ERROR: Fulfillment subscription encountered an error: %v", err)
						// Try to resubscribe
						resubCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
						newSub, resubErr := fulfillmentService.client.SubscribeFilterLogs(resubCtx, fulfillmentQuery, fulfillmentLogs)
						cancel()

						if resubErr != nil {
							s.logger.ErrorWithChain(chainID, "CRITICAL: Failed to resubscribe fulfillment listener: %v", resubErr)
						} else {
							fulfillmentSub = newSub
							s.logger.InfoWithChain(chainID, "Successfully resubscribed fulfillment listener")
						}
					}
				case <-ctx.Done():
					s.logger.DebugWithChain(chainID, "Fulfillment subscription monitor shutting down")
					return
				}
			}
		}()

		go fulfillmentService.processEventLogs(ctx, fulfillmentSub, fulfillmentLogs, contractAddress.Hex())
	}

	// Start settlement listeners with similar block tracking
	s.logger.Info("Starting live settlement event listeners")
	for chainID, settlementService := range s.settlementServices {
		chainID := chainID // Create a copy of the loop variable for the closure
		settlementService := settlementService

		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		// For live subscriptions, use the last processed block + 1 as the starting point
		var fromBlock uint64
		s.mu.Lock()
		if lastBlock, exists := s.settlementProgress[chainID]; exists && lastBlock > 0 {
			fromBlock = lastBlock + 1
			s.logger.DebugWithChain(chainID, "Setting settlement listener to start from block %d", fromBlock)
		} else {
			// If we don't have a stored last block, get the current one
			blockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			currentBlock, err := settlementService.client.BlockNumber(blockCtx)
			cancel()
			if err != nil {
				s.logger.InfoWithChain(chainID, "WARNING: Unable to get current block: %v", err)
			} else {
				fromBlock = currentBlock
				s.logger.DebugWithChain(chainID, "No stored progress found, setting settlement listener to start from current block %d", fromBlock)
			}
		}
		s.mu.Unlock()

		// Special handling for ZetaChain - use polling instead of subscription
		if chainID == 7000 {
			// Store the initial block to start polling from
			s.mu.Lock()
			s.settlementProgress[chainID] = fromBlock
			s.mu.Unlock()

			s.logger.InfoWithChain(chainID, "Setting up polling-based settlement monitoring for ZetaChain == starting from block %d", fromBlock)

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

		s.logger.InfoWithChain(chainID, "Settlement subscription filter: FromBlock=%v, Addresses=%s, Topics=%v", settlementQuery.FromBlock, contractAddress.Hex(), settlementQuery.Topics[0][0].Hex())

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
						s.logger.ErrorWithChain(chainID, "ERROR: Settlement subscription encountered an error: %v", err)
						// Try to resubscribe
						resubCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
						newSub, resubErr := settlementService.client.SubscribeFilterLogs(resubCtx, settlementQuery, settlementLogs)
						cancel()

						if resubErr != nil {
							s.logger.ErrorWithChain(chainID, "CRITICAL: Failed to resubscribe settlement listener: %v", resubErr)
						} else {
							settlementSub = newSub
							s.logger.DebugWithChain(chainID, "Successfully resubscribed settlement listener")
						}
					}
				case <-ctx.Done():
					s.logger.DebugWithChain(chainID, "Settlement subscription monitor shutting down")
					return
				}
			}
		}()

		go settlementService.processEventLogs(ctx, settlementSub, settlementLogs, contractAddress.Hex(), contractAddress)
	}

	s.logger.Info("All live event listeners started successfully")
	return nil
}

// StartPeriodicCatchup starts a timer-based periodic catchup service that runs alongside live listeners
// to detect any missed events while the service is running
func (s *EventCatchupService) StartPeriodicCatchup(ctx context.Context, cfg *config.Config) {
	interval := time.Duration(cfg.PeriodicCatchupInterval) * time.Minute
	s.logger.Notice("Starting periodic catchup service with interval of %v", interval)

	// Start the periodic catchup goroutine
	s.StartGoroutine("periodic-catchup", func() {
		s.runPeriodicCatchup(ctx, cfg)
	})
}

// runPeriodicCatchup runs the periodic catchup operations on a timer
func (s *EventCatchupService) runPeriodicCatchup(ctx context.Context, cfg *config.Config) {
	interval := time.Duration(cfg.PeriodicCatchupInterval) * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial catchup after a short delay to allow live listeners to start
	// Use a shorter delay if there are no services configured (like in tests)
	var initialDelay time.Duration
	if len(s.intentServices) == 0 {
		initialDelay = 1 * time.Second // Short delay for tests
	} else {
		initialDelay = 5 * time.Minute // Normal delay for production
	}
	s.logger.Info("Scheduling initial periodic catchup in %v", initialDelay)

	initialTimer := time.NewTimer(initialDelay)
	defer initialTimer.Stop()

	// Wait for initial delay or context cancellation
	select {
	case <-initialTimer.C:
		s.logger.Info("Running initial periodic catchup")
		s.performPeriodicCatchup(ctx, cfg)
	case <-ctx.Done():
		s.logger.Debug("Context cancelled during initial delay, stopping periodic catchup")
		return
	}

	// Continue with regular periodic catchup
	for {
		select {
		case <-ticker.C:
			s.logger.Info("Running scheduled periodic catchup")
			s.performPeriodicCatchup(ctx, cfg)
		case <-ctx.Done():
			s.logger.Debug("Context cancelled, stopping periodic catchup")
			return
		}
	}
}

// performPeriodicCatchup performs a single periodic catchup operation
func (s *EventCatchupService) performPeriodicCatchup(ctx context.Context, cfg *config.Config) {
	// Create a context with timeout for the entire periodic catchup operation
	timeout := time.Duration(cfg.PeriodicCatchupTimeout) * time.Minute
	catchupCtx, catchupCancel := context.WithTimeout(ctx, timeout)
	defer catchupCancel()

	s.logger.Info("Starting periodic catchup operation with timeout of %v", timeout)

	// Initialize periodic catchup progress for all chains if not already set
	s.periodicCatchupMu.Lock()
	for chainID := range s.intentServices {
		if _, exists := s.periodicCatchupProgress[chainID]; !exists {
			// Load the last processed block from the database as starting point
			lastBlock, err := s.LoadPeriodicCatchupProgress(catchupCtx, chainID)
			if err != nil {
				s.logger.ErrorWithChain(chainID, "Failed to load periodic catchup progress from DB: %v", err)
				// Use a default starting block if we can't get from DB
				s.periodicCatchupProgress[chainID] = cfg.ChainConfigs[chainID].DefaultBlock
				s.logger.InfoWithChain(chainID, "Using default block %d for periodic catchup", cfg.ChainConfigs[chainID].DefaultBlock)
			} else {
				s.logger.InfoWithChain(chainID, "Initialized periodic catchup progress to block %d", lastBlock)
			}
		}
	}
	s.periodicCatchupMu.Unlock()

	// Get current block numbers for all chains
	currentBlocks := make(map[uint64]uint64)
	for chainID, intentService := range s.intentServices {
		blockCtx, cancel := context.WithTimeout(catchupCtx, 30*time.Second)
		currentBlock, err := intentService.client.BlockNumber(blockCtx)
		cancel()

		if err != nil {
			s.logger.ErrorWithChain(chainID, "Failed to get current block number for periodic catchup: %v", err)
			continue
		}
		currentBlocks[chainID] = currentBlock
	}

	// Track any errors that occur during periodic catchup
	var catchupErrors []error

	// Perform intent periodic catchup
	if err := s.runPeriodicIntentCatchup(catchupCtx, cfg, currentBlocks); err != nil {
		catchupErrors = append(catchupErrors, fmt.Errorf("periodic intent catchup failed: %v", err))
		s.logger.Error("Periodic intent catchup encountered errors: %v", err)
	} else {
		s.logger.Info("Periodic intent catchup completed successfully")
	}

	// Perform fulfillment periodic catchup
	if err := s.runPeriodicFulfillmentCatchup(catchupCtx, cfg, currentBlocks); err != nil {
		catchupErrors = append(catchupErrors, fmt.Errorf("periodic fulfillment catchup failed: %v", err))
		s.logger.Error("Periodic fulfillment catchup encountered errors: %v", err)
	} else {
		s.logger.Info("Periodic fulfillment catchup completed successfully")
	}

	// Perform settlement periodic catchup
	if err := s.runPeriodicSettlementCatchup(catchupCtx, cfg, currentBlocks); err != nil {
		catchupErrors = append(catchupErrors, fmt.Errorf("periodic settlement catchup failed: %v", err))
		s.logger.Error("Periodic settlement catchup encountered errors: %v", err)
	} else {
		s.logger.Info("Periodic settlement catchup completed successfully")
	}

	// Log results
	if len(catchupErrors) > 0 {
		s.logger.Error("Periodic catchup completed with %d errors:", len(catchupErrors))
		for i, err := range catchupErrors {
			s.logger.Error("Periodic catchup error %d: %v", i+1, err)
		}
	} else {
		s.logger.Info("Periodic catchup completed successfully with no errors")
	}
}

// runPeriodicIntentCatchup performs periodic catchup for intent events
func (s *EventCatchupService) runPeriodicIntentCatchup(ctx context.Context, cfg *config.Config, currentBlocks map[uint64]uint64) error {
	var intentWg sync.WaitGroup
	intentErrors := make(chan error, len(s.intentServices))

	// Track number of chains that need catchup
	chainsToProcess := 0

	// Start intent catch-up for all chains in parallel
	for chainID, intentService := range s.intentServices {
		// Check if we should skip this chain due to recent failures
		if s.ShouldSkipPeriodicCatchup(chainID) {
			continue
		}

		// Check if we have current block information for this chain
		currentBlock, exists := currentBlocks[chainID]
		if !exists {
			s.logger.ErrorWithChain(chainID, "No current block information available for periodic catchup, skipping")
			s.RecordPeriodicCatchupFailure(chainID)
			continue
		}

		// Get the last block processed by periodic catchup
		lastBlock := s.GetPeriodicCatchupProgress(chainID)

		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		// Only process if we have blocks to catch up on
		if lastBlock >= currentBlock {
			s.logger.DebugWithChain(chainID, "No missed intent events to process in periodic catchup (last: %d, current: %d)", lastBlock, currentBlock)
			continue
		}

		// Use the last processed block as the starting point for periodic catchup
		startBlock := lastBlock
		s.logger.InfoWithChain(chainID, "Periodic catchup: Scanning from block %d to %d", startBlock+1, currentBlock)

		chainsToProcess++
		intentWg.Add(1)

		// Use a descriptive operation name
		opName := fmt.Sprintf("periodic_intent_catchup_chain_%d", chainID)
		s.trackCatchupOperation(opName)

		go func(chainID uint64, intentService *IntentService, startBlock, currentBlock uint64, opName string) {
			defer intentWg.Done()
			defer s.untrackCatchupOperation(opName)

			startTime := time.Now()
			s.logger.InfoWithChain(chainID, "Starting periodic intent event catch-up (blocks %d to %d)",
				startBlock+1, currentBlock)

			// Create a timeout context for this specific chain's catchup
			timeout := time.Duration(cfg.PeriodicCatchupTimeout) * time.Minute
			chainCtx, chainCancel := context.WithTimeout(ctx, timeout)
			defer chainCancel()

			// Track events found and processed for metrics
			var eventsFound, eventsProcessed int

			if err := s.catchUpOnIntentEvents(chainCtx, intentService, contractAddress, startBlock, currentBlock, opName); err != nil {
				intentErrors <- fmt.Errorf("failed to catch up on periodic intent events for chain %d: %v", chainID, err)
				s.logger.ErrorWithChain(chainID, "Periodic intent catchup failed: %v", err)
				s.RecordPeriodicCatchupFailure(chainID)

				// Record failure metrics
				if s.metricsService != nil {
					s.metricsService.RecordPeriodicCatchupFailure(chainID, "intent")
				}
				return
			}

			// Only update progress if the catchup operation succeeded
			s.UpdatePeriodicCatchupProgress(chainID, currentBlock)

			// Persist progress to database
			if err := s.SavePeriodicCatchupProgress(chainCtx, chainID, currentBlock); err != nil {
				s.logger.ErrorWithChain(chainID, "Failed to save periodic catchup progress: %v", err)
				// Don't fail the entire operation if DB save fails, but log the error
			}

			// Record success to reset failure count
			s.RecordPeriodicCatchupSuccess(chainID)

			// Record success metrics
			duration := time.Since(startTime)
			if s.metricsService != nil {
				s.metricsService.RecordPeriodicCatchupSuccess(chainID, "intent", duration, eventsFound, eventsProcessed)
			}

			s.logger.InfoWithChain(chainID, "Completed periodic intent event catch-up, updated progress to block %d", currentBlock)
		}(chainID, intentService, startBlock, currentBlock, opName)
	}

	// If there are no chains to process, we can return early
	if chainsToProcess == 0 {
		s.logger.Debug("No periodic intent catchup needed for any chain")
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
	timeout := time.Duration(cfg.PeriodicCatchupTimeout) * time.Minute
	select {
	case <-done:
		// Process any errors that were collected
		for err := range intentErrors {
			if err != nil {
				errs = append(errs, err)
				s.logger.Error("Periodic intent catchup error: %v", err)
			}
		}
	case <-ctx.Done():
		return fmt.Errorf("periodic intent catchup timed out after %v", timeout)
	}

	// Return combined errors if any
	if len(errs) > 0 {
		return fmt.Errorf("periodic intent catchup completed with %d errors", len(errs))
	}

	return nil
}

// runPeriodicFulfillmentCatchup performs periodic catchup for fulfillment events
func (s *EventCatchupService) runPeriodicFulfillmentCatchup(ctx context.Context, cfg *config.Config, currentBlocks map[uint64]uint64) error {
	var fulfillmentWg sync.WaitGroup
	fulfillmentErrors := make(chan error, len(s.fulfillmentServices))

	// Track number of chains that need catchup
	chainsToProcess := 0

	for chainID, fulfillmentService := range s.fulfillmentServices {
		// Check if we have current block information for this chain
		currentBlock, exists := currentBlocks[chainID]
		if !exists {
			s.logger.ErrorWithChain(chainID, "No current block information available for periodic catchup, skipping")
			continue
		}

		// Get the last block processed by periodic catchup
		lastBlock := s.GetPeriodicCatchupProgress(chainID)

		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		// Only process if we have blocks to catch up on
		if lastBlock >= currentBlock {
			s.logger.DebugWithChain(chainID, "No missed fulfillment events to process in periodic catchup (last: %d, current: %d)", lastBlock, currentBlock)
			continue
		}

		// Use the last processed block as the starting point for periodic catchup
		startBlock := lastBlock
		s.logger.InfoWithChain(chainID, "Periodic catchup: Scanning from block %d to %d", startBlock+1, currentBlock)

		chainsToProcess++
		fulfillmentWg.Add(1)

		// Use a descriptive operation name
		opName := fmt.Sprintf("periodic_fulfillment_catchup_chain_%d", chainID)
		s.trackCatchupOperation(opName)

		go func(chainID uint64, fulfillmentService *FulfillmentService, startBlock, currentBlock uint64, opName string) {
			defer fulfillmentWg.Done()
			defer s.untrackCatchupOperation(opName)

			s.logger.InfoWithChain(chainID, "Starting periodic fulfillment event catch-up (blocks %d to %d)",
				startBlock+1, currentBlock)

			// Create a timeout context for this specific chain's catchup
			timeout := time.Duration(cfg.PeriodicCatchupTimeout) * time.Minute
			chainCtx, chainCancel := context.WithTimeout(ctx, timeout)
			defer chainCancel()

			if err := s.catchUpOnFulfillmentEvents(chainCtx, fulfillmentService, contractAddress, startBlock, currentBlock, opName); err != nil {
				fulfillmentErrors <- fmt.Errorf("failed to catch up on periodic fulfillment events for chain %d: %v", chainID, err)
				s.logger.ErrorWithChain(chainID, "Periodic fulfillment catchup failed: %v", err)
				return
			}

			// Only update progress if the catchup operation succeeded
			s.UpdatePeriodicCatchupProgress(chainID, currentBlock)

			// Persist progress to database
			if err := s.SavePeriodicCatchupProgress(chainCtx, chainID, currentBlock); err != nil {
				s.logger.ErrorWithChain(chainID, "Failed to save periodic catchup progress: %v", err)
				// Don't fail the entire operation if DB save fails, but log the error
			}

			s.logger.InfoWithChain(chainID, "Completed periodic fulfillment event catch-up, updated progress to block %d", currentBlock)
		}(chainID, fulfillmentService, startBlock, currentBlock, opName)
	}

	// If there are no chains to process, we can return early
	if chainsToProcess == 0 {
		s.logger.Debug("No periodic fulfillment catchup needed for any chain")
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
	timeout := time.Duration(cfg.PeriodicCatchupTimeout) * time.Minute
	select {
	case <-done:
		// Process any errors that were collected
		for err := range fulfillmentErrors {
			if err != nil {
				errs = append(errs, err)
				s.logger.Error("Periodic fulfillment catchup error: %v", err)
			}
		}
	case <-ctx.Done():
		return fmt.Errorf("periodic fulfillment catchup timed out after %v", timeout)
	}

	// Return combined errors if any
	if len(errs) > 0 {
		return fmt.Errorf("periodic fulfillment catchup completed with %d errors", len(errs))
	}

	return nil
}

// runPeriodicSettlementCatchup performs periodic catchup for settlement events
func (s *EventCatchupService) runPeriodicSettlementCatchup(ctx context.Context, cfg *config.Config, currentBlocks map[uint64]uint64) error {
	var settlementWg sync.WaitGroup
	settlementErrors := make(chan error, len(s.settlementServices))

	// Track number of chains that need catchup
	chainsToProcess := 0

	for chainID, settlementService := range s.settlementServices {
		// Check if we have current block information for this chain
		currentBlock, exists := currentBlocks[chainID]
		if !exists {
			s.logger.ErrorWithChain(chainID, "No current block information available for periodic catchup, skipping")
			continue
		}

		// Get the last block processed by periodic catchup
		lastBlock := s.GetPeriodicCatchupProgress(chainID)

		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		// Only process if we have blocks to catch up on
		if lastBlock >= currentBlock {
			s.logger.DebugWithChain(chainID, "No missed settlement events to process in periodic catchup (last: %d, current: %d)", lastBlock, currentBlock)
			continue
		}

		// Use the last processed block as the starting point for periodic catchup
		startBlock := lastBlock
		s.logger.InfoWithChain(chainID, "Periodic catchup: Scanning from block %d to %d", startBlock+1, currentBlock)

		chainsToProcess++
		settlementWg.Add(1)

		// Use a descriptive operation name
		opName := fmt.Sprintf("periodic_settlement_catchup_chain_%d", chainID)
		s.trackCatchupOperation(opName)

		go func(chainID uint64, settlementService *SettlementService, startBlock, currentBlock uint64, opName string) {
			defer settlementWg.Done()
			defer s.untrackCatchupOperation(opName)

			s.logger.InfoWithChain(chainID, "Starting periodic settlement event catch-up (blocks %d to %d)",
				startBlock+1, currentBlock)

			// Create a timeout context for this specific chain's catchup
			timeout := time.Duration(cfg.PeriodicCatchupTimeout) * time.Minute
			chainCtx, chainCancel := context.WithTimeout(ctx, timeout)
			defer chainCancel()

			if err := s.catchUpOnSettlementEvents(chainCtx, settlementService, contractAddress, startBlock, currentBlock, opName); err != nil {
				settlementErrors <- fmt.Errorf("failed to catch up on periodic settlement events for chain %d: %v", chainID, err)
				s.logger.ErrorWithChain(chainID, "Periodic settlement catchup failed: %v", err)
				return
			}

			// Only update progress if the catchup operation succeeded
			s.UpdatePeriodicCatchupProgress(chainID, currentBlock)

			// Persist progress to database
			if err := s.SavePeriodicCatchupProgress(chainCtx, chainID, currentBlock); err != nil {
				s.logger.ErrorWithChain(chainID, "Failed to save periodic catchup progress: %v", err)
				// Don't fail the entire operation if DB save fails, but log the error
			}

			s.logger.InfoWithChain(chainID, "Completed periodic settlement event catch-up, updated progress to block %d", currentBlock)
		}(chainID, settlementService, startBlock, currentBlock, opName)
	}

	// If there are no chains to process, we can return early
	if chainsToProcess == 0 {
		s.logger.Debug("No periodic settlement catchup needed for any chain")
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
	timeout := time.Duration(cfg.PeriodicCatchupTimeout) * time.Minute
	select {
	case <-done:
		// Process any errors that were collected
		for err := range settlementErrors {
			if err != nil {
				errs = append(errs, err)
				s.logger.Error("Periodic settlement catchup error: %v", err)
			}
		}
	case <-ctx.Done():
		return fmt.Errorf("periodic settlement catchup timed out after %v", timeout)
	}

	// Return combined errors if any
	if len(errs) > 0 {
		return fmt.Errorf("periodic settlement catchup completed with %d errors", len(errs))
	}

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

// UpdatePeriodicCatchupProgress updates the progress of periodic catchup for a chain
func (s *EventCatchupService) UpdatePeriodicCatchupProgress(chainID, blockNumber uint64) {
	s.periodicCatchupMu.Lock()
	defer s.periodicCatchupMu.Unlock()
	s.periodicCatchupProgress[chainID] = blockNumber
	s.logger.Debug("Updated periodic catchup progress for chain %d to block %d", chainID, blockNumber)
}

// GetPeriodicCatchupProgress gets the last block processed by periodic catchup for a chain
func (s *EventCatchupService) GetPeriodicCatchupProgress(chainID uint64) uint64 {
	s.periodicCatchupMu.Lock()
	defer s.periodicCatchupMu.Unlock()
	return s.periodicCatchupProgress[chainID]
}

// SavePeriodicCatchupProgress persists the periodic catchup progress to the database
func (s *EventCatchupService) SavePeriodicCatchupProgress(ctx context.Context, chainID, blockNumber uint64) error {
	// Use a timeout for the database operation
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Retry logic for database operations
	maxRetries := 3
	baseDelay := 1 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := s.db.UpdatePeriodicCatchupBlock(dbCtx, chainID, blockNumber)
		if err == nil {
			s.logger.DebugWithChain(chainID, "Persisted periodic catchup progress to DB: block %d", blockNumber)
			return nil
		}

		s.logger.ErrorWithChain(chainID, "Failed to persist periodic catchup progress to DB (attempt %d/%d): %v", attempt, maxRetries, err)

		if attempt < maxRetries {
			// Exponential backoff
			delay := baseDelay * time.Duration(attempt)
			select {
			case <-time.After(delay):
				continue
			case <-dbCtx.Done():
				return fmt.Errorf("context cancelled while retrying database operation: %v", dbCtx.Err())
			}
		}
	}

	return fmt.Errorf("failed to persist periodic catchup progress after %d attempts", maxRetries)
}

// LoadPeriodicCatchupProgress loads the periodic catchup progress from the database
func (s *EventCatchupService) LoadPeriodicCatchupProgress(ctx context.Context, chainID uint64) (uint64, error) {
	// Use a timeout for the database operation
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	lastBlock, err := s.db.GetPeriodicCatchupBlock(dbCtx, chainID)
	if err != nil {
		return 0, err
	}

	s.periodicCatchupMu.Lock()
	s.periodicCatchupProgress[chainID] = lastBlock
	s.periodicCatchupMu.Unlock()

	s.logger.DebugWithChain(chainID, "Loaded periodic catchup progress from DB: block %d", lastBlock)
	return lastBlock, nil
}

// RecordPeriodicCatchupFailure records a failure for a chain to implement exponential backoff
func (s *EventCatchupService) RecordPeriodicCatchupFailure(chainID uint64) {
	s.periodicCatchupMu2.Lock()
	defer s.periodicCatchupMu2.Unlock()
	s.periodicCatchupFailures[chainID]++
	s.logger.InfoWithChain(chainID, "WARNING: Periodic catchup failure recorded (consecutive failures: %d)", s.periodicCatchupFailures[chainID])

	// Update metrics if available
	if s.metricsService != nil {
		s.metricsService.UpdatePeriodicCatchupFailureCount(chainID, s.periodicCatchupFailures[chainID])
	}
}

// RecordPeriodicCatchupSuccess resets the failure count for a chain
func (s *EventCatchupService) RecordPeriodicCatchupSuccess(chainID uint64) {
	s.periodicCatchupMu2.Lock()
	defer s.periodicCatchupMu2.Unlock()
	if s.periodicCatchupFailures[chainID] > 0 {
		s.logger.InfoWithChain(chainID, "Periodic catchup succeeded, resetting failure count from %d", s.periodicCatchupFailures[chainID])
		s.periodicCatchupFailures[chainID] = 0

		// Update metrics if available
		if s.metricsService != nil {
			s.metricsService.UpdatePeriodicCatchupFailureCount(chainID, 0)
		}
	}
}

// ShouldSkipPeriodicCatchup determines if a chain should be skipped due to recent failures
func (s *EventCatchupService) ShouldSkipPeriodicCatchup(chainID uint64) bool {
	s.periodicCatchupMu2.Lock()
	defer s.periodicCatchupMu2.Unlock()

	failureCount := s.periodicCatchupFailures[chainID]
	if failureCount == 0 {
		return false
	}

	// Exponential backoff: skip if we have 2 or more consecutive failures
	// This gives the chain time to recover from temporary issues
	if failureCount >= 2 {
		s.logger.InfoWithChain(chainID, "WARNING: Skipping periodic catchup due to %d consecutive failures", failureCount)
		return true
	}

	return false
}

// GetPeriodicCatchupHealth returns health information for periodic catchup
func (s *EventCatchupService) GetPeriodicCatchupHealth() map[string]interface{} {
	s.periodicCatchupMu2.Lock()
	defer s.periodicCatchupMu2.Unlock()

	health := make(map[string]interface{})
	chainHealth := make(map[string]interface{})

	for chainID, failureCount := range s.periodicCatchupFailures {
		chainIDStr := fmt.Sprintf("%d", chainID)
		chainHealth[chainIDStr] = map[string]interface{}{
			"consecutive_failures": failureCount,
			"is_healthy":           failureCount < 2,
			"last_failure_time":    time.Now().Add(-time.Duration(failureCount) * time.Hour), // Approximate
		}
	}

	health["chains"] = chainHealth
	health["total_chains"] = len(s.periodicCatchupFailures)
	health["healthy_chains"] = func() int {
		count := 0
		for _, failureCount := range s.periodicCatchupFailures {
			if failureCount < 2 {
				count++
			}
		}
		return count
	}()
	health["timestamp"] = time.Now()

	return health
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
		s.logger.Debug("[%s] Using smaller block range of %d for Ethereum mainnet", opName, maxBlockRange)
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
				s.logger.Debug("[%s] Error in bloom check: %v, will process range", opName, err)
			} else if !hasEvents {
				// Skip this chunk as it likely has no events
				s.logger.Debug("[%s] Fast-forwarding through block range %d-%d (no events detected)",
					opName, chunkStart+1, chunkEnd)

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

			s.logger.Debug("[%s] Fetching intent logs for blocks %d to %d", opName, chunkStart+1, chunkEnd)

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

			s.logger.Debug("[%s] Processing %d logs from blocks %d to %d", opName, len(logs), chunkStart+1, chunkEnd)

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

						s.logger.Debug("[%s] Processing intent log %d/%d: Block=%d, TxHash=%s",
							opName, i+j+1, len(logs), txlog.BlockNumber, txlog.TxHash.Hex())

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
							s.logger.Debug("[%s] Skipping existing intent: %s", opName, intentID)
							continue
						}

						// Process log with timeout
						processCtx, processCancel := context.WithTimeout(batchCtx, 20*time.Second)
						err = intentService.processLog(processCtx, txlog)
						processCancel()

						if err != nil {
							// Skip if intent already exists
							if strings.Contains(err.Error(), "duplicate key") {
								s.logger.Debug("[%s] Skipping duplicate intent: %s", opName, intentID)
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
					s.logger.Debug("[%s] Updated progress for chain %d to block %d", opName, intentService.chainID, lastBlock)
				}
			}

			// Update progress after processing each chunk
			s.UpdateIntentProgress(intentService.chainID, chunkEnd)

			// Persist progress to the database after each chunk
			dbUpdateCtx, dbUpdateCancel := context.WithTimeout(ctx, 10*time.Second)
			dbErr := s.db.UpdateLastProcessedBlock(dbUpdateCtx, intentService.chainID, chunkEnd)
			dbUpdateCancel()
			if dbErr != nil {
				s.logger.Debug("[%s] Warning: Failed to persist progress to DB: %v", opName, dbErr)
				// Continue processing even if DB update fails
			} else {
				s.logger.Debug("[%s] Persisted progress to DB: chain %d at block %d", opName, intentService.chainID, chunkEnd)
			}

			s.logger.Debug("[%s] Completed processing intent logs for blocks %d to %d", opName, chunkStart+1, chunkEnd)

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
		s.logger.Debug("[%s] Using smaller block range of %d for Ethereum mainnet", opName, maxBlockRange)
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

			s.logger.Debug("[%s] Fetching fulfillment logs for blocks %d to %d", opName, chunkStart+1, chunkEnd)

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

			s.logger.Debug("[%s] Processing %d logs from blocks %d to %d", opName, len(logs), chunkStart+1, chunkEnd)

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

						s.logger.Debug("[%s] Processing fulfillment log %d/%d: Block=%d, TxHash=%s",
							opName, i+j+1, len(logs), txlog.BlockNumber, txlog.TxHash.Hex())

						// Extract intent ID from the log
						intentID := txlog.Topics[1].Hex()

						// Check if intent exists (fulfillments need an intent)
						getIntentCtx, cancel := context.WithTimeout(batchCtx, 10*time.Second)
						_, err := s.db.GetIntent(getIntentCtx, intentID)
						cancel()

						if err != nil {
							if strings.Contains(err.Error(), "not found") {
								s.logger.Debug("[%s] Skipping fulfillment for non-existent intent: %s", opName, intentID)
								continue
							}
							s.logger.Debug("[%s] Failed to check for existing intent: %v", opName, err)
							continue
						}

						// Process log with timeout
						processCtx, processCancel := context.WithTimeout(batchCtx, 20*time.Second)
						err = fulfillmentService.processLog(processCtx, txlog)
						processCancel()

						if err != nil {
							// Skip if fulfillment already exists
							if strings.Contains(err.Error(), "duplicate key") {
								s.logger.Debug("[%s] Skipping duplicate fulfillment: %s", opName, intentID)
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
					s.logger.Debug("[%s] Updated progress for chain %d to block %d", opName, fulfillmentService.chainID, lastBlock)
				}
			}

			// Update progress after processing each chunk
			s.UpdateFulfillmentProgress(fulfillmentService.chainID, chunkEnd)

			// Persist progress to the database after each chunk
			dbUpdateCtx, dbUpdateCancel := context.WithTimeout(ctx, 10*time.Second)
			dbErr := s.db.UpdateLastProcessedBlock(dbUpdateCtx, fulfillmentService.chainID, chunkEnd)
			dbUpdateCancel()
			if dbErr != nil {
				s.logger.Debug("[%s] Warning: Failed to persist progress to DB: %v", opName, dbErr)
				// Continue processing even if DB update fails
			} else {
				s.logger.Debug("[%s] Persisted progress to DB: chain %d at block %d", opName, fulfillmentService.chainID, chunkEnd)
			}

			s.logger.Debug("[%s] Completed processing fulfillment logs for blocks %d to %d", opName, chunkStart+1, chunkEnd)

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
		s.logger.Debug("[%s] Using smaller block range of %d for Ethereum mainnet", opName, maxBlockRange)
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

			s.logger.Debug("[%s] Fetching settlement logs for blocks %d to %d", opName, chunkStart+1, chunkEnd)

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

			s.logger.Debug("[%s] Processing %d logs from blocks %d to %d", opName, len(logs), chunkStart+1, chunkEnd)

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

						s.logger.Debug("[%s] Processing settlement log %d/%d: Block=%d, TxHash=%s",
							opName, i+j+1, len(logs), txlog.BlockNumber, txlog.TxHash.Hex())

						// Extract intent ID from the log
						intentID := txlog.Topics[1].Hex()

						// Check if intent exists (settlements need an intent)
						getIntentCtx, cancel := context.WithTimeout(batchCtx, 10*time.Second)
						_, err := s.db.GetIntent(getIntentCtx, intentID)
						cancel()

						if err != nil {
							if strings.Contains(err.Error(), "not found") {
								s.logger.Debug("[%s] Skipping settlement for non-existent intent: %s", opName, intentID)
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
								s.logger.Debug("[%s] Skipping duplicate settlement: %s", opName, intentID)
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
					s.logger.Debug("[%s] Updated progress for chain %d to block %d", opName, settlementService.chainID, lastBlock)
				}
			}

			// Update progress after processing each chunk
			s.UpdateSettlementProgress(settlementService.chainID, chunkEnd)

			// Persist progress to the database after each chunk
			dbUpdateCtx, dbUpdateCancel := context.WithTimeout(ctx, 10*time.Second)
			dbErr := s.db.UpdateLastProcessedBlock(dbUpdateCtx, settlementService.chainID, chunkEnd)
			dbUpdateCancel()
			if dbErr != nil {
				s.logger.Debug("[%s] Warning: Failed to persist progress to DB: %v", opName, dbErr)
				// Continue processing even if DB update fails
			} else {
				s.logger.Debug("[%s] Persisted progress to DB: chain %d at block %d", opName, settlementService.chainID, chunkEnd)
			}

			s.logger.Debug("[%s] Completed processing settlement logs for blocks %d to %d", opName, chunkStart+1, chunkEnd)

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

	s.logger.Info("Starting ZetaChain polling for %s events with interval of %v", eventType, interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Track database persistence to avoid doing it every poll
	lastDbUpdateTime := time.Now()
	dbUpdateInterval := 5 * time.Minute

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Context cancelled, stopping ZetaChain %s event polling", eventType)
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
				s.logger.Error("ERROR: Failed to get current block for ZetaChain (attempt %d/%d): %v. Retrying in %v",
					retry+1, maxRetries, err, retryDelay)

				select {
				case <-time.After(retryDelay):
					continue
				case <-ctx.Done():
					return
				}
			}

			if err != nil {
				s.logger.Error("CRITICAL: Failed to get current block for ZetaChain after %d attempts. Skipping this polling cycle.", maxRetries)
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
						s.logger.Info("WARNING: Failed to persist %s progress to DB: %v", eventType, err)
					} else {
						s.logger.Debug("Persisted %s progress to DB: chain %d at block %d", eventType, chainID, lastProcessedBlock)
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

			s.logger.Debug("Polling ZetaChain for %s events from blocks %d to %d", eventType, lastProcessedBlock+1, endBlock)

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
				s.logger.Error("ERROR: Failed to filter logs for ZetaChain %s events (attempt %d/%d): %v. Retrying in %v",
					eventType, retry+1, maxRetries, err, retryDelay)

				select {
				case <-time.After(retryDelay):
					continue
				case <-ctx.Done():
					return
				}
			}

			if err != nil {
				s.logger.Error("CRITICAL: Failed to filter logs for ZetaChain %s events after %d attempts. Skipping this block range.",
					eventType, maxRetries)
				continue
			}

			// Process logs if any found
			processedCount := 0
			errorCount := 0
			if len(logs) > 0 {
				s.logger.Debug("Found %d new %s events in ZetaChain blocks %d to %d",
					len(logs), eventType, lastProcessedBlock+1, endBlock)

				// Process the logs with individual timeouts
				for _, logEntry := range logs {
					processCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
					err := processLogFunc(processCtx, logEntry)
					cancel()

					if err != nil {
						errorCount++
						if strings.Contains(err.Error(), "duplicate key") {
							// This is expected for duplicates, just log at debug level
							s.logger.Debug("Skipping duplicate %s event in tx: %s", eventType, logEntry.TxHash.Hex())
						} else {
							s.logger.Error("Failed to process ZetaChain %s log: %v", eventType, err)
						}
					} else {
						processedCount++
					}
				}
				s.logger.Info("Successfully processed %d/%d %s events (errors: %d)",
					processedCount, len(logs), eventType, errorCount)
			} else {
				s.logger.Info("No new %s events found in ZetaChain blocks %d to %d",
					eventType, lastProcessedBlock+1, endBlock)
			}

			// Update the last processed block
			updateProgressFunc(endBlock)

			// Persist progress to the database
			dbUpdateCtx, dbUpdateCancel := context.WithTimeout(ctx, 10*time.Second)
			if err := s.db.UpdateLastProcessedBlock(dbUpdateCtx, chainID, endBlock); err != nil {
				s.logger.Info("WARNING: Failed to persist %s progress to DB: %v", eventType, err)
			} else {
				s.logger.Debug("Persisted %s progress to DB: chain %d at block %d", eventType, chainID, endBlock)
				lastDbUpdateTime = time.Now()
			}
			dbUpdateCancel()
		}
	}
}

// StartSubscriptionSupervisor starts a background goroutine that periodically checks
// if services are still running and restarts them if needed
func (s *EventCatchupService) StartSubscriptionSupervisor(ctx context.Context, cfg *config.Config) {
	s.logger.Info("Starting subscription supervisor to monitor service health")

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
			s.logger.Info("Subscription supervisor checking service health...")

			// Check intent services
			for chainID, intentService := range s.intentServices {
				// Skip health check for ZetaChain as it's using polling
				if chainID == zetaChainID {
					s.logger.Info("ZetaChain intent service using polling mechanism - skipping subscription check")
					continue
				}

				activeGoroutines := intentService.ActiveGoroutines()
				s.logger.Debug("Intent service for chain %d: %d active goroutines", chainID, activeGoroutines)

				if activeGoroutines == 0 {
					s.logger.Info("WARNING: Intent service for chain %d has 0 active goroutines, restarting", chainID)
					contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

					// Create a context with timeout for restart
					restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					err := intentService.Restart(restartCtx, contractAddress)
					if err != nil {
						s.logger.Error("Failed to restart intent service for chain %d: %v", chainID, err)
					} else {
						s.logger.Info("RECOVERY: Successfully restarted intent service for chain %d", chainID)
					}
					cancel()
				}
			}

			// Check fulfillment services
			for chainID, fulfillmentService := range s.fulfillmentServices {
				// Skip health check for ZetaChain as it's using polling
				if chainID == zetaChainID {
					s.logger.Info("ZetaChain fulfillment service using polling mechanism - skipping subscription check")
					continue
				}

				count := fulfillmentService.GetSubscriptionCount()
				s.logger.Info("Fulfillment service for chain %d: %d active subscriptions", chainID, count)

				if count == 0 {
					s.logger.Info("WARNING: Fulfillment service for chain %d has no active subscriptions, restarting", chainID)
					contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

					// Create a context with timeout for restart
					restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					err := fulfillmentService.Restart(restartCtx, contractAddress)
					if err != nil {
						s.logger.Error("Failed to restart fulfillment service for chain %d: %v", chainID, err)
					} else {
						s.logger.Info("Successfully restarted fulfillment service for chain %d", chainID)
					}
					cancel()
				}
			}

			// Check settlement services
			for chainID, settlementService := range s.settlementServices {
				// Skip health check for ZetaChain as it's using polling
				if chainID == zetaChainID {
					s.logger.Info("ZetaChain settlement service using polling mechanism - skipping subscription check")
					continue
				}

				count := settlementService.GetSubscriptionCount()
				s.logger.Info("Settlement service for chain %d: %d active subscriptions", chainID, count)

				if count == 0 {
					s.logger.Info("WARNING: Settlement service for chain %d has no active subscriptions, restarting", chainID)
					contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

					// Create a context with timeout for restart
					restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					err := settlementService.Restart(restartCtx, contractAddress)
					if err != nil {
						s.logger.Error("Failed to restart settlement service for chain %d: %v", chainID, err)
					} else {
						s.logger.Info("Successfully restarted settlement service for chain %d", chainID)
					}
					cancel()
				}
			}

			// Check ZetaChain health by getting block number
			if client, ok := s.intentServices[zetaChainID]; ok && client != nil {
				s.logger.Info("Checking ZetaChain polling health...")
				blockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				_, err := client.client.BlockNumber(blockCtx)
				cancel()

				if err != nil {
					s.logger.Info("WARNING: ZetaChain polling health check failed: %v", err)
					client.UpdatePollingHealth(false)
				} else {
					s.logger.Info("ZetaChain polling health check passed")
					client.UpdatePollingHealth(true)
				}
			}

		case <-reconnectTicker.C:
			// Perform a complete refresh of all WebSocket connections every 2 hours
			timeSinceLastReconnect := time.Since(lastFullReconnect)
			s.logger.Info("Performing scheduled full reconnection of all services (last reconnect: %v ago)", timeSinceLastReconnect)
			lastFullReconnect = time.Now()

			// Force reconnect all intent services (except ZetaChain)
			for chainID, intentService := range s.intentServices {
				// Skip ZetaChain as it's using polling
				if chainID == zetaChainID {
					s.logger.Info("Skipping ZetaChain intent service reconnection (using polling)")
					continue
				}

				s.logger.Info("Scheduled reconnect: Restarting intent service for chain %d", chainID)
				contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

				// First, unsubscribe from all existing subscriptions
				intentService.UnsubscribeAll()

				// Create a context with timeout for restart
				restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				err := intentService.Restart(restartCtx, contractAddress)
				if err != nil {
					s.logger.Error("Failed to reconnect intent service for chain %d: %v", chainID, err)
				} else {
					s.logger.Info("Scheduled reconnect: Successfully reconnected intent service for chain %d", chainID)
				}
				cancel()
			}

			// Force reconnect all fulfillment services (except ZetaChain)
			for chainID, fulfillmentService := range s.fulfillmentServices {
				// Skip ZetaChain as it's using polling
				if chainID == zetaChainID {
					s.logger.Info("Skipping ZetaChain fulfillment service reconnection (using polling)")
					continue
				}

				s.logger.Info("Scheduled reconnect: Restarting fulfillment service for chain %d", chainID)
				contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

				// First, unsubscribe from all existing subscriptions
				fulfillmentService.UnsubscribeAll()

				// Create a context with timeout for restart
				restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				err := fulfillmentService.Restart(restartCtx, contractAddress)
				if err != nil {
					s.logger.Error("Failed to reconnect fulfillment service for chain %d: %v", chainID, err)
				} else {
					s.logger.Info("Scheduled reconnect: Successfully reconnected fulfillment service for chain %d", chainID)
				}
				cancel()
			}

			// Force reconnect all settlement services (except ZetaChain)
			for chainID, settlementService := range s.settlementServices {
				// Skip ZetaChain as it's using polling
				if chainID == zetaChainID {
					s.logger.Info("Skipping ZetaChain settlement service reconnection (using polling)")
					continue
				}

				s.logger.Info("Scheduled reconnect: Restarting settlement service for chain %d", chainID)
				contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

				// First, unsubscribe from all existing subscriptions
				settlementService.UnsubscribeAll()

				// Create a context with timeout for restart
				restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				err := settlementService.Restart(restartCtx, contractAddress)
				if err != nil {
					s.logger.Error("Failed to reconnect settlement service for chain %d: %v", chainID, err)
				} else {
					s.logger.Info("Scheduled reconnect: Successfully reconnected settlement service for chain %d", chainID)
				}
				cancel()
			}

		case <-ctx.Done():
			s.logger.Debug("Subscription supervisor shutting down")
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

	s.logger.Info("Shutting down EventCatchupService...")

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
		s.logger.Info("EventCatchupService shutdown completed successfully")
		return nil
	case <-time.After(timeout):
		s.logger.Error("EventCatchupService shutdown timed out after %v", timeout)
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
		s.logger.Debug("Cannot start goroutine %s: service is shutdown", name)
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
				s.logger.Error("CRITICAL: Panic in goroutine %s: %v", name, r)
			}
		}()

		fn()
	}()
}

// ActiveGoroutines returns the current count of active goroutines
func (s *EventCatchupService) ActiveGoroutines() int32 {
	return atomic.LoadInt32(&s.activeGoroutines)
}
