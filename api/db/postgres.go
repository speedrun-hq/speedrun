package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/zeta-chain/zetafast/api/models"
)

// PostgresDB implements the Database interface using PostgreSQL
type PostgresDB struct {
	db *sql.DB
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(databaseURL string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	postgresDB := &PostgresDB{db: db}

	// Initialize the database schema
	if err := postgresDB.InitDB(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	return postgresDB, nil
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// Ping checks if the database connection is alive
func (p *PostgresDB) Ping() error {
	return p.db.Ping()
}

// Exec executes a query without returning any rows
func (p *PostgresDB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return p.db.ExecContext(ctx, query, args...)
}

// QueryRow executes a query that is expected to return at most one row
func (p *PostgresDB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return p.db.QueryRowContext(ctx, query, args...)
}

// Query executes a query that returns rows
func (p *PostgresDB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, query, args...)
}

// GetIntent retrieves an intent by ID
func (p *PostgresDB) GetIntent(ctx context.Context, id string) (*models.Intent, error) {
	fmt.Printf("Getting intent with ID: %s\n", id)

	query := `
		SELECT id, source_chain, destination_chain, token, amount, recipient, intent_fee, status, created_at, updated_at
		FROM intents
		WHERE id = $1
	`

	var intent models.Intent
	err := p.db.QueryRowContext(ctx, query, id).Scan(
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
		fmt.Printf("Intent not found: %s\n", id)
		return nil, fmt.Errorf("intent not found: %s", id)
	}
	if err != nil {
		fmt.Printf("Error getting intent %s: %v\n", id, err)
		return nil, fmt.Errorf("failed to get intent: %v", err)
	}

	fmt.Printf("Found intent - ID: %s, SourceChain: %d, DestinationChain: %d, Status: %s\n",
		intent.ID,
		intent.SourceChain,
		intent.DestinationChain,
		intent.Status)

	return &intent, nil
}

// CreateIntent creates a new intent
func (p *PostgresDB) CreateIntent(ctx context.Context, intent *models.Intent) error {
	query := `
		INSERT INTO intents (
			id, source_chain, destination_chain, token, amount, recipient, intent_fee, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	fmt.Printf("Creating intent - ID: %s, SourceChain: %d, DestinationChain: %d, Status: %s, CreatedAt: %v, UpdatedAt: %v\n",
		intent.ID,
		intent.SourceChain,
		intent.DestinationChain,
		intent.Status,
		intent.CreatedAt,
		intent.UpdatedAt)

	// Ensure created_at and updated_at are set
	if intent.CreatedAt.IsZero() {
		intent.CreatedAt = time.Now()
	}
	if intent.UpdatedAt.IsZero() {
		intent.UpdatedAt = time.Now()
	}

	_, err := p.db.ExecContext(ctx, query,
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
	if err != nil {
		fmt.Printf("Error creating intent: %v\n", err)
		return fmt.Errorf("failed to create intent: %v", err)
	}
	return nil
}

// UpdateIntentStatus updates the status of an intent
func (p *PostgresDB) UpdateIntentStatus(ctx context.Context, id string, status models.IntentStatus) error {
	// Get current intent status
	var currentStatus string
	err := p.db.QueryRowContext(ctx, "SELECT status FROM intents WHERE id = $1", id).Scan(&currentStatus)
	if err != nil {
		log.Printf("Error getting current status for intent %s: %v", id, err)
		return fmt.Errorf("failed to get current intent status: %v", err)
	}

	// Update the status
	query := `
		UPDATE intents
		SET status = $1,
			updated_at = NOW()
		WHERE id = $2
	`
	result, err := p.db.ExecContext(ctx, query, string(status), id)
	if err != nil {
		log.Printf("Error updating status for intent %s: %v", id, err)
		return fmt.Errorf("failed to update intent status: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected for intent %s: %v", id, err)
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		log.Printf("No rows affected when updating status for intent %s", id)
		return fmt.Errorf("intent not found: %s", id)
	}

	log.Printf("Intent status updated - ID: %s, Previous: %s, New: %s", id, currentStatus, status)
	return nil
}

// GetFulfillment retrieves a fulfillment by ID
func (p *PostgresDB) GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error) {
	query := `
		SELECT id, intent_id, tx_hash, status, created_at, updated_at
		FROM fulfillments
		WHERE id = $1
	`

	var fulfillment models.Fulfillment
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&fulfillment.ID,
		&fulfillment.IntentID,
		&fulfillment.TxHash,
		&fulfillment.Status,
		&fulfillment.CreatedAt,
		&fulfillment.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("fulfillment not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get fulfillment: %v", err)
	}
	return &fulfillment, nil
}

// CreateFulfillment creates a new fulfillment
func (p *PostgresDB) CreateFulfillment(ctx context.Context, fulfillment *models.Fulfillment) error {
	query := `
		INSERT INTO fulfillments (
			id, intent_id, tx_hash, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := p.db.ExecContext(ctx, query,
		fulfillment.ID,
		fulfillment.IntentID,
		fulfillment.TxHash,
		fulfillment.Status,
		fulfillment.CreatedAt,
		fulfillment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create fulfillment: %v", err)
	}
	return nil
}

// ListFulfillments retrieves all fulfillments
func (p *PostgresDB) ListFulfillments(ctx context.Context) ([]*models.Fulfillment, error) {
	query := `
		SELECT id, intent_id, tx_hash, status, created_at, updated_at
		FROM fulfillments
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query fulfillments: %v", err)
	}
	defer rows.Close()

	var fulfillments []*models.Fulfillment
	for rows.Next() {
		var f models.Fulfillment
		err := rows.Scan(
			&f.ID,
			&f.IntentID,
			&f.TxHash,
			&f.Status,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fulfillment: %v", err)
		}
		fulfillments = append(fulfillments, &f)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fulfillments: %v", err)
	}
	return fulfillments, nil
}

// GetTotalFulfilledAmount gets the total amount fulfilled for an intent
func (p *PostgresDB) GetTotalFulfilledAmount(ctx context.Context, intentID string) (string, error) {
	// First, check if there's a completed fulfillment for this intent
	query := `
		SELECT COUNT(*)
		FROM fulfillments
		WHERE intent_id = $1 AND status = $2
	`

	var count int
	err := p.db.QueryRowContext(ctx, query, intentID, models.FulfillmentStatusCompleted).Scan(&count)
	if err != nil {
		return "0", fmt.Errorf("failed to check fulfillments: %v", err)
	}

	// If there's at least one completed fulfillment, return the intent amount
	if count > 0 {
		intent, err := p.GetIntent(ctx, intentID)
		if err != nil {
			return "0", fmt.Errorf("failed to get intent: %v", err)
		}
		return intent.Amount, nil
	}

	// Otherwise, return 0
	return "0", nil
}

// ListIntents retrieves all intents
func (p *PostgresDB) ListIntents(ctx context.Context) ([]*models.Intent, error) {
	query := `
		SELECT id, source_chain, destination_chain, token, amount, recipient, intent_fee, status, created_at, updated_at
		FROM intents
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query intents: %v", err)
	}
	defer rows.Close()

	var intents []*models.Intent
	for rows.Next() {
		var intent models.Intent
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
			return nil, fmt.Errorf("failed to scan intent: %v", err)
		}
		intents = append(intents, &intent)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating intents: %v", err)
	}
	return intents, nil
}

// GetLastProcessedBlock gets the last processed block number for a chain
func (p *PostgresDB) GetLastProcessedBlock(ctx context.Context, chainID uint64) (uint64, error) {
	query := `
		SELECT block_number
		FROM last_processed_blocks
		WHERE chain_id = $1
	`

	var blockNumber uint64
	err := p.db.QueryRowContext(ctx, query, chainID).Scan(&blockNumber)
	if err == sql.ErrNoRows {
		// If no record exists, create one with a default value of 0
		err = p.UpdateLastProcessedBlock(ctx, chainID, 0)
		if err != nil {
			return 0, fmt.Errorf("failed to create default last processed block: %v", err)
		}
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get last processed block: %v", err)
	}
	return blockNumber, nil
}

// UpdateLastProcessedBlock updates the last processed block number for a chain
func (p *PostgresDB) UpdateLastProcessedBlock(ctx context.Context, chainID uint64, blockNumber uint64) error {
	query := `
		INSERT INTO last_processed_blocks (chain_id, block_number, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (chain_id) DO UPDATE
		SET block_number = $2,
			updated_at = NOW()
		WHERE last_processed_blocks.block_number < $2
	`

	_, err := p.db.ExecContext(ctx, query, chainID, blockNumber)
	if err != nil {
		return fmt.Errorf("failed to update last processed block: %v", err)
	}
	return nil
}

// InitDB initializes the database schema
func (p *PostgresDB) InitDB(ctx context.Context) error {
	// Read schema file
	schema := `
		-- Create intents table
		CREATE TABLE IF NOT EXISTS intents (
			id VARCHAR(66) PRIMARY KEY,
			source_chain BIGINT NOT NULL,
			destination_chain BIGINT NOT NULL,
			token VARCHAR(42) NOT NULL,
			amount VARCHAR(78) NOT NULL,
			recipient VARCHAR(42) NOT NULL,
			intent_fee VARCHAR(78) NOT NULL,
			status VARCHAR(20) NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);

		-- Create fulfillments table
		CREATE TABLE IF NOT EXISTS fulfillments (
			id VARCHAR(66) PRIMARY KEY,
			intent_id VARCHAR(66) NOT NULL REFERENCES intents(id),
			tx_hash VARCHAR(66) NOT NULL,
			status VARCHAR(20) NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);

		-- Table to store last processed block numbers
		CREATE TABLE IF NOT EXISTS last_processed_blocks (
			chain_id BIGINT PRIMARY KEY,
			block_number BIGINT NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		-- Create indexes
		CREATE INDEX IF NOT EXISTS idx_intents_status ON intents(status);
		CREATE INDEX IF NOT EXISTS idx_fulfillments_intent_id ON fulfillments(intent_id);
		CREATE INDEX IF NOT EXISTS idx_fulfillments_status ON fulfillments(status);
	`

	// Execute schema
	_, err := p.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to initialize database schema: %v", err)
	}

	return nil
}
