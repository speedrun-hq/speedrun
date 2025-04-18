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
	"github.com/speedrun-hq/speedrun/api/db"
	"github.com/speedrun-hq/speedrun/api/models"
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
	client         *ethclient.Client
	clientResolver ClientResolver
	db             db.Database
	abi            abi.ABI
	chainID        uint64
	subs           map[string]ethereum.Subscription
}

// NewFulfillmentService creates a new FulfillmentService instance
func NewFulfillmentService(client *ethclient.Client, clientResolver ClientResolver, db db.Database, intentFulfilledEventABI string, chainID uint64) (*FulfillmentService, error) {
	// Parse the contract ABI
	parsedABI, err := abi.JSON(strings.NewReader(intentFulfilledEventABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %v", err)
	}

	return &FulfillmentService{
		client:         client,
		clientResolver: clientResolver,
		db:             db,
		abi:            parsedABI,
		chainID:        chainID,
		subs:           make(map[string]ethereum.Subscription),
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
		return err
	}

	return nil
}

// processLog processes a single fulfillment event log
func (s *FulfillmentService) processLog(ctx context.Context, vLog types.Log) error {
	if err := s.validateLog(vLog); err != nil {
		return err
	}

	event, err := s.extractEventData(vLog)
	if err != nil {
		return err
	}

	// Get related intent to associate with fulfillment
	intent, err := s.db.GetIntent(ctx, event.IntentID)
	if err != nil {
		return fmt.Errorf("failed to get intent: %v", err)
	}

	// Important: Use the destination chain client for fulfillment events
	// Fulfillment events happen on the destination chain
	var client *ethclient.Client
	if s.clientResolver != nil && intent.DestinationChain != 0 {
		// Try to get the destination chain client
		destClient, err := s.clientResolver.GetClient(intent.DestinationChain)
		if err == nil {
			client = destClient
		} else {
			log.Printf("Warning: Failed to get destination chain client: %v, using default client", err)
			client = s.client
		}
	} else {
		client = s.client
	}

	fulfillment, err := event.ToFulfillment(client)
	if err != nil {
		log.Printf("Warning: Failed to get block timestamp: %v", err)
		// Continue with what we have
	}

	// Add a warning log if the chain IDs don't match and we're using the default client
	if intent.DestinationChain != s.chainID && client == s.client {
		log.Printf("Warning: Using client for chain %d to fetch timestamp for fulfillment event on chain %d",
			s.chainID, intent.DestinationChain)
	}

	// Ensure we have all the necessary data
	if fulfillment.Asset == "" {
		fulfillment.Asset = intent.Token
	}
	if fulfillment.Amount == "" {
		fulfillment.Amount = intent.Amount
	}
	if fulfillment.Receiver == "" {
		fulfillment.Receiver = intent.Recipient
	}

	// Convert padded blockchain addresses to standard Ethereum addresses
	if len(fulfillment.Asset) > 42 && strings.HasPrefix(fulfillment.Asset, "0x") {
		// Extract last 40 chars and add 0x prefix
		fulfillment.Asset = "0x" + fulfillment.Asset[len(fulfillment.Asset)-40:]
	}

	if len(fulfillment.Receiver) > 42 && strings.HasPrefix(fulfillment.Receiver, "0x") {
		// Extract last 40 chars and add 0x prefix
		fulfillment.Receiver = "0x" + fulfillment.Receiver[len(fulfillment.Receiver)-40:]
	}

	// Save fulfillment directly to database, preserving the block timestamp
	if err := s.db.CreateFulfillment(ctx, fulfillment); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			log.Printf("Skipping duplicate fulfillment: %s", event.IntentID)
			return nil
		}
		return fmt.Errorf("failed to create fulfillment: %v", err)
	}

	// Update intent status
	if err := s.db.UpdateIntentStatus(ctx, event.IntentID, models.IntentStatusFulfilled); err != nil {
		return fmt.Errorf("failed to update intent status: %v", err)
	}

	return nil
}

func (s *FulfillmentService) validateLog(vLog types.Log) error {
	if len(vLog.Topics) < IntentFulfilledRequiredTopics {
		return fmt.Errorf("invalid log: expected at least %d topics, got %d", IntentFulfilledRequiredTopics, len(vLog.Topics))
	}
	return nil
}

func (s *FulfillmentService) extractEventData(vLog types.Log) (*models.IntentFulfilledEvent, error) {
	amount := new(big.Int).SetBytes(vLog.Data)

	// Format addresses properly by extracting the standard Ethereum address from padded topics
	assetAddr := vLog.Topics[2].Hex()
	if len(assetAddr) > 42 && strings.HasPrefix(assetAddr, "0x") {
		assetAddr = "0x" + assetAddr[len(assetAddr)-40:]
	}

	receiverAddr := vLog.Topics[3].Hex()
	if len(receiverAddr) > 42 && strings.HasPrefix(receiverAddr, "0x") {
		receiverAddr = "0x" + receiverAddr[len(receiverAddr)-40:]
	}

	event := &models.IntentFulfilledEvent{
		IntentID:    vLog.Topics[1].Hex(),
		Asset:       assetAddr,
		Amount:      amount,
		Receiver:    receiverAddr,
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

	// For API-created fulfillments, try to get the block timestamp from the transaction
	// This provides more accurate timestamps even for API-created fulfillments
	var timestamp time.Time

	// Use the destination chain client if possible, as fulfillments happen on the destination chain
	var client *ethclient.Client
	if s.clientResolver != nil && intent.DestinationChain != 0 {
		destClient, err := s.clientResolver.GetClient(intent.DestinationChain)
		if err == nil {
			client = destClient
		} else {
			log.Printf("Warning: Failed to get destination chain client for manual fulfillment: %v, using default client", err)
			client = s.client
		}
	} else {
		client = s.client
	}

	// If the destination chain doesn't match our service chain and we're using the default client,
	// log a warning about potentially incorrect timestamps
	if intent.DestinationChain != s.chainID && client == s.client && txHash != "" {
		log.Printf("Warning: Manual fulfillment using client for chain %d to fetch timestamp for transaction on chain %d",
			s.chainID, intent.DestinationChain)
	}

	if txHash != "" && strings.HasPrefix(txHash, "0x") {
		txHashObj := common.HexToHash(txHash)
		_, isPending, err := client.TransactionByHash(ctx, txHashObj)
		if err == nil && !isPending {
			// If we can get the transaction, try to get its receipt to find the block number
			receipt, err := client.TransactionReceipt(ctx, txHashObj)
			if err == nil {
				// If we have the receipt, get the block to find the timestamp
				block, err := client.BlockByNumber(ctx, big.NewInt(int64(receipt.BlockNumber.Uint64())))
				if err == nil {
					timestamp = time.Unix(int64(block.Time()), 0)
					log.Printf("Using blockchain timestamp for manual fulfillment of intent %s: %s (block #%d, tx: %s)",
						intentID, timestamp.Format(time.RFC3339), receipt.BlockNumber.Uint64(), txHash)
				} else {
					log.Printf("Warning: Failed to get block for timestamp in manual fulfillment of intent %s (tx: %s): %v, using current time",
						intentID, txHash, err)
					timestamp = time.Now()
				}
			} else {
				log.Printf("Warning: Failed to get transaction receipt in manual fulfillment of intent %s (tx: %s): %v, using current time",
					intentID, txHash, err)
				timestamp = time.Now()
			}
		} else {
			if err != nil {
				log.Printf("Warning: Failed to get transaction in manual fulfillment of intent %s (tx: %s): %v, using current time",
					intentID, txHash, err)
			} else if isPending {
				log.Printf("Warning: Transaction is still pending in manual fulfillment of intent %s (tx: %s), using current time",
					intentID, txHash)
			}
			timestamp = time.Now()
		}
	} else {
		// No valid txHash, use current time
		log.Printf("Warning: No valid transaction hash provided for manual fulfillment of intent %s, using current time", intentID)
		timestamp = time.Now()
	}

	fulfillment := &models.Fulfillment{
		ID:        intentID,
		Asset:     intent.Token,
		Amount:    intent.Amount,
		Receiver:  intent.Recipient,
		TxHash:    txHash,
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
	}

	// Convert padded blockchain addresses to standard Ethereum addresses
	if len(fulfillment.Asset) > 42 && strings.HasPrefix(fulfillment.Asset, "0x") {
		// Extract last 40 chars and add 0x prefix
		fulfillment.Asset = "0x" + fulfillment.Asset[len(fulfillment.Asset)-40:]
	}

	if len(fulfillment.Receiver) > 42 && strings.HasPrefix(fulfillment.Receiver, "0x") {
		// Extract last 40 chars and add 0x prefix
		fulfillment.Receiver = "0x" + fulfillment.Receiver[len(fulfillment.Receiver)-40:]
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
