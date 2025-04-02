package db

import (
	"context"
	"database/sql"
	"fmt"

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

	return &PostgresDB{db: db}, nil
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
	_, err := p.db.ExecContext(ctx, query, string(status), id)
	return err
}

// GetFulfillment retrieves a fulfillment by ID
func (p *PostgresDB) GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error) {
	query := `
		SELECT id, intent_id, fulfiller, target_chain, amount, status, tx_hash, block_number, created_at, updated_at
		FROM fulfillments
		WHERE id = $1
	`

	var fulfillment models.Fulfillment
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&fulfillment.ID,
		&fulfillment.IntentID,
		&fulfillment.Fulfiller,
		&fulfillment.TargetChain,
		&fulfillment.Amount,
		&fulfillment.Status,
		&fulfillment.TxHash,
		&fulfillment.BlockNumber,
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
			id, intent_id, fulfiller, target_chain, amount, status, tx_hash, block_number, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := p.db.ExecContext(ctx, query,
		fulfillment.ID,
		fulfillment.IntentID,
		fulfillment.Fulfiller,
		fulfillment.TargetChain,
		fulfillment.Amount,
		fulfillment.Status,
		fulfillment.TxHash,
		fulfillment.BlockNumber,
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
		SELECT id, intent_id, fulfiller, target_chain, amount, status, tx_hash, block_number, created_at, updated_at
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
		var fulfillment models.Fulfillment
		err := rows.Scan(
			&fulfillment.ID,
			&fulfillment.IntentID,
			&fulfillment.Fulfiller,
			&fulfillment.TargetChain,
			&fulfillment.Amount,
			&fulfillment.Status,
			&fulfillment.TxHash,
			&fulfillment.BlockNumber,
			&fulfillment.CreatedAt,
			&fulfillment.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fulfillment: %v", err)
		}
		fulfillments = append(fulfillments, &fulfillment)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fulfillments: %v", err)
	}
	return fulfillments, nil
}

// GetTotalFulfilledAmount gets the total amount fulfilled for an intent
func (p *PostgresDB) GetTotalFulfilledAmount(ctx context.Context, intentID string) (string, error) {
	query := `
		SELECT COALESCE(SUM(amount), '0')
		FROM fulfillments
		WHERE intent_id = $1 AND status = $2
	`

	var total string
	err := p.db.QueryRowContext(ctx, query, intentID, models.FulfillmentStatusCompleted).Scan(&total)
	if err != nil {
		return "0", fmt.Errorf("failed to get total fulfilled amount: %v", err)
	}
	return total, nil
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
