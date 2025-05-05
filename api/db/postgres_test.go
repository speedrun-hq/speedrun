package db

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*PostgresDB, sqlmock.Sqlmock) {
	// Create SQL mock
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "Failed to create mock DB")

	postgresDB := &PostgresDB{db: db}
	return postgresDB, mock
}

func TestCreateIntent(t *testing.T) {
	postgresDB, mock := setupTestDB(t)
	defer postgresDB.Close()

	now := time.Now().UTC().Truncate(time.Microsecond)

	intent := &models.Intent{
		ID:               "0x1234567890123456789012345678901234567890123456789012345678901234",
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

	// Setup expectations
	mock.ExpectExec(`INSERT INTO intents`).
		WithArgs(
			intent.ID,
			intent.SourceChain,
			intent.DestinationChain,
			intent.Token,
			intent.Amount,
			intent.Recipient,
			intent.Sender,
			intent.IntentFee,
			string(intent.Status),
			intent.CreatedAt,
			intent.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Run test
	err := postgresDB.CreateIntent(context.Background(), intent)
	assert.NoError(t, err)

	// Verify expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetIntent(t *testing.T) {
	postgresDB, mock := setupTestDB(t)
	defer postgresDB.Close()

	intentID := "0x1234567890123456789012345678901234567890123456789012345678901234"
	now := time.Now().UTC().Truncate(time.Microsecond)

	expectedIntent := &models.Intent{
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

	// Setup the expected rows
	rows := sqlmock.NewRows([]string{
		"id", "source_chain", "destination_chain", "token", "amount",
		"recipient", "sender", "intent_fee", "status", "created_at", "updated_at",
	}).
		AddRow(
			expectedIntent.ID, expectedIntent.SourceChain, expectedIntent.DestinationChain,
			expectedIntent.Token, expectedIntent.Amount, expectedIntent.Recipient,
			expectedIntent.Sender, expectedIntent.IntentFee, string(expectedIntent.Status),
			expectedIntent.CreatedAt, expectedIntent.UpdatedAt,
		)

	// Setup expectations
	mock.ExpectQuery(`SELECT .* FROM intents WHERE id = \$1`).
		WithArgs(intentID).
		WillReturnRows(rows)

	// Run test
	intent, err := postgresDB.GetIntent(context.Background(), intentID)
	assert.NoError(t, err)
	assert.Equal(t, expectedIntent.ID, intent.ID)
	assert.Equal(t, expectedIntent.SourceChain, intent.SourceChain)
	assert.Equal(t, expectedIntent.Amount, intent.Amount)
	assert.Equal(t, expectedIntent.Status, intent.Status)
	assert.Equal(t, expectedIntent.CreatedAt, intent.CreatedAt)

	// Verify expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateIntentStatus(t *testing.T) {
	postgresDB, mock := setupTestDB(t)
	defer postgresDB.Close()

	intentID := "0x1234567890123456789012345678901234567890123456789012345678901234"
	status := models.IntentStatusSettled

	// Setup expectations
	mock.ExpectExec(`UPDATE intents SET status = \$1, updated_at = NOW\(\) WHERE id = \$2`).
		WithArgs(
			string(status),
			intentID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Run test
	err := postgresDB.UpdateIntentStatus(context.Background(), intentID, status)
	assert.NoError(t, err)

	// Verify expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateSettlement(t *testing.T) {
	postgresDB, mock := setupTestDB(t)
	defer postgresDB.Close()

	now := time.Now().UTC().Truncate(time.Microsecond)

	settlement := &models.Settlement{
		ID:           "0x1234567890123456789012345678901234567890123456789012345678901234",
		Asset:        "0x1234567890123456789012345678901234567890",
		Amount:       "1000000000000000000", // 1 ETH
		Receiver:     "0x9876543210987654321098765432109876543210",
		Fulfilled:    true,
		Fulfiller:    "0x5678901234567890123456789012345678901234",
		ActualAmount: "900000000000000000", // 0.9 ETH
		PaidTip:      "100000000000000000", // 0.1 ETH
		TxHash:       "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Setup expectations
	mock.ExpectExec(`INSERT INTO settlements`).
		WithArgs(
			settlement.ID,
			settlement.Asset,
			settlement.Amount,
			settlement.Receiver,
			settlement.Fulfilled,
			settlement.Fulfiller,
			settlement.ActualAmount,
			settlement.PaidTip,
			settlement.TxHash,
			settlement.CreatedAt,
			settlement.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Run test
	err := postgresDB.CreateSettlement(context.Background(), settlement)
	assert.NoError(t, err)

	// Verify expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSettlement(t *testing.T) {
	postgresDB, mock := setupTestDB(t)
	defer postgresDB.Close()

	intentID := "0x1234567890123456789012345678901234567890123456789012345678901234"
	now := time.Now().UTC().Truncate(time.Microsecond)

	expectedSettlement := &models.Settlement{
		ID:           intentID,
		Asset:        "0x1234567890123456789012345678901234567890",
		Amount:       "1000000000000000000", // 1 ETH
		Receiver:     "0x9876543210987654321098765432109876543210",
		Fulfilled:    true,
		Fulfiller:    "0x5678901234567890123456789012345678901234",
		ActualAmount: "900000000000000000", // 0.9 ETH
		PaidTip:      "100000000000000000", // 0.1 ETH
		TxHash:       "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Setup the expected rows
	rows := sqlmock.NewRows([]string{
		"id", "asset", "amount", "receiver", "fulfilled", "fulfiller",
		"actual_amount", "paid_tip", "tx_hash", "created_at", "updated_at",
	}).
		AddRow(
			expectedSettlement.ID, expectedSettlement.Asset, expectedSettlement.Amount,
			expectedSettlement.Receiver, expectedSettlement.Fulfilled, expectedSettlement.Fulfiller,
			expectedSettlement.ActualAmount, expectedSettlement.PaidTip, expectedSettlement.TxHash,
			expectedSettlement.CreatedAt, expectedSettlement.UpdatedAt,
		)

	// Setup expectations
	mock.ExpectQuery(`SELECT .* FROM settlements WHERE id = \$1`).
		WithArgs(intentID).
		WillReturnRows(rows)

	// Run test
	settlement, err := postgresDB.GetSettlement(context.Background(), intentID)
	assert.NoError(t, err)
	assert.Equal(t, expectedSettlement.ID, settlement.ID)
	assert.Equal(t, expectedSettlement.Asset, settlement.Asset)
	assert.Equal(t, expectedSettlement.Amount, settlement.Amount)
	assert.Equal(t, expectedSettlement.Fulfilled, settlement.Fulfilled)
	assert.Equal(t, expectedSettlement.CreatedAt, settlement.CreatedAt)

	// Verify expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetLastProcessedBlock(t *testing.T) {
	postgresDB, mock := setupTestDB(t)
	defer postgresDB.Close()

	chainID := uint64(1)
	expectedBlockNumber := uint64(12345)

	// Setup the expected rows
	rows := sqlmock.NewRows([]string{"block_number"}).
		AddRow(expectedBlockNumber)

	// Setup expectations
	mock.ExpectQuery(`SELECT block_number FROM last_processed_blocks WHERE chain_id = \$1`).
		WithArgs(chainID).
		WillReturnRows(rows)

	// Run test
	blockNumber, err := postgresDB.GetLastProcessedBlock(context.Background(), chainID)
	assert.NoError(t, err)
	assert.Equal(t, expectedBlockNumber, blockNumber)

	// Verify expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateLastProcessedBlock(t *testing.T) {
	postgresDB, mock := setupTestDB(t)
	defer postgresDB.Close()

	chainID := uint64(1)
	blockNumber := uint64(12345)

	// Setup expectations
	mock.ExpectExec(`INSERT INTO last_processed_blocks .* ON CONFLICT .* DO UPDATE`).
		WithArgs(
			chainID,
			blockNumber,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Run test
	err := postgresDB.UpdateLastProcessedBlock(context.Background(), chainID, blockNumber)
	assert.NoError(t, err)

	// Verify expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
