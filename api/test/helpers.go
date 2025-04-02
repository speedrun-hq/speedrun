package test

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeta-chain/zetafast/api/models"
	"github.com/zeta-chain/zetafast/api/test/mocks"
)

// TestHelper provides common testing utilities
type TestHelper struct {
	MockDB *mocks.MockDB
}

// NewTestHelper creates a new TestHelper instance
func NewTestHelper() *TestHelper {
	return &TestHelper{
		MockDB: mocks.NewMockDB(),
	}
}

// CreateTestIntent creates a test intent
func (h *TestHelper) CreateTestIntent() *models.Intent {
	now := time.Now()
	return &models.Intent{
		ID:               "test-intent-id",
		SourceChain:      7001,
		DestinationChain: 42161, // Arbitrum
		Token:            "0x1234567890123456789012345678901234567890",
		Amount:           "1000000000000000000", // 1 ETH
		Recipient:        "0x0987654321098765432109876543210987654321",
		IntentFee:        "100000000000000000", // 0.1 ETH
		Status:           models.IntentStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// CreateTestFulfillment creates a test fulfillment
func (h *TestHelper) CreateTestFulfillment(intentID string) *models.Fulfillment {
	now := time.Now()
	return &models.Fulfillment{
		ID:          "test-fulfillment-id",
		IntentID:    intentID,
		Fulfiller:   "0x0987654321098765432109876543210987654321",
		TargetChain: 42161,
		Amount:      "1000000000000000000", // 1 ETH
		Status:      models.FulfillmentStatusPending,
		TxHash:      "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		BlockNumber: 12345678,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// CreateTestIntentInitiatedEvent creates a test intent initiated event
func (h *TestHelper) CreateTestIntentInitiatedEvent() *models.IntentInitiatedEvent {
	return &models.IntentInitiatedEvent{
		IntentID:    "test-intent-id",
		Asset:       "0x1234567890123456789012345678901234567890",
		Amount:      big.NewInt(1000000000000000000), // 1 ETH
		TargetChain: 42161,                           // Arbitrum
		Receiver:    common.FromHex("0x0987654321098765432109876543210987654321"),
		Tip:         big.NewInt(100000000000000000), // 0.1 ETH
		Salt:        big.NewInt(0),
		ChainID:     7001, // ZetaChain
		BlockNumber: 12345678,
		TxHash:      "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
	}
}

// CreateTestFulfillmentEvent creates a test fulfillment event
func (h *TestHelper) CreateTestFulfillmentEvent() *models.FulfillmentEvent {
	return &models.FulfillmentEvent{
		IntentID:    "test-intent-id",
		TargetChain: 42161,
		Receiver:    "0x0987654321098765432109876543210987654321",
		Amount:      "1000000000000000000", // 1 ETH
		TxHash:      "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		BlockNumber: 12345678,
	}
}

// SetupTestData sets up test data in the mock database
func (h *TestHelper) SetupTestData() error {
	ctx := context.Background()

	// Create test intent
	intent := h.CreateTestIntent()
	if err := h.MockDB.CreateIntent(ctx, intent); err != nil {
		return err
	}

	// Create test fulfillment
	fulfillment := h.CreateTestFulfillment(intent.ID)
	if err := h.MockDB.CreateFulfillment(ctx, fulfillment); err != nil {
		return err
	}

	return nil
}
