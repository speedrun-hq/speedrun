package services

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/zeta-chain/zetafast/api/models"
	"github.com/zeta-chain/zetafast/api/test/mocks"
)

func TestNewFulfillmentService(t *testing.T) {
	// Create a mock database and eth client
	mockDB := mocks.NewMockDB()
	ethClient := createMockEthClient()

	// Create chain clients map
	clients := map[uint64]*ethclient.Client{
		42161: ethClient, // Arbitrum
		7001:  ethClient, // ZetaChain
	}

	// Contract addresses
	contractAddresses := map[uint64]string{
		42161: "0x1234567890123456789012345678901234567890",
		7001:  "0x0987654321098765432109876543210987654321",
	}

	// Create a fulfillment service with a valid ABI
	abi := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"address","name":"fulfiller","type":"address"}],"name":"IntentFulfilled","type":"event"}]`
	service, err := NewFulfillmentService(clients, contractAddresses, mockDB, abi)
	assert.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, mockDB, service.db)

	// Test with invalid ABI
	service, err = NewFulfillmentService(clients, contractAddresses, mockDB, "invalid abi")
	assert.Error(t, err)
	assert.Nil(t, service)
}

func TestFulfillmentServiceStartListening(t *testing.T) {
	// Skip this test for now as it requires a real Ethereum client
	t.Skip("Skipping test that requires a real Ethereum client")

	// Create a mock database and eth client
	mockDB := mocks.NewMockDB()
	ethClient := createMockEthClient()

	// Create chain clients map
	clients := map[uint64]*ethclient.Client{
		42161: ethClient, // Arbitrum
	}

	// Contract addresses
	contractAddresses := map[uint64]string{
		42161: "0x1234567890123456789012345678901234567890",
	}

	// Create a fulfillment service
	service, err := NewFulfillmentService(clients, contractAddresses, mockDB, `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"address","name":"fulfiller","type":"address"}],"name":"IntentFulfilled","type":"event"}]`)
	assert.NoError(t, err)

	// Test starting to listen for events
	ctx := context.Background()
	err = service.StartListening(ctx)
	assert.NoError(t, err)
}

func TestProcessFulfillmentEvent(t *testing.T) {
	// Create a mock database
	mockDB := mocks.NewMockDB()

	// Create a test intent first
	amount := new(big.Int)
	amount.SetString("1000000000000000000", 10)
	tip := new(big.Int)
	tip.SetString("100000000000000000", 10)
	salt := new(big.Int)
	salt.SetString("0", 10)

	intent := &models.Intent{
		ID:               "test-intent-id",
		SourceChain:      7001,
		DestinationChain: 42161,
		Token:            "0x1234567890123456789012345678901234567890",
		Amount:           amount.String(),
		Recipient:        "0x0987654321098765432109876543210987654321",
		IntentFee:        tip.String(),
		Status:           models.IntentStatusPending,
	}

	// Save the intent to the database
	ctx := context.Background()
	err := mockDB.CreateIntent(ctx, intent)
	assert.NoError(t, err)

	// Create a test fulfillment event
	fulfillmentAmount := new(big.Int)
	fulfillmentAmount.SetString("1000000000000000000", 10) // Full amount

	event := &models.FulfillmentEvent{
		IntentID:    intent.ID,
		TargetChain: 42161,
		Receiver:    common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12").Hex(),
		Amount:      fulfillmentAmount.String(),
		TxHash:      common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890").Hex(),
		BlockNumber: 12345678,
	}

	// Test processing the event
	fulfillment := event.ToFulfillment()
	err = mockDB.CreateFulfillment(ctx, fulfillment)
	assert.NoError(t, err)

	// Verify fulfillment was created
	retrievedFulfillment, err := mockDB.GetFulfillment(ctx, fulfillment.ID)
	assert.NoError(t, err)
	assert.Equal(t, fulfillment.ID, retrievedFulfillment.ID)
	assert.Equal(t, fulfillment.IntentID, retrievedFulfillment.IntentID)
	assert.Equal(t, fulfillment.Amount, retrievedFulfillment.Amount)
	assert.Equal(t, fulfillment.Fulfiller, retrievedFulfillment.Fulfiller)
	assert.Equal(t, event.TargetChain, retrievedFulfillment.TargetChain)
	assert.Equal(t, fulfillment.TxHash, retrievedFulfillment.TxHash)

	// Verify intent status was updated
	updatedIntent, err := mockDB.GetIntent(ctx, intent.ID)
	assert.NoError(t, err)
	assert.Equal(t, models.IntentStatusFulfilled, updatedIntent.Status)
}

func TestCreateFulfillment(t *testing.T) {
	// Create mock database
	mockDB := mocks.NewMockDB()

	// Create service
	clients := map[uint64]*ethclient.Client{
		42161: {},
	}
	contractAddresses := map[uint64]string{
		42161: "0x1234567890123456789012345678901234567890",
	}
	contractABI := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":true,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"IntentFulfilled","type":"event"}]`

	service, err := NewFulfillmentService(clients, contractAddresses, mockDB, contractABI)
	assert.NoError(t, err)

	// Create a test intent
	now := time.Now()
	intent := &models.Intent{
		ID:               "0x1234567890123456789012345678901234567890123456789012345678901234",
		SourceChain:      7001,
		DestinationChain: 42161,
		Token:            "0x1234567890123456789012345678901234567890",
		Amount:           "1000000000000000000",
		Recipient:        "0x0987654321098765432109876543210987654321",
		IntentFee:        "100000000000000000",
		Status:           models.IntentStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Store intent in database
	err = mockDB.CreateIntent(context.Background(), intent)
	assert.NoError(t, err)

	// Create fulfillment
	fulfillment, err := service.CreateFulfillment(
		context.Background(),
		intent.ID,
		"0x0987654321098765432109876543210987654321",
		"1000000000000000000",
	)
	assert.NoError(t, err)
	assert.NotNil(t, fulfillment)
	assert.Equal(t, intent.ID, fulfillment.IntentID)
	assert.Equal(t, "0x0987654321098765432109876543210987654321", fulfillment.Fulfiller)
	assert.Equal(t, "1000000000000000000", fulfillment.Amount)
	assert.Equal(t, models.FulfillmentStatusPending, fulfillment.Status)
}

func TestGetFulfillment(t *testing.T) {
	// Create mock database
	mockDB := mocks.NewMockDB()

	// Create service
	clients := map[uint64]*ethclient.Client{
		42161: {},
	}
	contractAddresses := map[uint64]string{
		42161: "0x1234567890123456789012345678901234567890",
	}
	contractABI := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":true,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"IntentFulfilled","type":"event"}]`

	service, err := NewFulfillmentService(clients, contractAddresses, mockDB, contractABI)
	assert.NoError(t, err)

	// Create a test intent first
	now := time.Now()
	intent := &models.Intent{
		ID:               "0x1234567890123456789012345678901234567890123456789012345678901234",
		SourceChain:      7001,
		DestinationChain: 42161,
		Token:            "0x1234567890123456789012345678901234567890",
		Amount:           "1000000000000000000",
		Recipient:        "0x0987654321098765432109876543210987654321",
		IntentFee:        "100000000000000000",
		Status:           models.IntentStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Store intent in database
	err = mockDB.CreateIntent(context.Background(), intent)
	assert.NoError(t, err)

	// Create a test fulfillment
	fulfillment := &models.Fulfillment{
		ID:          "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		IntentID:    intent.ID,
		Fulfiller:   "0x0987654321098765432109876543210987654321",
		TargetChain: 42161,
		Amount:      "1000000000000000000",
		Status:      models.FulfillmentStatusCompleted,
		TxHash:      "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		BlockNumber: 12345678,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Store fulfillment in database
	err = mockDB.CreateFulfillment(context.Background(), fulfillment)
	assert.NoError(t, err)

	// Get fulfillment
	retrieved, err := service.GetFulfillment(context.Background(), fulfillment.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, fulfillment.ID, retrieved.ID)
	assert.Equal(t, fulfillment.IntentID, retrieved.IntentID)
	assert.Equal(t, fulfillment.Fulfiller, retrieved.Fulfiller)
	assert.Equal(t, fulfillment.Amount, retrieved.Amount)
	assert.Equal(t, fulfillment.Status, retrieved.Status)
}

func TestListFulfillments(t *testing.T) {
	// Create mock database
	mockDB := mocks.NewMockDB()

	// Create service
	clients := map[uint64]*ethclient.Client{
		42161: {},
	}
	contractAddresses := map[uint64]string{
		42161: "0x1234567890123456789012345678901234567890",
	}
	contractABI := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":true,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"IntentFulfilled","type":"event"}]`

	service, err := NewFulfillmentService(clients, contractAddresses, mockDB, contractABI)
	assert.NoError(t, err)

	// Create a test intent
	now := time.Now()
	intent := &models.Intent{
		ID:               "0x1234567890123456789012345678901234567890123456789012345678901234",
		SourceChain:      7001,
		DestinationChain: 42161,
		Token:            "0x1234567890123456789012345678901234567890",
		Amount:           "1000000000000000000",
		Recipient:        "0x0987654321098765432109876543210987654321",
		IntentFee:        "100000000000000000",
		Status:           models.IntentStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Store intent in database
	err = mockDB.CreateIntent(context.Background(), intent)
	assert.NoError(t, err)

	// Create multiple fulfillments
	for i := 0; i < 3; i++ {
		fulfillment := &models.Fulfillment{
			ID:          fmt.Sprintf("test-fulfillment-id-%d", i),
			IntentID:    intent.ID,
			Fulfiller:   "0x0987654321098765432109876543210987654321",
			TargetChain: 42161,
			Amount:      "1000000000000000000",
			Status:      models.FulfillmentStatusPending,
			TxHash:      fmt.Sprintf("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567%03d", i),
			BlockNumber: uint64(12345678 + i),
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		err = mockDB.CreateFulfillment(context.Background(), fulfillment)
		assert.NoError(t, err)
	}

	// Test listing fulfillments
	fulfillments, err := service.ListFulfillments(context.Background())
	assert.NoError(t, err)
	assert.Len(t, fulfillments, 3)
}

func TestListFulfillmentsAndFlow(t *testing.T) {
	// Create mock database
	mockDB := mocks.NewMockDB()

	// Create service
	clients := map[uint64]*ethclient.Client{
		42161: {},
	}
	contractAddresses := map[uint64]string{
		42161: "0x1234567890123456789012345678901234567890",
	}
	contractABI := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":true,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"IntentFulfilled","type":"event"}]`

	service, err := NewFulfillmentService(clients, contractAddresses, mockDB, contractABI)
	assert.NoError(t, err)

	// Create a test intent
	now := time.Now()
	intent := &models.Intent{
		ID:               "0x1234567890123456789012345678901234567890123456789012345678901234",
		SourceChain:      7001,
		DestinationChain: 42161,
		Token:            "0x1234567890123456789012345678901234567890",
		Amount:           "3000000000000000000",
		Recipient:        "0x0987654321098765432109876543210987654321",
		IntentFee:        "100000000000000000",
		Status:           models.IntentStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Store intent in database
	err = mockDB.CreateIntent(context.Background(), intent)
	assert.NoError(t, err)

	// Create multiple fulfillments
	for i := 0; i < 3; i++ {
		fulfillment := &models.Fulfillment{
			ID:          fmt.Sprintf("test-fulfillment-id-%d", i),
			IntentID:    intent.ID,
			Fulfiller:   "0x0987654321098765432109876543210987654321",
			TargetChain: 42161,
			Amount:      "500000000000000000",
			Status:      models.FulfillmentStatusPending,
			TxHash:      fmt.Sprintf("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567%03d", i),
			BlockNumber: uint64(12345678 + i),
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		err = mockDB.CreateFulfillment(context.Background(), fulfillment)
		assert.NoError(t, err)
	}

	// Test listing fulfillments
	fulfillments, err := service.ListFulfillments(context.Background())
	assert.NoError(t, err)
	assert.Len(t, fulfillments, 3)

	// Create fulfillment
	fulfillment, err := service.CreateFulfillment(
		context.Background(),
		intent.ID,
		"0x0987654321098765432109876543210987654321",
		"500000000000000000",
	)
	assert.NoError(t, err)
	assert.NotNil(t, fulfillment)

	// Verify fulfillment was created
	retrievedFulfillment, err := service.GetFulfillment(context.Background(), fulfillment.ID)
	assert.NoError(t, err)
	assert.Equal(t, fulfillment.ID, retrievedFulfillment.ID)
	assert.Equal(t, fulfillment.IntentID, retrievedFulfillment.IntentID)
	assert.Equal(t, fulfillment.Amount, retrievedFulfillment.Amount)
	assert.Equal(t, fulfillment.Fulfiller, retrievedFulfillment.Fulfiller)
	assert.Equal(t, fulfillment.Status, retrievedFulfillment.Status)

	// List fulfillments again
	fulfillments, err = service.ListFulfillments(context.Background())
	assert.NoError(t, err)
	assert.Len(t, fulfillments, 4)
}
