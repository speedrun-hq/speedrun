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
	logger         logger.Logger
}

// NewSettlementService creates a new SettlementService instance
func NewSettlementService(
	client *ethclient.Client,
	clientResolver ClientResolver,
	db db.Database,
	intentSettledEventABI string,
	chainID uint64,
	logger logger.Logger,
) (*SettlementService, error) {
	// Parse the contract ABI
	parsedABI, err := abi.JSON(strings.NewReader(intentSettledEventABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %v", err)
	}

	return &SettlementService{
		client:         client,
		clientResolver: clientResolver,
		db:             db,
		abi:            parsedABI,
		chainID:        chainID,
		subs:           make(map[string]ethereum.Subscription),
		logger:         logger,
	}, nil
}

func (s *SettlementService) StartListening(ctx context.Context, contractAddress common.Address) error {
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

	s.logger.InfoWithChain(s.chainID, "Successfully subscribed to settlement events for contract %s",
		contractAddress.Hex())

	go s.processEventLogs(ctx, sub, logs, subID, contractAddress)
	return nil
}

func (s *SettlementService) processEventLogs(ctx context.Context, sub ethereum.Subscription, logs chan types.Log, subID string, contractAddress common.Address) {
	defer func() {
		sub.Unsubscribe()
		// Remove the subscription from the map when done
		s.mu.Lock()
		delete(s.subs, subID)
		s.mu.Unlock()
		s.logger.InfoWithChain(s.chainID, "Ended settlement event log processing, subscription %s", subID)
	}()

	s.logger.NoticeWithChain(s.chainID, "Starting settlement event log processing, subscription %s", subID)

	// Add a ticker for debugging to periodically log subscription status
	debugTicker := time.NewTicker(30 * time.Second)
	defer debugTicker.Stop()

	for {
		select {
		case err := <-sub.Err():
			if err != nil {
				s.logger.ErrorWithChain(s.chainID, "Settlement subscription %s error: %v", subID, err)
				// Try to resubscribe
				newSub, err := s.handleSubscriptionError(ctx, sub, logs, subID, contractAddress)
				if err != nil {
					s.logger.ErrorWithChain(s.chainID, "CRITICAL: Failed to resubscribe settlement service: %v", err)
					return
				}
				// Update the subscription and continue the loop
				sub = newSub
			}
		case vLog, ok := <-logs:
			if !ok {
				s.logger.ErrorWithChain(s.chainID, "ERROR: Settlement log channel closed unexpectedly for %s", subID)
				return
			}

			s.logger.InfoWithChain(s.chainID, "SETTLEMENT EVENT RECEIVED: Block %d, TxHash %s", vLog.BlockNumber, vLog.TxHash.Hex())

			if err := s.processLog(ctx, vLog); err != nil {
				s.logger.Error("Error processing settlement log: %v", err)
				continue
			}
		case <-debugTicker.C:
			// Extra debugging info
			s.logger.DebugWithChain(s.chainID, "Settlement subscription %s still active", subID)
		case <-ctx.Done():
			s.logger.DebugWithChain(s.chainID, "Context cancelled, stopping settlement event processing")
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
			s.logger.DebugWithChain(s.chainID, "Successfully resubscribed to settlement events")
			return newSub, nil
		}

		// If we reach here, resubscription failed
		backoffTime := time.Duration(1<<attempt) * time.Second
		if backoffTime > 30*time.Second {
			backoffTime = 30 * time.Second
		}
		s.logger.Debug("Settlement service resubscription attempt %d/%d failed: %v. Retrying in %v",
			attempt+1, maxRetries, err, backoffTime)

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
			s.logger.Info("Warning: Failed to get destination chain client: %v, using default client", err)
			client = s.client
		}
	} else {
		client = s.client
	}

	settlement, err := event.ToSettlement(client)
	if err != nil {
		s.logger.Info("Warning: Failed to get block timestamp: %v", err)
		// Continue with what we have
	}

	// Add a warning log if the chain IDs don't match and we're using the default client
	if intent.DestinationChain != s.chainID && client == s.client {
		s.logger.Info("Warning: Using client for chain %d to fetch timestamp for settlement event on chain %d",
			s.chainID, intent.DestinationChain)
	}

	// Process the event
	return s.CreateSettlement(ctx, settlement)
}

func (s *SettlementService) validateLog(vLog types.Log) error {
	if len(vLog.Topics) < IntentSettledRequiredTopics {
		return fmt.Errorf("invalid log: expected at least %d topics, got %d", IntentSettledRequiredTopics, len(vLog.Topics))
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
			s.logger.Info("Warning: Invalid call data in settlement event: %v", unpacked[5])
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
	if err := s.db.CreateSettlement(ctx, settlement); err != nil {
		return fmt.Errorf("failed to create settlement: %v", err)
	}

	// Update intent status
	if err := s.db.UpdateIntentStatus(ctx, settlement.ID, models.IntentStatusSettled); err != nil {
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

// UnsubscribeAll unsubscribes from all active subscriptions
func (s *SettlementService) UnsubscribeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.DebugWithChain(s.chainID, "Unsubscribing from all settlement subscriptions (%d active subscriptions)", len(s.subs))

	for id, sub := range s.subs {
		sub.Unsubscribe()
		s.logger.DebugWithChain(s.chainID, "Unsubscribed from settlement subscription %s", id)
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
		return fmt.Errorf("failed to get intent: %v", err)
	}
	if intent == nil {
		return fmt.Errorf("intent not found: %s", intentID)
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
				s.logger.Info("Warning: Failed to get destination chain client for manual settlement: %v, using default client", err)
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
