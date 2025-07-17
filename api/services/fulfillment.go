package services

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/speedrun-hq/speedrun/api/logger"

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
	logger         logger.Logger

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
	logger logger.Logger,
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
		logger:         logger,
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

	s.logger.InfoWithChain(s.chainID, "Successfully subscribed to fulfillment events for contract %s",
		contractAddress.Hex())

	s.startGoroutine("fulfillment-processor", func() {
		s.processEventLogs(s.cleanupCtx, sub, logs, subID)
	})
	return nil
}

// processEventLogs handles the event processing loop for the subscription.
// It manages subscription errors, log processing, and context cancellation.
func (s *FulfillmentService) processEventLogs(ctx context.Context, sub ethereum.Subscription, logs chan types.Log, subID string) {
	// Get the contract address from subID (which we set to contract address hex)
	contractAddress := common.HexToAddress(subID)

	defer func() {
		sub.Unsubscribe()
		// Remove the subscription from the map when done
		s.mu.Lock()
		delete(s.subs, subID)
		s.mu.Unlock()
		s.logger.DebugWithChain(s.chainID, "Ended fulfillment event log processing, subscription %s", subID)
	}()

	s.logger.NoticeWithChain(s.chainID, "Starting fulfillment event log processing, subscription %s", subID)

	// Add a ticker for debugging to periodically log subscription status
	debugTicker := time.NewTicker(30 * time.Second)
	defer debugTicker.Stop()

	for {
		select {
		case err := <-sub.Err():
			if err != nil {
				s.logger.ErrorWithChain(s.chainID, "Fulfillment subscription %s error: %v", subID, err)
				// Try to resubscribe
				if err := s.handleSubscriptionError(ctx, sub, logs, subID, contractAddress); err != nil {
					s.logger.ErrorWithChain(s.chainID, "CRITICAL: Failed to resubscribe fulfillment service: %v", err)
					return
				}
			}
		case vLog, ok := <-logs:
			if !ok {
				s.logger.ErrorWithChain(s.chainID, "Fulfillment log channel closed unexpectedly for %s", subID)
				return
			}

			s.logger.InfoWithChain(s.chainID, "FULFILLMENT EVENT RECEIVED: Block %d, TxHash %s", vLog.BlockNumber, vLog.TxHash.Hex())

			if err := s.processLog(ctx, vLog); err != nil {
				s.logger.ErrorWithChain(s.chainID, "Error processing fulfillment log: %v", err)
				continue
			}
		case <-debugTicker.C:
			// Extra debugging info
			s.logger.DebugWithChain(s.chainID, "Fulfillment subscription %s still active", subID)
		case <-ctx.Done():
			s.logger.DebugWithChain(s.chainID, "Context cancelled, stopping fulfillment event processing")
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
			s.logger.Debug("Successfully resubscribed to fulfillment events")
			return nil
		}

		// If we reach here, resubscription failed
		backoffTime := time.Duration(1<<attempt) * time.Second
		if backoffTime > 30*time.Second {
			backoffTime = 30 * time.Second
		}
		s.logger.Debug("Resubscription attempt %d/%d failed: %v. Retrying in %v",
			attempt+1, maxRetries, err, backoffTime)

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
			s.logger.Info("Warning: Failed to get destination chain client: %v, using default client", err)
			client = s.client
		}
	} else {
		client = s.client
	}

	fulfillment, err := event.ToFulfillment(client)
	if err != nil {
		s.logger.Info("Warning: Failed to get block timestamp: %v", err)
		// Continue with what we have
	}

	// Add a warning log if the chain IDs don't match and we're using the default client
	if intent.DestinationChain != s.chainID && client == s.client {
		s.logger.Debug("Warning: Using client for chain %d to fetch timestamp for fulfillment event on chain %d",
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
			s.logger.Debug("Skipping duplicate fulfillment: %s", event.IntentID)
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
		return fmt.Errorf("invalid log: expected at least %d topics, got %d", IntentFulfilledRequiredTopics, len(vLog.Topics))
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
				s.logger.Info("Warning: Invalid call data in fulfillment event: %v", unpacked[1])
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
		s.logger.Info("Warning: Error processing fulfillment event data: %v", err)
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
			s.logger.Info("Warning: Failed to get destination chain client for manual fulfillment: %v, using default client", err)
			client = s.client
		}
	} else {
		client = s.client
	}

	// If the destination chain doesn't match our service chain and we're using the default client,
	// log a warning about potentially incorrect timestamps
	if intent.DestinationChain != s.chainID && client == s.client && txHash != "" {
		s.logger.Info("Warning: Manual fulfillment using client for chain %d to fetch timestamp for transaction on chain %d",
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
					s.logger.Debug("Using blockchain timestamp for manual fulfillment of intent %s: %s (block #%d, tx: %s)",
						intentID, timestamp.Format(time.RFC3339), receipt.BlockNumber.Uint64(), txHash)
				} else {
					s.logger.Info("Warning: Failed to get block for timestamp in manual fulfillment of intent %s (tx: %s): %v, using current time",
						intentID, txHash, err)
					timestamp = time.Now()
				}
			} else {
				s.logger.Info("Warning: Failed to get transaction receipt in manual fulfillment of intent %s (tx: %s): %v, using current time",
					intentID, txHash, err)
				timestamp = time.Now()
			}
		} else {
			if err != nil {
				s.logger.Info("Warning: Failed to get transaction in manual fulfillment of intent %s (tx: %s): %v, using current time",
					intentID, txHash, err)
			} else if isPending {
				s.logger.Info("Warning: Transaction is still pending in manual fulfillment of intent %s (tx: %s), using current time",
					intentID, txHash)
			}
			timestamp = time.Now()
		}
	} else {
		// No valid txHash, use current time
		s.logger.Info("Warning: No valid transaction hash provided for manual fulfillment of intent %s, using current time", intentID)
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
func (s *FulfillmentService) CreateCallFulfillment(ctx context.Context, intentID, txHash string, callData string) error {
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
			s.logger.Info("Warning: Failed to get destination chain client for manual fulfillment: %v, using default client", err)
			client = s.client
		}
	} else {
		client = s.client
	}

	// If the destination chain doesn't match our service chain and we're using the default client,
	// log a warning about potentially incorrect timestamps
	if intent.DestinationChain != s.chainID && client == s.client && txHash != "" {
		s.logger.Info("Warning: Manual fulfillment using client for chain %d to fetch timestamp for transaction on chain %d",
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
					s.logger.Debug("Using blockchain timestamp for manual fulfillment of intent %s: %s (block #%d, tx: %s)",
						intentID, timestamp.Format(time.RFC3339), receipt.BlockNumber.Uint64(), txHash)
				} else {
					s.logger.Info("Warning: Failed to get block for timestamp in manual fulfillment of intent %s (tx: %s): %v, using current time",
						intentID, txHash, err)
					timestamp = time.Now()
				}
			} else {
				s.logger.Info("Warning: Failed to get transaction receipt in manual fulfillment of intent %s (tx: %s): %v, using current time",
					intentID, txHash, err)
				timestamp = time.Now()
			}
		} else {
			if err != nil {
				s.logger.Info("Warning: Failed to get transaction in manual fulfillment of intent %s (tx: %s): %v, using current time",
					intentID, txHash, err)
			} else if isPending {
				s.logger.Info("Warning: Transaction is still pending in manual fulfillment of intent %s (tx: %s), using current time",
					intentID, txHash)
			}
			timestamp = time.Now()
		}
	} else {
		// No valid txHash, use current time
		s.logger.Info("Warning: No valid transaction hash provided for manual fulfillment of intent %s, using current time", intentID)
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

// UnsubscribeAll unsubscribes from all active subscriptions
func (s *FulfillmentService) UnsubscribeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug("Unsubscribing from all fulfillment subscriptions for chain %d (%d active subscriptions)",
		s.chainID, len(s.subs))

	for id, sub := range s.subs {
		sub.Unsubscribe()
		s.logger.Debug("Unsubscribed from fulfillment subscription %s on chain %d", id, s.chainID)
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

	s.logger.InfoWithChain(s.chainID, "Shutting down FulfillmentService...")

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
		s.logger.InfoWithChain(s.chainID, "FulfillmentService shutdown completed successfully")
		return nil
	case <-time.After(timeout):
		s.logger.ErrorWithChain(s.chainID, "FulfillmentService shutdown timed out after %v", timeout)
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
		s.logger.DebugWithChain(s.chainID, "Cannot start goroutine %s: service is shutdown", name)
		return
	}
	s.shutdownMu.RUnlock()

	s.goroutineWg.Add(1)

	go func() {
		defer func() {
			s.goroutineWg.Done()

			// Recover from panics
			if r := recover(); r != nil {
				s.logger.Error("CRITICAL: Panic in goroutine %s: %v", name, r)
			}
		}()

		fn()
	}()
}
