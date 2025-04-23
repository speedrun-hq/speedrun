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
	LogBatchProcessTimeout = 2 * time.Minute

	// MonitoringInterval is how often to log the status of ongoing operations
	MonitoringInterval = 30 * time.Second
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

	// INTENT CATCHUP
	log.Printf("Starting intent event catchup")
	if err := s.runIntentCatchup(ctx, cfg, currentBlocks); err != nil {
		return fmt.Errorf("intent catchup failed: %v", err)
	}
	log.Printf("All intent services have completed catchup")

	// FULFILLMENT CATCHUP
	log.Printf("Starting fulfillment catchup")
	if err := s.runFulfillmentCatchup(ctx, cfg, currentBlocks); err != nil {
		return fmt.Errorf("fulfillment catchup failed: %v", err)
	}
	log.Printf("All fulfillment services have completed catchup")

	// SETTLEMENT CATCHUP
	log.Printf("Starting settlement catchup")
	if err := s.runSettlementCatchup(ctx, cfg, currentBlocks); err != nil {
		return fmt.Errorf("settlement catchup failed: %v", err)
	}
	log.Printf("All settlement services have completed catchup")

	// Update last processed blocks for all chains only after all services have completed
	for chainID, currentBlock := range currentBlocks {
		updateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		if err := s.db.UpdateLastProcessedBlock(updateCtx, chainID, currentBlock); err != nil {
			cancel()
			return fmt.Errorf("failed to update last processed block for chain %d: %v", chainID, err)
		}
		cancel()
	}

	// Start live subscriptions for all services
	if err := s.startLiveSubscriptions(ctx, cfg); err != nil {
		return err
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

// startLiveSubscriptions starts the live event listeners for all services
func (s *EventCatchupService) startLiveSubscriptions(ctx context.Context, cfg *config.Config) error {
	// Start intent listeners
	log.Printf("Starting live intent event listeners")
	for chainID, intentService := range s.intentServices {
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)
		log.Printf("Starting intent event listener for chain %d at contract %s",
			chainID, contractAddress.Hex())

		intentQuery := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{intentService.abi.Events[IntentInitiatedEventName].ID},
			},
		}

		intentLogs := make(chan types.Log)
		subCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		intentSub, err := intentService.client.SubscribeFilterLogs(subCtx, intentQuery, intentLogs)
		cancel()

		if err != nil {
			return fmt.Errorf("failed to subscribe to intent logs for chain %d: %v", chainID, err)
		}
		log.Printf("Successfully subscribed to intent events for chain %d", chainID)

		go intentService.processEventLogs(ctx, intentSub, intentLogs, fmt.Sprintf("chain_%d", chainID))
	}

	// Start fulfillment listeners
	log.Printf("Starting live fulfillment event listeners")
	for chainID, fulfillmentService := range s.fulfillmentServices {
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)
		fulfillmentQuery := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{fulfillmentService.abi.Events[IntentFulfilledEventName].ID},
			},
		}

		fulfillmentLogs := make(chan types.Log)
		subCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		fulfillmentSub, err := fulfillmentService.client.SubscribeFilterLogs(subCtx, fulfillmentQuery, fulfillmentLogs)
		cancel()

		if err != nil {
			return fmt.Errorf("failed to subscribe to fulfillment logs for chain %d: %v", chainID, err)
		}

		go fulfillmentService.processEventLogs(ctx, fulfillmentSub, fulfillmentLogs)
	}

	// Start settlement listeners
	log.Printf("Starting live settlement event listeners")
	for chainID, settlementService := range s.settlementServices {
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)
		settlementQuery := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{settlementService.abi.Events[IntentSettledEventName].ID},
			},
		}

		settlementLogs := make(chan types.Log)
		subCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		settlementSub, err := settlementService.client.SubscribeFilterLogs(subCtx, settlementQuery, settlementLogs)
		cancel()

		if err != nil {
			return fmt.Errorf("failed to subscribe to settlement logs for chain %d: %v", chainID, err)
		}

		go settlementService.processEventLogs(ctx, settlementSub, settlementLogs)
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

// catchUpOnIntentEvents processes missed intent events for a specific chain
func (s *EventCatchupService) catchUpOnIntentEvents(ctx context.Context, intentService *IntentService, contractAddress common.Address, fromBlock, toBlock uint64, opName string) error {
	// Use a max range of 5,000 blocks per query to stay well under the 10,000 limit
	const maxBlockRange = uint64(5000)

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

			log.Printf("[%s] Fetching intent logs for blocks %d to %d", opName, chunkStart+1, chunkEnd)

			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(int64(chunkStart + 1)),
				ToBlock:   big.NewInt(int64(chunkEnd)),
				Addresses: []common.Address{contractAddress},
				Topics: [][]common.Hash{
					{intentService.abi.Events[IntentInitiatedEventName].ID},
				},
			}

			// Add timeout for FilterLogs
			filterCtx, filterCancel := context.WithTimeout(chunkCtx, 60*time.Second)
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
	// Use a max range of 5,000 blocks per query to stay well under the 10,000 limit
	const maxBlockRange = uint64(5000)

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
					{fulfillmentService.abi.Events[IntentFulfilledEventName].ID},
				},
			}

			// Add timeout for FilterLogs
			filterCtx, filterCancel := context.WithTimeout(chunkCtx, 60*time.Second)
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
	// Use a max range of 5,000 blocks per query to stay well under the 10,000 limit
	const maxBlockRange = uint64(5000)

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
					{settlementService.abi.Events[IntentSettledEventName].ID},
				},
			}

			// Add timeout for FilterLogs
			filterCtx, filterCancel := context.WithTimeout(chunkCtx, 60*time.Second)
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
