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
	"github.com/zeta-chain/zetafast/api/config"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/models"
	"github.com/zeta-chain/zetafast/api/utils"
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
	client  *ethclient.Client
	db      db.Database
	abi     abi.ABI
	chainID uint64
	subs    map[string]ethereum.Subscription
}

func NewIntentService(client *ethclient.Client, db db.Database, intentInitiatedEventABI string, chainID uint64) (*IntentService, error) {
	parsedABI, err := abi.JSON(strings.NewReader(intentInitiatedEventABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}

	return &IntentService{
		client:  client,
		db:      db,
		abi:     parsedABI,
		chainID: chainID,
		subs:    make(map[string]ethereum.Subscription),
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
	// First, catch up on any missed events
	if err := s.catchUpOnMissedEvents(ctx, contractAddress); err != nil {
		return fmt.Errorf("failed to catch up on missed events: %v", err)
	}

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

// catchUpOnMissedEvents fetches and processes any events that were missed during downtime
func (s *IntentService) catchUpOnMissedEvents(ctx context.Context, contractAddress common.Address) error {
	log.Printf("Catching up on missed events for chain %d, contract %s", s.chainID, contractAddress.Hex())

	// Get the last processed block number from the database
	lastBlock, err := s.db.GetLastProcessedBlock(ctx, s.chainID)
	if err != nil {
		log.Printf("Error getting last processed block: %v", err)
		return fmt.Errorf("failed to get last processed block: %v", err)
	}

	// If no last processed block is found, use the default block from config
	if lastBlock == 0 {
		// Get the default block from config
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Printf("Error loading config: %v", err)
			return fmt.Errorf("failed to load config: %v", err)
		}

		chainConfig, ok := cfg.ChainConfigs[s.chainID]
		if !ok {
			log.Printf("No config found for chain %d", s.chainID)
			return fmt.Errorf("no config found for chain %d", s.chainID)
		}

		lastBlock = chainConfig.DefaultBlock
		log.Printf("Using default block %d for chain %d", lastBlock, s.chainID)
	}

	log.Printf("Last processed block: %d", lastBlock)

	// Get the current block number
	currentBlock, err := s.client.BlockNumber(ctx)
	if err != nil {
		log.Printf("Error getting current block number: %v", err)
		return fmt.Errorf("failed to get current block number: %v", err)
	}
	log.Printf("Current block: %d", currentBlock)

	// If we're up to date, no need to catch up
	if lastBlock >= currentBlock {
		log.Printf("No missed events to process")
		return nil
	}

	// Fetch logs for the missed blocks
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(lastBlock + 1)),
		ToBlock:   big.NewInt(int64(currentBlock)),
		Addresses: []common.Address{contractAddress},
		Topics: [][]common.Hash{
			{s.abi.Events[IntentInitiatedEventName].ID},
		},
	}
	log.Printf("Fetching logs with query: FromBlock=%d, ToBlock=%d, Address=%s, EventID=%s",
		query.FromBlock, query.ToBlock, query.Addresses[0].Hex(), query.Topics[0][0].Hex())

	logs, err := s.client.FilterLogs(ctx, query)
	if err != nil {
		log.Printf("Error fetching intent initiated logs: %v", err)
		return fmt.Errorf("failed to fetch intent initiated logs: %v", err)
	}
	log.Printf("Found %d intent initiated logs to process", len(logs))

	// Process each missed log
	for i, txlog := range logs {
		log.Printf("Processing missed log %d/%d: Block=%d, TxHash=%s", i+1, len(logs), txlog.BlockNumber, txlog.TxHash.Hex())
		if err := s.processLog(ctx, txlog); err != nil {
			log.Printf("Error processing intent initiated log: %v", err)
			return fmt.Errorf("failed to process intent initiated log: %v", err)
		}
		log.Printf("Successfully processed intent initiated log %d/%d", i+1, len(logs))
	}

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

	log.Printf("Extracted event - IntentID: %s, Asset: %s, Amount: %s, TargetChain: %d, Receiver: %x, Tip: %s, Salt: %s",
		event.IntentID,
		event.Asset,
		event.Amount.String(),
		event.TargetChain,
		event.Receiver,
		event.Tip.String(),
		event.Salt.String())

	// Use the target chain from the event data
	event.ChainID = s.chainID

	intent := event.ToIntent()
	log.Printf("Created intent - ID: %s, SourceChain: %d, DestinationChain: %d, Status: %s, CreatedAt: %v, UpdatedAt: %v",
		intent.ID,
		intent.SourceChain,
		intent.DestinationChain,
		intent.Status,
		intent.CreatedAt,
		intent.UpdatedAt)

	// Check if intent already exists
	existingIntent, err := s.db.GetIntent(ctx, intent.ID)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("failed to check for existing intent: %v", err)
	}

	// Skip if intent already exists
	if existingIntent != nil {
		log.Printf("Skipping existing intent: %s", intent.ID)
		return nil
	}

	if err := s.db.CreateIntent(ctx, intent); err != nil {
		// Skip if intent already exists
		if strings.Contains(err.Error(), "duplicate key") {
			log.Printf("Skipping duplicate intent: %s", intent.ID)
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

	log.Printf("Extracting event data from block %d, tx %s", vLog.BlockNumber, vLog.TxHash.Hex())
	log.Printf("Topics count: %d", len(vLog.Topics))
	for i, topic := range vLog.Topics {
		log.Printf("Topic[%d]: %x", i, topic)
	}
	log.Printf("Raw event data: %x", vLog.Data)

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
	log.Printf("Unpacking event data...")
	unpacked, err := s.abi.Unpack(IntentInitiatedEventName, vLog.Data)
	if err != nil {
		log.Printf("Error unpacking event data: %v", err)
		log.Printf("Event signature: %s", s.abi.Events[IntentInitiatedEventName].ID.Hex())
		log.Printf("Event inputs: %+v", s.abi.Events[IntentInitiatedEventName].Inputs)
		return nil, fmt.Errorf("failed to unpack event data: %v", err)
	}

	log.Printf("Unpacked data length: %d", len(unpacked))
	for i, data := range unpacked {
		log.Printf("Unpacked[%d]: %+v (type: %T)", i, data, data)
	}

	if len(unpacked) < IntentInitiatedRequiredFields {
		return nil, fmt.Errorf("invalid event data: expected %d fields, got %d", IntentInitiatedRequiredFields, len(unpacked))
	}

	if err := s.validateEventFields(unpacked, event); err != nil {
		return nil, err
	}

	log.Printf("Successfully extracted event data - IntentID: %s, Asset: %s, Amount: %s, TargetChain: %d, Receiver: %x, Tip: %s, Salt: %s",
		event.IntentID,
		event.Asset,
		event.Amount.String(),
		event.TargetChain,
		event.Receiver,
		event.Tip.String(),
		event.Salt.String())

	return event, nil
}

// validateEventFields validates each field of the event data.
func (s *IntentService) validateEventFields(unpacked []interface{}, event *models.IntentInitiatedEvent) error {
	var ok bool

	log.Printf("Validating amount field...")
	event.Amount, ok = unpacked[0].(*big.Int)
	if !ok || event.Amount == nil {
		log.Printf("Invalid amount: %+v (type: %T)", unpacked[0], unpacked[0])
		return fmt.Errorf("invalid amount in event data")
	}
	log.Printf("Amount validated: %s", event.Amount.String())

	log.Printf("Validating target chain field...")
	targetChainBig, ok := unpacked[1].(*big.Int)
	if !ok || targetChainBig == nil {
		log.Printf("Invalid target chain: %+v (type: %T)", unpacked[1], unpacked[1])
		return fmt.Errorf("invalid target chain in event data")
	}
	event.TargetChain = targetChainBig.Uint64()
	log.Printf("Target chain validated: %d", event.TargetChain)

	log.Printf("Validating receiver field...")
	event.Receiver, ok = unpacked[2].([]byte)
	if !ok || len(event.Receiver) == 0 {
		log.Printf("Invalid receiver: %+v (type: %T)", unpacked[2], unpacked[2])
		return fmt.Errorf("invalid receiver in event data")
	}
	log.Printf("Receiver validated: %x", event.Receiver)

	log.Printf("Validating tip field...")
	event.Tip, ok = unpacked[3].(*big.Int)
	if !ok || event.Tip == nil {
		log.Printf("Invalid tip: %+v (type: %T)", unpacked[3], unpacked[3])
		return fmt.Errorf("invalid tip in event data")
	}
	log.Printf("Tip validated: %s", event.Tip.String())

	log.Printf("Validating salt field...")
	event.Salt, ok = unpacked[4].(*big.Int)
	if !ok || event.Salt == nil {
		log.Printf("Invalid salt: %+v (type: %T)", unpacked[4], unpacked[4])
		return fmt.Errorf("invalid salt in event data")
	}
	log.Printf("Salt validated: %s", event.Salt.String())

	return nil
}

// GetIntent retrieves an intent by ID
func (s *IntentService) GetIntent(ctx context.Context, id string) (*models.Intent, error) {
	// Get intent from database
	intent, err := s.db.GetIntent(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get intent: %v", err)
	}

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
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.db.CreateIntent(ctx, intent); err != nil {
		return nil, err
	}

	return intent, nil
}
