package db

import (
	"context"
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
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	CreateIntent(ctx context.Context, intent *models.Intent) error
	GetIntent(ctx context.Context, id string) (*models.Intent, error)
	ListIntents(ctx context.Context) ([]*models.Intent, error)
	CreateFulfillment(ctx context.Context, fulfillment *models.Fulfillment) error
	GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error)
	ListFulfillments(ctx context.Context) ([]*models.Fulfillment, error)
	GetTotalFulfilledAmount(ctx context.Context, intentID string) (string, error)
	UpdateIntentStatus(ctx context.Context, id string, status models.IntentStatus) error
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

// CreateIntent creates a new intent in the database
func (db *DB) CreateIntent(ctx context.Context, intent *models.Intent) error {
	query := `
		INSERT INTO intents (
			id, source_chain, destination_chain, token, amount,
			recipient, intent_fee, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := db.ExecContext(ctx, query,
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
	)
	return err
}

// GetIntent retrieves an intent by ID
func (db *DB) GetIntent(ctx context.Context, id string) (*models.Intent, error) {
	query := `
		SELECT id, source_chain, destination_chain, token, amount,
			recipient, intent_fee, status, created_at, updated_at
		FROM intents WHERE id = $1
	`
	intent := &models.Intent{}
	err := db.QueryRowContext(ctx, query, id).Scan(
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
		return nil, err
	}
	return intent, nil
}

// ListIntents retrieves all intents
func (db *DB) ListIntents(ctx context.Context) ([]*models.Intent, error) {
	query := `
		SELECT id, source_chain, destination_chain, token, amount,
			recipient, intent_fee, status, created_at, updated_at
		FROM intents ORDER BY created_at DESC
	`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		intents = append(intents, intent)
	}
	return intents, nil
}

// CreateFulfillment creates a new fulfillment in the database
func (db *DB) CreateFulfillment(ctx context.Context, fulfillment *models.Fulfillment) error {
	query := `
		INSERT INTO fulfillments (
			id, intent_id, amount, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.ExecContext(ctx, query,
		fulfillment.ID,
		fulfillment.IntentID,
		fulfillment.Amount,
		fulfillment.Status,
		fulfillment.CreatedAt,
		fulfillment.UpdatedAt,
	)
	return err
}

// GetFulfillment retrieves a fulfillment by ID
func (db *DB) GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error) {
	query := `
		SELECT id, intent_id, amount, status, created_at, updated_at
		FROM fulfillments WHERE id = $1
	`
	fulfillment := &models.Fulfillment{}
	err := db.QueryRowContext(ctx, query, id).Scan(
		&fulfillment.ID,
		&fulfillment.IntentID,
		&fulfillment.Amount,
		&fulfillment.Status,
		&fulfillment.CreatedAt,
		&fulfillment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return fulfillment, nil
}

// GetTotalFulfilledAmount calculates the total amount fulfilled for an intent
func (db *DB) GetTotalFulfilledAmount(ctx context.Context, intentID string) (string, error) {
	query := `
		SELECT COALESCE(SUM(amount), '0')
		FROM fulfillments
		WHERE intent_id = $1 AND status = 'completed'
	`
	var total string
	err := db.QueryRowContext(ctx, query, intentID).Scan(&total)
	if err != nil {
		return "0", err
	}
	return total, nil
}

// UpdateIntentStatus updates the status of an intent
func (db *DB) UpdateIntentStatus(ctx context.Context, id string, status models.IntentStatus) error {
	query := `
		UPDATE intents
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := db.ExecContext(ctx, query, string(status), id)
	return err
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
