package services

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/models"
	"github.com/zeta-chain/zetafast/api/utils"
)

// Constants for event processing
const (
	// EventName is the name of the intent initiated event
	EventName = "IntentInitiated"

	// RequiredTopics is the minimum number of topics required in a log
	RequiredTopics = 3

	// RequiredFields is the number of fields expected in the event data
	RequiredFields = 5
)

// IntentService handles monitoring and processing of intent events from the blockchain.
// It subscribes to intent events, processes them, and stores them in the database.
type IntentService struct {
	client *ethclient.Client
	db     db.Database
	abi    abi.ABI
}

// NewIntentService creates a new IntentService instance with the provided dependencies.
// It parses the ABI string and initializes the service with the given client and database.
//
// Parameters:
//   - client: The Ethereum client to use for blockchain interactions
//   - db: The database interface for storing intents
//   - intentInitiatedEventABI: The ABI string for the IntentInitiated event
//
// Returns:
//   - *IntentService: The initialized service
//   - error: Any error that occurred during initialization
func NewIntentService(client *ethclient.Client, db db.Database, intentInitiatedEventABI string) (*IntentService, error) {
	parsedABI, err := abi.JSON(strings.NewReader(intentInitiatedEventABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}

	return &IntentService{
		client: client,
		db:     db,
		abi:    parsedABI,
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
			{s.abi.Events[EventName].ID},
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
			{s.abi.Events[EventName].ID},
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

	chainID, err := s.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %v", err)
	}
	event.ChainID = chainID.Uint64()

	intent := event.ToIntent()
	if err := s.db.CreateIntent(ctx, intent); err != nil {
		return fmt.Errorf("failed to store intent in database: %v", err)
	}

	return nil
}

// validateLog checks if the log has the required structure and data.
func (s *IntentService) validateLog(vLog types.Log) error {
	if len(vLog.Topics) < RequiredTopics {
		return fmt.Errorf("invalid log: expected at least %d topics, got %d", RequiredTopics, len(vLog.Topics))
	}
	return nil
}

// extractEventData extracts and validates the event data from the log.
func (s *IntentService) extractEventData(vLog types.Log) (*models.IntentInitiatedEvent, error) {
	event := &models.IntentInitiatedEvent{
		IntentID:    vLog.Topics[1].Hex(),
		Asset:       common.HexToAddress(vLog.Topics[2].Hex()).Hex(),
		BlockNumber: vLog.BlockNumber,
		TxHash:      vLog.TxHash.Hex(),
	}

	unpacked, err := s.abi.Unpack(EventName, vLog.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %v", err)
	}

	if len(unpacked) < RequiredFields {
		return nil, fmt.Errorf("invalid event data: expected %d fields, got %d", RequiredFields, len(unpacked))
	}

	if err := s.validateEventFields(unpacked, event); err != nil {
		return nil, err
	}

	return event, nil
}

// validateEventFields validates each field of the event data.
func (s *IntentService) validateEventFields(unpacked []interface{}, event *models.IntentInitiatedEvent) error {
	var ok bool

	event.Amount, ok = unpacked[0].(*big.Int)
	if !ok || event.Amount == nil {
		return fmt.Errorf("invalid amount in event data")
	}

	event.TargetChain, ok = unpacked[1].(uint64)
	if !ok {
		return fmt.Errorf("invalid target chain in event data")
	}

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

// GetIntent retrieves an intent by ID
func (s *IntentService) GetIntent(ctx context.Context, id string) (*models.Intent, error) {
	return s.db.GetIntent(ctx, id)
}

// ListIntents retrieves all intents
func (s *IntentService) ListIntents(ctx context.Context) ([]*models.Intent, error) {
	return s.db.ListIntents(ctx)
}

// CreateIntent creates a new intent
func (s *IntentService) CreateIntent(ctx context.Context, id string, sourceChain uint64, destinationChain uint64, token, amount, recipient, intentFee string) (*models.Intent, error) {
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

	// Validate intent fee
	if err := utils.ValidateAmount(intentFee); err != nil {
		return nil, fmt.Errorf("invalid intent fee: %v", err)
	}

	intent := &models.Intent{
		ID:               id,
		SourceChain:      sourceChain,
		DestinationChain: destinationChain,
		Token:            token,
		Amount:           amount,
		Recipient:        recipient,
		IntentFee:        intentFee,
		Status:           models.IntentStatusPending,
	}

	if err := s.db.CreateIntent(ctx, intent); err != nil {
		return nil, err
	}

	return intent, nil
}
