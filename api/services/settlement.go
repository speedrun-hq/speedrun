package services

import (
	"context"
	"errors"
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
	// IntentSettledEventName is the name of the intent settled event
	IntentSettledEventName = "IntentSettled"

	// IntentSettledWithCallEventName is the name of the intent settled with call event
	IntentSettledWithCallEventName = "IntentSettledWithCall"

	// IntentSettledRequiredTopics is the minimum number of topics required in a log
	IntentSettledRequiredTopics = 3

	// IntentSettledRequiredFields is the number of fields expected in the event data
	IntentSettledRequiredFields = 5

	// IntentSettledWithCallRequiredFields is the number of fields expected in the event data for call intents
	IntentSettledWithCallRequiredFields = 6
)

// SettlementService handles monitoring and processing of settlement events
type SettlementService struct {
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

// NewSettlementService creates a new SettlementService instance
func NewSettlementService(
	client *ethclient.Client,
	clientResolver ClientResolver,
	db db.Database,
	intentSettledEventABI string,
	chainID uint64,
	logger zerolog.Logger,
) (*SettlementService, error) {
	logger = logger.With().Uint64(logging.FieldChain, chainID).Logger()

	// Parse the contract ABI
	parsedABI, err := abi.JSON(strings.NewReader(intentSettledEventABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %v", err)
	}

	// Create cleanup context
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())

	return &SettlementService{
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

func (s *SettlementService) StartListening(ctx context.Context, contractAddress common.Address) error {
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
				s.abi.Events[IntentSettledEventName].ID,
				s.abi.Events[IntentSettledWithCallEventName].ID,
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
		Msg("Successfully subscribed to settlement events")

	s.startGoroutine("settlement-processor", func() {
		s.processEventLogs(s.cleanupCtx, sub, logs, subID, contractAddress)
	})
	return nil
}

func (s *SettlementService) processEventLogs(
	ctx context.Context,
	sub ethereum.Subscription,
	logs chan types.Log,
	subID string,
	contractAddress common.Address,
) {
	defer func() {
		sub.Unsubscribe()
		// Remove the subscription from the map when done
		s.mu.Lock()
		delete(s.subs, subID)
		s.mu.Unlock()
		s.logger.Info().
			Str("subscription_id", subID).
			Msg("Ended settlement event log processing")
	}()

	s.logger.Info().
		Str("subscription_id", subID).
		Msg("Starting settlement event log processing")

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
					Msg("Settlement subscription error")
				// Try to resubscribe
				newSub, err := s.handleSubscriptionError(ctx, sub, logs, subID, contractAddress)
				if err != nil {
					s.logger.Error().Err(err).Msg("CRITICAL: Failed to resubscribe settlement service")
					return
				}
				// Update the subscription and continue the loop
				sub = newSub
			}
		case vLog, ok := <-logs:
			if !ok {
				s.logger.Error().
					Str("subscription_id", subID).
					Msg("Settlement log channel closed unexpectedly")
				return
			}

			s.logger.Info().
				Uint64(logging.FieldBlock, vLog.BlockNumber).
				Str("tx_hash", vLog.TxHash.Hex()).
				Msg("SETTLEMENT EVENT RECEIVED")

			if err := s.processLog(ctx, vLog); err != nil {
				s.logger.Error().Err(err).Msg("Error processing settlement log")
				continue
			}
		case <-debugTicker.C:
			// Extra debugging info
			s.logger.Debug().
				Str("subscription_id", subID).
				Msg("Settlement subscription still active")
		case <-ctx.Done():
			s.logger.Debug().Msg("Context cancelled, stopping settlement event processing")
			return
		}
	}
}

func (s *SettlementService) handleSubscriptionError(
	ctx context.Context,
	oldSub ethereum.Subscription,
	logs chan types.Log,
	subID string,
	contractAddress common.Address,
) (ethereum.Subscription, error) {
	oldSub.Unsubscribe()
	s.mu.Lock()
	delete(s.subs, subID)
	s.mu.Unlock()

	// Implement exponential backoff for retry
	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Check if context is cancelled
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		query := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddress},
			Topics: [][]common.Hash{
				{
					s.abi.Events[IntentSettledEventName].ID,
					s.abi.Events[IntentSettledWithCallEventName].ID,
				},
			},
		}

		// Try to resubscribe
		newSub, err := s.client.SubscribeFilterLogs(ctx, query, logs)
		if err == nil {
			// Store the new subscription
			s.mu.Lock()
			s.subs[subID] = newSub
			s.mu.Unlock()
			s.logger.Debug().Msg("Successfully resubscribed to settlement events")
			return newSub, nil
		}

		// If we reach here, resubscription failed
		backoffTime := time.Duration(1<<attempt) * time.Second
		if backoffTime > 30*time.Second {
			backoffTime = 30 * time.Second
		}
		s.logger.Debug().
			Int("attempt", attempt+1).
			Int("max_attempts", maxRetries).
			Err(err).
			Dur("backoff_time", backoffTime).
			Msg("Settlement service resubscription attempt failed")

		select {
		case <-time.After(backoffTime):
			// Continue with next retry
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("failed to resubscribe to settlement events after %d attempts", maxRetries)
}

func (s *SettlementService) processLog(ctx context.Context, vLog types.Log) error {
	if err := s.validateLog(vLog); err != nil {
		return err
	}

	event, err := s.extractEventData(vLog)
	if err != nil {
		return err
	}

	// Get related intent to associate with settlement
	intent, err := s.db.GetIntent(ctx, event.IntentID)
	if err != nil {
		return fmt.Errorf("failed to get intent: %v", err)
	}

	// Important: Use the destination chain client for settlement events
	// Settlement events happen on the destination chain
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

	settlement, err := event.ToSettlement(client)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get block timestamp")
		// Continue with what we have
	}

	// Add a warning log if the chain IDs don't match and we're using the default client
	if intent.DestinationChain != s.chainID && client == s.client {
		s.logger.Warn().
			Uint64("service_chain", s.chainID).
			Uint64("destination_chain", intent.DestinationChain).
			Msg("Using client for different chain to fetch timestamp for settlement event")
	}

	// Process the event
	return s.CreateSettlement(ctx, settlement)
}

func (s *SettlementService) validateLog(vLog types.Log) error {
	if len(vLog.Topics) < IntentSettledRequiredTopics {
		return fmt.Errorf(
			"invalid log: expected at least %d topics, got %d",
			IntentSettledRequiredTopics,
			len(vLog.Topics),
		)
	}

	// Check if the event signature matches one of our expected event types
	eventSig := vLog.Topics[0]
	if eventSig != s.abi.Events[IntentSettledEventName].ID &&
		eventSig != s.abi.Events[IntentSettledWithCallEventName].ID {
		return fmt.Errorf("invalid event signature: %s", eventSig.Hex())
	}

	return nil
}

func (s *SettlementService) extractEventData(vLog types.Log) (*models.IntentSettledEvent, error) {
	// Determine if this is a standard settlement or a call settlement
	isCallSettlement := vLog.Topics[0] == s.abi.Events[IntentSettledWithCallEventName].ID

	// Parse indexed parameters from topics
	intentID := vLog.Topics[1].Hex()

	// Convert asset address to proper format
	assetAddr := common.BytesToAddress(vLog.Topics[2].Bytes())
	asset := assetAddr.Hex()

	// Convert receiver address to proper format
	receiverAddr := common.BytesToAddress(vLog.Topics[3].Bytes())
	receiver := receiverAddr.Hex()

	// Determine which event name to use for unpacking
	eventName := IntentSettledEventName
	if isCallSettlement {
		eventName = IntentSettledWithCallEventName
	}

	// Parse non-indexed parameters from data
	unpacked, err := s.abi.Unpack(eventName, vLog.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %v", err)
	}

	// Check required fields based on event type
	requiredFields := IntentSettledRequiredFields
	if isCallSettlement {
		requiredFields = IntentSettledWithCallRequiredFields
	}

	if len(unpacked) < requiredFields {
		return nil, fmt.Errorf("invalid event data: expected at least %d fields, got %d", requiredFields, len(unpacked))
	}

	// Extract values from unpacked data
	// The order should match the non-indexed parameters in the event definition
	amount := unpacked[0].(*big.Int)
	fulfilled := unpacked[1].(bool)
	fulfillerAddr := unpacked[2].(common.Address)
	fulfiller := fulfillerAddr.Hex()
	actualAmount := unpacked[3].(*big.Int)
	paidTip := unpacked[4].(*big.Int)

	// Create event with basic fields
	event := &models.IntentSettledEvent{
		IntentID:     intentID,
		Asset:        asset,
		Amount:       amount,
		Receiver:     receiver,
		Fulfilled:    fulfilled,
		Fulfiller:    fulfiller,
		ActualAmount: actualAmount,
		PaidTip:      paidTip,
		BlockNumber:  vLog.BlockNumber,
		TxHash:       vLog.TxHash.Hex(),
		IsCall:       isCallSettlement,
	}

	// Extract call data if present
	if isCallSettlement && len(unpacked) > 5 {
		if callData, ok := unpacked[5].([]byte); ok {
			event.Data = callData
		} else {
			s.logger.Warn().
				Interface("call_data", unpacked[5]).
				Msg("Invalid call data in settlement event")
		}
	}

	return event, nil
}

// GetSettlement get settlement from database
func (s *SettlementService) GetSettlement(ctx context.Context, id string) (*models.Settlement, error) {
	settlement, err := s.db.GetSettlement(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get settlement: %v", err)
	}

	return settlement, nil
}

// ListSettlements lists all settlements from the database
func (s *SettlementService) ListSettlements(ctx context.Context) ([]*models.Settlement, error) {
	settlements, err := s.db.ListSettlements(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list settlements: %v", err)
	}

	return settlements, nil
}

// CreateSettlement creates a new settlement
func (s *SettlementService) CreateSettlement(ctx context.Context, settlement *models.Settlement) error {
	// Check if settlement already exists
	existingSettlement, err := s.db.GetSettlement(ctx, settlement.ID)
	if err != nil {
		if !errors.Is(err, db.ErrNotFound) {
			return fmt.Errorf("failed to check for existing settlement: %v", err)
		}
	} else if existingSettlement != nil {
		s.logger.Debug().
			Str("settlement_id", settlement.ID).
			Msg("Settlement already exists, skipping creation")
		return nil
	}

	// Create the settlement
	if err := s.db.CreateSettlement(ctx, settlement); err != nil {
		return fmt.Errorf("failed to create settlement: %v", err)
	}

	// Mark failed if settlement indicates unsuccessful fulfillment; otherwise settled
	status := models.IntentStatusSettled
	if !settlement.Fulfilled {
		status = models.IntentStatusFailed
	}
	if err := s.db.UpdateIntentStatus(ctx, settlement.ID, status); err != nil {
		return fmt.Errorf("failed to update intent status: %v", err)
	}

	return nil
}

// GetSubscriptionCount returns the number of active subscriptions
func (s *SettlementService) GetSubscriptionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.subs)
}

// Restart properly restarts the service by shutting down existing goroutines and starting new ones
func (s *SettlementService) Restart(ctx context.Context, contractAddress common.Address) error {
	s.logger.Info().Msg("Restarting settlement service...")

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
func (s *SettlementService) UnsubscribeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug().
		Int("active_subscriptions", len(s.subs)).
		Msg("Unsubscribing from all settlement subscriptions")

	for id, sub := range s.subs {
		sub.Unsubscribe()
		s.logger.Debug().
			Str("subscription_id", id).
			Msg("Unsubscribed from settlement subscription")
		delete(s.subs, id)
	}
}

// CreateCallSettlement creates a new settlement with call data
func (s *SettlementService) CreateCallSettlement(
	ctx context.Context,
	intentID,
	asset,
	amount,
	receiver string,
	fulfilled bool,
	fulfiller,
	actualAmount,
	paidTip,
	txHash,
	callData string,
) error {
	// Validate intent exists
	intent, err := s.db.GetIntent(ctx, intentID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return fmt.Errorf("intent not found: %s", intentID)
		}
		return fmt.Errorf("failed to get intent: %v", err)
	}

	// Verify this is a call intent
	if !intent.IsCall {
		return fmt.Errorf("intent is not a call intent: %s", intentID)
	}

	// Get block timestamp if available
	var timestamp time.Time
	if txHash != "" && strings.HasPrefix(txHash, "0x") {
		var client *ethclient.Client
		if s.clientResolver != nil && intent.DestinationChain != 0 {
			destClient, err := s.clientResolver.GetClient(intent.DestinationChain)
			if err == nil {
				client = destClient
			} else {
				s.logger.Warn().Err(err).Msg("Failed to get destination chain client for manual settlement, using default client")
				client = s.client
			}
		} else {
			client = s.client
		}

		txHashObj := common.HexToHash(txHash)
		_, isPending, err := client.TransactionByHash(ctx, txHashObj)
		if err == nil && !isPending {
			receipt, err := client.TransactionReceipt(ctx, txHashObj)
			if err == nil {
				block, err := client.BlockByNumber(ctx, big.NewInt(int64(receipt.BlockNumber.Uint64())))
				if err == nil {
					timestamp = time.Unix(int64(block.Time()), 0)
				} else {
					timestamp = time.Now()
				}
			} else {
				timestamp = time.Now()
			}
		} else {
			timestamp = time.Now()
		}
	} else {
		timestamp = time.Now()
	}

	settlement := &models.Settlement{
		ID:           intentID,
		Asset:        asset,
		Amount:       amount,
		Receiver:     receiver,
		Fulfilled:    fulfilled,
		Fulfiller:    fulfiller,
		ActualAmount: actualAmount,
		PaidTip:      paidTip,
		TxHash:       txHash,
		CreatedAt:    timestamp,
		UpdatedAt:    timestamp,
		IsCall:       true,
		CallData:     callData,
	}

	return s.CreateSettlement(ctx, settlement)
}

// Shutdown gracefully shuts down the service and waits for all goroutines to complete
func (s *SettlementService) Shutdown(timeout time.Duration) error {
	s.shutdownMu.Lock()
	if s.isShutdown {
		s.shutdownMu.Unlock()
		return nil // Already shutdown
	}
	s.isShutdown = true
	s.shutdownMu.Unlock()

	s.logger.Info().Msg("Shutting down SettlementService...")

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
		s.logger.Info().Msg("SettlementService shutdown completed successfully")
		return nil
	case <-time.After(timeout):
		s.logger.Error().
			Dur("timeout", timeout).
			Msg("SettlementService shutdown timed out")
		return fmt.Errorf("shutdown timed out after %v", timeout)
	}
}

// IsShutdown returns whether the service is in shutdown state
func (s *SettlementService) IsShutdown() bool {
	s.shutdownMu.RLock()
	defer s.shutdownMu.RUnlock()
	return s.isShutdown
}

// startGoroutine safely starts a goroutine with proper cleanup tracking
func (s *SettlementService) startGoroutine(name string, fn func()) {
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
					Msg("Panic in goroutine")
			}
		}()

		fn()
	}()
}

// ActiveGoroutines returns the current count of active goroutines
func (s *SettlementService) ActiveGoroutines() int32 {
	return atomic.LoadInt32(&s.activeGoroutines)
}
