package services

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/zeta-chain/zetafast/api/config"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/models"
	"github.com/zeta-chain/zetafast/api/test/mocks"
)

// TestIntentFulfillmentFlow tests the complete flow from intent creation to fulfillment
func TestIntentFulfillmentFlow(t *testing.T) {
	// Create mock database
	mockDB := mocks.NewMockDB()

	// Create mock Ethereum clients
	clients := make(map[uint64]*ethclient.Client)
	clients[7001] = createMockEthClient()
	clients[42161] = createMockEthClient()
	clients[8453] = createMockEthClient()

	// Create contract addresses map
	contractAddresses := make(map[uint64]string)
	contractAddresses[7001] = "0x1234567890123456789012345678901234567890"
	contractAddresses[42161] = "0x1234567890123456789012345678901234567890"
	contractAddresses[8453] = "0x0987654321098765432109876543210987654321"

	// Create test config
	testConfig := &config.Config{
		ChainConfigs: map[uint64]*config.ChainConfig{
			42161: {
				DefaultBlock: 322207320, // Arbitrum
			},
			8453: {
				DefaultBlock: 28411000, // Base
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

	// Intent contract ABI
	intentABI := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":false,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"targetChain","type":"uint256"},{"indexed":false,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"tip","type":"uint256"},{"indexed":false,"internalType":"bytes32","name":"salt","type":"bytes32"}],"name":"IntentInitiated","type":"event"}]`

	// Fulfillment contract ABI
	fulfillmentABI := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":true,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"IntentFulfilled","type":"event"}]`

	// Create intent service
	intentService, err := NewIntentService(clients[7001], mockDB, intentABI, 7001)
	assert.NoError(t, err)
	assert.NotNil(t, intentService)

	// Create fulfillment service
	fulfillmentService, err := NewFulfillmentService(clients, contractAddresses, db.DBInterface(mockDB), fulfillmentABI, defaultBlocks)
	assert.NoError(t, err)
	assert.NotNil(t, fulfillmentService)

	// Create a test intent event
	amount := new(big.Int)
	amount.SetString("2000000000000000000", 10) // 2 ETH
	tip := new(big.Int)
	tip.SetString("100000000000000000", 10) // 0.1 ETH
	salt := new(big.Int)
	salt.SetString("0", 10)

	intentID := "0x1234567890123456789012345678901234567890123456789012345678901234"
	event := &models.IntentInitiatedEvent{
		IntentID:    intentID,
		Asset:       common.HexToAddress("0x1234567890123456789012345678901234567890").Hex(),
		Amount:      amount,
		TargetChain: 42161,
		Receiver:    common.FromHex("0x0987654321098765432109876543210987654321"),
		Tip:         tip,
		Salt:        salt,
		ChainID:     7001,
		BlockNumber: 12345678,
		TxHash:      common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890").Hex(),
	}

	// Process the intent event
	ctx := context.Background()
	intent := event.ToIntent()
	err = mockDB.CreateIntent(ctx, intent)
	assert.NoError(t, err)

	// Verify intent was created with correct status
	retrievedIntent, err := mockDB.GetIntent(ctx, intent.ID)
	assert.NoError(t, err)
	assert.Equal(t, models.IntentStatusPending, retrievedIntent.Status)

	// Create a fulfillment event
	fulfillmentLog := types.Log{
		Address: common.HexToAddress(contractAddresses[42161]),
		Topics: []common.Hash{
			common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234"), // Event ID
			common.HexToHash(intentID),                                     // Intent ID
			common.HexToHash("0x1234567890123456789012345678901234567890"), // Asset
			common.HexToHash("0x0987654321098765432109876543210987654321"), // Receiver
		},
		Data:        common.FromHex("0x0000000000000000000000000000000000000000000000000de0b6b3a7640000"), // Amount: 1 ETH
		BlockNumber: 12345679,
		TxHash:      common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
	}

	// Process the fulfillment log
	err = fulfillmentService.processLog(ctx, fulfillmentService.clients[42161], fulfillmentLog)
	assert.NoError(t, err)

	// Verify fulfillment was created
	fulfillments, err := mockDB.ListFulfillments(ctx)
	assert.NoError(t, err)
	assert.Len(t, fulfillments, 1)
	assert.Equal(t, intentID, fulfillments[0].IntentID)
	assert.Equal(t, fulfillmentLog.TxHash.Hex(), fulfillments[0].TxHash)
	assert.Equal(t, models.FulfillmentStatusCompleted, fulfillments[0].Status)

	// Verify intent status is still pending
	updatedIntent, err := mockDB.GetIntent(ctx, intent.ID)
	assert.NoError(t, err)
	assert.Equal(t, models.IntentStatusPending, updatedIntent.Status)

	// Create a second fulfillment for the same intent (partial fulfillment)
	fulfillmentLog2 := types.Log{
		Address: common.HexToAddress(contractAddresses[42161]),
		Topics: []common.Hash{
			common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234"), // Event ID
			common.HexToHash(intentID),                                     // Intent ID
			common.HexToHash("0x1234567890123456789012345678901234567890"), // Asset
			common.HexToHash("0x0987654321098765432109876543210987654321"), // Receiver
		},
		Data:        common.FromHex("0x0000000000000000000000000000000000000000000000000de0b6b3a7640000"), // Amount: 1 ETH
		BlockNumber: 12345680,
		TxHash:      common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567891"),
	}

	// Process the second fulfillment log
	err = fulfillmentService.processLog(ctx, fulfillmentService.clients[42161], fulfillmentLog2)
	assert.NoError(t, err)

	// Verify both fulfillments exist
	fulfillments, err = mockDB.ListFulfillments(ctx)
	assert.NoError(t, err)
	assert.Len(t, fulfillments, 2)

	// Verify total fulfilled amount
	totalFulfilled, err := mockDB.GetTotalFulfilledAmount(ctx, intentID)
	assert.NoError(t, err)
	assert.Equal(t, "2000000000000000000", totalFulfilled) // 2 ETH total

	// Verify intent status is now fulfilled
	updatedIntent, err = mockDB.GetIntent(ctx, intent.ID)
	assert.NoError(t, err)
	assert.Equal(t, models.IntentStatusFulfilled, updatedIntent.Status)
}
