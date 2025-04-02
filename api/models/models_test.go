package models

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestIntentToResponse(t *testing.T) {
	// Create a test intent
	now := time.Now()
	intent := &Intent{
		ID:               "test-intent-id",
		SourceChain:      7001,
		DestinationChain: 42161,
		Token:            "0x1234567890123456789012345678901234567890",
		Amount:           "1000000000000000000",
		Recipient:        "0x0987654321098765432109876543210987654321",
		IntentFee:        "100000000000000000",
		Status:           IntentStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Convert to response
	response := intent.ToResponse()

	// Verify response fields
	assert.Equal(t, intent.ID, response.ID)
	assert.Equal(t, intent.SourceChain, response.SourceChain)
	assert.Equal(t, intent.DestinationChain, response.DestinationChain)
	assert.Equal(t, intent.Token, response.Token)
	assert.Equal(t, intent.Amount, response.Amount)
	assert.Equal(t, intent.Recipient, response.Recipient)
	assert.Equal(t, intent.IntentFee, response.IntentFee)
	assert.Equal(t, string(intent.Status), response.Status)
	assert.Equal(t, intent.CreatedAt, response.CreatedAt)
	assert.Equal(t, intent.UpdatedAt, response.UpdatedAt)
}

func TestFulfillmentEventToFulfillment(t *testing.T) {
	// Create a test fulfillment event
	event := &FulfillmentEvent{
		IntentID:    "test-intent-id",
		TargetChain: 42161,
		Receiver:    "0x0987654321098765432109876543210987654321",
		Amount:      "1000000000000000000",
		TxHash:      "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		BlockNumber: 12345678,
	}

	// Convert to fulfillment
	fulfillment := event.ToFulfillment()

	// Verify fulfillment fields
	assert.Equal(t, event.TxHash, fulfillment.ID)
	assert.Equal(t, event.IntentID, fulfillment.IntentID)
	assert.Equal(t, event.TxHash, fulfillment.TxHash)
	assert.Equal(t, FulfillmentStatusPending, fulfillment.Status)
}

func TestIntentInitiatedEventToIntent(t *testing.T) {
	// Create test values
	intentID := common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234")
	amount := new(big.Int)
	amount.SetString("1000000000000000000", 10) // 1 ETH
	tip := new(big.Int)
	tip.SetString("100000000000000000", 10) // 0.1 ETH
	salt := new(big.Int)
	salt.SetString("123456789", 10)

	// Create a test intent initiated event
	event := &IntentInitiatedEvent{
		IntentID:    intentID.Hex(),
		Asset:       "0x1234567890123456789012345678901234567890",
		Amount:      amount,
		TargetChain: 42161,
		Receiver:    common.FromHex("0x0987654321098765432109876543210987654321"),
		Tip:         tip,
		Salt:        salt,
		ChainID:     7001,
		BlockNumber: 12345678,
		TxHash:      "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
	}

	// Convert to intent
	intent := event.ToIntent()

	// Verify intent fields
	assert.Equal(t, event.IntentID, intent.ID)
	assert.Equal(t, event.ChainID, intent.SourceChain)
	assert.Equal(t, event.Asset, intent.Token)
	assert.Equal(t, amount.String(), intent.Amount)
	assert.Equal(t, common.BytesToAddress(event.Receiver).Hex(), intent.Recipient)
	assert.Equal(t, tip.String(), intent.IntentFee)
	assert.Equal(t, IntentStatusPending, intent.Status)
}
