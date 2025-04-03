package mocks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zeta-chain/zetafast/api/models"
)

func TestMockDB(t *testing.T) {
	// Create a mock database
	db := NewMockDB()

	// Create a test intent
	now := time.Now()
	intent := &models.Intent{
		ID:               "test-intent-id",
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

	// Create a test fulfillment
	fulfillment := &models.Fulfillment{
		ID:        "test-fulfillment-id",
		IntentID:  intent.ID,
		TxHash:    "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		Status:    models.FulfillmentStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	ctx := context.Background()

	// Test creating an intent
	err := db.CreateIntent(ctx, intent)
	assert.NoError(t, err)

	// Test getting an intent
	retrievedIntent, err := db.GetIntent(ctx, intent.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedIntent)
	assert.Equal(t, intent.ID, retrievedIntent.ID)
	assert.Equal(t, intent.SourceChain, retrievedIntent.SourceChain)
	assert.Equal(t, intent.DestinationChain, retrievedIntent.DestinationChain)
	assert.Equal(t, intent.Token, retrievedIntent.Token)
	assert.Equal(t, intent.Amount, retrievedIntent.Amount)
	assert.Equal(t, intent.Recipient, retrievedIntent.Recipient)
	assert.Equal(t, intent.IntentFee, retrievedIntent.IntentFee)
	assert.Equal(t, intent.Status, retrievedIntent.Status)

	// Test listing intents
	intents, err := db.ListIntents(ctx)
	assert.NoError(t, err)
	assert.Len(t, intents, 1)
	assert.Equal(t, intent.ID, intents[0].ID)

	// Test creating a fulfillment
	err = db.CreateFulfillment(ctx, fulfillment)
	assert.NoError(t, err)

	// Test getting a fulfillment
	retrievedFulfillment, err := db.GetFulfillment(ctx, fulfillment.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedFulfillment)
	assert.Equal(t, fulfillment.ID, retrievedFulfillment.ID)
	assert.Equal(t, fulfillment.IntentID, retrievedFulfillment.IntentID)
	assert.Equal(t, fulfillment.TxHash, retrievedFulfillment.TxHash)
	assert.Equal(t, fulfillment.Status, retrievedFulfillment.Status)

	// Test listing fulfillments
	fulfillments, err := db.ListFulfillments(ctx)
	assert.NoError(t, err)
	assert.Len(t, fulfillments, 1)
	assert.Equal(t, fulfillment.ID, fulfillments[0].ID)

	// Test getting non-existent intent
	retrievedIntent, err = db.GetIntent(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Equal(t, errors.New("intent not found"), err)
	assert.Nil(t, retrievedIntent)

	// Test getting non-existent fulfillment
	retrievedFulfillment, err = db.GetFulfillment(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Equal(t, errors.New("fulfillment not found"), err)
	assert.Nil(t, retrievedFulfillment)
}
