package services

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"sync"

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
		currentBlock, err := intentService.client.BlockNumber(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current block number for chain %d: %v", chainID, err)
		}
		currentBlocks[chainID] = currentBlock
	}

	// Create a wait group to track all intent catch-up goroutines
	var intentWg sync.WaitGroup
	intentErrors := make(chan error, len(s.intentServices))

	// Start intent catch-up for all chains in parallel
	for chainID, intentService := range s.intentServices {
		lastBlock := s.intentProgress[chainID]
		currentBlock := currentBlocks[chainID]
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

			if err := s.catchUpOnIntentEvents(ctx, intentService, contractAddress, lastBlock, currentBlock); err != nil {
				intentErrors <- fmt.Errorf("failed to catch up on intent events for chain %d: %v", chainID, err)
				return
			}

			// Update progress
			s.UpdateIntentProgress(chainID, currentBlock)
			log.Printf("Completed intent event catch-up for chain %d", chainID)
		}(chainID, intentService, lastBlock, currentBlock)
	}

	// Wait for all intent catch-ups to complete
	go func() {
		intentWg.Wait()
		close(intentErrors)
	}()

	// Check for any errors from intent catch-ups
	for err := range intentErrors {
		if err != nil {
			log.Printf("Error during intent catch-up: %v", err)
		}
	}

	log.Printf("All intent services have completed catch-up")

	log.Printf("Starting fulfillment catch-up")

	// Initialize progress tracking for all chains
	s.mu.Lock()
	for chainID := range s.fulfillmentServices {
		lastBlock, err := s.db.GetLastProcessedBlock(ctx, chainID)
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

	for chainID, fulfillmentService := range s.fulfillmentServices {
		lastBlock := s.fulfillmentProgress[chainID]
		currentBlock := currentBlocks[chainID]
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)
		if lastBlock >= currentBlock {
			log.Printf("No missed events to process for chain %d", chainID)
			continue
		}

		fulfillmentWg.Add(1)
		go func(chainID uint64, fulfillmentService *FulfillmentService, lastBlock, currentBlock uint64) {
			defer fulfillmentWg.Done()

			log.Printf("Starting fulfillment event catch-up for chain %d (blocks %d to %d)",
				chainID, lastBlock+1, currentBlock)

			if err := s.catchUpOnFulfillmentEvents(ctx, fulfillmentService, contractAddress, lastBlock, currentBlock); err != nil {
				fulfillmentErrors <- fmt.Errorf("failed to catch up on fulfillment events for chain %d: %v", chainID, err)
				return
			}

			// Update progress
			s.UpdateFulfillmentProgress(chainID, currentBlock)
			log.Printf("Completed fulfillment event catch-up for chain %d", chainID)
		}(chainID, fulfillmentService, lastBlock, currentBlock)
	}

	// Wait for all fulfillment catch-ups to complete
	go func() {
		fulfillmentWg.Wait()
		close(fulfillmentErrors)
	}()

	// Check for any errors from fulfillment catch-ups
	for err := range fulfillmentErrors {
		if err != nil {
			log.Printf("Error during fulfillment catch-up: %v", err)
		}
	}

	log.Printf("All fulfillment services have completed catch-up")

	log.Printf("Starting settlement catch-up")

	// Initialize progress tracking for all chains
	s.mu.Lock()
	for chainID := range s.settlementServices {
		lastBlock, err := s.db.GetLastProcessedBlock(ctx, chainID)
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

	for chainID, settlementService := range s.settlementServices {
		lastBlock := s.settlementProgress[chainID]
		currentBlock := currentBlocks[chainID]
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)
		if lastBlock >= currentBlock {
			log.Printf("No missed events to process for chain %d", chainID)
			continue
		}

		settlementWg.Add(1)
		go func(chainID uint64, settlementService *SettlementService, lastBlock, currentBlock uint64) {
			defer settlementWg.Done()

			log.Printf("Starting settlement event catch-up for chain %d (blocks %d to %d)",
				chainID, lastBlock+1, currentBlock)

			if err := s.catchUpOnSettlementEvents(ctx, settlementService, contractAddress, lastBlock, currentBlock); err != nil {
				settlementErrors <- fmt.Errorf("failed to catch up on settlement events for chain %d: %v", chainID, err)
				return
			}

			// Update progress
			s.UpdateSettlementProgress(chainID, currentBlock)
			log.Printf("Completed settlement event catch-up for chain %d", chainID)
		}(chainID, settlementService, lastBlock, currentBlock)
	}

	// Wait for all settlement catch-ups to complete
	go func() {
		settlementWg.Wait()
		close(settlementErrors)
	}()

	// Check for any errors from settlement catch-ups
	for err := range settlementErrors {
		if err != nil {
			log.Printf("Error during settlement catch-up: %v", err)
		}
	}

	log.Printf("All settlement services have completed catch-up")

	// Update last processed blocks for all chains only after all services have completed
	for chainID, currentBlock := range currentBlocks {
		if err := s.db.UpdateLastProcessedBlock(ctx, chainID, currentBlock); err != nil {
			return fmt.Errorf("failed to update last processed block for chain %d: %v", chainID, err)
		}
	}

	// Start live event intentlisteners for each chain
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
		intentSub, err := intentService.client.SubscribeFilterLogs(ctx, intentQuery, intentLogs)
		if err != nil {
			return fmt.Errorf("failed to subscribe to intent logs for chain %d: %v", chainID, err)
		}
		log.Printf("Successfully subscribed to intent events for chain %d", chainID)

		go intentService.processEventLogs(ctx, intentSub, intentLogs, fmt.Sprintf("chain_%d", chainID))
	}

	// Start live event fulfillment listeners for each chain
	for chainID, fulfillmentService := range s.fulfillmentServices {
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)
		fulfillmentQuery := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{fulfillmentService.abi.Events[IntentFulfilledEventName].ID},
			},
		}

		fulfillmentLogs := make(chan types.Log)
		fulfillmentSub, err := fulfillmentService.client.SubscribeFilterLogs(ctx, fulfillmentQuery, fulfillmentLogs)
		if err != nil {
			return fmt.Errorf("failed to subscribe to fulfillment logs for chain %d: %v", chainID, err)
		}

		go fulfillmentService.processEventLogs(ctx, fulfillmentSub, fulfillmentLogs)
	}

	// Start live event settlement listeners for each chain
	for chainID, settlementService := range s.settlementServices {
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)
		settlementQuery := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{settlementService.abi.Events[IntentSettledEventName].ID},
			},
		}

		settlementLogs := make(chan types.Log)
		settlementSub, err := settlementService.client.SubscribeFilterLogs(ctx, settlementQuery, settlementLogs)
		if err != nil {
			return fmt.Errorf("failed to subscribe to settlement logs for chain %d: %v", chainID, err)
		}

		go settlementService.processEventLogs(ctx, settlementSub, settlementLogs)
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

	// Process in chunks to avoid RPC provider l	imitations
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
