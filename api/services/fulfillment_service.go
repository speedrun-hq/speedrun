package services

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/models"
)

// Constants for event processing
const (
	// IntentFulfilledEventName is the name of the intent fulfilled event
	IntentFulfilledEventName = "IntentFulfilled"

	// IntentFulfilledRequiredTopics is the minimum number of topics required in a log
	IntentFulfilledRequiredTopics = 3

	// IntentFulfilledRequiredFields is the number of fields expected in the event data
	IntentFulfilledRequiredFields = 3
)

// FulfillmentService handles monitoring and processing of fulfillment events
type FulfillmentService struct {
	client  *ethclient.Client
	db      db.Database
	abi     abi.ABI
	chainID uint64
	subs    map[string]ethereum.Subscription
}

// NewFulfillmentService creates a new FulfillmentService instance
func NewFulfillmentService(client *ethclient.Client, db db.Database, intentFulfilledEventABI string, chainID uint64) (*FulfillmentService, error) {
	// Parse the contract ABI
	parsedABI, err := abi.JSON(strings.NewReader(intentFulfilledEventABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %v", err)
	}

	return &FulfillmentService{
		client:  client,
		db:      db,
		abi:     parsedABI,
		chainID: chainID,
		subs:    make(map[string]ethereum.Subscription),
	}, nil
}

// StartListening starts listening for fulfillment events on all chains
func (s *FulfillmentService) StartListening(ctx context.Context, contractAddress common.Address) error {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics: [][]common.Hash{
			{s.abi.Events[IntentFulfilledEventName].ID},
		},
	}

	logs := make(chan types.Log)
	sub, err := s.client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return fmt.Errorf("failed to subscribe to logs: %v", err)
	}

	go s.processEventLogs(ctx, sub, logs)
	return nil
}

// processEventLogs handles the event processing loop for the subscription.
// It manages subscription errors, log processing, and context cancellation.
func (s *FulfillmentService) processEventLogs(ctx context.Context, sub ethereum.Subscription, logs chan types.Log) {
	defer sub.Unsubscribe()

	for {
		select {
		case err := <-sub.Err():
			if err != nil {
				log.Printf("Error in subscription: %v", err)
				if err := s.handleSubscriptionError(ctx, sub, logs); err != nil {
					return
				}
			}
		case vLog := <-logs:
			if err := s.processLog(ctx, vLog); err != nil {
				log.Printf("Error processing log: %v", err)
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}

// handleSubscriptionError attempts to recover from a subscription error by resubscribing.
func (s *FulfillmentService) handleSubscriptionError(ctx context.Context, oldSub ethereum.Subscription, logs chan types.Log) error {
	oldSub.Unsubscribe()

	// Get the contract address from the old subscription
	contractAddress := common.HexToAddress("0x0") // Default value
	if sub, ok := oldSub.(interface{ Query() ethereum.FilterQuery }); ok {
		if len(sub.Query().Addresses) > 0 {
			contractAddress = sub.Query().Addresses[0]
		}
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics: [][]common.Hash{
			{s.abi.Events[IntentInitiatedEventName].ID},
		},
	}

	_, err := s.client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		log.Printf("Failed to resubscribe: %v", err)
		return err
	}

	return nil
}

// processLog processes a single fulfillment event log
func (s *FulfillmentService) processLog(ctx context.Context, vLog types.Log) error {
	log.Printf("Processing log - Block: %d, TxHash: %s, Topics: %v", vLog.BlockNumber, vLog.TxHash.Hex(), vLog.Topics)
	if err := s.validateLog(vLog); err != nil {
		log.Printf("Log validation failed: %v", err)
		return err
	}
	event, err := s.extractEventData(vLog)
	if err != nil {
		log.Printf("Failed to extract event data: %v", err)
		return err
	}

	log.Printf("Extracted event - IntentID: %s, Asset: %s, Amount: %s, Receiver: %s, TxHash: %s",
		event.IntentID,
		event.Asset,
		event.Amount.String(),
		event.Receiver,
		event.TxHash)

	fulfillment := event.ToFulfillment()
	log.Printf("Created fulfillment - ID: %s, Asset: %s, Amount: %s, Receiver: %s, TxHash: %s",
		fulfillment.ID,
		fulfillment.Asset,
		fulfillment.Amount,
		fulfillment.Receiver,
		fulfillment.TxHash)

	// Process the event
	return s.CreateFulfillment(ctx, event.IntentID, fulfillment.TxHash)
}

func (s *FulfillmentService) validateLog(vLog types.Log) error {
	if len(vLog.Topics) < IntentFulfilledRequiredTopics {
		return fmt.Errorf("invalid log: expected at least %d topics, got %d", IntentFulfilledRequiredTopics, len(vLog.Topics))
	}
	return nil
}

func (s *FulfillmentService) extractEventData(vLog types.Log) (*models.IntentFulfilledEvent, error) {
	amount := new(big.Int).SetBytes(vLog.Data)
	event := &models.IntentFulfilledEvent{
		IntentID:    vLog.Topics[1].Hex(),
		Asset:       vLog.Topics[2].Hex(),
		Amount:      amount,
		Receiver:    vLog.Topics[3].Hex(),
		BlockNumber: vLog.BlockNumber,
		TxHash:      vLog.TxHash.Hex(),
	}

	return event, nil
}

// GetFulfillment retrieves a fulfillment by ID
func (s *FulfillmentService) GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error) {

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
func (s *FulfillmentService) CreateFulfillment(ctx context.Context, intentID, txHash string) error {
	// Validate intent exists
	intent, err := s.db.GetIntent(ctx, intentID)
	if err != nil {
		return fmt.Errorf("failed to get intent: %v", err)
	}
	if intent == nil {
		return fmt.Errorf("intent not found: %s", intentID)
	}

	// Create fulfillment
	now := time.Now()
	fulfillment := &models.Fulfillment{
		ID:        intentID,
		Asset:     intent.Token,
		Amount:    intent.Amount,
		Receiver:  intent.Recipient,
		TxHash:    txHash,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save fulfillment
	if err := s.db.CreateFulfillment(ctx, fulfillment); err != nil {
		return fmt.Errorf("failed to create fulfillment: %v", err)
	}

	// Update intent status
	if err := s.db.UpdateIntentStatus(ctx, intentID, models.IntentStatusFulfilled); err != nil {
		return fmt.Errorf("failed to update intent status: %v", err)
	}

	return nil
}
