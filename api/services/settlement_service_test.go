package services

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/speedrun-hq/speedrun/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDB for testing
type MockDB struct {
	mock.Mock
}

func (m *MockDB) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDB) Ping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	mockArgs := m.Called(ctx, query, args)
	return mockArgs.Get(0).(sql.Result), mockArgs.Error(1)
}

func (m *MockDB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	mockArgs := m.Called(ctx, query, args)
	return mockArgs.Get(0).(*sql.Row)
}

func (m *MockDB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	mockArgs := m.Called(ctx, query, args)
	return mockArgs.Get(0).(*sql.Rows), mockArgs.Error(1)
}

func (m *MockDB) PrepareStatements(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDB) CreateIntent(ctx context.Context, intent *models.Intent) error {
	args := m.Called(ctx, intent)
	return args.Error(0)
}

func (m *MockDB) GetIntent(ctx context.Context, id string) (*models.Intent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Intent), args.Error(1)
}

func (m *MockDB) ListIntents(ctx context.Context) ([]*models.Intent, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func (m *MockDB) ListIntentsPaginated(ctx context.Context, page, pageSize int, status string) ([]*models.Intent, int, error) {
	args := m.Called(ctx, page, pageSize, status)
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}

func (m *MockDB) ListIntentsPaginatedOptimized(ctx context.Context, page, pageSize int, status string) ([]*models.Intent, int, error) {
	args := m.Called(ctx, page, pageSize, status)
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}

func (m *MockDB) ListIntentsBySender(ctx context.Context, sender string) ([]*models.Intent, error) {
	args := m.Called(ctx, sender)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func (m *MockDB) ListIntentsBySenderPaginated(ctx context.Context, sender string, page, pageSize int) ([]*models.Intent, int, error) {
	args := m.Called(ctx, sender, page, pageSize)
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}

func (m *MockDB) ListIntentsBySenderPaginatedOptimized(ctx context.Context, sender string, page, pageSize int) ([]*models.Intent, int, error) {
	args := m.Called(ctx, sender, page, pageSize)
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}

func (m *MockDB) ListIntentsByRecipient(ctx context.Context, recipient string) ([]*models.Intent, error) {
	args := m.Called(ctx, recipient)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func (m *MockDB) ListIntentsByRecipientPaginated(ctx context.Context, recipient string, page, pageSize int) ([]*models.Intent, int, error) {
	args := m.Called(ctx, recipient, page, pageSize)
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}

func (m *MockDB) ListIntentsByRecipientPaginatedOptimized(ctx context.Context, recipient string, page, pageSize int) ([]*models.Intent, int, error) {
	args := m.Called(ctx, recipient, page, pageSize)
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}

func (m *MockDB) ListIntentsKeysetPaginated(ctx context.Context, lastTimestamp time.Time, lastID string, pageSize int, status string) ([]*models.Intent, bool, error) {
	args := m.Called(ctx, lastTimestamp, lastID, pageSize, status)
	return args.Get(0).([]*models.Intent), args.Bool(1), args.Error(2)
}

func (m *MockDB) UpdateIntentStatus(ctx context.Context, id string, status models.IntentStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockDB) CreateFulfillment(ctx context.Context, fulfillment *models.Fulfillment) error {
	args := m.Called(ctx, fulfillment)
	return args.Error(0)
}

func (m *MockDB) GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Fulfillment), args.Error(1)
}

func (m *MockDB) ListFulfillments(ctx context.Context) ([]*models.Fulfillment, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Fulfillment), args.Error(1)
}

func (m *MockDB) ListFulfillmentsPaginated(ctx context.Context, page, pageSize int) ([]*models.Fulfillment, int, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]*models.Fulfillment), args.Int(1), args.Error(2)
}

func (m *MockDB) ListFulfillmentsPaginatedOptimized(ctx context.Context, page, pageSize int) ([]*models.Fulfillment, int, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]*models.Fulfillment), args.Int(1), args.Error(2)
}

func (m *MockDB) GetTotalFulfilledAmount(ctx context.Context, intentID string) (string, error) {
	args := m.Called(ctx, intentID)
	return args.String(0), args.Error(1)
}

func (m *MockDB) CreateSettlement(ctx context.Context, settlement *models.Settlement) error {
	args := m.Called(ctx, settlement)
	return args.Error(0)
}

func (m *MockDB) GetSettlement(ctx context.Context, id string) (*models.Settlement, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Settlement), args.Error(1)
}

func (m *MockDB) ListSettlements(ctx context.Context) ([]*models.Settlement, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Settlement), args.Error(1)
}

func (m *MockDB) ListSettlementsPaginated(ctx context.Context, page, pageSize int) ([]*models.Settlement, int, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]*models.Settlement), args.Int(1), args.Error(2)
}

func (m *MockDB) ListSettlementsPaginatedOptimized(ctx context.Context, page, pageSize int) ([]*models.Settlement, int, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]*models.Settlement), args.Int(1), args.Error(2)
}

func (m *MockDB) GetLastProcessedBlock(ctx context.Context, chainID uint64) (uint64, error) {
	args := m.Called(ctx, chainID)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockDB) UpdateLastProcessedBlock(ctx context.Context, chainID uint64, blockNumber uint64) error {
	args := m.Called(ctx, chainID, blockNumber)
	return args.Error(0)
}

func (m *MockDB) InitDB(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// SQLResultMock is a mock implementation of sql.Result
type SQLResultMock struct {
	mock.Mock
}

func (m *SQLResultMock) LastInsertId() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *SQLResultMock) RowsAffected() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func TestSettlementService_ExtractEventData(t *testing.T) {
	// Setup test ABI
	intentSettledEventABI := `[
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "internalType": "bytes32", "name": "intentId", "type": "bytes32"},
				{"indexed": true, "internalType": "address", "name": "asset", "type": "address"},
				{"indexed": false, "internalType": "uint256", "name": "amount", "type": "uint256"},
				{"indexed": true, "internalType": "address", "name": "receiver", "type": "address"},
				{"indexed": false, "internalType": "bool", "name": "fulfilled", "type": "bool"},
				{"indexed": false, "internalType": "address", "name": "fulfiller", "type": "address"},
				{"indexed": false, "internalType": "uint256", "name": "actualAmount", "type": "uint256"},
				{"indexed": false, "internalType": "uint256", "name": "paidTip", "type": "uint256"}
			],
			"name": "IntentSettled",
			"type": "event"
		}
	]`

	parsedABI, err := abi.JSON(strings.NewReader(intentSettledEventABI))
	assert.NoError(t, err)

	settlementService := &SettlementService{
		abi:     parsedABI,
		chainID: 1,
	}

	// Create a log entry with data
	intentID := "0x1234567890123456789012345678901234567890123456789012345678901234"
	asset := "0x1234567890123456789012345678901234567890"
	receiver := "0x9876543210987654321098765432109876543210"

	topics := []common.Hash{
		parsedABI.Events[IntentSettledEventName].ID,               // event signature
		common.HexToHash(intentID),                                // intentId (indexed)
		common.BytesToHash(common.HexToAddress(asset).Bytes()),    // asset (indexed)
		common.BytesToHash(common.HexToAddress(receiver).Bytes()), // receiver (indexed)
	}

	// Create the test data for non-indexed fields
	amount := big.NewInt(1000000000000000000) // 1 ETH
	fulfilled := true
	fulfiller := common.HexToAddress("0x5678901234567890123456789012345678901234")
	actualAmount := big.NewInt(900000000000000000) // 0.9 ETH
	paidTip := big.NewInt(100000000000000000)      // 0.1 ETH

	// Pack the data
	data, err := parsedABI.Events[IntentSettledEventName].Inputs.NonIndexed().Pack(
		amount,
		fulfilled,
		fulfiller,
		actualAmount,
		paidTip,
	)
	assert.NoError(t, err)

	// Create the log
	log := types.Log{
		Address:     common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Topics:      topics,
		Data:        data,
		BlockNumber: 12345,
		TxHash:      common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
		TxIndex:     0,
		BlockHash:   common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Index:       0,
		Removed:     false,
	}

	// Extract the event data
	event, err := settlementService.extractEventData(log)
	assert.NoError(t, err)
	assert.NotNil(t, event)

	// Verify the extracted data
	assert.Equal(t, intentID, event.IntentID)
	assert.Equal(t, asset, event.Asset)
	assert.Equal(t, amount, event.Amount)
	assert.Equal(t, receiver, event.Receiver)
	assert.Equal(t, fulfilled, event.Fulfilled)
	assert.Equal(t, fulfiller.Hex(), event.Fulfiller)
	assert.Equal(t, actualAmount, event.ActualAmount)
	assert.Equal(t, paidTip, event.PaidTip)
	assert.Equal(t, uint64(12345), event.BlockNumber)
	assert.Equal(t, "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", event.TxHash)
}

func TestSettlementService_ProcessLog(t *testing.T) {
	// Setup mock database
	mockDB := new(MockDB)
	mockSQL := new(SQLResultMock)

	// Mock SQL result for testing
	_ = sqlmock.NewResult(1, 1)

	// Mock client setup
	client, err := ethclient.Dial("https://eth-sepolia.g.alchemy.com/v2/demo") // Using public endpoint
	if err != nil {
		t.Skip("Skipping test due to failure to connect to Ethereum node")
		return
	}

	mockResolver := new(mocks.MockClientResolver)

	// Setup test ABI
	intentSettledEventABI := `[
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "internalType": "bytes32", "name": "intentId", "type": "bytes32"},
				{"indexed": true, "internalType": "address", "name": "asset", "type": "address"},
				{"indexed": false, "internalType": "uint256", "name": "amount", "type": "uint256"},
				{"indexed": true, "internalType": "address", "name": "receiver", "type": "address"},
				{"indexed": false, "internalType": "bool", "name": "fulfilled", "type": "bool"},
				{"indexed": false, "internalType": "address", "name": "fulfiller", "type": "address"},
				{"indexed": false, "internalType": "uint256", "name": "actualAmount", "type": "uint256"},
				{"indexed": false, "internalType": "uint256", "name": "paidTip", "type": "uint256"}
			],
			"name": "IntentSettled",
			"type": "event"
		}
	]`

	parsedABI, err := abi.JSON(strings.NewReader(intentSettledEventABI))
	assert.NoError(t, err)

	settlementService := &SettlementService{
		client:         client,
		clientResolver: mockResolver,
		db:             mockDB,
		abi:            parsedABI,
		chainID:        1,
	}

	// Create a log entry with data
	intentID := "0x1234567890123456789012345678901234567890123456789012345678901234"
	asset := "0x1234567890123456789012345678901234567890"
	receiver := "0x9876543210987654321098765432109876543210"

	topics := []common.Hash{
		parsedABI.Events[IntentSettledEventName].ID,               // event signature
		common.HexToHash(intentID),                                // intentId (indexed)
		common.BytesToHash(common.HexToAddress(asset).Bytes()),    // asset (indexed)
		common.BytesToHash(common.HexToAddress(receiver).Bytes()), // receiver (indexed)
	}

	// Create the test data for non-indexed fields
	amount := big.NewInt(1000000000000000000) // 1 ETH
	fulfilled := true
	fulfiller := common.HexToAddress("0x5678901234567890123456789012345678901234")
	actualAmount := big.NewInt(900000000000000000) // 0.9 ETH
	paidTip := big.NewInt(100000000000000000)      // 0.1 ETH

	// Pack the data
	data, err := parsedABI.Events[IntentSettledEventName].Inputs.NonIndexed().Pack(
		amount,
		fulfilled,
		fulfiller,
		actualAmount,
		paidTip,
	)
	assert.NoError(t, err)

	// Create the log
	log := types.Log{
		Address:     common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Topics:      topics,
		Data:        data,
		BlockNumber: 12345,
		TxHash:      common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
		TxIndex:     0,
		BlockHash:   common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Index:       0,
		Removed:     false,
	}

	t.Run("Process settlement log", func(t *testing.T) {
		// Mock an existing intent
		existingIntent := &models.Intent{
			ID:               intentID,
			SourceChain:      1,
			DestinationChain: 2,
			Token:            asset,
			Amount:           "1000000000000000000",
			Recipient:        receiver,
			Status:           models.IntentStatusFulfilled,
		}

		// Mock client resolver to return the dest chain client
		mockResolver.On("GetClient", uint64(2)).
			Return(client, nil).
			Once()

		// The database should find the existing intent
		mockDB.On("GetIntent", mock.Anything, intentID).
			Return(existingIntent, nil).
			Once()

		// Mock SQL result for CreateSettlement
		mockSQL.On("LastInsertId").Return(int64(1), nil)
		mockSQL.On("RowsAffected").Return(int64(1), nil)

		// The settlement should be created
		mockDB.On("CreateSettlement", mock.Anything, mock.MatchedBy(func(s *models.Settlement) bool {
			return s.ID == intentID &&
				s.Asset == asset &&
				s.Amount == amount.String() &&
				s.Receiver == receiver &&
				s.Fulfilled == fulfilled &&
				s.Fulfiller == fulfiller.Hex()
		})).Return(nil).Once()

		// The intent status should be updated
		mockDB.On("UpdateIntentStatus", mock.Anything, intentID, models.IntentStatusSettled).
			Return(nil).
			Once()

		// Process the log
		err = settlementService.processLog(context.Background(), log)
		assert.NoError(t, err)

		// Verify mocks were called
		mockDB.AssertExpectations(t)
		mockResolver.AssertExpectations(t)
	})

	t.Run("Intent not found", func(t *testing.T) {
		// Reset mocks
		mockDB = new(MockDB)
		mockResolver = new(mocks.MockClientResolver)
		settlementService.db = mockDB
		settlementService.clientResolver = mockResolver

		// The database should fail to find the intent
		mockDB.On("GetIntent", mock.Anything, intentID).
			Return(nil, fmt.Errorf("intent not found")).
			Once()

		// Process the log
		err = settlementService.processLog(context.Background(), log)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get intent")

		// Verify mocks
		mockDB.AssertExpectations(t)
	})
}
