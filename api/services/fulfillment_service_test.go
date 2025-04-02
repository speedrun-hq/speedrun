package services

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/zeta-chain/zetafast/api/config"
	"github.com/zeta-chain/zetafast/api/db"
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

	// Create test config
	testConfig := &config.Config{
		ChainConfigs: map[uint64]*config.ChainConfig{
			42161: {
				DefaultBlock: 322207320, // Arbitrum
			},
			7001: {
				DefaultBlock: 1000000, // ZetaChain
			},
		},
	}

	// Create default blocks map from test config
	defaultBlocks := make(map[uint64]uint64)
	for chainID, chainConfig := range testConfig.ChainConfigs {
		defaultBlocks[chainID] = chainConfig.DefaultBlock
	}

	// Create a fulfillment service
	abi := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":true,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"IntentFulfilled","type":"event"}]`
	service, err := NewFulfillmentService(clients, contractAddresses, db.DBInterface(mockDB), abi, defaultBlocks)
	assert.NoError(t, err)
	assert.NotNil(t, service)

	// Test with invalid contract address
	invalidContractAddresses := map[uint64]string{
		42161: "invalid-address",
	}
	_, err = NewFulfillmentService(clients, invalidContractAddresses, db.DBInterface(mockDB), abi, defaultBlocks)
	assert.Error(t, err)

	// Test with invalid ABI
	invalidABI := "invalid-abi"
	_, err = NewFulfillmentService(clients, contractAddresses, db.DBInterface(mockDB), invalidABI, defaultBlocks)
	assert.Error(t, err)
}

func TestProcessFulfillmentEvent(t *testing.T) {
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

	// Create test config
	testConfig := &config.Config{
		ChainConfigs: map[uint64]*config.ChainConfig{
			42161: {
				DefaultBlock: 322207320, // Arbitrum
			},
			7001: {
				DefaultBlock: 1000000, // ZetaChain
			},
		},
	}

	// Create default blocks map from test config
	defaultBlocks := make(map[uint64]uint64)
	for chainID, chainConfig := range testConfig.ChainConfigs {
		defaultBlocks[chainID] = chainConfig.DefaultBlock
	}

	// Create a fulfillment service
	abi := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":true,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"IntentFulfilled","type":"event"}]`
	service, err := NewFulfillmentService(clients, contractAddresses, db.DBInterface(mockDB), abi, defaultBlocks)
	assert.NoError(t, err)
	assert.NotNil(t, service)

	// Test processing a fulfillment event
	ctx := context.Background()
	event := &IntentFulfilledEvent{
		IntentID: common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234"),
		Asset:    common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Amount:   big.NewInt(1000000000000000000), // 1 ETH
		Receiver: common.HexToAddress("0x0987654321098765432109876543210987654321"),
	}

	// Create the intent first
	intent := &models.Intent{
		ID:               event.IntentID.Hex(),
		SourceChain:      7001,
		DestinationChain: 42161,
		Token:            event.Asset.Hex(),
		Amount:           "2000000000000000000", // 2 ETH
		Recipient:        event.Receiver.Hex(),
		IntentFee:        "100000000000000000", // 0.1 ETH
		Status:           models.IntentStatusPending,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err = mockDB.CreateIntent(ctx, intent)
	assert.NoError(t, err)

	// Create a test log entry
	log := types.Log{
		Topics: []common.Hash{
			common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234"),
			event.IntentID,
			common.BytesToHash(event.Asset.Bytes()),
			common.BytesToHash(event.Receiver.Bytes()),
		},
		Data: common.FromHex("0x0000000000000000000000000000000000000000000000000de0b6b3a7640000"), // Amount: 1 ETH
	}

	// Process the log
	err = service.processLog(ctx, service.clients[42161], log)
	assert.NoError(t, err)
}

func TestCreateFulfillment(t *testing.T) {
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

	// Create test config
	testConfig := &config.Config{
		ChainConfigs: map[uint64]*config.ChainConfig{
			42161: {
				DefaultBlock: 322207320, // Arbitrum
			},
			7001: {
				DefaultBlock: 1000000, // ZetaChain
			},
		},
	}

	// Create default blocks map from test config
	defaultBlocks := make(map[uint64]uint64)
	for chainID, chainConfig := range testConfig.ChainConfigs {
		defaultBlocks[chainID] = chainConfig.DefaultBlock
	}

	// Create a fulfillment service
	abi := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":true,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"IntentFulfilled","type":"event"}]`
	service, err := NewFulfillmentService(clients, contractAddresses, db.DBInterface(mockDB), abi, defaultBlocks)
	assert.NoError(t, err)
	assert.NotNil(t, service)

	// Test creating a fulfillment
	ctx := context.Background()
	intentID := "0x1234567890123456789012345678901234567890123456789012345678901234"
	txHash := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	// Create the intent first
	intent := &models.Intent{
		ID:               intentID,
		SourceChain:      7001,
		DestinationChain: 42161,
		Token:            "0x1234567890123456789012345678901234567890",
		Amount:           "2000000000000000000", // 2 ETH
		Recipient:        "0x0987654321098765432109876543210987654321",
		IntentFee:        "100000000000000000", // 0.1 ETH
		Status:           models.IntentStatusPending,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err = mockDB.CreateIntent(ctx, intent)
	assert.NoError(t, err)

	fulfillment, err := service.CreateFulfillment(ctx, intentID, txHash)
	assert.NoError(t, err)
	assert.NotNil(t, fulfillment)
}

func TestGetFulfillment(t *testing.T) {
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

	// Create test config
	testConfig := &config.Config{
		ChainConfigs: map[uint64]*config.ChainConfig{
			42161: {
				DefaultBlock: 322207320, // Arbitrum
			},
			7001: {
				DefaultBlock: 1000000, // ZetaChain
			},
		},
	}

	// Create default blocks map from test config
	defaultBlocks := make(map[uint64]uint64)
	for chainID, chainConfig := range testConfig.ChainConfigs {
		defaultBlocks[chainID] = chainConfig.DefaultBlock
	}

	// Create a fulfillment service
	abi := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":true,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"IntentFulfilled","type":"event"}]`
	service, err := NewFulfillmentService(clients, contractAddresses, db.DBInterface(mockDB), abi, defaultBlocks)
	assert.NoError(t, err)
	assert.NotNil(t, service)

	// Test getting a fulfillment
	ctx := context.Background()
	fulfillmentID := "0x1234567890123456789012345678901234567890123456789012345678901234"
	intentID := "0x1234567890123456789012345678901234567890123456789012345678901234"

	// Create the intent first
	intent := &models.Intent{
		ID:               intentID,
		SourceChain:      7001,
		DestinationChain: 42161,
		Token:            "0x1234567890123456789012345678901234567890",
		Amount:           "2000000000000000000", // 2 ETH
		Recipient:        "0x0987654321098765432109876543210987654321",
		IntentFee:        "100000000000000000", // 0.1 ETH
		Status:           models.IntentStatusPending,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err = mockDB.CreateIntent(ctx, intent)
	assert.NoError(t, err)

	// Create the fulfillment
	fulfillment := &models.Fulfillment{
		ID:        fulfillmentID,
		IntentID:  intentID,
		TxHash:    fulfillmentID,
		Status:    models.FulfillmentStatusCompleted,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = mockDB.CreateFulfillment(ctx, fulfillment)
	assert.NoError(t, err)

	// Now get the fulfillment
	retrievedFulfillment, err := service.GetFulfillment(ctx, fulfillmentID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedFulfillment)
	assert.Equal(t, fulfillmentID, retrievedFulfillment.ID)
	assert.Equal(t, intentID, retrievedFulfillment.IntentID)
	assert.Equal(t, fulfillmentID, retrievedFulfillment.TxHash)
	assert.Equal(t, models.FulfillmentStatusCompleted, retrievedFulfillment.Status)
}

func TestListFulfillments(t *testing.T) {
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

	// Default blocks
	defaultBlocks := map[uint64]uint64{
		42161: 1000000,
		7001:  2000000,
	}

	// Create a fulfillment service
	abi := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":true,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"IntentFulfilled","type":"event"}]`
	service, err := NewFulfillmentService(clients, contractAddresses, db.DBInterface(mockDB), abi, defaultBlocks)
	assert.NoError(t, err)
	assert.NotNil(t, service)

	// Test listing fulfillments
	ctx := context.Background()
	fulfillments, err := service.ListFulfillments(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, fulfillments)
}

func TestListFulfillmentsAndFlow(t *testing.T) {
	// Create mock database
	mockDB := mocks.NewMockDB()

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
	err := mockDB.CreateIntent(context.Background(), intent)
	assert.NoError(t, err)

	// Create service
	clients := map[uint64]*ethclient.Client{
		42161: {},
	}
	contractAddresses := map[uint64]string{
		42161: "0x1234567890123456789012345678901234567890",
	}
	contractABI := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":true,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"IntentFulfilled","type":"event"}]`

	// Default blocks
	defaultBlocks := map[uint64]uint64{
		42161: 1000000,
	}

	service, err := NewFulfillmentService(clients, contractAddresses, db.DBInterface(mockDB), contractABI, defaultBlocks)
	assert.NoError(t, err)

	// Test listing fulfillments
	fulfillments, err := service.ListFulfillments(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, fulfillments, "should have no fulfillments initially")

	// Create a fulfillment
	txHash := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	fulfillment, err := service.CreateFulfillment(
		context.Background(),
		intent.ID,
		txHash,
	)
	assert.NoError(t, err)
	assert.NotNil(t, fulfillment)

	// List fulfillments again
	fulfillments, err = service.ListFulfillments(context.Background())
	assert.NoError(t, err)
	assert.Len(t, fulfillments, 1, "should have one fulfillment")
	assert.Equal(t, intent.ID, fulfillments[0].IntentID)
	assert.Equal(t, txHash, fulfillments[0].TxHash)
	assert.Equal(t, models.FulfillmentStatusPending, fulfillments[0].Status)
}
