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
	}
}

// StartListening starts the coordinated event listening process
func (s *EventCatchupService) StartListening(ctx context.Context) error {
	log.Println("Starting event catchup service")

	// Define timeouts and retry settings
	const (
		phaseTimeout     = 5 * time.Minute  // Max time to wait for each phase
		operationTimeout = 30 * time.Second // Max time for individual operations
		maxRetries       = 3                // Max retries for failed operations
		retryBackoff     = 5 * time.Second  // Backoff between retries
	)

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

		// Add timeout for database operation
		dbCtx, dbCancel := context.WithTimeout(ctx, operationTimeout)
		lastBlock, err := s.db.GetLastProcessedBlock(dbCtx, chainID)
		dbCancel()

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

	// Get current block numbers for all chains with retries
	currentBlocks := make(map[uint64]uint64)
	for chainID, intentService := range s.intentServices {
		var currentBlock uint64
		var blockErr error

		// Retry block number query with exponential backoff
		for attempt := 0; attempt < maxRetries; attempt++ {
			blockCtx, blockCancel := context.WithTimeout(ctx, operationTimeout)
			currentBlock, blockErr = intentService.client.BlockNumber(blockCtx)
			blockCancel()

			if blockErr == nil {
				break // Success
			}

			// If this is not the last attempt, wait before retrying
			if attempt < maxRetries-1 {
				retryDelay := retryBackoff * time.Duration(attempt+1)
				log.Printf("Failed to get current block for chain %d (attempt %d/%d): %v. Retrying in %v...",
					chainID, attempt+1, maxRetries, blockErr, retryDelay)
				time.Sleep(retryDelay)
			}
		}

		if blockErr != nil {
			log.Printf("ERROR: Failed to get current block for chain %d after %d attempts: %v",
				chainID, maxRetries, blockErr)
			continue // Skip this chain but continue with others
		}

		currentBlocks[chainID] = currentBlock
	}

	if len(currentBlocks) == 0 {
		return fmt.Errorf("failed to get current block number for any chain")
	}

	// INTENT PHASE
	log.Printf("Starting intent catch-up phase")

	// Create a wait group to track all intent catch-up goroutines
	var intentWg sync.WaitGroup
	intentErrors := make(chan error, len(s.intentServices))

	// Create a context with timeout for intent phase
	intentCtx, intentCancel := context.WithTimeout(ctx, phaseTimeout)
	defer intentCancel()

	// Create a done channel to signal completion
	intentDone := make(chan struct{})

	// Start intent catch-up for all chains in parallel
	for chainID, intentService := range s.intentServices {
		currentBlock, ok := currentBlocks[chainID]
		if !ok {
			log.Printf("Skipping intent catch-up for chain %d: couldn't get current block", chainID)
			continue
		}

		lastBlock := s.intentProgress[chainID]
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		if lastBlock >= currentBlock {
			log.Printf("No missed events to process for chain %d", chainID)
			continue
		}

		intentWg.Add(1)
		go func(chainID uint64, intentService *IntentService, lastBlock, currentBlock uint64) {
			defer intentWg.Done()

			log.Printf("Starting intent event catch-up for chain %d (blocks %d to %d)",
				chainID, lastBlock+1, currentBlock)

			if err := s.catchUpOnIntentEvents(intentCtx, intentService, contractAddress, lastBlock, currentBlock); err != nil {
				intentErrors <- fmt.Errorf("failed to catch up on intent events for chain %d: %v", chainID, err)
				return
			}

			// Update progress
			s.UpdateIntentProgress(chainID, currentBlock)
			log.Printf("Completed intent event catch-up for chain %d", chainID)
		}(chainID, intentService, lastBlock, currentBlock)
	}

	// Wait for all intent catch-ups to complete with timeout
	go func() {
		intentWg.Wait()
		close(intentDone)
	}()

	// Wait for completion or timeout
	select {
	case <-intentDone:
		log.Printf("All intent catch-up tasks completed")
	case <-intentCtx.Done():
		log.Printf("WARNING: Intent catch-up phase timed out after %v, proceeding to next phase anyway", phaseTimeout)
	}

	// Drain errors channel
	close(intentErrors)
	errCount := 0
	for err := range intentErrors {
		errCount++
		log.Printf("Error during intent catch-up: %v", err)
	}

	if errCount > 0 {
		log.Printf("WARNING: %d errors occurred during intent catch-up phase", errCount)
	}

	log.Printf("Intent catch-up phase completed, proceeding to fulfillment phase")

	// FULFILLMENT PHASE
	log.Printf("Starting fulfillment catch-up")

	// Initialize progress tracking for all chains
	s.mu.Lock()
	for chainID := range s.fulfillmentServices {
		dbCtx, dbCancel := context.WithTimeout(ctx, operationTimeout)
		lastBlock, err := s.db.GetLastProcessedBlock(dbCtx, chainID)
		dbCancel()

		if err != nil {
			s.mu.Unlock()
			log.Printf("WARNING: Failed to get last processed block for chain %d: %v", chainID, err)
			continue // Skip this chain but continue with others
		}
		if lastBlock < cfg.ChainConfigs[chainID].DefaultBlock {
			lastBlock = cfg.ChainConfigs[chainID].DefaultBlock
		}
		s.fulfillmentProgress[chainID] = lastBlock
	}
	s.mu.Unlock()

	// Create context with timeout for fulfillment phase
	fulfillmentCtx, fulfillmentCancel := context.WithTimeout(ctx, phaseTimeout)
	defer fulfillmentCancel()

	var fulfillmentWg sync.WaitGroup
	fulfillmentErrors := make(chan error, len(s.fulfillmentServices))
	fulfillmentDone := make(chan struct{})

	for chainID, fulfillmentService := range s.fulfillmentServices {
		currentBlock, ok := currentBlocks[chainID]
		if !ok {
			log.Printf("Skipping fulfillment catch-up for chain %d: couldn't get current block", chainID)
			continue
		}

		lastBlock := s.fulfillmentProgress[chainID]
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		if lastBlock >= currentBlock {
			log.Printf("No missed fulfillment events to process for chain %d", chainID)
			continue
		}

		fulfillmentWg.Add(1)
		go func(chainID uint64, fulfillmentService *FulfillmentService, lastBlock, currentBlock uint64) {
			defer fulfillmentWg.Done()

			log.Printf("Starting fulfillment event catch-up for chain %d (blocks %d to %d)",
				chainID, lastBlock+1, currentBlock)

			if err := s.catchUpOnFulfillmentEvents(fulfillmentCtx, fulfillmentService, contractAddress, lastBlock, currentBlock); err != nil {
				fulfillmentErrors <- fmt.Errorf("failed to catch up on fulfillment events for chain %d: %v", chainID, err)
				return
			}

			// Update progress
			s.UpdateFulfillmentProgress(chainID, currentBlock)
			log.Printf("Completed fulfillment event catch-up for chain %d", chainID)
		}(chainID, fulfillmentService, lastBlock, currentBlock)
	}

	// Wait for all fulfillment catch-ups to complete with timeout
	go func() {
		fulfillmentWg.Wait()
		close(fulfillmentDone)
	}()

	// Wait for completion or timeout
	select {
	case <-fulfillmentDone:
		log.Printf("All fulfillment catch-up tasks completed")
	case <-fulfillmentCtx.Done():
		log.Printf("WARNING: Fulfillment catch-up phase timed out after %v, proceeding to next phase anyway", phaseTimeout)
	}

	// Drain errors channel
	close(fulfillmentErrors)
	errCount = 0
	for err := range fulfillmentErrors {
		errCount++
		log.Printf("Error during fulfillment catch-up: %v", err)
	}

	if errCount > 0 {
		log.Printf("WARNING: %d errors occurred during fulfillment catch-up phase", errCount)
	}

	log.Printf("Fulfillment catch-up phase completed, proceeding to settlement phase")

	// SETTLEMENT PHASE
	log.Printf("Starting settlement catch-up")

	// Initialize progress tracking for all chains
	s.mu.Lock()
	for chainID := range s.settlementServices {
		dbCtx, dbCancel := context.WithTimeout(ctx, operationTimeout)
		lastBlock, err := s.db.GetLastProcessedBlock(dbCtx, chainID)
		dbCancel()

		if err != nil {
			s.mu.Unlock()
			log.Printf("WARNING: Failed to get last processed block for chain %d: %v", chainID, err)
			continue // Skip this chain but continue with others
		}
		if lastBlock < cfg.ChainConfigs[chainID].DefaultBlock {
			lastBlock = cfg.ChainConfigs[chainID].DefaultBlock
		}
		s.settlementProgress[chainID] = lastBlock
	}
	s.mu.Unlock()

	// Create context with timeout for settlement phase
	settlementCtx, settlementCancel := context.WithTimeout(ctx, phaseTimeout)
	defer settlementCancel()

	var settlementWg sync.WaitGroup
	settlementErrors := make(chan error, len(s.settlementServices))
	settlementDone := make(chan struct{})

	for chainID, settlementService := range s.settlementServices {
		currentBlock, ok := currentBlocks[chainID]
		if !ok {
			log.Printf("Skipping settlement catch-up for chain %d: couldn't get current block", chainID)
			continue
		}

		lastBlock := s.settlementProgress[chainID]
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)

		if lastBlock >= currentBlock {
			log.Printf("No missed settlement events to process for chain %d", chainID)
			continue
		}

		settlementWg.Add(1)
		go func(chainID uint64, settlementService *SettlementService, lastBlock, currentBlock uint64) {
			defer settlementWg.Done()

			log.Printf("Starting settlement event catch-up for chain %d (blocks %d to %d)",
				chainID, lastBlock+1, currentBlock)

			if err := s.catchUpOnSettlementEvents(settlementCtx, settlementService, contractAddress, lastBlock, currentBlock); err != nil {
				settlementErrors <- fmt.Errorf("failed to catch up on settlement events for chain %d: %v", chainID, err)
				return
			}

			// Update progress
			s.UpdateSettlementProgress(chainID, currentBlock)
			log.Printf("Completed settlement event catch-up for chain %d", chainID)
		}(chainID, settlementService, lastBlock, currentBlock)
	}

	// Wait for all settlement catch-ups to complete with timeout
	go func() {
		settlementWg.Wait()
		close(settlementDone)
	}()

	// Wait for completion or timeout
	select {
	case <-settlementDone:
		log.Printf("All settlement catch-up tasks completed")
	case <-settlementCtx.Done():
		log.Printf("WARNING: Settlement catch-up phase timed out after %v, proceeding anyway", phaseTimeout)
	}

	// Drain errors channel
	close(settlementErrors)
	errCount = 0
	for err := range settlementErrors {
		errCount++
		log.Printf("Error during settlement catch-up: %v", err)
	}

	if errCount > 0 {
		log.Printf("WARNING: %d errors occurred during settlement catch-up phase", errCount)
	}

	log.Printf("All catch-up phases have completed, updating last processed blocks")

	// Update last processed blocks for all chains only after all services have completed
	for chainID, currentBlock := range currentBlocks {
		dbCtx, dbCancel := context.WithTimeout(ctx, operationTimeout)
		err := s.db.UpdateLastProcessedBlock(dbCtx, chainID, currentBlock)
		dbCancel()

		if err != nil {
			log.Printf("WARNING: Failed to update last processed block for chain %d: %v", chainID, err)
			// Continue anyway
		}
	}

	log.Printf("Starting real-time event subscriptions")

	// SUBSCRIPTION PHASE - Start live event listeners for each chain
	// Setup a timeout for the subscription phase
	subCtx, subCancel := context.WithTimeout(ctx, operationTimeout)
	defer subCancel()

	errCount = 0
	successCount := 0

	// Start intent event listeners
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

		// Set up the subscription with retries
		var intentSub ethereum.Subscription
		var subErr error

		for attempt := 0; attempt < maxRetries; attempt++ {
			intentLogs := make(chan types.Log)
			intentSub, subErr = intentService.client.SubscribeFilterLogs(subCtx, intentQuery, intentLogs)

			if subErr == nil {
				log.Printf("Successfully subscribed to intent events for chain %d", chainID)
				successCount++
				go intentService.processEventLogs(ctx, intentSub, intentLogs)
				break
			}

			// If not last attempt, wait and retry
			if attempt < maxRetries-1 {
				retryDelay := retryBackoff * time.Duration(attempt+1)
				log.Printf("WARNING: Failed to subscribe to intent logs for chain %d (attempt %d/%d): %v. Retrying in %v...",
					chainID, attempt+1, maxRetries, subErr, retryDelay)
				time.Sleep(retryDelay)
			} else {
				log.Printf("ERROR: Failed to subscribe to intent logs for chain %d after %d attempts: %v",
					chainID, maxRetries, subErr)
				errCount++
			}
		}
	}

	// Start fulfillment event listeners
	for chainID, fulfillmentService := range s.fulfillmentServices {
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)
		fulfillmentQuery := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{fulfillmentService.abi.Events[IntentFulfilledEventName].ID},
			},
		}

		// Set up the subscription with retries
		var fulfillmentSub ethereum.Subscription
		var subErr error

		for attempt := 0; attempt < maxRetries; attempt++ {
			fulfillmentLogs := make(chan types.Log)
			fulfillmentSub, subErr = fulfillmentService.client.SubscribeFilterLogs(subCtx, fulfillmentQuery, fulfillmentLogs)

			if subErr == nil {
				log.Printf("Successfully subscribed to fulfillment events for chain %d", chainID)
				successCount++
				go fulfillmentService.processEventLogs(ctx, fulfillmentSub, fulfillmentLogs)
				break
			}

			// If not last attempt, wait and retry
			if attempt < maxRetries-1 {
				retryDelay := retryBackoff * time.Duration(attempt+1)
				log.Printf("WARNING: Failed to subscribe to fulfillment logs for chain %d (attempt %d/%d): %v. Retrying in %v...",
					chainID, attempt+1, maxRetries, subErr, retryDelay)
				time.Sleep(retryDelay)
			} else {
				log.Printf("ERROR: Failed to subscribe to fulfillment logs for chain %d after %d attempts: %v",
					chainID, maxRetries, subErr)
				errCount++
			}
		}
	}

	// Start settlement event listeners
	for chainID, settlementService := range s.settlementServices {
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)
		settlementQuery := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{settlementService.abi.Events[IntentSettledEventName].ID},
			},
		}

		// Set up the subscription with retries
		var settlementSub ethereum.Subscription
		var subErr error

		for attempt := 0; attempt < maxRetries; attempt++ {
			settlementLogs := make(chan types.Log)
			settlementSub, subErr = settlementService.client.SubscribeFilterLogs(subCtx, settlementQuery, settlementLogs)

			if subErr == nil {
				log.Printf("Successfully subscribed to settlement events for chain %d", chainID)
				successCount++
				go settlementService.processEventLogs(ctx, settlementSub, settlementLogs)
				break
			}

			// If not last attempt, wait and retry
			if attempt < maxRetries-1 {
				retryDelay := retryBackoff * time.Duration(attempt+1)
				log.Printf("WARNING: Failed to subscribe to settlement logs for chain %d (attempt %d/%d): %v. Retrying in %v...",
					chainID, attempt+1, maxRetries, subErr, retryDelay)
				time.Sleep(retryDelay)
			} else {
				log.Printf("ERROR: Failed to subscribe to settlement logs for chain %d after %d attempts: %v",
					chainID, maxRetries, subErr)
				errCount++
			}
		}
	}

	if errCount > 0 {
		log.Printf("WARNING: Failed to set up %d event subscriptions", errCount)
	}
	log.Printf("Successfully established %d event subscriptions", successCount)

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
func (s *EventCatchupService) catchUpOnIntentEvents(ctx context.Context, intentService *IntentService, contractAddress common.Address, fromBlock, toBlock uint64) error {
	// Use a max range of 5,000 blocks per query to stay well under the 10,000 limit
	const maxBlockRange = uint64(5000)

	// Process in chunks to avoid RPC provider limitations
	for chunkStart := fromBlock; chunkStart < toBlock; chunkStart += maxBlockRange {
		chunkEnd := chunkStart + maxBlockRange
		if chunkEnd > toBlock {
			chunkEnd = toBlock
		}

		log.Printf("Fetching intent logs for blocks %d to %d", chunkStart+1, chunkEnd)

		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(chunkStart + 1)),
			ToBlock:   big.NewInt(int64(chunkEnd)),
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{intentService.abi.Events[IntentInitiatedEventName].ID},
			},
		}

		logs, err := intentService.client.FilterLogs(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to fetch intent logs for range %d-%d: %v", chunkStart+1, chunkEnd, err)
		}

		// Process logs in batches to report progress
		batchSize := 100
		for i := 0; i < len(logs); i += batchSize {
			end := i + batchSize
			if end > len(logs) {
				end = len(logs)
			}

			batch := logs[i:end]
			for j, txlog := range batch {
				log.Printf("Processing intent log %d/%d: Block=%d, TxHash=%s",
					i+j+1, len(logs), txlog.BlockNumber, txlog.TxHash.Hex())

				// Extract intent ID from the log
				intentID := txlog.Topics[1].Hex()

				// Check if intent already exists
				existingIntent, err := s.db.GetIntent(ctx, intentID)
				if err != nil && !strings.Contains(err.Error(), "not found") {
					return fmt.Errorf("failed to check for existing intent: %v", err)
				}

				// Skip if intent already exists
				if existingIntent != nil {
					log.Printf("Skipping existing intent: %s", intentID)
					continue
				}

				if err := intentService.processLog(ctx, txlog); err != nil {
					// Skip if intent already exists
					if strings.Contains(err.Error(), "duplicate key") {
						log.Printf("Skipping duplicate intent: %s", intentID)
						continue
					}
					return fmt.Errorf("failed to process intent log: %v", err)
				}
			}

			// Update progress after each batch
			if len(batch) > 0 {
				lastBlock := batch[len(batch)-1].BlockNumber
				s.UpdateIntentProgress(intentService.chainID, lastBlock)
				log.Printf("Updated progress for chain %d to block %d", intentService.chainID, lastBlock)
			}
		}

		// Update progress after processing each chunk
		s.UpdateIntentProgress(intentService.chainID, chunkEnd)
		log.Printf("Completed processing intent logs for blocks %d to %d", chunkStart+1, chunkEnd)
	}

	return nil
}

// catchUpOnFulfillmentEvents processes missed fulfillment events for a specific chain
func (s *EventCatchupService) catchUpOnFulfillmentEvents(ctx context.Context, fulfillmentService *FulfillmentService, contractAddress common.Address, fromBlock, toBlock uint64) error {
	// Use a max range of 5,000 blocks per query to stay well under the 10,000 limit
	const maxBlockRange = uint64(5000)

	// Process in chunks to avoid RPC provider limitations
	for chunkStart := fromBlock; chunkStart < toBlock; chunkStart += maxBlockRange {
		chunkEnd := chunkStart + maxBlockRange
		if chunkEnd > toBlock {
			chunkEnd = toBlock
		}

		log.Printf("Fetching fulfillment logs for blocks %d to %d", chunkStart+1, chunkEnd)

		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(chunkStart + 1)),
			ToBlock:   big.NewInt(int64(chunkEnd)),
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{fulfillmentService.abi.Events[IntentFulfilledEventName].ID},
			},
		}

		logs, err := fulfillmentService.client.FilterLogs(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to fetch fulfillment logs for range %d-%d: %v", chunkStart+1, chunkEnd, err)
		}

		// Process logs in batches to report progress
		batchSize := 100
		for i := 0; i < len(logs); i += batchSize {
			end := i + batchSize
			if end > len(logs) {
				end = len(logs)
			}

			batch := logs[i:end]
			for j, txlog := range batch {
				log.Printf("Processing fulfillment log %d/%d: Block=%d, TxHash=%s",
					i+j+1, len(logs), txlog.BlockNumber, txlog.TxHash.Hex())

				// Extract intent ID from the log
				intentID := txlog.Topics[1].Hex()

				// Check if intent already exists
				_, err := s.db.GetIntent(ctx, intentID)
				if err != nil && !strings.Contains(err.Error(), "not found") {
					log.Printf("failed to check for existing intent: %v", err)
					continue
				}

				if err := fulfillmentService.processLog(ctx, txlog); err != nil {
					// Skip if fulfillment already exists
					if strings.Contains(err.Error(), "duplicate key") {
						log.Printf("Skipping duplicate fulfillment: %s", intentID)
						continue
					}
					return fmt.Errorf("failed to process fulfillment log: %v", err)
				}
			}

			// Update progress after each batch
			if len(batch) > 0 {
				lastBlock := batch[len(batch)-1].BlockNumber
				s.UpdateFulfillmentProgress(fulfillmentService.chainID, lastBlock)
				log.Printf("Updated progress for chain %d to block %d", fulfillmentService.chainID, lastBlock)
			}
		}

		// Update progress after processing each chunk
		s.UpdateFulfillmentProgress(fulfillmentService.chainID, chunkEnd)
		log.Printf("Completed processing fulfillment logs for blocks %d to %d", chunkStart+1, chunkEnd)
	}

	return nil
}

// catchUpOnSettlementEvents processes missed settlement events for a specific chain
func (s *EventCatchupService) catchUpOnSettlementEvents(ctx context.Context, settlementService *SettlementService, contractAddress common.Address, fromBlock, toBlock uint64) error {
	// Use a max range of 9,000 blocks per query to stay well under the 10,000 limit
	const maxBlockRange = uint64(5000)

	// Process in chunks to avoid RPC provider limitations
	for chunkStart := fromBlock; chunkStart < toBlock; chunkStart += maxBlockRange {
		chunkEnd := chunkStart + maxBlockRange
		if chunkEnd > toBlock {
			chunkEnd = toBlock
		}

		log.Printf("Fetching settlement logs for blocks %d to %d", chunkStart+1, chunkEnd)

		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(chunkStart + 1)),
			ToBlock:   big.NewInt(int64(chunkEnd)),
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{settlementService.abi.Events[IntentSettledEventName].ID},
			},
		}

		logs, err := settlementService.client.FilterLogs(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to fetch settlement logs for range %d-%d: %v", chunkStart+1, chunkEnd, err)
		}

		// Process logs in batches to report progress
		batchSize := 100
		for i := 0; i < len(logs); i += batchSize {
			end := i + batchSize
			if end > len(logs) {
				end = len(logs)
			}

			batch := logs[i:end]
			for j, txlog := range batch {
				log.Printf("Processing settlement log %d/%d: Block=%d, TxHash=%s",
					i+j+1, len(logs), txlog.BlockNumber, txlog.TxHash.Hex())

				// Extract intent ID from the log
				intentID := txlog.Topics[1].Hex()

				// Check if intent already exists
				_, err := s.db.GetIntent(ctx, intentID)
				if err != nil && !strings.Contains(err.Error(), "not found") {
					return fmt.Errorf("failed to check for existing intent: %v", err)
				}

				if err := settlementService.processLog(ctx, txlog); err != nil {
					// Skip if settlement already exists
					if strings.Contains(err.Error(), "duplicate key") {
						log.Printf("Skipping duplicate settlement: %s", intentID)
						continue
					}
					return fmt.Errorf("failed to process settlement log: %v", err)
				}
			}

			// Update progress after each batch
			if len(batch) > 0 {
				lastBlock := batch[len(batch)-1].BlockNumber
				s.UpdateSettlementProgress(settlementService.chainID, lastBlock)
				log.Printf("Updated progress for chain %d to block %d", settlementService.chainID, lastBlock)
			}
		}

		// Update progress after processing each chunk
		s.UpdateSettlementProgress(settlementService.chainID, chunkEnd)
		log.Printf("Completed processing settlement logs for blocks %d to %d", chunkStart+1, chunkEnd)
	}

	return nil
}
