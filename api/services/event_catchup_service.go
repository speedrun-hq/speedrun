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
	"github.com/zeta-chain/zetafast/api/config"
	"github.com/zeta-chain/zetafast/api/db"
)

// EventCatchupService coordinates the catch-up process between intent and fulfillment services
type EventCatchupService struct {
	intentServices     map[uint64]*IntentService
	fulfillmentService *FulfillmentService
	db                 db.Database
	mu                 sync.Mutex
	intentProgress     map[uint64]uint64 // chainID -> last processed block
}

// NewEventCatchupService creates a new EventCatchupService instance
func NewEventCatchupService(intentServices map[uint64]*IntentService, fulfillmentService *FulfillmentService, db db.Database) *EventCatchupService {
	return &EventCatchupService{
		intentServices:     intentServices,
		fulfillmentService: fulfillmentService,
		db:                 db,
		intentProgress:     make(map[uint64]uint64),
	}
}

// StartListening starts the coordinated event listening process
func (s *EventCatchupService) StartListening(ctx context.Context, contractAddress common.Address) error {
	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// Initialize progress tracking for all chains
	s.mu.Lock()
	for chainID := range s.intentServices {
		lastBlock, err := s.db.GetLastProcessedBlock(ctx, chainID)
		if err != nil {
			s.mu.Unlock()
			return fmt.Errorf("failed to get last processed block for chain %d: %v", chainID, err)
		}
		if lastBlock < cfg.ChainConfigs[chainID].DefaultBlock {
			lastBlock = cfg.ChainConfigs[chainID].DefaultBlock
		}
		s.intentProgress[chainID] = lastBlock
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

	// Catch up intent events for all chains
	for chainID, intentService := range s.intentServices {
		lastBlock := s.intentProgress[chainID]
		currentBlock := currentBlocks[chainID]

		if lastBlock >= currentBlock {
			log.Printf("No missed events to process for chain %d", chainID)
			continue
		}

		log.Printf("Starting intent event catch-up for chain %d (blocks %d to %d)",
			chainID, lastBlock+1, currentBlock)
		if err := s.catchUpOnIntentEvents(ctx, intentService, contractAddress, lastBlock, currentBlock); err != nil {
			return fmt.Errorf("failed to catch up on intent events for chain %d: %v", chainID, err)
		}

		// Update progress
		s.UpdateIntentProgress(chainID, currentBlock)
	}

	// Wait for all intent services to catch up
	maxBlock := uint64(0)
	for _, block := range currentBlocks {
		if block > maxBlock {
			maxBlock = block
		}
	}
	if err := s.waitForIntentCatchup(ctx, maxBlock); err != nil {
		return fmt.Errorf("failed to wait for intent catchup: %v", err)
	}

	// Start fulfillment catch-up for each chain
	for chainID, currentBlock := range currentBlocks {
		lastBlock := cfg.ChainConfigs[chainID].DefaultBlock
		if lastBlock >= currentBlock {
			continue
		}

		log.Printf("Starting fulfillment event catch-up for chain %d (blocks %d to %d)",
			chainID, lastBlock+1, currentBlock)
		if err := s.catchUpOnFulfillmentEvents(ctx, contractAddress, lastBlock, currentBlock); err != nil {
			return fmt.Errorf("failed to catch up on fulfillment events for chain %d: %v", chainID, err)
		}
	}

	// Update last processed blocks for all chains only after all services have completed
	for chainID, currentBlock := range currentBlocks {
		if err := s.db.UpdateLastProcessedBlock(ctx, chainID, currentBlock); err != nil {
			return fmt.Errorf("failed to update last processed block for chain %d: %v", chainID, err)
		}
	}

	// Start live event listeners
	return s.startLiveListeners(ctx, contractAddress)
}

// waitForIntentCatchup waits for all intent services to catch up to the target block
func (s *EventCatchupService) waitForIntentCatchup(ctx context.Context, targetBlock uint64) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			s.mu.Lock()
			allCaughtUp := true
			for chainID, progress := range s.intentProgress {
				if progress < targetBlock {
					log.Printf("Waiting for chain %d to catch up (progress: %d, target: %d)",
						chainID, progress, targetBlock)
					allCaughtUp = false
					break
				}
			}
			s.mu.Unlock()

			if allCaughtUp {
				log.Printf("All intent services have caught up to block %d", targetBlock)
				return nil
			}
		}
	}
}

// UpdateIntentProgress updates the progress of an intent service
func (s *EventCatchupService) UpdateIntentProgress(chainID, blockNumber uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.intentProgress[chainID] = blockNumber
}

// catchUpOnIntentEvents processes missed intent events for a specific chain
func (s *EventCatchupService) catchUpOnIntentEvents(ctx context.Context, intentService *IntentService, contractAddress common.Address, fromBlock, toBlock uint64) error {
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock + 1)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: []common.Address{contractAddress},
		Topics: [][]common.Hash{
			{intentService.abi.Events[IntentInitiatedEventName].ID},
		},
	}

	logs, err := intentService.client.FilterLogs(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to fetch intent logs: %v", err)
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

	return nil
}

// catchUpOnFulfillmentEvents processes missed fulfillment events
func (s *EventCatchupService) catchUpOnFulfillmentEvents(ctx context.Context, contractAddress common.Address, fromBlock, toBlock uint64) error {
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock + 1)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: []common.Address{contractAddress},
		Topics: [][]common.Hash{
			{s.fulfillmentService.abi.Events[IntentFulfilledEventName].ID},
		},
	}

	logs, err := s.fulfillmentService.client.FilterLogs(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to fetch fulfillment logs: %v", err)
	}

	for i, txlog := range logs {
		log.Printf("Processing fulfillment log %d/%d: Block=%d, TxHash=%s",
			i+1, len(logs), txlog.BlockNumber, txlog.TxHash.Hex())

		// Extract intent ID from the log
		intentID := txlog.Topics[1].Hex()

		// Check if fulfillment already exists
		existingFulfillment, err := s.db.GetFulfillment(ctx, intentID)
		if err != nil {
			// Only return error if it's not a "not found" error
			if !strings.Contains(err.Error(), "not found") {
				return fmt.Errorf("failed to check for existing fulfillment: %v", err)
			}
		}

		// Skip if fulfillment already exists
		if existingFulfillment != nil {
			log.Printf("Skipping existing fulfillment: %s", intentID)
			continue
		}

		if err := s.fulfillmentService.processLog(ctx, txlog); err != nil {
			return fmt.Errorf("failed to process fulfillment log: %v", err)
		}
	}

	return nil
}

// startLiveListeners starts the live event listeners for both services
func (s *EventCatchupService) startLiveListeners(ctx context.Context, contractAddress common.Address) error {
	// Start intent event listeners for each chain
	for chainID, intentService := range s.intentServices {
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

		go intentService.processEventLogs(ctx, intentSub, intentLogs)
	}

	// Start fulfillment event listener
	fulfillmentQuery := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics: [][]common.Hash{
			{s.fulfillmentService.abi.Events[IntentFulfilledEventName].ID},
		},
	}

	fulfillmentLogs := make(chan types.Log)
	fulfillmentSub, err := s.fulfillmentService.client.SubscribeFilterLogs(ctx, fulfillmentQuery, fulfillmentLogs)
	if err != nil {
		return fmt.Errorf("failed to subscribe to fulfillment logs: %v", err)
	}

	go s.fulfillmentService.processEventLogs(ctx, fulfillmentSub, fulfillmentLogs)

	return nil
}
