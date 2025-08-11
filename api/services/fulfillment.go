package services

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
	"github.com/speedrun-hq/speedrun/api/db"
	"github.com/speedrun-hq/speedrun/api/logging"
	"github.com/speedrun-hq/speedrun/api/models"
)

// Constants for event processing
const (
	// IntentFulfilledEventName is the name of the intent fulfilled event
	IntentFulfilledEventName = "IntentFulfilled"

	// IntentFulfilledWithCallEventName is the name of the intent fulfilled with call event
	IntentFulfilledWithCallEventName = "IntentFulfilledWithCall"

	// IntentFulfilledRequiredTopics is the minimum number of topics required in a log
	IntentFulfilledRequiredTopics = 3

	// IntentFulfilledWithCallRequiredFields is the number of fields expected in the event data for call intents
	IntentFulfilledWithCallRequiredFields = 4
)

// FulfillmentService handles monitoring and processing of fulfillment events
type FulfillmentService struct {
	client         *ethclient.Client
	clientResolver ClientResolver
	db             db.Database
	abi            abi.ABI
	chainID        uint64
	subs           map[string]ethereum.Subscription
	mu             sync.Mutex
	logger         zerolog.Logger

	// Goroutine tracking
	activeGoroutines int32 // Counter for active goroutines

	// Goroutine cleanup management
	cleanupCtx    context.Context    // Context for cleanup operations
	cleanupCancel context.CancelFunc // Cancel function for cleanup context
	goroutineWg   sync.WaitGroup     // WaitGroup to track all goroutines
	isShutdown    bool               // Flag to prevent new goroutines after shutdown
	shutdownMu    sync.RWMutex       // Mutex for shutdown operations
}

// NewFulfillmentService creates a new FulfillmentService instance
func NewFulfillmentService(
	client *ethclient.Client,
	clientResolver ClientResolver,
	db db.Database,
	intentFulfilledEventABI string,
	chainID uint64,
	logger zerolog.Logger,
) (*FulfillmentService, error) {
	// Parse the contract ABI
	parsedABI, err := abi.JSON(strings.NewReader(intentFulfilledEventABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %v", err)
	}

	// Create cleanup context
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())

	return &FulfillmentService{
		client:         client,
		clientResolver: clientResolver,
		db:             db,
		abi:            parsedABI,
		chainID:        chainID,
		subs:           make(map[string]ethereum.Subscription),
		logger:         logger.With().Uint64(logging.FieldChain, chainID).Logger(),
		cleanupCtx:     cleanupCtx,
		cleanupCancel:  cleanupCancel,
	}, nil
}

// StartListening starts listening for fulfillment events on all chains
func (s *FulfillmentService) StartListening(ctx context.Context, contractAddress common.Address) error {
	// Check if service is shutdown
	if s.IsShutdown() {
		return fmt.Errorf("cannot start listening: service is shutdown")
	}

	// Check if service is already running - prevent multiple starts
	activeGoroutines := atomic.LoadInt32(&s.activeGoroutines)
	if activeGoroutines > 0 {
		s.logger.Info().
			Int32("active_goroutines", activeGoroutines).
			Msg("Service already running, skipping start")
		return nil
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics: [][]common.Hash{
			{
				s.abi.Events[IntentFulfilledEventName].ID,
				s.abi.Events[IntentFulfilledWithCallEventName].ID,
			},
		},
	}

	logs := make(chan types.Log)
	sub, err := s.client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return fmt.Errorf("failed to subscribe to logs: %v", err)
	}

	// Store the subscription with a unique ID
	subID := contractAddress.Hex()
	s.mu.Lock()
	s.subs[subID] = sub
	s.mu.Unlock()

	s.logger.Info().
		Str("contract", contractAddress.Hex()).
		Msg("Successfully subscribed to fulfillment events")

	s.startGoroutine("fulfillment-processor", func() {
		s.processEventLogs(s.cleanupCtx, sub, logs, subID)
	})
	return nil
}

// processEventLogs handles the event processing loop for the subscription.
// It manages subscription errors, log processing, and context cancellation.
func (s *FulfillmentService) processEventLogs(
	ctx context.Context,
	sub ethereum.Subscription,
	logs chan types.Log,
	subID string,
) {
	// Get the contract address from subID (which we set to contract address hex)
	contractAddress := common.HexToAddress(subID)

	defer func() {
		sub.Unsubscribe()
		// Remove the subscription from the map when done
		s.mu.Lock()
		delete(s.subs, subID)
		s.mu.Unlock()
		s.logger.Debug().
			Str("subscription_id", subID).
			Msg("Ended fulfillment event log processing")
	}()

	s.logger.Info().
		Str("subscription_id", subID).
		Msg("Starting fulfillment event log processing")

	// Add a ticker for debugging to periodically log subscription status
	debugTicker := time.NewTicker(30 * time.Second)
	defer debugTicker.Stop()

	for {
		select {
		case err := <-sub.Err():
			if err != nil {
				s.logger.Error().
					Str("subscription_id", subID).
					Err(err).
					Msg("Fulfillment subscription error")
				// Try to resubscribe
				if err := s.handleSubscriptionError(ctx, sub, logs, subID, contractAddress); err != nil {
					s.logger.Error().Err(err).Msg("CRITICAL: Failed to resubscribe fulfillment service")
					return
				}
			}
		case vLog, ok := <-logs:
			if !ok {
				s.logger.Error().
					Str("subscription_id", subID).
					Msg("Fulfillment log channel closed unexpectedly")
				return
			}

			s.logger.Info().
				Uint64(logging.FieldBlock, vLog.BlockNumber).
				Str("tx_hash", vLog.TxHash.Hex()).
				Msg("FULFILLMENT EVENT RECEIVED")

			if err := s.processLog(ctx, vLog); err != nil {
				s.logger.Error().Err(err).Msg("Error processing fulfillment log")
				continue
			}
		case <-debugTicker.C:
			// Extra debugging info
			s.logger.Debug().
				Str("subscription_id", subID).
				Msg("Fulfillment subscription still active")
		case <-ctx.Done():
			s.logger.Debug().Msg("Context cancelled, stopping fulfillment event processing")
			return
		}
	}
}

// handleSubscriptionError attempts to recover from a subscription error by resubscribing.
func (s *FulfillmentService) handleSubscriptionError(
	ctx context.Context,
	oldSub ethereum.Subscription,
	logs chan types.Log,
	subID string,
	contractAddress common.Address,
) error {
	oldSub.Unsubscribe()
	s.mu.Lock()
	delete(s.subs, subID)
	s.mu.Unlock()

	// Implement exponential backoff for retry
	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Check if context is cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Create a new query with both event types
		query := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{
					s.abi.Events[IntentFulfilledEventName].ID,
					s.abi.Events[IntentFulfilledWithCallEventName].ID,
				},
			},
		}

		// Try to resubscribe
		newSub, err := s.client.SubscribeFilterLogs(ctx, query, logs)
		if err == nil {
			// Update the subscription
			s.mu.Lock()
			s.subs[subID] = newSub
			s.mu.Unlock()
			s.logger.Debug().Msg("Successfully resubscribed to fulfillment events")
			return nil
		}

		// If we reach here, resubscription failed
		backoffTime := time.Duration(1<<attempt) * time.Second
		if backoffTime > 30*time.Second {
			backoffTime = 30 * time.Second
		}
		s.logger.Debug().
			Int("attempt", attempt+1).
			Int("max_retries", maxRetries).
			Err(err).
			Dur("backoff_time", backoffTime).
			Msg("Resubscription attempt failed, retrying")

		select {
		case <-time.After(backoffTime):
			// Continue with next retry
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("failed to resubscribe after %d attempts", maxRetries)
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
			s.logger.Warn().Err(err).Msg("Failed to get destination chain client, using default client")
			client = s.client
		}
	} else {
		client = s.client
	}

	fulfillment, err := event.ToFulfillment(client)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get block timestamp")
		// Continue with what we have
	}

	// Add a warning log if the chain IDs don't match and we're using the default client
	if intent.DestinationChain != s.chainID && client == s.client {
		s.logger.Warn().
			Uint64("service_chain", s.chainID).
			Uint64("destination_chain", intent.DestinationChain).
			Msg("Using client for different chain to fetch timestamp for fulfillment event")
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
			s.logger.Debug().
				Str(logging.FieldIntent, event.IntentID).
				Msg("Skipping duplicate fulfillment")
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
	// Check if the log has the minimum required topics
	if len(vLog.Topics) < IntentFulfilledRequiredTopics {
		return fmt.Errorf(
			"invalid log: expected at least %d topics, got %d",
			IntentFulfilledRequiredTopics,
			len(vLog.Topics),
		)
	}

	// Check if the event signature matches one of our expected event types
	eventSig := vLog.Topics[0]
	if eventSig != s.abi.Events[IntentFulfilledEventName].ID &&
		eventSig != s.abi.Events[IntentFulfilledWithCallEventName].ID {
		return fmt.Errorf("invalid event signature: %s", eventSig.Hex())
	}

	return nil
}

func (s *FulfillmentService) extractEventData(vLog types.Log) (*models.IntentFulfilledEvent, error) {
	// Determine if this is a standard fulfillment or call fulfillment
	isCallFulfillment := vLog.Topics[0] == s.abi.Events[IntentFulfilledWithCallEventName].ID

	// Extract common data
	event := &models.IntentFulfilledEvent{
		IntentID:    vLog.Topics[1].Hex(),
		BlockNumber: vLog.BlockNumber,
		TxHash:      vLog.TxHash.Hex(),
		IsCall:      isCallFulfillment,
	}

	// Format addresses properly by extracting the standard Ethereum address from padded topics
	assetAddr := vLog.Topics[2].Hex()
	if len(assetAddr) > 42 && strings.HasPrefix(assetAddr, "0x") {
		assetAddr = "0x" + assetAddr[len(assetAddr)-40:]
	}
	event.Asset = assetAddr

	receiverAddr := vLog.Topics[3].Hex()
	if len(receiverAddr) > 42 && strings.HasPrefix(receiverAddr, "0x") {
		receiverAddr = "0x" + receiverAddr[len(receiverAddr)-40:]
	}
	event.Receiver = receiverAddr

	// Unpack data fields
	var err error
	if isCallFulfillment {
		// Unpack data for call fulfillment which should include amount and call data
		var unpacked []interface{}
		unpacked, err = s.abi.Unpack(IntentFulfilledWithCallEventName, vLog.Data)
		if err == nil && len(unpacked) >= IntentFulfilledWithCallRequiredFields {
			// Extract amount
			if amount, ok := unpacked[0].(*big.Int); ok && amount != nil {
				event.Amount = amount
			} else {
				err = fmt.Errorf("invalid amount in event data")
			}

			// Extract call data
			if callData, ok := unpacked[1].([]byte); ok {
				event.Data = callData
			} else {
				s.logger.Warn().
					Interface("call_data", unpacked[1]).
					Msg("Invalid call data in fulfillment event")
			}
		} else if err == nil {
			err = fmt.Errorf("insufficient data fields: expected %d, got %d",
				IntentFulfilledWithCallRequiredFields, len(unpacked))
		}
	} else {
		// Standard fulfillment just has the amount in the data
		event.Amount = new(big.Int).SetBytes(vLog.Data)
	}

	if err != nil {
		s.logger.Warn().Err(err).Msg("Error processing fulfillment event data")
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
			s.logger.Warn().Err(err).Msg("Failed to get destination chain client for manual fulfillment, using default client")
			client = s.client
		}
	} else {
		client = s.client
	}

	// If the destination chain doesn't match our service chain and we're using the default client,
	// log a warning about potentially incorrect timestamps
	if intent.DestinationChain != s.chainID && client == s.client && txHash != "" {
		s.logger.Warn().
			Uint64("service_chain", s.chainID).
			Uint64("destination_chain", intent.DestinationChain).
			Msg("Manual fulfillment using client for different chain to fetch timestamp")
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
					s.logger.Debug().
						Str(logging.FieldIntent, intentID).
						Str("timestamp", timestamp.Format(time.RFC3339)).
						Uint64(logging.FieldBlock, receipt.BlockNumber.Uint64()).
						Str("tx_hash", txHash).
						Msg("Using blockchain timestamp for manual fulfillment")
				} else {
					s.logger.Warn().
						Str(logging.FieldIntent, intentID).
						Str("tx_hash", txHash).
						Err(err).
						Msg("Failed to get block for timestamp in manual fulfillment, using current time")
					timestamp = time.Now()
				}
			} else {
				s.logger.Warn().
					Str(logging.FieldIntent, intentID).
					Str("tx_hash", txHash).
					Err(err).
					Msg("Failed to get transaction receipt in manual fulfillment, using current time")
				timestamp = time.Now()
			}
		} else {
			if err != nil {
				s.logger.Warn().
					Str(logging.FieldIntent, intentID).
					Str("tx_hash", txHash).
					Err(err).
					Msg("Failed to get transaction in manual fulfillment, using current time")
			} else if isPending {
				s.logger.Warn().
					Str(logging.FieldIntent, intentID).
					Str("tx_hash", txHash).
					Msg("Transaction is still pending in manual fulfillment, using current time")
			}
			timestamp = time.Now()
		}
	} else {
		// No valid txHash, use current time
		s.logger.Warn().
			Str(logging.FieldIntent, intentID).
			Msg("No valid transaction hash provided for manual fulfillment, using current time")
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
		IsCall:    intent.IsCall,
		CallData:  intent.CallData,
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

// CreateCallFulfillment creates a new fulfillment for an intent with call
func (s *FulfillmentService) CreateCallFulfillment(
	ctx context.Context,
	intentID, txHash string,
	callData string,
) error {
	// Validate intent exists
	intent, err := s.db.GetIntent(ctx, intentID)
	if err != nil {
		return fmt.Errorf("failed to get intent: %v", err)
	}
	if intent == nil {
		return fmt.Errorf("intent not found: %s", intentID)
	}

	// Verify this is a call intent
	if !intent.IsCall {
		return fmt.Errorf("intent is not a call intent: %s", intentID)
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
			s.logger.Warn().Err(err).Msg("Failed to get destination chain client for manual fulfillment, using default client")
			client = s.client
		}
	} else {
		client = s.client
	}

	// If the destination chain doesn't match our service chain and we're using the default client,
	// log a warning about potentially incorrect timestamps
	if intent.DestinationChain != s.chainID && client == s.client && txHash != "" {
		s.logger.Warn().
			Uint64("service_chain", s.chainID).
			Uint64("destination_chain", intent.DestinationChain).
			Msg("Manual fulfillment using client for different chain to fetch timestamp")
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
					s.logger.Debug().
						Str(logging.FieldIntent, intentID).
						Str("timestamp", timestamp.Format(time.RFC3339)).
						Uint64(logging.FieldBlock, receipt.BlockNumber.Uint64()).
						Str("tx_hash", txHash).
						Msg("Using blockchain timestamp for manual fulfillment")
				} else {
					s.logger.Warn().
						Str(logging.FieldIntent, intentID).
						Str("tx_hash", txHash).
						Err(err).
						Msg("Failed to get block for timestamp in manual fulfillment, using current time")
					timestamp = time.Now()
				}
			} else {
				s.logger.Warn().
					Str(logging.FieldIntent, intentID).
					Str("tx_hash", txHash).
					Err(err).
					Msg("Failed to get transaction receipt in manual fulfillment, using current time")
				timestamp = time.Now()
			}
		} else {
			if err != nil {
				s.logger.Warn().
					Str(logging.FieldIntent, intentID).
					Str("tx_hash", txHash).
					Err(err).
					Msg("Failed to get transaction in manual fulfillment, using current time")
			} else if isPending {
				s.logger.Warn().
					Str(logging.FieldIntent, intentID).
					Str("tx_hash", txHash).
					Msg("Transaction is still pending in manual fulfillment, using current time")
			}
			timestamp = time.Now()
		}
	} else {
		// No valid txHash, use current time
		s.logger.Warn().
			Str(logging.FieldIntent, intentID).
			Msg("No valid transaction hash provided for manual fulfillment, using current time")
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
		IsCall:    true,
		CallData:  callData,
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

// GetSubscriptionCount returns the number of active subscriptions
func (s *FulfillmentService) GetSubscriptionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.subs)
}

// Restart properly restarts the service by shutting down existing goroutines and starting new ones
func (s *FulfillmentService) Restart(ctx context.Context, contractAddress common.Address) error {
	s.logger.Info().Msg("Restarting fulfillment service...")

	// Check if service is shutdown
	if s.IsShutdown() {
		return fmt.Errorf("cannot restart: service is shutdown")
	}

	// Cancel the cleanup context to signal all existing goroutines to stop
	s.cleanupCancel()

	// Wait for existing goroutines to complete with a short timeout
	done := make(chan struct{})
	go func() {
		s.goroutineWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Debug().Msg("Existing goroutines stopped successfully")
	case <-time.After(5 * time.Second):
		s.logger.Warn().Msg("Timeout waiting for existing goroutines to stop")
	}

	// Unsubscribe from all subscriptions
	s.UnsubscribeAll()

	// Create a new cleanup context
	s.cleanupCtx, s.cleanupCancel = context.WithCancel(context.Background())

	// Reset goroutine counter
	atomic.StoreInt32(&s.activeGoroutines, 0)

	// Start the service again
	return s.StartListening(ctx, contractAddress)
}

// UnsubscribeAll unsubscribes from all active subscriptions
func (s *FulfillmentService) UnsubscribeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug().
		Int("active_subscriptions", len(s.subs)).
		Msg("Unsubscribing from all fulfillment subscriptions")

	for id, sub := range s.subs {
		sub.Unsubscribe()
		s.logger.Debug().
			Str("subscription_id", id).
			Msg("Unsubscribed from fulfillment subscription")
		delete(s.subs, id)
	}
}

// Shutdown gracefully shuts down the service and waits for all goroutines to complete
func (s *FulfillmentService) Shutdown(timeout time.Duration) error {
	s.shutdownMu.Lock()
	if s.isShutdown {
		s.shutdownMu.Unlock()
		return nil // Already shutdown
	}
	s.isShutdown = true
	s.shutdownMu.Unlock()

	s.logger.Info().Msg("Shutting down FulfillmentService...")

	// Cancel the cleanup context to signal all goroutines to stop
	s.cleanupCancel()

	// Unsubscribe from all subscriptions
	s.UnsubscribeAll()

	// Wait for all goroutines to complete with timeout
	done := make(chan struct{})
	go func() {
		s.goroutineWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info().Msg("FulfillmentService shutdown completed successfully")
		return nil
	case <-time.After(timeout):
		s.logger.Error().
			Dur("timeout", timeout).
			Msg("FulfillmentService shutdown timed out")
		return fmt.Errorf("shutdown timed out after %v", timeout)
	}
}

// IsShutdown returns whether the service is in shutdown state
func (s *FulfillmentService) IsShutdown() bool {
	s.shutdownMu.RLock()
	defer s.shutdownMu.RUnlock()
	return s.isShutdown
}

// startGoroutine safely starts a goroutine with proper cleanup tracking
func (s *FulfillmentService) startGoroutine(name string, fn func()) {
	s.shutdownMu.RLock()
	if s.isShutdown {
		s.shutdownMu.RUnlock()
		s.logger.Debug().
			Str("goroutine_name", name).
			Msg("Cannot start goroutine: service is shutdown")
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
				s.logger.Error().
					Str("goroutine_name", name).
					Any("panic", r).
					Msg("CRITICAL: Panic in goroutine")
			}
		}()

		fn()
	}()
}

// ActiveGoroutines returns the current count of active goroutines
func (s *FulfillmentService) ActiveGoroutines() int32 {
	return atomic.LoadInt32(&s.activeGoroutines)
}
