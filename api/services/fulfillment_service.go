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
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/models"
	"github.com/zeta-chain/zetafast/api/utils"
)

// IntentFulfilledEvent represents the event emitted when an intent is fulfilled
type IntentFulfilledEvent struct {
	IntentID common.Hash
	Asset    common.Address
	Amount   *big.Int
	Receiver common.Address
	TxHash   common.Hash
}

// IntentSettledEvent represents the event emitted when an intent is settled
type IntentSettledEvent struct {
	IntentID  string
	Asset     string
	Receiver  string
	TxHash    string
	Amount    *big.Int
	Fulfilled bool
}

// ChainClient represents a client for a specific chain
type ChainClient struct {
	client          *ethclient.Client
	contractAddress common.Address
	chainID         uint64
	abi             abi.ABI
}

// FulfillmentService handles monitoring and processing of fulfillment events
type FulfillmentService struct {
	clients       map[uint64]*ChainClient
	db            db.DBInterface
	mu            sync.RWMutex
	abi           abi.ABI
	subscriptions map[uint64]ethereum.Subscription
	defaultBlocks map[uint64]uint64
}

// NewFulfillmentService creates a new FulfillmentService instance
func NewFulfillmentService(clients map[uint64]*ethclient.Client, contractAddresses map[uint64]string, db db.DBInterface, contractABI string, defaultBlocks map[uint64]uint64) (*FulfillmentService, error) {
	// Parse the contract ABI
	parsedABI, err := abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %v", err)
	}

	// Create chain clients
	chainClients := make(map[uint64]*ChainClient)
	for chainID, client := range clients {
		contractAddr, ok := contractAddresses[chainID]
		if !ok {
			return nil, fmt.Errorf("no contract address for chain %d", chainID)
		}

		chainClients[chainID] = &ChainClient{
			client:          client,
			contractAddress: common.HexToAddress(contractAddr),
			chainID:         chainID,
			abi:             parsedABI,
		}
	}

	return &FulfillmentService{
		clients:       chainClients,
		db:            db,
		abi:           parsedABI,
		subscriptions: make(map[uint64]ethereum.Subscription),
		defaultBlocks: defaultBlocks,
	}, nil
}

// StartListening starts listening for fulfillment events on all chains
func (s *FulfillmentService) StartListening(ctx context.Context) error {
	log.Printf("Starting fulfillment service listener for %d chains", len(s.clients))
	log.Printf("Default blocks configuration: %+v", s.defaultBlocks)

	for chainID, client := range s.clients {
		log.Printf("Setting up listener for chain %d at contract %s", chainID, client.contractAddress.Hex())

		// First, catch up on any missed events
		if err := s.catchUpOnMissedEvents(ctx, client, chainID); err != nil {
			log.Printf("Error catching up on missed events for chain %d: %v", chainID, err)
			return fmt.Errorf("failed to catch up on missed events for chain %d: %v", chainID, err)
		}

		// Start a goroutine for each chain
		go s.processChainLogs(ctx, client)
		log.Printf("Started log processor for chain %d", chainID)
	}
	log.Printf("Successfully started all chain listeners")
	return nil
}

// catchUpOnMissedEvents fetches and processes any events that were missed during downtime
func (s *FulfillmentService) catchUpOnMissedEvents(ctx context.Context, client *ChainClient, chainID uint64) error {
	log.Printf("Catching up on missed events for chain %d at contract %s", chainID, client.contractAddress.Hex())

	// Get last processed block
	lastBlock, err := s.db.GetLastProcessedBlock(ctx, chainID)
	if err != nil {
		log.Printf("Error getting last processed block for chain %d: %v", chainID, err)
		return fmt.Errorf("failed to get last processed block for chain %d: %v", chainID, err)
	}
	log.Printf("Chain %d - Last processed block from database: %d", chainID, lastBlock)

	// If no block was found, use the default block from config
	if lastBlock == 0 {
		defaultBlock, ok := s.defaultBlocks[chainID]
		if ok {
			lastBlock = defaultBlock
			log.Printf("Using default block %d for chain %d", defaultBlock, chainID)
			// Update the last processed block with the default value
			if err := s.db.UpdateLastProcessedBlock(ctx, chainID, defaultBlock); err != nil {
				log.Printf("Error updating last processed block with default value for chain %d: %v", chainID, err)
				return fmt.Errorf("failed to update last processed block with default value: %v", err)
			}
			log.Printf("Successfully updated last processed block to default value %d for chain %d", defaultBlock, chainID)
		} else {
			log.Printf("No default block configured for chain %d", chainID)
		}
	}

	// Get current block
	currentBlock, err := client.client.BlockNumber(ctx)
	if err != nil {
		log.Printf("Error getting current block for chain %d: %v", chainID, err)
		return fmt.Errorf("failed to get current block for chain %d: %v", chainID, err)
	}

	log.Printf("Chain %d - Processing blocks from %d to %d", chainID, lastBlock, currentBlock)

	// If there's a gap, fetch all events in that range
	if lastBlock < currentBlock {
		log.Printf("Chain %d - Fetching logs from block %d to %d", chainID, lastBlock+1, currentBlock)
		log.Printf("Chain %d - Using filter query: Address=%s, Topics=[%s]",
			chainID,
			client.contractAddress.Hex(),
			s.abi.Events["IntentFulfilled"].ID.Hex())

		logs, err := client.client.FilterLogs(ctx, ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(lastBlock + 1)),
			ToBlock:   big.NewInt(int64(currentBlock)),
			Addresses: []common.Address{client.contractAddress},
			Topics:    [][]common.Hash{{s.abi.Events["IntentFulfilled"].ID}},
		})
		if err != nil {
			log.Printf("Error fetching logs for chain %d: %v", chainID, err)
			return fmt.Errorf("failed to fetch logs for chain %d: %v", chainID, err)
		}

		log.Printf("Found %d events to process for chain %d", len(logs), chainID)

		// Process each log
		for i, eventLog := range logs {
			log.Printf("Processing event %d/%d for chain %d", i+1, len(logs), chainID)
			log.Printf("Event details - Block: %d, TxHash: %s, LogIndex: %d",
				eventLog.BlockNumber,
				eventLog.TxHash.Hex(),
				eventLog.Index)
			log.Printf("Event contract: %s", eventLog.Address.Hex())

			if err := s.processLog(ctx, client, eventLog); err != nil {
				log.Printf("Error processing log for chain %d: %v", chainID, err)
				return fmt.Errorf("failed to process missed log: %v", err)
			}
			log.Printf("Successfully processed event %d/%d for chain %d", i+1, len(logs), chainID)
		}

		// Update last processed block
		if err := s.db.UpdateLastProcessedBlock(ctx, chainID, currentBlock); err != nil {
			log.Printf("Error updating last processed block for chain %d: %v", chainID, err)
			return fmt.Errorf("failed to update last processed block for chain %d: %v", chainID, err)
		}
		log.Printf("Successfully updated last processed block to %d for chain %d", currentBlock, chainID)

		log.Printf("Successfully caught up on chain %d - Processed %d events", chainID, len(logs))
	} else {
		log.Printf("Chain %d is up to date at block %d", chainID, currentBlock)
	}

	return nil
}

// processChainLogs processes logs for a specific chain
func (s *FulfillmentService) processChainLogs(ctx context.Context, client *ChainClient) {
	log.Printf("Starting log processor for chain %d", client.chainID)

	query := ethereum.FilterQuery{
		Addresses: []common.Address{client.contractAddress},
		Topics: [][]common.Hash{
			{s.abi.Events["IntentFulfilled"].ID},
		},
	}

	logs := make(chan types.Log)
	sub, err := client.client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		log.Printf("Error subscribing to logs for chain %d: %v", client.chainID, err)
		return
	}

	log.Printf("Successfully subscribed to logs for chain %d", client.chainID)
	log.Printf("Chain %d - Listening for events at contract %s", client.chainID, client.contractAddress.Hex())

	// Store the subscription
	s.mu.Lock()
	s.subscriptions[client.chainID] = sub
	s.mu.Unlock()

	eventCount := 0
	for {
		select {
		case err := <-sub.Err():
			log.Printf("Subscription error for chain %d: %v", client.chainID, err)
			// Try to resubscribe
			if err := s.handleSubscriptionError(ctx, client, sub, logs); err != nil {
				log.Printf("Failed to resubscribe for chain %d: %v", client.chainID, err)
				return
			}
			log.Printf("Successfully resubscribed to logs for chain %d", client.chainID)
		case eventLog := <-logs:
			eventCount++
			log.Printf("Chain %d - Received event #%d - Block: %d, TxHash: %s", client.chainID, eventCount, eventLog.BlockNumber, eventLog.TxHash.Hex())
			if err := s.processLog(ctx, client, eventLog); err != nil {
				log.Printf("Error processing log for chain %d: %v", client.chainID, err)
			} else {
				log.Printf("Successfully processed event #%d for chain %d", eventCount, client.chainID)
			}
			// Update the last processed block number
			if err := s.db.UpdateLastProcessedBlock(ctx, client.chainID, eventLog.BlockNumber); err != nil {
				log.Printf("Error updating last processed block for chain %d: %v", client.chainID, err)
			} else {
				log.Printf("Successfully updated last processed block to %d for chain %d", eventLog.BlockNumber, client.chainID)
			}
		case <-ctx.Done():
			log.Printf("Stopping log processor for chain %d - Processed %d events", client.chainID, eventCount)
			sub.Unsubscribe()
			return
		}
	}
}

// handleSubscriptionError attempts to recover from a subscription error by resubscribing
func (s *FulfillmentService) handleSubscriptionError(ctx context.Context, client *ChainClient, oldSub ethereum.Subscription, logs chan types.Log) error {
	oldSub.Unsubscribe()

	query := ethereum.FilterQuery{
		Addresses: []common.Address{client.contractAddress},
		Topics: [][]common.Hash{
			{s.abi.Events["IntentFulfilled"].ID},
		},
	}

	newSub, err := client.client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return fmt.Errorf("failed to resubscribe: %v", err)
	}

	// Update the subscription
	s.mu.Lock()
	s.subscriptions[client.chainID] = newSub
	s.mu.Unlock()

	return nil
}

// parseIntentFulfilledEvent parses a log entry into an IntentFulfilledEvent
func (s *FulfillmentService) parseIntentFulfilledEvent(eventLog types.Log) (*IntentFulfilledEvent, error) {
	log.Printf("Raw event data - Block: %d, TxHash: %s", eventLog.BlockNumber, eventLog.TxHash.Hex())
	log.Printf("Raw event data - Address: %s", eventLog.Address.Hex())
	log.Printf("Raw event data - Topics count: %d", len(eventLog.Topics))
	for i, topic := range eventLog.Topics {
		log.Printf("Topic[%d]: %s", i, topic.Hex())
	}
	log.Printf("Raw event data - Data: %x", eventLog.Data)

	// Validate topics count
	if len(eventLog.Topics) < 4 {
		return nil, fmt.Errorf("invalid number of topics: expected at least 4, got %d", len(eventLog.Topics))
	}

	// Create a new event instance with indexed parameters from topics
	event := &IntentFulfilledEvent{
		IntentID: eventLog.Topics[1],                            // intentId is indexed (topic 1)
		Asset:    common.HexToAddress(eventLog.Topics[2].Hex()), // asset is indexed (topic 2)
		Receiver: common.HexToAddress(eventLog.Topics[3].Hex()), // receiver is indexed (topic 3)
		TxHash:   eventLog.TxHash,
	}

	// Decode the non-indexed data (amount)
	type NonIndexed struct {
		Amount *big.Int
	}
	var decoded NonIndexed
	err := s.abi.UnpackIntoInterface(&decoded, "IntentFulfilled", eventLog.Data)
	if err != nil {
		log.Printf("Error unpacking event data: %v", err)
		log.Printf("Event signature: %s", s.abi.Events["IntentFulfilled"].ID.Hex())
		log.Printf("Event inputs: %+v", s.abi.Events["IntentFulfilled"].Inputs)
		return nil, fmt.Errorf("failed to unpack log data: %v", err)
	}
	event.Amount = decoded.Amount

	log.Printf("Parsed event data - IntentID: %s", event.IntentID.Hex())
	log.Printf("Parsed event data - Asset: %s", event.Asset.Hex())
	log.Printf("Parsed event data - Amount: %s", event.Amount.String())
	log.Printf("Parsed event data - Receiver: %s", event.Receiver.Hex())

	// Validate receiver address
	if err := utils.ValidateAddress(event.Receiver.Hex()); err != nil {
		log.Printf("Invalid receiver address: %v", err)
		return nil, fmt.Errorf("invalid receiver address: %v", err)
	}

	return event, nil
}

// parseIntentSettledEvent parses a log entry into an IntentSettledEvent
func (s *FulfillmentService) parseIntentSettledEvent(eventLog types.Log) (*IntentSettledEvent, error) {
	// Validate topics count
	if len(eventLog.Topics) < 4 {
		return nil, fmt.Errorf("invalid number of topics: expected at least 4, got %d", len(eventLog.Topics))
	}

	// Create a new event instance with indexed parameters from topics
	event := &IntentSettledEvent{
		IntentID: eventLog.Topics[1].Hex(),                            // intentId is indexed (topic 1)
		Asset:    common.HexToAddress(eventLog.Topics[2].Hex()).Hex(), // asset is indexed (topic 2)
		Receiver: common.HexToAddress(eventLog.Topics[3].Hex()).Hex(), // receiver is indexed (topic 3)
		TxHash:   eventLog.TxHash.Hex(),
	}

	// Decode the non-indexed data (amount and fulfilled)
	type NonIndexed struct {
		Amount    *big.Int
		Fulfilled bool
	}
	var decoded NonIndexed
	err := s.abi.UnpackIntoInterface(&decoded, "IntentSettled", eventLog.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack log data: %v", err)
	}
	event.Amount = decoded.Amount
	event.Fulfilled = decoded.Fulfilled

	return event, nil
}

// processLog processes a single fulfillment event log
func (s *FulfillmentService) processLog(ctx context.Context, client *ChainClient, eventLog types.Log) error {
	log.Printf("Processing event - Chain: %d, Contract: %s",
		client.chainID,
		client.contractAddress.Hex())
	log.Printf("Event details - Block: %d, TxHash: %s, LogIndex: %d",
		eventLog.BlockNumber,
		eventLog.TxHash.Hex(),
		eventLog.Index)

	// Determine event type based on the first topic
	eventID := eventLog.Topics[0]

	if eventID == s.abi.Events["IntentFulfilled"].ID {
		// Handle IntentFulfilled event
		event, err := s.parseIntentFulfilledEvent(eventLog)
		if err != nil {
			return fmt.Errorf("failed to parse fulfillment event: %w", err)
		}
		return s.processFulfillmentEvent(ctx, event)
	} else if eventID == s.abi.Events["IntentSettled"].ID {
		// Handle IntentSettled event
		event, err := s.parseIntentSettledEvent(eventLog)
		if err != nil {
			return fmt.Errorf("failed to parse settlement event: %w", err)
		}
		return s.processSettlementEvent(ctx, event)
	}

	return fmt.Errorf("unknown event type: %s", eventID.Hex())
}

// processSettlementEvent handles an IntentSettled event
func (s *FulfillmentService) processSettlementEvent(ctx context.Context, event *IntentSettledEvent) error {
	// Get the intent
	intent, err := s.db.GetIntent(ctx, event.IntentID)
	if err != nil {
		return fmt.Errorf("failed to get intent: %w", err)
	}

	log.Printf("Processing settlement event for intent %s - Current status: %s", intent.ID, intent.Status)

	// Update intent status based on whether it was fulfilled
	var newStatus models.IntentStatus
	if event.Fulfilled {
		newStatus = models.IntentStatusFulfilled
		log.Printf("Intent %s was fulfilled by a fulfiller, updating status to %s", intent.ID, newStatus)
	} else {
		newStatus = models.IntentStatusCompleted
		log.Printf("Intent %s was settled without a fulfiller, updating status to %s", intent.ID, newStatus)
	}

	// Update intent status
	err = s.db.UpdateIntentStatus(ctx, intent.ID, newStatus)
	if err != nil {
		log.Printf("Failed to update intent %s status to %s: %v", intent.ID, newStatus, err)
		return fmt.Errorf("failed to update intent status: %w", err)
	}

	log.Printf("Successfully processed settlement event for intent %s - Final status: %s", intent.ID, newStatus)
	return nil
}

// processFulfillmentEvent handles an IntentFulfilled event
func (s *FulfillmentService) processFulfillmentEvent(ctx context.Context, event *IntentFulfilledEvent) error {
	// Get the intent
	intent, err := s.db.GetIntent(ctx, event.IntentID.Hex())
	if err != nil {
		return fmt.Errorf("failed to get intent: %w", err)
	}

	log.Printf("Processing fulfillment event for intent %s - Current status: %s", intent.ID, intent.Status)

	// Create fulfillment
	fulfillment := &models.Fulfillment{
		ID:        event.TxHash.Hex(),
		IntentID:  event.IntentID.Hex(),
		TxHash:    event.TxHash.Hex(),
		Status:    models.FulfillmentStatusCompleted,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.db.CreateFulfillment(ctx, fulfillment)
	if err != nil {
		log.Printf("Failed to create fulfillment for intent %s: %v", intent.ID, err)
		return fmt.Errorf("failed to create fulfillment: %w", err)
	}

	log.Printf("Created fulfillment for intent %s - TxHash: %s", intent.ID, fulfillment.TxHash)

	// Get the number of completed fulfillments
	fulfillments, err := s.db.ListFulfillments(ctx)
	if err != nil {
		return fmt.Errorf("failed to list fulfillments: %w", err)
	}

	completedCount := 0
	for _, f := range fulfillments {
		if f.IntentID == intent.ID && f.Status == models.FulfillmentStatusCompleted {
			completedCount++
		}
	}

	log.Printf("Intent %s has %d completed fulfillments", intent.ID, completedCount)

	// Mark intent as fulfilled after the second fulfillment
	if completedCount >= 2 {
		log.Printf("Intent %s has reached required number of fulfillments (%d), marking as fulfilled", intent.ID, completedCount)
		err = s.db.UpdateIntentStatus(ctx, intent.ID, models.IntentStatusFulfilled)
		if err != nil {
			log.Printf("Failed to update intent %s status to fulfilled: %v", intent.ID, err)
			return fmt.Errorf("failed to update intent status: %w", err)
		}
		log.Printf("Successfully marked intent %s as fulfilled", intent.ID)
	} else {
		log.Printf("Intent %s needs %d more fulfillments to be marked as fulfilled", intent.ID, 2-completedCount)
	}

	return nil
}

// Stop stops the fulfillment service
func (s *FulfillmentService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Close all subscriptions
	for chainID, sub := range s.subscriptions {
		if sub != nil {
			sub.Unsubscribe()
		}
		delete(s.subscriptions, chainID)
	}

	// Close all clients
	for chainID, client := range s.clients {
		if client.client != nil {
			client.client.Close()
		}
		delete(s.clients, chainID)
	}
}

// GetFulfillment retrieves a fulfillment by ID
func (s *FulfillmentService) GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error) {
	// Validate bytes32 format
	if !utils.IsValidBytes32(id) {
		return nil, fmt.Errorf("invalid fulfillment ID format")
	}

	// Get fulfillment from database
	fulfillment, err := s.db.GetFulfillment(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get fulfillment: %v", err)
	}

	return fulfillment, nil
}

// ListFulfillments retrieves all fulfillments
func (s *FulfillmentService) ListFulfillments(ctx context.Context) ([]*models.Fulfillment, error) {
	// Get fulfillments from database
	fulfillments, err := s.db.ListFulfillments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list fulfillments: %v", err)
	}

	return fulfillments, nil
}

// CreateFulfillment creates a new fulfillment
func (s *FulfillmentService) CreateFulfillment(ctx context.Context, intentID, txHash string) (*models.Fulfillment, error) {
	// Validate intent exists
	intent, err := s.db.GetIntent(ctx, intentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get intent: %v", err)
	}
	if intent == nil {
		return nil, fmt.Errorf("intent not found: %s", intentID)
	}

	// Create fulfillment
	now := time.Now()
	fulfillment := &models.Fulfillment{
		ID:        txHash,
		IntentID:  intentID,
		TxHash:    txHash,
		Status:    models.FulfillmentStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save fulfillment
	if err := s.db.CreateFulfillment(ctx, fulfillment); err != nil {
		return nil, fmt.Errorf("failed to create fulfillment: %v", err)
	}

	return fulfillment, nil
}
