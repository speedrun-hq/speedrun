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
	"github.com/speedrun-hq/speedrun/api/db"
	"github.com/speedrun-hq/speedrun/api/models"
)

// Constants for event processing
const (
	// IntentSettledEventName is the name of the intent settled event
	IntentSettledEventName = "IntentSettled"

	// IntentSettledRequiredTopics is the minimum number of topics required in a log
	IntentSettledRequiredTopics = 3

	// IntentSettledRequiredFields is the number of fields expected in the event data
	IntentSettledRequiredFields = 5
)

type SettlementService struct {
	client         *ethclient.Client
	clientResolver ClientResolver
	db             db.Database
	abi            abi.ABI
	chainID        uint64
	subs           map[string]ethereum.Subscription
	mu             sync.Mutex
}

func NewSettlementService(client *ethclient.Client, clientResolver ClientResolver, db db.Database, intentSettledEventABI string, chainID uint64) (*SettlementService, error) {
	parsedABI, err := abi.JSON(strings.NewReader(intentSettledEventABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}

	return &SettlementService{
		client:         client,
		clientResolver: clientResolver,
		db:             db,
		abi:            parsedABI,
		chainID:        chainID,
		subs:           make(map[string]ethereum.Subscription),
	}, nil
}

func (s *SettlementService) StartListening(ctx context.Context, contractAddress common.Address) error {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics: [][]common.Hash{
			{s.abi.Events[IntentSettledEventName].ID},
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

func (s *SettlementService) processEventLogs(ctx context.Context, sub ethereum.Subscription, logs chan types.Log) {
	defer sub.Unsubscribe()

	for {
		select {
		case err := <-sub.Err():
			if err != nil {
				log.Printf("Error in subscription: %v", err)
				newSub, err := s.handleSubscriptionError(ctx, sub, logs)
				if err != nil {
					return
				}
				// Update the subscription and continue the loop
				sub = newSub
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

func (s *SettlementService) handleSubscriptionError(ctx context.Context, oldSub ethereum.Subscription, logs chan types.Log) (ethereum.Subscription, error) {
	oldSub.Unsubscribe()

	// Get the contract address from the old subscription
	contractAddress := common.HexToAddress("0x0") // Default value
	if sub, ok := oldSub.(interface{ Query() ethereum.FilterQuery }); ok {
		if len(sub.Query().Addresses) > 0 {
			contractAddress = sub.Query().Addresses[0]
		}
	}

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
				{s.abi.Events[IntentSettledEventName].ID},
			},
		}

		// Try to resubscribe
		newSub, err := s.client.SubscribeFilterLogs(ctx, query, logs)
		if err == nil {
			log.Printf("Successfully resubscribed to settlement events for chain %d", s.chainID)
			return newSub, nil
		}

		// If we reach here, resubscription failed
		backoffTime := time.Duration(1<<attempt) * time.Second
		if backoffTime > 30*time.Second {
			backoffTime = 30 * time.Second
		}
		log.Printf("Settlement service resubscription attempt %d/%d failed: %v. Retrying in %v",
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
			log.Printf("Warning: Failed to get destination chain client: %v, using default client", err)
			client = s.client
		}
	} else {
		client = s.client
	}

	settlement, err := event.ToSettlement(client)
	if err != nil {
		log.Printf("Warning: Failed to get block timestamp: %v", err)
		// Continue with what we have
	}

	// Add a warning log if the chain IDs don't match and we're using the default client
	if intent.DestinationChain != s.chainID && client == s.client {
		log.Printf("Warning: Using client for chain %d to fetch timestamp for settlement event on chain %d",
			s.chainID, intent.DestinationChain)
	}

	// Process the event
	return s.CreateSettlement(ctx, settlement)
}

func (s *SettlementService) validateLog(vLog types.Log) error {
	if len(vLog.Topics) < IntentSettledRequiredTopics {
		return fmt.Errorf("invalid log: expected at least %d topics, got %d", IntentSettledRequiredTopics, len(vLog.Topics))
	}
	return nil
}

func (s *SettlementService) extractEventData(vLog types.Log) (*models.IntentSettledEvent, error) {
	// Parse indexed parameters from topics
	intentID := vLog.Topics[1].Hex()

	// Convert asset address to proper format
	assetAddr := common.BytesToAddress(vLog.Topics[2].Bytes())
	asset := assetAddr.Hex()

	// Convert receiver address to proper format
	receiverAddr := common.BytesToAddress(vLog.Topics[3].Bytes())
	receiver := receiverAddr.Hex()

	// Parse non-indexed parameters from data
	// The data field contains all non-indexed parameters in order
	// We need to use the ABI to decode the data field
	unpacked, err := s.abi.Unpack(IntentSettledEventName, vLog.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %v", err)
	}

	if len(unpacked) < 5 {
		return nil, fmt.Errorf("invalid event data: expected at least 5 fields, got %d", len(unpacked))
	}

	// Extract values from unpacked data
	// The order should match the non-indexed parameters in the event definition
	amount := unpacked[0].(*big.Int)
	fulfilled := unpacked[1].(bool)
	fulfillerAddr := unpacked[2].(common.Address)
	fulfiller := fulfillerAddr.Hex()
	actualAmount := unpacked[3].(*big.Int)
	paidTip := unpacked[4].(*big.Int)

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
	}
	return event, nil
}

// Get settlement from database
func (s *SettlementService) GetSettlement(ctx context.Context, id string) (*models.Settlement, error) {
	settlement, err := s.db.GetSettlement(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get settlement: %v", err)
	}

	return settlement, nil
}

// List settlements
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

	log.Printf("Unsubscribing from all settlement subscriptions for chain %d (%d active subscriptions)",
		s.chainID, len(s.subs))

	for id, sub := range s.subs {
		sub.Unsubscribe()
		log.Printf("Unsubscribed from settlement subscription %s on chain %d", id, s.chainID)
		delete(s.subs, id)
	}
}
