package services

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"sync"
	"time"

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

	// Add these constants for block range optimization
	// Default max range for most chains
	DefaultMaxBlockRange = uint64(5000)

	// Smaller range for Ethereum mainnet (chain ID 1) since it has higher transaction density
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
}

// NewEventCatchupService creates a new EventCatchupService instance
func NewEventCatchupService(intentServices map[uint64]*IntentService, fulfillmentServices map[uint64]*FulfillmentService, settlementServices map[uint64]*SettlementService, db db.Database) *EventCatchupService {
	return &EventCatchupService{
		intentServices:      intentServices,
		fulfillmentServices: fulfillmentServices,
		settlementServices:  settlementServices,
		db:                  db,
		intentProgress:      make(map[uint64]uint64),
		fulfillmentProgress: make(map[uint64]uint64),
		settlementProgress:  make(map[uint64]uint64),
		activeCatchups:      make(map[string]bool),
	}
}

// StartListening starts the coordinated event listening process
func (s *EventCatchupService) StartListening(ctx context.Context) error {
	log.Println("Starting event catchup service")

	// Start a monitoring goroutine to periodically log the status
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	defer monitorCancel()
	go s.monitorCatchupProgress(monitorCtx)

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// Print information about active chains for debugging
	for chainID, chainConfig := range cfg.ChainConfigs {
		log.Printf("Configured chain %d with contract address %s",
			chainID, chainConfig.ContractAddr)
	}

	// Initialize progress tracking for all chains
	s.mu.Lock()
	for chainID := range s.intentServices {
		log.Printf("Initializing intent progress tracking for chain %d", chainID)
		lastBlock, err := s.db.GetLastProcessedBlock(ctx, chainID)
		if err != nil {
			s.mu.Unlock()
			return fmt.Errorf("failed to get last processed block for chain %d: %v", chainID, err)
		}
		if lastBlock < cfg.ChainConfigs[chainID].DefaultBlock {
			log.Printf("Last processed block %d is less than default block %d for chain %d, using default",
				lastBlock, cfg.ChainConfigs[chainID].DefaultBlock, chainID)
			lastBlock = cfg.ChainConfigs[chainID].DefaultBlock
		}
		s.intentProgress[chainID] = lastBlock
		log.Printf("Setting intent progress for chain %d to block %d", chainID, lastBlock)
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
		log.Printf("Current block for chain %d: %d", chainID, currentBlock)
	}

	// Track any errors that occur during catchup
	var catchupErrors []error

	// INTENT CATCHUP
	log.Printf("Starting intent event catchup")
	if err := s.runIntentCatchup(ctx, cfg, currentBlocks); err != nil {
		// Store the error but continue with fulfillment and settlement catchup
		catchupErrors = append(catchupErrors, fmt.Errorf("intent catchup failed: %v", err))
		log.Printf("WARNING: Intent catchup encountered errors: %v, continuing with fulfillment catchup", err)
	} else {
		log.Printf("All intent services have completed catchup successfully")
	}

	// FULFILLMENT CATCHUP
	log.Printf("Starting fulfillment catchup")
	if err := s.runFulfillmentCatchup(ctx, cfg, currentBlocks); err != nil {
		// Store the error but continue with settlement catchup
		catchupErrors = append(catchupErrors, fmt.Errorf("fulfillment catchup failed: %v", err))
		log.Printf("WARNING: Fulfillment catchup encountered errors: %v, continuing with settlement catchup", err)
	} else {
		log.Printf("All fulfillment services have completed catchup successfully")
	}

	// SETTLEMENT CATCHUP
	log.Printf("Starting settlement catchup")
	if err := s.runSettlementCatchup(ctx, cfg, currentBlocks); err != nil {
		// Store the error
		catchupErrors = append(catchupErrors, fmt.Errorf("settlement catchup failed: %v", err))
		log.Printf("WARNING: Settlement catchup encountered errors: %v", err)
	} else {
		log.Printf("All settlement services have completed catchup successfully")
	}

	// Only attempt to update processed blocks for chains that completed successfully
	for chainID, currentBlock := range currentBlocks {
		updateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		if err := s.db.UpdateLastProcessedBlock(updateCtx, chainID, currentBlock); err != nil {
			log.Printf("WARNING: Failed to update last processed block for chain %d: %v", chainID, err)
			// Don't return an error here, just log the warning
		} else {
			log.Printf("Updated last processed block for chain %d to %d", chainID, currentBlock)
		}
		cancel()
	}

	// Start live subscriptions for all services
	if err := s.StartLiveEventListeners(ctx, cfg); err != nil {
		catchupErrors = append(catchupErrors, fmt.Errorf("failed to start live subscriptions: %v", err))
		log.Printf("WARNING: Failed to start some live subscriptions: %v", err)
	}

	// If there were any errors during the catchup process, log them but don't fail
	if len(catchupErrors) > 0 {
		log.Printf("Catchup process completed with %d errors:", len(catchupErrors))
		for i, err := range catchupErrors {
			log.Printf("Catchup error %d: %v", i+1, err)
		}
		// Still return an error if there were any issues, but services will be running
		return fmt.Errorf("catchup process completed with %d errors", len(catchupErrors))
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

			log.Printf("CATCHUP STATUS: %d active operations", activeOps)
			if activeOps > 0 {
				log.Printf("Active operations: %v", activeList)
			}

			// Log intent service goroutines if available
			for chainID, service := range s.intentServices {
				if service != nil {
					activeGoroutines := service.ActiveGoroutines()
					log.Printf("Intent service for chain %d: %d active goroutines",
						chainID, activeGoroutines)
				}
			}
		case <-ctx.Done():
			log.Printf("Stopping catchup monitoring")
			return
		}
	}
}

// trackCatchupOperation adds an operation to the active operations map
func (s *EventCatchupService) trackCatchupOperation(operation string) {
	s.catchupMu.Lock()
	defer s.catchupMu.Unlock()
	s.activeCatchups[operation] = true
	log.Printf("Starting catchup operation: %s", operation)
}

// untrackCatchupOperation removes an operation from the active operations map
func (s *EventCatchupService) untrackCatchupOperation(operation string) {
	s.catchupMu.Lock()
	defer s.catchupMu.Unlock()
	delete(s.activeCatchups, operation)
	log.Printf("Completed catchup operation: %s", operation)
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
			log.Printf("No missed events to process for chain %d", chainID)
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

			log.Printf("Starting intent event catch-up for chain %d (blocks %d to %d)",
				chainID, lastBlock+1, currentBlock)

			// Create a timeout context for this specific chain's catchup
			chainCtx, chainCancel := context.WithTimeout(catchupCtx, CatchupOperationTimeout)
			defer chainCancel()

			if err := s.catchUpOnIntentEvents(chainCtx, intentService, contractAddress, lastBlock, currentBlock, opName); err != nil {
				intentErrors <- fmt.Errorf("failed to catch up on intent events for chain %d: %v", chainID, err)
				log.Printf("ERROR: Intent catchup for chain %d failed: %v", chainID, err)
				return
			}

			// Update progress
			s.UpdateIntentProgress(chainID, currentBlock)
			log.Printf("Completed intent event catch-up for chain %d", chainID)
		}(chainID, intentService, lastBlock, currentBlock, opName)
	}

	// If there are no chains to process, we can return early
	if chainsToProcess == 0 {
		log.Printf("No intent catchup needed for any chain")
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
				log.Printf("Intent catchup error: %v", err)
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
			log.Printf("No missed fulfillment events to process for chain %d", chainID)
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

			log.Printf("Starting fulfillment event catch-up for chain %d (blocks %d to %d)",
				chainID, lastBlock+1, currentBlock)

			// Create a timeout context for this specific chain's catchup
			chainCtx, chainCancel := context.WithTimeout(catchupCtx, CatchupOperationTimeout)
			defer chainCancel()

			if err := s.catchUpOnFulfillmentEvents(chainCtx, fulfillmentService, contractAddress, lastBlock, currentBlock, opName); err != nil {
				fulfillmentErrors <- fmt.Errorf("failed to catch up on fulfillment events for chain %d: %v", chainID, err)
				log.Printf("ERROR: Fulfillment catchup for chain %d failed: %v", chainID, err)
				return
			}

			// Update progress
			s.UpdateFulfillmentProgress(chainID, currentBlock)
			log.Printf("Completed fulfillment event catch-up for chain %d", chainID)
		}(chainID, fulfillmentService, lastBlock, currentBlock, opName)
	}

	// If there are no chains to process, we can return early
	if chainsToProcess == 0 {
		log.Printf("No fulfillment catchup needed for any chain")
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
				log.Printf("Fulfillment catchup error: %v", err)
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
			log.Printf("No missed settlement events to process for chain %d", chainID)
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

			log.Printf("Starting settlement event catch-up for chain %d (blocks %d to %d)",
				chainID, lastBlock+1, currentBlock)

			// Create a timeout context for this specific chain's catchup
			chainCtx, chainCancel := context.WithTimeout(catchupCtx, CatchupOperationTimeout)
			defer chainCancel()

			if err := s.catchUpOnSettlementEvents(chainCtx, settlementService, contractAddress, lastBlock, currentBlock, opName); err != nil {
				settlementErrors <- fmt.Errorf("failed to catch up on settlement events for chain %d: %v", chainID, err)
				log.Printf("ERROR: Settlement catchup for chain %d failed: %v", chainID, err)
				return
			}

			// Update progress
			s.UpdateSettlementProgress(chainID, currentBlock)
			log.Printf("Completed settlement event catch-up for chain %d", chainID)
		}(chainID, settlementService, lastBlock, currentBlock, opName)
	}

	// If there are no chains to process, we can return early
	if chainsToProcess == 0 {
		log.Printf("No settlement catchup needed for any chain")
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
				log.Printf("Settlement catchup error: %v", err)
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
	log.Printf("Starting live intent event listeners")
	for chainID, intentService := range s.intentServices {
		chainID := chainID // Create a copy of the loop variable for the closure
		intentService := intentService

		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)
		log.Printf("Starting intent event listener for chain %d at contract %s",
			chainID, contractAddress.Hex())

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
				log.Printf("WARNING: Unable to get current block for chain %d: %v", chainID, err)
			} else {
				fromBlock = currentBlock
				log.Printf("No stored progress found, setting intent listener for chain %d to start from current block %d", chainID, fromBlock)
			}
		}
		s.mu.Unlock()

		// Special handling for ZetaChain - use polling instead of subscription
		if chainID == 7000 {
			// Store the initial block to start polling from
			s.mu.Lock()
			s.intentProgress[chainID] = fromBlock
			s.mu.Unlock()

			log.Printf("Setting up polling-based event monitoring for ZetaChain (chain ID %d) starting from block %d",
				chainID, fromBlock)

			// Start polling goroutine
			go s.pollZetachainEvents(ctx, intentService, contractAddress, cfg.ChainConfigs[chainID].BlockInterval)
			continue
		}

		intentQuery := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{intentService.abi.Events[IntentInitiatedEventName].ID},
			},
		}

		// Set FromBlock explicitly to avoid processing old events
		if fromBlock > 0 {
			intentQuery.FromBlock = big.NewInt(int64(fromBlock))
		}

		log.Printf("Intent subscription filter for chain %d: FromBlock=%v, Addresses=%s, Topics=%v",
			chainID, intentQuery.FromBlock, contractAddress.Hex(), intentQuery.Topics[0][0].Hex())

		intentLogs := make(chan types.Log)
		// Create a context that remains alive for the subscription
		// Don't create a timeout context for the subscription as it needs to run indefinitely
		intentSub, err := intentService.client.SubscribeFilterLogs(ctx, intentQuery, intentLogs)
		if err != nil {
			return fmt.Errorf("failed to subscribe to intent logs for chain %d: %v", chainID, err)
		}
		log.Printf("Successfully subscribed to intent events for chain %d", chainID)

		// Add monitoring goroutine for subscription errors
		go func() {
			for {
				select {
				case err := <-intentSub.Err():
					if err != nil {
						log.Printf("ERROR: Intent subscription for chain %d encountered an error: %v", chainID, err)
						// Try to resubscribe
						resubCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
						newSub, resubErr := intentService.client.SubscribeFilterLogs(resubCtx, intentQuery, intentLogs)
						cancel()

						if resubErr != nil {
							log.Printf("CRITICAL: Failed to resubscribe intent listener for chain %d: %v", chainID, resubErr)
						} else {
							intentSub = newSub
							log.Printf("Successfully resubscribed intent listener for chain %d", chainID)
						}
					}
				case <-ctx.Done():
					log.Printf("Intent subscription monitor for chain %d shutting down", chainID)
					return
				}
			}
		}()

		go intentService.processEventLogs(ctx, intentSub, intentLogs, fmt.Sprintf("chain_%d", chainID))
	}

	// Start fulfillment listeners with similar block tracking
	log.Printf("Starting live fulfillment event listeners")
	for chainID, fulfillmentService := range s.fulfillmentServices {
		chainID := chainID // Create a copy of the loop variable for the closure
		fulfillmentService := fulfillmentService

		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		// For live subscriptions, use the last processed block + 1 as the starting point
		var fromBlock uint64
		s.mu.Lock()
		if lastBlock, exists := s.fulfillmentProgress[chainID]; exists && lastBlock > 0 {
			fromBlock = lastBlock + 1
			log.Printf("Setting fulfillment listener for chain %d to start from block %d", chainID, fromBlock)
		} else {
			// If we don't have a stored last block, get the current one
			blockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			currentBlock, err := fulfillmentService.client.BlockNumber(blockCtx)
			cancel()
			if err != nil {
				log.Printf("WARNING: Unable to get current block for chain %d: %v", chainID, err)
			} else {
				fromBlock = currentBlock
				log.Printf("No stored progress found, setting fulfillment listener for chain %d to start from current block %d", chainID, fromBlock)
			}
		}
		s.mu.Unlock()

		// Special handling for ZetaChain - use polling instead of subscription
		if chainID == 7000 {
			// Store the initial block to start polling from
			s.mu.Lock()
			s.fulfillmentProgress[chainID] = fromBlock
			s.mu.Unlock()

			log.Printf("Setting up polling-based fulfillment monitoring for ZetaChain (chain ID %d) starting from block %d",
				chainID, fromBlock)

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

		log.Printf("Fulfillment subscription filter for chain %d: FromBlock=%v, Addresses=%s, Topics=%v",
			chainID, fulfillmentQuery.FromBlock, contractAddress.Hex(), fulfillmentQuery.Topics[0][0].Hex())

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
						log.Printf("ERROR: Fulfillment subscription for chain %d encountered an error: %v", chainID, err)
						// Try to resubscribe
						resubCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
						newSub, resubErr := fulfillmentService.client.SubscribeFilterLogs(resubCtx, fulfillmentQuery, fulfillmentLogs)
						cancel()

						if resubErr != nil {
							log.Printf("CRITICAL: Failed to resubscribe fulfillment listener for chain %d: %v", chainID, resubErr)
						} else {
							fulfillmentSub = newSub
							log.Printf("Successfully resubscribed fulfillment listener for chain %d", chainID)
						}
					}
				case <-ctx.Done():
					log.Printf("Fulfillment subscription monitor for chain %d shutting down", chainID)
					return
				}
			}
		}()

		go fulfillmentService.processEventLogs(ctx, fulfillmentSub, fulfillmentLogs, contractAddress.Hex())
	}

	// Start settlement listeners with similar block tracking
	log.Printf("Starting live settlement event listeners")
	for chainID, settlementService := range s.settlementServices {
		chainID := chainID // Create a copy of the loop variable for the closure
		settlementService := settlementService

		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		// For live subscriptions, use the last processed block + 1 as the starting point
		var fromBlock uint64
		s.mu.Lock()
		if lastBlock, exists := s.settlementProgress[chainID]; exists && lastBlock > 0 {
			fromBlock = lastBlock + 1
			log.Printf("Setting settlement listener for chain %d to start from block %d", chainID, fromBlock)
		} else {
			// If we don't have a stored last block, get the current one
			blockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			currentBlock, err := settlementService.client.BlockNumber(blockCtx)
			cancel()
			if err != nil {
				log.Printf("WARNING: Unable to get current block for chain %d: %v", chainID, err)
			} else {
				fromBlock = currentBlock
				log.Printf("No stored progress found, setting settlement listener for chain %d to start from current block %d", chainID, fromBlock)
			}
		}
		s.mu.Unlock()

		// Special handling for ZetaChain - use polling instead of subscription
		if chainID == 7000 {
			// Store the initial block to start polling from
			s.mu.Lock()
			s.settlementProgress[chainID] = fromBlock
			s.mu.Unlock()

			log.Printf("Setting up polling-based settlement monitoring for ZetaChain (chain ID %d) starting from block %d",
				chainID, fromBlock)

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

		log.Printf("Settlement subscription filter for chain %d: FromBlock=%v, Addresses=%s, Topics=%v",
			chainID, settlementQuery.FromBlock, contractAddress.Hex(), settlementQuery.Topics[0][0].Hex())

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
						log.Printf("ERROR: Settlement subscription for chain %d encountered an error: %v", chainID, err)
						// Try to resubscribe
						resubCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
						newSub, resubErr := settlementService.client.SubscribeFilterLogs(resubCtx, settlementQuery, settlementLogs)
						cancel()

						if resubErr != nil {
							log.Printf("CRITICAL: Failed to resubscribe settlement listener for chain %d: %v", chainID, resubErr)
						} else {
							settlementSub = newSub
							log.Printf("Successfully resubscribed settlement listener for chain %d", chainID)
						}
					}
				case <-ctx.Done():
					log.Printf("Settlement subscription monitor for chain %d shutting down", chainID)
					return
				}
			}
		}()

		go settlementService.processEventLogs(ctx, settlementSub, settlementLogs, contractAddress.Hex(), contractAddress)
	}

	log.Printf("All live event listeners started successfully")
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
				log.Printf("WARNING: Failed to check sample block %d for events: %v, will process full range", blockNum, err)
				return true, nil // Error getting logs, so process the range to be safe
			}

			// If we found any logs in the sample block, there might be more in the range
			if len(logs) > 0 {
				return true, nil
			}
		}
	}

	// If we didn't find any logs in the sample blocks, we'll skip this range
	log.Printf("Sample block check: no events found in sample blocks, likely no events in range %d-%d, skipping",
		startBlock, endBlock)
	return false, nil
}

// Modify the intent events function to use bloom filtering on Ethereum
func (s *EventCatchupService) catchUpOnIntentEvents(ctx context.Context, intentService *IntentService, contractAddress common.Address, fromBlock, toBlock uint64, opName string) error {
	// Use a chain-specific block range - smaller for Ethereum mainnet
	var maxBlockRange uint64
	if intentService.chainID == 1 { // Ethereum mainnet
		maxBlockRange = EthereumMaxBlockRange
		log.Printf("[%s] Using smaller block range of %d for Ethereum mainnet", opName, maxBlockRange)
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
				log.Printf("[%s] Error in bloom check: %v, will process range", opName, err)
			} else if !hasEvents {
				// Skip this chunk as it likely has no events
				log.Printf("[%s] Fast-forwarding through block range %d-%d (no events detected)",
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

			log.Printf("[%s] Fetching intent logs for blocks %d to %d", opName, chunkStart+1, chunkEnd)

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

			log.Printf("[%s] Processing %d logs from blocks %d to %d", opName, len(logs), chunkStart+1, chunkEnd)

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

						log.Printf("[%s] Processing intent log %d/%d: Block=%d, TxHash=%s",
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
							log.Printf("[%s] Skipping existing intent: %s", opName, intentID)
							continue
						}

						// Process log with timeout
						processCtx, processCancel := context.WithTimeout(batchCtx, 20*time.Second)
						err = intentService.processLog(processCtx, txlog)
						processCancel()

						if err != nil {
							// Skip if intent already exists
							if strings.Contains(err.Error(), "duplicate key") {
								log.Printf("[%s] Skipping duplicate intent: %s", opName, intentID)
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
					log.Printf("[%s] Updated progress for chain %d to block %d", opName, intentService.chainID, lastBlock)
				}
			}

			// Update progress after processing each chunk
			s.UpdateIntentProgress(intentService.chainID, chunkEnd)

			// Persist progress to the database after each chunk
			dbUpdateCtx, dbUpdateCancel := context.WithTimeout(ctx, 10*time.Second)
			dbErr := s.db.UpdateLastProcessedBlock(dbUpdateCtx, intentService.chainID, chunkEnd)
			dbUpdateCancel()
			if dbErr != nil {
				log.Printf("[%s] Warning: Failed to persist progress to DB: %v", opName, dbErr)
				// Continue processing even if DB update fails
			} else {
				log.Printf("[%s] Persisted progress to DB: chain %d at block %d", opName, intentService.chainID, chunkEnd)
			}

			log.Printf("[%s] Completed processing intent logs for blocks %d to %d", opName, chunkStart+1, chunkEnd)

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
		log.Printf("[%s] Using smaller block range of %d for Ethereum mainnet", opName, maxBlockRange)
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

			log.Printf("[%s] Fetching fulfillment logs for blocks %d to %d", opName, chunkStart+1, chunkEnd)

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

			log.Printf("[%s] Processing %d logs from blocks %d to %d", opName, len(logs), chunkStart+1, chunkEnd)

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

						log.Printf("[%s] Processing fulfillment log %d/%d: Block=%d, TxHash=%s",
							opName, i+j+1, len(logs), txlog.BlockNumber, txlog.TxHash.Hex())

						// Extract intent ID from the log
						intentID := txlog.Topics[1].Hex()

						// Check if intent exists (fulfillments need an intent)
						getIntentCtx, cancel := context.WithTimeout(batchCtx, 10*time.Second)
						_, err := s.db.GetIntent(getIntentCtx, intentID)
						cancel()

						if err != nil {
							if strings.Contains(err.Error(), "not found") {
								log.Printf("[%s] Skipping fulfillment for non-existent intent: %s", opName, intentID)
								continue
							}
							log.Printf("[%s] Failed to check for existing intent: %v", opName, err)
							continue
						}

						// Process log with timeout
						processCtx, processCancel := context.WithTimeout(batchCtx, 20*time.Second)
						err = fulfillmentService.processLog(processCtx, txlog)
						processCancel()

						if err != nil {
							// Skip if fulfillment already exists
							if strings.Contains(err.Error(), "duplicate key") {
								log.Printf("[%s] Skipping duplicate fulfillment: %s", opName, intentID)
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
					log.Printf("[%s] Updated progress for chain %d to block %d", opName, fulfillmentService.chainID, lastBlock)
				}
			}

			// Update progress after processing each chunk
			s.UpdateFulfillmentProgress(fulfillmentService.chainID, chunkEnd)

			// Persist progress to the database after each chunk
			dbUpdateCtx, dbUpdateCancel := context.WithTimeout(ctx, 10*time.Second)
			dbErr := s.db.UpdateLastProcessedBlock(dbUpdateCtx, fulfillmentService.chainID, chunkEnd)
			dbUpdateCancel()
			if dbErr != nil {
				log.Printf("[%s] Warning: Failed to persist progress to DB: %v", opName, dbErr)
				// Continue processing even if DB update fails
			} else {
				log.Printf("[%s] Persisted progress to DB: chain %d at block %d", opName, fulfillmentService.chainID, chunkEnd)
			}

			log.Printf("[%s] Completed processing fulfillment logs for blocks %d to %d", opName, chunkStart+1, chunkEnd)

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
		log.Printf("[%s] Using smaller block range of %d for Ethereum mainnet", opName, maxBlockRange)
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

			log.Printf("[%s] Fetching settlement logs for blocks %d to %d", opName, chunkStart+1, chunkEnd)

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

			log.Printf("[%s] Processing %d logs from blocks %d to %d", opName, len(logs), chunkStart+1, chunkEnd)

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

						log.Printf("[%s] Processing settlement log %d/%d: Block=%d, TxHash=%s",
							opName, i+j+1, len(logs), txlog.BlockNumber, txlog.TxHash.Hex())

						// Extract intent ID from the log
						intentID := txlog.Topics[1].Hex()

						// Check if intent exists (settlements need an intent)
						getIntentCtx, cancel := context.WithTimeout(batchCtx, 10*time.Second)
						_, err := s.db.GetIntent(getIntentCtx, intentID)
						cancel()

						if err != nil {
							if strings.Contains(err.Error(), "not found") {
								log.Printf("[%s] Skipping settlement for non-existent intent: %s", opName, intentID)
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
								log.Printf("[%s] Skipping duplicate settlement: %s", opName, intentID)
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
					log.Printf("[%s] Updated progress for chain %d to block %d", opName, settlementService.chainID, lastBlock)
				}
			}

			// Update progress after processing each chunk
			s.UpdateSettlementProgress(settlementService.chainID, chunkEnd)

			// Persist progress to the database after each chunk
			dbUpdateCtx, dbUpdateCancel := context.WithTimeout(ctx, 10*time.Second)
			dbErr := s.db.UpdateLastProcessedBlock(dbUpdateCtx, settlementService.chainID, chunkEnd)
			dbUpdateCancel()
			if dbErr != nil {
				log.Printf("[%s] Warning: Failed to persist progress to DB: %v", opName, dbErr)
				// Continue processing even if DB update fails
			} else {
				log.Printf("[%s] Persisted progress to DB: chain %d at block %d", opName, settlementService.chainID, chunkEnd)
			}

			log.Printf("[%s] Completed processing settlement logs for blocks %d to %d", opName, chunkStart+1, chunkEnd)

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
	// Use the generic polling function
	s.pollChainEvents(ctx, "intent", 7000, intentService.client, contractAddress,
		[]common.Hash{intentService.abi.Events[IntentInitiatedEventName].ID, intentService.abi.Events[IntentInitiatedWithCallEventName].ID},
		intentService.processLog,
		func(blockNum uint64) { s.UpdateIntentProgress(7000, blockNum) },
		blockInterval)
}

// pollZetachainFulfillmentEvents polls for events on ZetaChain at regular intervals instead of using WebSocket subscription
func (s *EventCatchupService) pollZetachainFulfillmentEvents(ctx context.Context, fulfillmentService *FulfillmentService, contractAddress common.Address, blockInterval int64) {
	// Use the generic polling function
	s.pollChainEvents(ctx, "fulfillment", 7000, fulfillmentService.client, contractAddress,
		[]common.Hash{fulfillmentService.abi.Events[IntentFulfilledEventName].ID, fulfillmentService.abi.Events[IntentFulfilledWithCallEventName].ID},
		fulfillmentService.processLog,
		func(blockNum uint64) { s.UpdateFulfillmentProgress(7000, blockNum) },
		blockInterval)
}

// pollZetachainSettlementEvents polls for events on ZetaChain at regular intervals instead of using WebSocket subscription
func (s *EventCatchupService) pollZetachainSettlementEvents(ctx context.Context, settlementService *SettlementService, contractAddress common.Address, blockInterval int64) {
	// Use the generic polling function
	s.pollChainEvents(ctx, "settlement", 7000, settlementService.client, contractAddress,
		[]common.Hash{settlementService.abi.Events[IntentSettledEventName].ID, settlementService.abi.Events[IntentSettledWithCallEventName].ID},
		settlementService.processLog,
		func(blockNum uint64) { s.UpdateSettlementProgress(7000, blockNum) },
		blockInterval)
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
) {
	// Default to checking every 15 seconds if not specified in config
	interval := time.Duration(blockInterval) * time.Second
	if interval < 5*time.Second {
		interval = 15 * time.Second
	}

	maxRetries := 3
	baseRetryDelay := 5 * time.Second

	log.Printf("Starting ZetaChain polling for %s events with interval of %v", eventType, interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Track database persistence to avoid doing it every poll
	lastDbUpdateTime := time.Now()
	dbUpdateInterval := 5 * time.Minute

	for {
		select {
		case <-ctx.Done():
			log.Printf("Context cancelled, stopping ZetaChain %s event polling", eventType)
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
				log.Printf("ERROR: Failed to get current block for ZetaChain (attempt %d/%d): %v. Retrying in %v",
					retry+1, maxRetries, err, retryDelay)

				select {
				case <-time.After(retryDelay):
					continue
				case <-ctx.Done():
					return
				}
			}

			if err != nil {
				log.Printf("CRITICAL: Failed to get current block for ZetaChain after %d attempts. Skipping this polling cycle.", maxRetries)
				continue
			}

			// Skip if no new blocks
			if currentBlock <= lastProcessedBlock {
				if time.Since(lastDbUpdateTime) >= dbUpdateInterval {
					// Even if no new blocks, periodically update the DB to ensure we don't lose progress
					dbUpdateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
					if err := s.db.UpdateLastProcessedBlock(dbUpdateCtx, chainID, lastProcessedBlock); err != nil {
						log.Printf("WARNING: Failed to persist %s progress to DB: %v", eventType, err)
					} else {
						log.Printf("Persisted %s progress to DB: chain %d at block %d", eventType, chainID, lastProcessedBlock)
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

			log.Printf("Polling ZetaChain for %s events from blocks %d to %d", eventType, lastProcessedBlock+1, endBlock)

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
				log.Printf("ERROR: Failed to filter logs for ZetaChain %s events (attempt %d/%d): %v. Retrying in %v",
					eventType, retry+1, maxRetries, err, retryDelay)

				select {
				case <-time.After(retryDelay):
					continue
				case <-ctx.Done():
					return
				}
			}

			if err != nil {
				log.Printf("CRITICAL: Failed to filter logs for ZetaChain %s events after %d attempts. Skipping this block range.",
					eventType, maxRetries)
				continue
			}

			// Process logs if any found
			processedCount := 0
			errorCount := 0
			if len(logs) > 0 {
				log.Printf("Found %d new %s events in ZetaChain blocks %d to %d",
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
							log.Printf("DEBUG: Skipping duplicate %s event in tx: %s", eventType, logEntry.TxHash.Hex())
						} else {
							log.Printf("ERROR: Failed to process ZetaChain %s log: %v", eventType, err)
						}
					} else {
						processedCount++
					}
				}
				log.Printf("Successfully processed %d/%d %s events (errors: %d)",
					processedCount, len(logs), eventType, errorCount)
			} else {
				log.Printf("No new %s events found in ZetaChain blocks %d to %d",
					eventType, lastProcessedBlock+1, endBlock)
			}

			// Update the last processed block
			updateProgressFunc(endBlock)

			// Persist progress to the database
			dbUpdateCtx, dbUpdateCancel := context.WithTimeout(ctx, 10*time.Second)
			if err := s.db.UpdateLastProcessedBlock(dbUpdateCtx, chainID, endBlock); err != nil {
				log.Printf("WARNING: Failed to persist %s progress to DB: %v", eventType, err)
			} else {
				log.Printf("Persisted %s progress to DB: chain %d at block %d", eventType, chainID, endBlock)
				lastDbUpdateTime = time.Now()
			}
			dbUpdateCancel()
		}
	}
}

// StartSubscriptionSupervisor starts a background goroutine that periodically checks
// if services are still running and restarts them if needed
func (s *EventCatchupService) StartSubscriptionSupervisor(ctx context.Context, cfg *config.Config) {
	log.Printf("Starting subscription supervisor to monitor service health")

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
			log.Printf("Subscription supervisor checking service health...")

			// Check intent services
			for chainID, intentService := range s.intentServices {
				// Skip health check for ZetaChain as it's using polling
				if chainID == zetaChainID {
					log.Printf("ZetaChain intent service using polling mechanism - skipping subscription check")
					continue
				}

				activeGoroutines := intentService.ActiveGoroutines()
				log.Printf("Intent service for chain %d: %d active goroutines", chainID, activeGoroutines)

				if activeGoroutines == 0 {
					log.Printf("WARNING: Intent service for chain %d has 0 active goroutines, restarting", chainID)
					contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

					// Create a context with timeout for restart
					restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					err := intentService.StartListening(restartCtx, contractAddress)
					if err != nil {
						log.Printf("ERROR: Failed to restart intent service for chain %d: %v", chainID, err)
					} else {
						log.Printf("RECOVERY: Successfully restarted intent service for chain %d", chainID)
					}
					cancel()
				}
			}

			// Check fulfillment services
			for chainID, fulfillmentService := range s.fulfillmentServices {
				// Skip health check for ZetaChain as it's using polling
				if chainID == zetaChainID {
					log.Printf("ZetaChain fulfillment service using polling mechanism - skipping subscription check")
					continue
				}

				count := fulfillmentService.GetSubscriptionCount()
				log.Printf("Fulfillment service for chain %d: %d active subscriptions", chainID, count)

				if count == 0 {
					log.Printf("WARNING: Fulfillment service for chain %d has no active subscriptions, restarting", chainID)
					contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

					// Create a context with timeout for restart
					restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					err := fulfillmentService.StartListening(restartCtx, contractAddress)
					if err != nil {
						log.Printf("ERROR: Failed to restart fulfillment service for chain %d: %v", chainID, err)
					} else {
						log.Printf("RECOVERY: Successfully restarted fulfillment service for chain %d", chainID)
					}
					cancel()
				}
			}

			// Check settlement services
			for chainID, settlementService := range s.settlementServices {
				// Skip health check for ZetaChain as it's using polling
				if chainID == zetaChainID {
					log.Printf("ZetaChain settlement service using polling mechanism - skipping subscription check")
					continue
				}

				count := settlementService.GetSubscriptionCount()
				log.Printf("Settlement service for chain %d: %d active subscriptions", chainID, count)

				if count == 0 {
					log.Printf("WARNING: Settlement service for chain %d has no active subscriptions, restarting", chainID)
					contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

					// Create a context with timeout for restart
					restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					err := settlementService.StartListening(restartCtx, contractAddress)
					if err != nil {
						log.Printf("ERROR: Failed to restart settlement service for chain %d: %v", chainID, err)
					} else {
						log.Printf("RECOVERY: Successfully restarted settlement service for chain %d", chainID)
					}
					cancel()
				}
			}

			// Check ZetaChain health by getting block number
			if client, ok := s.intentServices[zetaChainID]; ok && client != nil {
				log.Printf("Checking ZetaChain polling health...")
				blockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				_, err := client.client.BlockNumber(blockCtx)
				cancel()

				if err != nil {
					log.Printf("WARNING: ZetaChain polling health check failed: %v", err)
				} else {
					log.Printf("ZetaChain polling health check passed")
				}
			}

		case <-reconnectTicker.C:
			// Perform a complete refresh of all WebSocket connections every 2 hours
			timeSinceLastReconnect := time.Since(lastFullReconnect)
			log.Printf("Performing scheduled full reconnection of all services (last reconnect: %v ago)", timeSinceLastReconnect)
			lastFullReconnect = time.Now()

			// Force reconnect all intent services (except ZetaChain)
			for chainID, intentService := range s.intentServices {
				// Skip ZetaChain as it's using polling
				if chainID == zetaChainID {
					log.Printf("Skipping ZetaChain intent service reconnection (using polling)")
					continue
				}

				log.Printf("Scheduled reconnect: Restarting intent service for chain %d", chainID)
				contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

				// First, unsubscribe from all existing subscriptions
				intentService.UnsubscribeAll()

				// Create a context with timeout for restart
				restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				err := intentService.StartListening(restartCtx, contractAddress)
				if err != nil {
					log.Printf("ERROR: Failed to reconnect intent service for chain %d: %v", chainID, err)
				} else {
					log.Printf("Scheduled reconnect: Successfully reconnected intent service for chain %d", chainID)
				}
				cancel()
			}

			// Force reconnect all fulfillment services (except ZetaChain)
			for chainID, fulfillmentService := range s.fulfillmentServices {
				// Skip ZetaChain as it's using polling
				if chainID == zetaChainID {
					log.Printf("Skipping ZetaChain fulfillment service reconnection (using polling)")
					continue
				}

				log.Printf("Scheduled reconnect: Restarting fulfillment service for chain %d", chainID)
				contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

				// First, unsubscribe from all existing subscriptions
				fulfillmentService.UnsubscribeAll()

				// Create a context with timeout for restart
				restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				err := fulfillmentService.StartListening(restartCtx, contractAddress)
				if err != nil {
					log.Printf("ERROR: Failed to reconnect fulfillment service for chain %d: %v", chainID, err)
				} else {
					log.Printf("Scheduled reconnect: Successfully reconnected fulfillment service for chain %d", chainID)
				}
				cancel()
			}

			// Force reconnect all settlement services (except ZetaChain)
			for chainID, settlementService := range s.settlementServices {
				// Skip ZetaChain as it's using polling
				if chainID == zetaChainID {
					log.Printf("Skipping ZetaChain settlement service reconnection (using polling)")
					continue
				}

				log.Printf("Scheduled reconnect: Restarting settlement service for chain %d", chainID)
				contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

				// First, unsubscribe from all existing subscriptions
				settlementService.UnsubscribeAll()

				// Create a context with timeout for restart
				restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				err := settlementService.StartListening(restartCtx, contractAddress)
				if err != nil {
					log.Printf("ERROR: Failed to reconnect settlement service for chain %d: %v", chainID, err)
				} else {
					log.Printf("Scheduled reconnect: Successfully reconnected settlement service for chain %d", chainID)
				}
				cancel()
			}

		case <-ctx.Done():
			log.Printf("Subscription supervisor shutting down")
			return
		}
	}
}
