package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/zeta-chain/zetafast/api/models"
)

// Database interface defines the methods that a database implementation must provide
type Database interface {
	Close() error
	Ping() error
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)
	CreateIntent(intent *models.Intent) error
	GetIntent(id string) (*models.Intent, error)
	ListIntents(page, limit int) ([]*models.Intent, int, error)
	CreateFulfillment(fulfillment *models.Fulfillment) error
	GetFulfillment(id string) (*models.Fulfillment, error)
	GetTotalFulfilledAmount(intentID string) (string, error)
	UpdateIntentStatus(id, status string) error
}

// DB implements the Database interface
type DB struct {
	*sql.DB
}

// NewDB creates a new database connection
func NewDB(dbURL string) (*DB, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
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

// CreateIntent inserts a new intent into the database
func (db *DB) CreateIntent(intent *models.Intent) error {
	query := `
		INSERT INTO intents (id, source_chain, destination_chain, token, amount, recipient, intent_fee, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	return db.QueryRow(
		query,
		intent.ID,
		intent.SourceChain,
		intent.DestinationChain,
		intent.Token,
		intent.Amount,
		intent.Recipient,
		intent.IntentFee,
		intent.Status,
		intent.CreatedAt,
		intent.UpdatedAt,
	).Scan(&intent.ID)
}

// ListIntents retrieves a list of intents with pagination
func (db *DB) ListIntents(page, limit int) ([]*models.Intent, int, error) {
	offset := (page - 1) * limit

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM intents`
	err := db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting intents: %v", err)
	}

	// Get paginated results
	query := `
		SELECT id, source_chain, destination_chain, token, amount, recipient, intent_fee, status, created_at, updated_at
		FROM intents
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying intents: %v", err)
	}
	defer rows.Close()

	var intents []*models.Intent
	for rows.Next() {
		intent := &models.Intent{}
		err := rows.Scan(
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
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning intent: %v", err)
		}
		intents = append(intents, intent)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating intents: %v", err)
	}

	return intents, total, nil
}

// InitSchema initializes the database schema
func (db *DB) InitSchema() error {
	// Read schema.sql file
	schema, err := os.ReadFile("db/schema.sql")
	if err != nil {
		return fmt.Errorf("error reading schema file: %v", err)
	}

	// Execute schema
	_, err = db.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("error executing schema: %v", err)
	}

	return nil
}
