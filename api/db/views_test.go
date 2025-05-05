package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test requires a PostgreSQL database
// We'll use a skip mechanism if the DB isn't available
func TestIntentLifecycleView(t *testing.T) {
	// Use test database URL
	const testDBURL = "postgresql://postgres:postgres@localhost:5432/speedrun_test?sslmode=disable"

	// Try to connect to the test database
	db, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Skip("Skipping view test, could not connect to test database:", err)
		return
	}
	defer db.Close()

	// Ping the database to make sure it's available
	if err := db.Ping(); err != nil {
		t.Skip("Skipping view test, could not ping test database:", err)
		return
	}

	// Create PostgresDB instance
	postgresDB := &PostgresDB{db: db}

	// Initialize the database schema with our views
	err = postgresDB.InitDB(context.Background())
	require.NoError(t, err, "Failed to initialize test database schema")

	// Clean any existing test data
	_, err = db.Exec("DELETE FROM settlements WHERE id LIKE 'test-%'")
	require.NoError(t, err, "Failed to clean settlements")

	_, err = db.Exec("DELETE FROM fulfillments WHERE id LIKE 'test-%'")
	require.NoError(t, err, "Failed to clean fulfillments")

	_, err = db.Exec("DELETE FROM intents WHERE id LIKE 'test-%'")
	require.NoError(t, err, "Failed to clean intents")

	// Setup test data
	ctx := context.Background()

	// Create test intent
	intentID := "test-0x1234567890123456789012345678901234567890123456789012345678901234"
	now := time.Now()

	intent := &models.Intent{
		ID:               intentID,
		SourceChain:      1,
		DestinationChain: 2,
		Token:            "0x1234567890123456789012345678901234567890",
		Amount:           "1000000000000000000", // 1 ETH
		Recipient:        "0x9876543210987654321098765432109876543210",
		Sender:           "0x5432109876543210987654321098765432109876",
		IntentFee:        "100000000000000000", // 0.1 ETH
		Status:           models.IntentStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	err = postgresDB.CreateIntent(ctx, intent)
	require.NoError(t, err, "Failed to create test intent")

	// Create test fulfillment (5 minutes later)
	fulfillmentTime := now.Add(5 * time.Minute)

	fulfillment := &models.Fulfillment{
		ID:        intentID,
		Asset:     "0x1234567890123456789012345678901234567890",
		Amount:    "1000000000000000000",
		Receiver:  "0x9876543210987654321098765432109876543210",
		TxHash:    "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		CreatedAt: fulfillmentTime,
		UpdatedAt: fulfillmentTime,
	}

	err = postgresDB.CreateFulfillment(ctx, fulfillment)
	require.NoError(t, err, "Failed to create test fulfillment")

	// Update intent status to fulfilled
	err = postgresDB.UpdateIntentStatus(ctx, intentID, models.IntentStatusFulfilled)
	require.NoError(t, err, "Failed to update intent status to fulfilled")

	// Create test settlement (10 minutes after intent creation)
	settlementTime := now.Add(10 * time.Minute)

	settlement := &models.Settlement{
		ID:           intentID,
		Asset:        "0x1234567890123456789012345678901234567890",
		Amount:       "1000000000000000000",
		Receiver:     "0x9876543210987654321098765432109876543210",
		Fulfilled:    true,
		Fulfiller:    "0x5678901234567890123456789012345678901234",
		ActualAmount: "900000000000000000",
		PaidTip:      "100000000000000000",
		TxHash:       "0xfedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321",
		CreatedAt:    settlementTime,
		UpdatedAt:    settlementTime,
	}

	err = postgresDB.CreateSettlement(ctx, settlement)
	require.NoError(t, err, "Failed to create test settlement")

	// Update intent status to settled
	err = postgresDB.UpdateIntentStatus(ctx, intentID, models.IntentStatusSettled)
	require.NoError(t, err, "Failed to update intent status to settled")

	// Query the intent_lifecycle_view
	rows, err := db.Query("SELECT * FROM intent_lifecycle_view WHERE id = $1", intentID)
	require.NoError(t, err, "Failed to query intent_lifecycle_view")
	defer rows.Close()

	// Verify data
	require.True(t, rows.Next(), "No rows returned from intent_lifecycle_view")

	var (
		id                  string
		sourceChain         int64
		destChain           int64
		token               string
		amount              string
		recipient           string
		sender              string
		intentFee           string
		status              string
		intentCreatedAt     time.Time
		dbFulfillmentTime   sql.NullTime
		dbSettlementTime    sql.NullTime
		settlementFulfilled sql.NullBool
		fulfiller           sql.NullString
		actualAmount        sql.NullString
		paidTip             sql.NullString
		totalTime           sql.NullFloat64
		timeToFulfillment   sql.NullFloat64
	)

	err = rows.Scan(
		&id, &sourceChain, &destChain, &token, &amount, &recipient, &sender, &intentFee,
		&status, &intentCreatedAt, &dbFulfillmentTime, &dbSettlementTime, &settlementFulfilled,
		&fulfiller, &actualAmount, &paidTip, &totalTime, &timeToFulfillment,
	)
	require.NoError(t, err, "Failed to scan row from intent_lifecycle_view")

	// Assert the data
	assert.Equal(t, intentID, id)
	assert.Equal(t, "settled", status)
	assert.True(t, dbFulfillmentTime.Valid, "Fulfillment time should be valid")
	assert.True(t, dbSettlementTime.Valid, "Settlement time should be valid")
	assert.True(t, settlementFulfilled.Valid && settlementFulfilled.Bool, "Settlement should be marked as fulfilled")
	assert.True(t, totalTime.Valid && totalTime.Float64 > 0, "Total processing time should be positive")
	assert.True(t, timeToFulfillment.Valid && timeToFulfillment.Float64 > 0, "Time to fulfillment should be positive")

	// Clean up test data
	_, err = db.Exec("DELETE FROM settlements WHERE id = $1", intentID)
	assert.NoError(t, err, "Failed to clean up test settlement")

	_, err = db.Exec("DELETE FROM fulfillments WHERE id = $1", intentID)
	assert.NoError(t, err, "Failed to clean up test fulfillment")

	_, err = db.Exec("DELETE FROM intents WHERE id = $1", intentID)
	assert.NoError(t, err, "Failed to clean up test intent")
}
