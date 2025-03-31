package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/zeta-chain/zetafast/api/models"
)

type DB struct {
	*sql.DB
}

func NewDB(connStr string) (*DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &DB{db}, nil
}

// CreateFulfillment inserts a new fulfillment into the database
func (db *DB) CreateFulfillment(fulfillment *models.Fulfillment) error {
	query := `
		INSERT INTO fulfillments (id, intent_id, fulfiller, amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	return db.QueryRow(
		query,
		fulfillment.ID,
		fulfillment.IntentID,
		fulfillment.Fulfiller,
		fulfillment.Amount,
		fulfillment.Status,
		fulfillment.CreatedAt,
		fulfillment.UpdatedAt,
	).Scan(&fulfillment.ID)
}

// GetFulfillment retrieves a fulfillment by ID
func (db *DB) GetFulfillment(id string) (*models.Fulfillment, error) {
	fulfillment := &models.Fulfillment{}
	query := `
		SELECT id, intent_id, fulfiller, amount, status, created_at, updated_at
		FROM fulfillments
		WHERE id = $1`

	err := db.QueryRow(query, id).Scan(
		&fulfillment.ID,
		&fulfillment.IntentID,
		&fulfillment.Fulfiller,
		&fulfillment.Amount,
		&fulfillment.Status,
		&fulfillment.CreatedAt,
		&fulfillment.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("fulfillment not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error querying fulfillment: %v", err)
	}

	return fulfillment, nil
}

// GetIntent retrieves an intent by ID
func (db *DB) GetIntent(id string) (*models.Intent, error) {
	intent := &models.Intent{}
	query := `
		SELECT id, source_chain, destination_chain, token, amount, recipient, intent_fee, status, created_at, updated_at
		FROM intents
		WHERE id = $1`

	err := db.QueryRow(query, id).Scan(
		&intent.ID,
		&intent.SourceChain,
		&intent.DestinationChain,
		&intent.Token,
		&intent.Amount,
		&intent.Recipient,
		&intent.IntentFee,
		&intent.Status,
		&intent.CreatedAt,
		&intent.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("intent not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error querying intent: %v", err)
	}

	return intent, nil
}

// GetTotalFulfilledAmount gets the total amount fulfilled for an intent
func (db *DB) GetTotalFulfilledAmount(intentID string) (string, error) {
	var total string
	query := `
		SELECT COALESCE(SUM(amount), '0')
		FROM fulfillments
		WHERE intent_id = $1 AND status = $2`

	err := db.QueryRow(query, intentID, models.FulfillmentStatusAccepted).Scan(&total)
	if err != nil {
		return "0", fmt.Errorf("error getting total fulfilled amount: %v", err)
	}

	return total, nil
}

// UpdateIntentStatus updates the status of an intent
func (db *DB) UpdateIntentStatus(id, status string) error {
	query := `
		UPDATE intents
		SET status = $1, updated_at = NOW()
		WHERE id = $2`

	result, err := db.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("error updating intent status: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %v", err)
	}

	if rows == 0 {
		return fmt.Errorf("intent not found")
	}

	return nil
}
