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
	"github.com/speedrun-hq/speedrun/api/utils"
)

// Constants for event processing
const (
	// IntentInitiatedEventName is the name of the intent initiated event
	IntentInitiatedEventName = "IntentInitiated"

	// IntentInitiatedRequiredTopics is the minimum number of topics required in a log
	IntentInitiatedRequiredTopics = 3

	// IntentInitiatedRequiredFields is the number of fields expected in the event data
	IntentInitiatedRequiredFields = 5
)

// IntentService handles monitoring and processing of intent events from the blockchain.
// It subscribes to intent events, processes them, and stores them in the database.
type IntentService struct {
	client         *ethclient.Client
	clientResolver ClientResolver
	db             db.Database
	abi            abi.ABI
	chainID        uint64
	subs           map[string]ethereum.Subscription
}

func NewIntentService(client *ethclient.Client, clientResolver ClientResolver, db db.Database, intentInitiatedEventABI string, chainID uint64) (*IntentService, error) {
	parsedABI, err := abi.JSON(strings.NewReader(intentInitiatedEventABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}

	return &IntentService{
		client:         client,
		clientResolver: clientResolver,
		db:             db,
		abi:            parsedABI,
		chainID:        chainID,
		subs:           make(map[string]ethereum.Subscription),
	}, nil
}

// StartListening starts a goroutine to listen for intent events from the specified contract address.
// It sets up a subscription to the blockchain and processes events as they arrive.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - contractAddress: The address of the contract to listen to
//
// Returns:
//   - error: Any error that occurred during setup
func (s *IntentService) StartListening(ctx context.Context, contractAddress common.Address) error {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics: [][]common.Hash{
			{s.abi.Events[IntentInitiatedEventName].ID},
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
func (s *IntentService) processEventLogs(ctx context.Context, sub ethereum.Subscription, logs chan types.Log) {
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
func (s *IntentService) handleSubscriptionError(ctx context.Context, oldSub ethereum.Subscription, logs chan types.Log) error {
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

// processLog processes a single log entry from the blockchain.
// It validates the log, extracts event data, and stores the intent in the database.
func (s *IntentService) processLog(ctx context.Context, vLog types.Log) error {
	if err := s.validateLog(vLog); err != nil {
		return err
	}

	event, err := s.extractEventData(vLog)
	if err != nil {
		return err
	}

	// Use the target chain from the event data
	event.ChainID = s.chainID

	// Important: Use the correct chain client for intent events
	// Intent events happen on the source chain, so we need to use the source chain client
	var client *ethclient.Client
	if s.clientResolver != nil {
		// Try to get the source chain client
		sourceClient, err := s.clientResolver.GetClient(event.ChainID)
		if err == nil {
			client = sourceClient
		} else {
			log.Printf("Warning: Failed to get source chain client: %v, using default client", err)
			client = s.client
		}
	} else {
		client = s.client
	}

	intent, err := event.ToIntent(client)
	if err != nil {
		log.Printf("Warning: Failed to get block timestamp: %v", err)
		// Continue with what we have
	}

	// Add a warning log if the chain IDs don't match and we're using the default client
	if intent.SourceChain != s.chainID && client == s.client {
		log.Printf("Warning: Using client for chain %d to fetch timestamp for intent event on chain %d",
			s.chainID, intent.SourceChain)
	}

	// Check if intent already exists
	existingIntent, err := s.db.GetIntent(ctx, intent.ID)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("failed to check for existing intent: %v", err)
	}

	// Skip if intent already exists
	if existingIntent != nil {
		return nil
	}

	if err := s.db.CreateIntent(ctx, intent); err != nil {
		// Skip if intent already exists
		if strings.Contains(err.Error(), "duplicate key") {
			return nil
		}
		return fmt.Errorf("failed to store intent in database: %v", err)
	}

	return nil
}

// validateLog checks if the log has the required structure and data.
func (s *IntentService) validateLog(vLog types.Log) error {
	if len(vLog.Topics) < IntentInitiatedRequiredTopics {
		return fmt.Errorf("invalid log: expected at least %d topics, got %d", IntentInitiatedRequiredTopics, len(vLog.Topics))
	}
	return nil
}

// extractEventData extracts and validates the event data from the log.
func (s *IntentService) extractEventData(vLog types.Log) (*models.IntentInitiatedEvent, error) {
	event := &models.IntentInitiatedEvent{
		BlockNumber: vLog.BlockNumber,
		TxHash:      vLog.TxHash.Hex(),
	}

	// Parse indexed parameters from topics
	if len(vLog.Topics) < 3 {
		return nil, fmt.Errorf("invalid log: expected at least 3 topics, got %d", len(vLog.Topics))
	}

	// Topic[0] is the event signature
	// Topic[1] is the indexed intentId
	// Topic[2] is the indexed asset address
	event.IntentID = vLog.Topics[1].Hex()
	event.Asset = common.HexToAddress(vLog.Topics[2].Hex()).Hex()

	// Parse non-indexed parameters from data
	unpacked, err := s.abi.Unpack(IntentInitiatedEventName, vLog.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %v", err)
	}

	if len(unpacked) < IntentInitiatedRequiredFields {
		return nil, fmt.Errorf("invalid event data: expected %d fields, got %d", IntentInitiatedRequiredFields, len(unpacked))
	}

	if err := s.validateEventFields(unpacked, event); err != nil {
		return nil, err
	}

	// Get the sender address from the transaction
	tx, _, err := s.client.TransactionByHash(context.Background(), vLog.TxHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %v", err)
	}

	// Get the sender address from the transaction
	signer := types.LatestSignerForChainID(big.NewInt(int64(s.chainID)))
	sender, err := signer.Sender(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender address: %v", err)
	}

	event.Sender = sender.Hex()

	return event, nil
}

// validateEventFields validates each field of the event data.
func (s *IntentService) validateEventFields(unpacked []interface{}, event *models.IntentInitiatedEvent) error {
	var ok bool

	event.Amount, ok = unpacked[0].(*big.Int)
	if !ok || event.Amount == nil {
		return fmt.Errorf("invalid amount in event data")
	}

	targetChainBig, ok := unpacked[1].(*big.Int)
	if !ok || targetChainBig == nil {
		return fmt.Errorf("invalid target chain in event data")
	}
	event.TargetChain = targetChainBig.Uint64()

	event.Receiver, ok = unpacked[2].([]byte)
	if !ok || len(event.Receiver) == 0 {
		return fmt.Errorf("invalid receiver in event data")
	}

	event.Tip, ok = unpacked[3].(*big.Int)
	if !ok || event.Tip == nil {
		return fmt.Errorf("invalid tip in event data")
	}

	event.Salt, ok = unpacked[4].(*big.Int)
	if !ok || event.Salt == nil {
		return fmt.Errorf("invalid salt in event data")
	}

	return nil
}

// GetIntent retrieves an intent from the database
func (s *IntentService) GetIntent(ctx context.Context, id string) (*models.Intent, error) {
	// First, check if the intent exists in the database
	intent, err := s.db.GetIntent(ctx, id)
	if err != nil {
		// Log detailed error for debugging
		log.Printf("ERROR: Failed to get intent %s from database: %v", id, err)

		// Check if the error is "not found"
		if strings.Contains(err.Error(), "not found") {
			// Try to check on-chain via RPC if this intent exists
			log.Printf("Intent not found in database, attempting to check on-chain for intent %s", id)

			// Here you would typically query the blockchain or other sources
			// For now, we're just improving error logging
			return nil, fmt.Errorf("intent not found: %s (not in database)", id)
		}

		return nil, fmt.Errorf("error retrieving intent: %v", err)
	}

	// Log success
	log.Printf("Successfully retrieved intent %s from database", id)
	return intent, nil
}

// ListIntents retrieves all intents
func (s *IntentService) ListIntents(ctx context.Context) ([]*models.Intent, error) {
	intents, err := s.db.ListIntents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list intents: %v", err)
	}

	return intents, nil
}

// GetIntentsBySender retrieves all intents for a specific sender address
func (s *IntentService) GetIntentsBySender(ctx context.Context, sender string) ([]*models.Intent, error) {
	intents, err := s.db.ListIntentsBySender(ctx, sender)
	if err != nil {
		return nil, fmt.Errorf("failed to list intents by sender: %v", err)
	}
	return intents, nil
}

// GetIntentsByRecipient retrieves all intents for a specific recipient address
func (s *IntentService) GetIntentsByRecipient(ctx context.Context, recipient string) ([]*models.Intent, error) {
	intents, err := s.db.ListIntentsByRecipient(ctx, recipient)
	if err != nil {
		return nil, fmt.Errorf("failed to list intents by recipient: %v", err)
	}
	return intents, nil
}

// CreateIntent creates a new intent
func (s *IntentService) CreateIntent(ctx context.Context, id string, sourceChain uint64, destinationChain uint64, token, amount, recipient, sender, intentFee string, timestamp ...time.Time) (*models.Intent, error) {
	// Validate chain IDs
	if err := utils.ValidateChain(sourceChain); err != nil {
		return nil, fmt.Errorf("invalid source chain: %v", err)
	}
	if err := utils.ValidateChain(destinationChain); err != nil {
		return nil, fmt.Errorf("invalid destination chain: %v", err)
	}

	// Validate token address
	if err := utils.ValidateAddress(token); err != nil {
		return nil, fmt.Errorf("invalid token address: %v", err)
	}

	// Validate amount
	if err := utils.ValidateAmount(amount); err != nil {
		return nil, fmt.Errorf("invalid amount: %v", err)
	}

	// Validate recipient address
	if err := utils.ValidateAddress(recipient); err != nil {
		return nil, fmt.Errorf("invalid recipient address: %v", err)
	}

	// Validate sender address
	if err := utils.ValidateAddress(sender); err != nil {
		return nil, fmt.Errorf("invalid sender address: %v", err)
	}

	// Validate intent fee
	if err := utils.ValidateAmount(intentFee); err != nil {
		return nil, fmt.Errorf("invalid intent fee: %v", err)
	}

	// For API-created intents, we use the current time
	// For blockchain events, the block timestamp should be used and passed as a parameter
	var now time.Time
	if len(timestamp) > 0 && !timestamp[0].IsZero() {
		now = timestamp[0]
	} else {
		now = time.Now()
	}

	intent := &models.Intent{
		ID:               id,
		SourceChain:      sourceChain,
		DestinationChain: destinationChain,
		Token:            token,
		Amount:           amount,
		Recipient:        recipient,
		Sender:           sender,
		IntentFee:        intentFee,
		Status:           models.IntentStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.db.CreateIntent(ctx, intent); err != nil {
		return nil, err
	}

	return intent, nil
}
