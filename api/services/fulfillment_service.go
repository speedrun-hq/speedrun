package services

import (
	"context"
	"fmt"
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
	clients       map[string]*ChainClient
	db            db.Database
	mu            sync.RWMutex
	abi           abi.ABI
	subscriptions map[string]ethereum.Subscription
}

// NewFulfillmentService creates a new FulfillmentService instance
func NewFulfillmentService(clients map[uint64]*ethclient.Client, contractAddresses map[uint64]string, db db.Database, contractABI string) (*FulfillmentService, error) {
	// Parse the contract ABI
	parsedABI, err := abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %v", err)
	}

	// Create chain clients
	chainClients := make(map[string]*ChainClient)
	for chainID, client := range clients {
		contractAddr, ok := contractAddresses[chainID]
		if !ok {
			return nil, fmt.Errorf("no contract address for chain %d", chainID)
		}

		chainClients[fmt.Sprintf("%d", chainID)] = &ChainClient{
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
		subscriptions: make(map[string]ethereum.Subscription),
	}, nil
}

// StartListening starts listening for fulfillment events on all chains
func (s *FulfillmentService) StartListening(ctx context.Context) error {
	for _, client := range s.clients {
		// Start a goroutine for each chain
		go s.processChainLogs(ctx, client)
	}
	return nil
}

// processChainLogs processes logs for a specific chain
func (s *FulfillmentService) processChainLogs(ctx context.Context, client *ChainClient) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{client.contractAddress},
		Topics: [][]common.Hash{
			{s.abi.Events["IntentFulfilled"].ID},
		},
	}

	logs := make(chan types.Log)
	sub, err := client.client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		fmt.Printf("Failed to subscribe to logs: %v\n", err)
		return
	}

	for {
		select {
		case err := <-sub.Err():
			fmt.Printf("Subscription error: %v\n", err)
		case log := <-logs:
			if err := s.processLog(ctx, client, log); err != nil {
				fmt.Printf("Failed to process log: %v\n", err)
			}
		case <-ctx.Done():
			sub.Unsubscribe()
			return
		}
	}
}

// processLog processes a log entry from the blockchain
func (s *FulfillmentService) processLog(ctx context.Context, client *ChainClient, log types.Log) error {
	// Decode the log data into our event struct
	var event IntentFulfilledEvent
	err := s.abi.UnpackIntoInterface(&event, "IntentFulfilled", log.Data)
	if err != nil {
		return fmt.Errorf("failed to unpack log data: %v", err)
	}

	// Extract indexed parameters from topics
	if len(log.Topics) < 4 {
		return fmt.Errorf("invalid number of topics in log: %d", len(log.Topics))
	}

	event.IntentID = log.Topics[1]
	event.Asset = common.HexToAddress(log.Topics[2].Hex())
	event.Receiver = common.HexToAddress(log.Topics[3].Hex())

	// Validate receiver address
	if err := utils.ValidateAddress(event.Receiver.Hex()); err != nil {
		return fmt.Errorf("invalid receiver address: %v", err)
	}

	// Get the intent from database
	intent, err := s.db.GetIntent(ctx, event.IntentID.Hex())
	if err != nil {
		return fmt.Errorf("failed to get intent: %v", err)
	}

	// Validate intent
	if err := utils.ValidateIntent(intent); err != nil {
		return fmt.Errorf("invalid intent: %v", err)
	}

	// Get total fulfilled amount
	totalFulfilled, err := s.db.GetTotalFulfilledAmount(ctx, event.IntentID.Hex())
	if err != nil {
		return fmt.Errorf("failed to get total fulfilled amount: %v", err)
	}

	// Validate fulfillment amount
	if err := utils.ValidateFulfillmentAmount(event.Amount.String(), intent.Amount, totalFulfilled); err != nil {
		return fmt.Errorf("invalid fulfillment amount: %v", err)
	}

	// Create fulfillment record
	fulfillment := &models.Fulfillment{
		ID:          log.TxHash.Hex(), // Use transaction hash as ID for uniqueness
		IntentID:    event.IntentID.Hex(),
		Fulfiller:   event.Receiver.Hex(),
		TargetChain: client.chainID,
		Amount:      event.Amount.String(),
		Status:      models.FulfillmentStatusCompleted,
		TxHash:      log.TxHash.Hex(),
		BlockNumber: log.BlockNumber,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store fulfillment in database
	if err := s.db.CreateFulfillment(ctx, fulfillment); err != nil {
		return fmt.Errorf("failed to create fulfillment: %v", err)
	}

	// Get new total fulfilled amount
	newTotal, err := s.db.GetTotalFulfilledAmount(ctx, event.IntentID.Hex())
	if err != nil {
		return fmt.Errorf("failed to get new total fulfilled amount: %v", err)
	}

	// Update intent status to fulfilled only if the total amount has been fulfilled
	if newTotal == intent.Amount {
		if err := s.db.UpdateIntentStatus(ctx, event.IntentID.Hex(), models.IntentStatusFulfilled); err != nil {
			return fmt.Errorf("failed to update intent status: %v", err)
		}
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

// CreateFulfillment creates a new fulfillment for an intent
func (s *FulfillmentService) CreateFulfillment(ctx context.Context, intentID, fulfiller, amount string) (*models.Fulfillment, error) {
	// Validate fulfiller address
	if err := utils.ValidateAddress(fulfiller); err != nil {
		return nil, fmt.Errorf("invalid fulfiller address: %v", err)
	}

	// Validate amount
	if err := utils.ValidateAmount(amount); err != nil {
		return nil, fmt.Errorf("invalid amount: %v", err)
	}

	// Validate intent ID format
	if !utils.IsValidBytes32(intentID) {
		return nil, fmt.Errorf("invalid intent ID format")
	}

	// Get the intent
	intent, err := s.db.GetIntent(ctx, intentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get intent: %v", err)
	}

	// Validate intent
	if err := utils.ValidateIntent(intent); err != nil {
		return nil, fmt.Errorf("invalid intent: %v", err)
	}

	// Get total amount already fulfilled
	totalFulfilled, err := s.db.GetTotalFulfilledAmount(ctx, intentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get total fulfilled amount: %v", err)
	}

	// Validate fulfillment amount
	if err := utils.ValidateFulfillmentAmount(amount, intent.Amount, totalFulfilled); err != nil {
		return nil, fmt.Errorf("invalid fulfillment amount: %v", err)
	}

	// Generate a transaction hash-like ID for the fulfillment
	// We use the intent ID as a base and add a timestamp to make it unique
	fulfillmentID := common.HexToHash(fmt.Sprintf("%s%d", intentID, time.Now().UnixNano())).Hex()

	// Create fulfillment
	fulfillment := &models.Fulfillment{
		ID:          fulfillmentID,
		IntentID:    intentID,
		Fulfiller:   fulfiller,
		TargetChain: intent.DestinationChain,
		Amount:      amount,
		Status:      models.FulfillmentStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save to database
	if err := s.db.CreateFulfillment(ctx, fulfillment); err != nil {
		return nil, fmt.Errorf("failed to create fulfillment: %v", err)
	}

	// Check if intent is fully fulfilled
	newTotal, err := s.db.GetTotalFulfilledAmount(ctx, intentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get new total fulfilled amount: %v", err)
	}

	// Update intent status if fully fulfilled
	if newTotal == intent.Amount {
		if err := s.db.UpdateIntentStatus(ctx, intentID, models.IntentStatusFulfilled); err != nil {
			return nil, fmt.Errorf("failed to update intent status: %v", err)
		}
	}

	return fulfillment, nil
}
