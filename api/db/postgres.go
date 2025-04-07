package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/speedrun-hq/speedrun/api/models"
)

//go:embed schema.sql
var schemaFS embed.FS

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

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

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

// InitDB initializes the database schema
func (p *PostgresDB) InitDB(ctx context.Context) error {
	// Read schema file from embedded filesystem
	schemaBytes, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read embedded schema file: %v", err)
	}

	// Execute schema
	_, err = p.db.ExecContext(ctx, string(schemaBytes))
	if err != nil {
		return fmt.Errorf("failed to initialize database schema: %v", err)
	}

	return nil
}

// GetIntent retrieves an intent by ID
func (p *PostgresDB) GetIntent(ctx context.Context, id string) (*models.Intent, error) {
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
		return nil, fmt.Errorf("intent not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get intent: %v", err)
	}

	return &intent, nil
}

// CreateIntent creates a new intent
func (p *PostgresDB) CreateIntent(ctx context.Context, intent *models.Intent) error {
	query := `
		INSERT INTO intents (
			id, source_chain, destination_chain, token, amount, recipient, intent_fee, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

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
		return fmt.Errorf("failed to create intent: %v", err)
	}
	return nil
}

// UpdateIntentStatus updates the status of an intent
func (p *PostgresDB) UpdateIntentStatus(ctx context.Context, id string, status models.IntentStatus) error {
	query := `
		UPDATE intents
		SET status = $1,
			updated_at = NOW()
		WHERE id = $2
	`
	result, err := p.db.ExecContext(ctx, query, string(status), id)
	if err != nil {
		return fmt.Errorf("failed to update intent status: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("intent not found: %s", id)
	}

	return nil
}

// GetFulfillment retrieves a fulfillment by ID
func (p *PostgresDB) GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error) {
	query := `
		SELECT id, asset, amount, receiver, tx_hash, created_at, updated_at
		FROM fulfillments
		WHERE id = $1
	`

	var fulfillment models.Fulfillment
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&fulfillment.ID,
		&fulfillment.Asset,
		&fulfillment.Amount,
		&fulfillment.Receiver,
		&fulfillment.TxHash,
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
			id, asset, amount, receiver, tx_hash, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	// Ensure timestamps are set
	if fulfillment.CreatedAt.IsZero() {
		fulfillment.CreatedAt = time.Now()
	}
	if fulfillment.UpdatedAt.IsZero() {
		fulfillment.UpdatedAt = time.Now()
	}

	_, err := p.db.ExecContext(ctx, query,
		fulfillment.ID,
		fulfillment.Asset,
		fulfillment.Amount,
		fulfillment.Receiver,
		fulfillment.TxHash,
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
		SELECT id, asset, amount, receiver, tx_hash, created_at, updated_at
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
			&f.Asset,
			&f.Amount,
			&f.Receiver,
			&f.TxHash,
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
		WHERE id = $1
	`

	var count int
	err := p.db.QueryRowContext(ctx, query, intentID).Scan(&count)
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

// GetSettlement retrieves a settlement by ID
func (p *PostgresDB) GetSettlement(ctx context.Context, id string) (*models.Settlement, error) {
	query := `
		SELECT id, asset, amount, receiver, fulfilled, fulfiller, actual_amount, paid_tip, tx_hash, created_at, updated_at
		FROM settlements
		WHERE id = $1
	`

	var settlement models.Settlement
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&settlement.ID,
		&settlement.Asset,
		&settlement.Amount,
		&settlement.Receiver,
		&settlement.Fulfilled,
		&settlement.Fulfiller,
		&settlement.ActualAmount,
		&settlement.PaidTip,
		&settlement.TxHash,
		&settlement.CreatedAt,
		&settlement.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("settlement not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get settlement: %v", err)
	}
	return &settlement, nil
}

// ListSettlements retrieves all settlements
func (p *PostgresDB) ListSettlements(ctx context.Context) ([]*models.Settlement, error) {
	query := `
		SELECT id, asset, amount, receiver, fulfilled, fulfiller, actual_amount, paid_tip, tx_hash, created_at, updated_at
		FROM settlements
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query settlements: %v", err)
	}
	defer rows.Close()

	var settlements []*models.Settlement
	for rows.Next() {
		var s models.Settlement
		err := rows.Scan(
			&s.ID,
			&s.Asset,
			&s.Amount,
			&s.Receiver,
			&s.Fulfilled,
			&s.Fulfiller,
			&s.ActualAmount,
			&s.PaidTip,
			&s.TxHash,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan settlement: %v", err)
		}
		settlements = append(settlements, &s)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating settlements: %v", err)
	}
	return settlements, nil
}

// CreateSettlement creates a new settlement
func (p *PostgresDB) CreateSettlement(ctx context.Context, settlement *models.Settlement) error {
	query := `
		INSERT INTO settlements (
			id, asset, amount, receiver, fulfilled, fulfiller, actual_amount, paid_tip, tx_hash, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	// Ensure timestamps are set
	if settlement.CreatedAt.IsZero() {
		settlement.CreatedAt = time.Now()
	}
	if settlement.UpdatedAt.IsZero() {
		settlement.UpdatedAt = time.Now()
	}

	_, err := p.db.ExecContext(ctx, query,
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
	)
	if err != nil {
		return fmt.Errorf("failed to create settlement: %v", err)
	}
	return nil
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
	`

	_, err := p.db.ExecContext(ctx, query, chainID, blockNumber)
	if err != nil {
		return fmt.Errorf("failed to update last processed block: %v", err)
	}
	return nil
}
