package services

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/zeta-chain/zetafast/api/models"
	"github.com/zeta-chain/zetafast/api/test/mocks"
)

func TestNewIntentService(t *testing.T) {
	// Create a mock database and eth client
	mockDB := mocks.NewMockDB()
	ethClient := createMockEthClient()

	// Create an intent service with a valid ABI
	abi := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":false,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"targetChain","type":"uint256"},{"indexed":false,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"tip","type":"uint256"},{"indexed":false,"internalType":"bytes32","name":"salt","type":"bytes32"}],"name":"IntentInitiated","type":"event"}]`
	service, err := NewIntentService(ethClient, mockDB, abi)
	assert.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, mockDB, service.db)

	// Test with invalid ABI
	service, err = NewIntentService(ethClient, mockDB, "invalid abi")
	assert.Error(t, err)
	assert.Nil(t, service)
}

func TestIntentServiceStartListening(t *testing.T) {
	// Skip this test for now as it requires a real Ethereum client
	t.Skip("Skipping test that requires a real Ethereum client")

	// Create a mock database and eth client
	mockDB := mocks.NewMockDB()
	ethClient := createMockEthClient()

	// Create an intent service
	service, err := NewIntentService(ethClient, mockDB, `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":false,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"targetChain","type":"uint256"},{"indexed":false,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"tip","type":"uint256"},{"indexed":false,"internalType":"bytes32","name":"salt","type":"bytes32"}],"name":"IntentInitiated","type":"event"}]`)
	assert.NoError(t, err)

	// Test starting to listen for events
	ctx := context.Background()
	contractAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")
	err = service.StartListening(ctx, contractAddress)
	assert.NoError(t, err)
}

func TestProcessIntentEvent(t *testing.T) {
	// Create a mock database
	mockDB := mocks.NewMockDB()

	// Create a test event
	amount := new(big.Int)
	amount.SetString("1000000000000000000", 10)
	tip := new(big.Int)
	tip.SetString("100000000000000000", 10)
	salt := new(big.Int)
	salt.SetString("0", 10)

	event := &models.IntentInitiatedEvent{
		IntentID:    "test-intent-id",
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

	// Test processing the event
	ctx := context.Background()
	intent := event.ToIntent()
	err := mockDB.CreateIntent(ctx, intent)
	assert.NoError(t, err)

	// Verify intent was created
	retrievedIntent, err := mockDB.GetIntent(ctx, intent.ID)
	assert.NoError(t, err)
	assert.Equal(t, intent.ID, retrievedIntent.ID)
	assert.Equal(t, intent.SourceChain, retrievedIntent.SourceChain)
	assert.Equal(t, intent.DestinationChain, retrievedIntent.DestinationChain)
	assert.Equal(t, intent.Token, retrievedIntent.Token)
	assert.Equal(t, intent.Amount, retrievedIntent.Amount)
	assert.Equal(t, intent.Recipient, retrievedIntent.Recipient)
	assert.Equal(t, intent.IntentFee, retrievedIntent.IntentFee)
	assert.Equal(t, models.IntentStatusPending, retrievedIntent.Status)
}
