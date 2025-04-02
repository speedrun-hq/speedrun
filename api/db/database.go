package db

import (
	"context"
	"database/sql"

	"github.com/zeta-chain/zetafast/api/models"
)

// DBInterface defines the methods for database operations
type DBInterface interface {
	// Intent operations
	CreateIntent(ctx context.Context, intent *models.Intent) error
	GetIntent(ctx context.Context, id string) (*models.Intent, error)
	ListIntents(ctx context.Context) ([]*models.Intent, error)
	UpdateIntentStatus(ctx context.Context, id string, status models.IntentStatus) error

	// Fulfillment operations
	CreateFulfillment(ctx context.Context, fulfillment *models.Fulfillment) error
	GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error)
	ListFulfillments(ctx context.Context) ([]*models.Fulfillment, error)
	GetTotalFulfilledAmount(ctx context.Context, intentID string) (string, error)

	// Block tracking operations
	GetLastProcessedBlock(ctx context.Context, chainID uint64) (uint64, error)
	UpdateLastProcessedBlock(ctx context.Context, chainID uint64, blockNumber uint64) error

	// Database connection management
	Close() error
}

// PostgresDatabase implements the DBInterface using PostgreSQL
type PostgresDatabase struct {
	db *sql.DB
}

// NewPostgresDatabase creates a new PostgreSQL database connection
func NewPostgresDatabase(dsn string) (*PostgresDatabase, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return &PostgresDatabase{db: db}, nil
}

// Close closes the database connection
func (d *PostgresDatabase) Close() error {
	return d.db.Close()
}

// GetLastProcessedBlock gets the last processed block number for a chain
func (d *PostgresDatabase) GetLastProcessedBlock(ctx context.Context, chainID uint64) (uint64, error) {
	var blockNumber uint64
	err := d.db.QueryRowContext(ctx, "SELECT block_number FROM last_processed_blocks WHERE chain_id = $1", chainID).Scan(&blockNumber)
	if err == sql.ErrNoRows {
		return 0, nil // Return 0 if no record exists
	}
	return blockNumber, err
}

// UpdateLastProcessedBlock updates the last processed block number for a chain
func (d *PostgresDatabase) UpdateLastProcessedBlock(ctx context.Context, chainID uint64, blockNumber uint64) error {
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO last_processed_blocks (chain_id, block_number)
		VALUES ($1, $2)
		ON CONFLICT (chain_id) DO UPDATE
		SET block_number = $2, updated_at = CURRENT_TIMESTAMP
	`, chainID, blockNumber)
	return err
}

// CreateIntent creates a new intent in the database
func (d *PostgresDatabase) CreateIntent(ctx context.Context, intent *models.Intent) error {
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO intents (id, source_chain, destination_chain, token, amount, recipient, intent_fee, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, intent.ID, intent.SourceChain, intent.DestinationChain, intent.Token, intent.Amount, intent.Recipient, intent.IntentFee, intent.Status, intent.CreatedAt, intent.UpdatedAt)
	return err
}

// GetIntent retrieves an intent by ID
func (d *PostgresDatabase) GetIntent(ctx context.Context, id string) (*models.Intent, error) {
	intent := &models.Intent{}
	err := d.db.QueryRowContext(ctx, `
		SELECT id, source_chain, destination_chain, token, amount, recipient, intent_fee, status
		FROM intents WHERE id = $1
	`, id).Scan(&intent.ID, &intent.SourceChain, &intent.DestinationChain, &intent.Token, &intent.Amount, &intent.Recipient, &intent.IntentFee, &intent.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return intent, err
}

// ListIntents retrieves all intents
func (d *PostgresDatabase) ListIntents(ctx context.Context) ([]*models.Intent, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, source_chain, destination_chain, token, amount, recipient, intent_fee, status
		FROM intents
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var intents []*models.Intent
	for rows.Next() {
		intent := &models.Intent{}
		if err := rows.Scan(&intent.ID, &intent.SourceChain, &intent.DestinationChain, &intent.Token, &intent.Amount, &intent.Recipient, &intent.IntentFee, &intent.Status); err != nil {
			return nil, err
		}
		intents = append(intents, intent)
	}
	return intents, nil
}

// UpdateIntentStatus updates the status of an intent
func (d *PostgresDatabase) UpdateIntentStatus(ctx context.Context, id string, status models.IntentStatus) error {
	_, err := d.db.ExecContext(ctx, `
		UPDATE intents SET status = $1 WHERE id = $2
	`, status, id)
	return err
}

// CreateFulfillment creates a new fulfillment in the database
func (d *PostgresDatabase) CreateFulfillment(ctx context.Context, fulfillment *models.Fulfillment) error {
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO fulfillments (id, intent_id, tx_hash, status)
		VALUES ($1, $2, $3, $4)
	`, fulfillment.ID, fulfillment.IntentID, fulfillment.TxHash, fulfillment.Status)
	return err
}

// GetFulfillment retrieves a fulfillment by ID
func (d *PostgresDatabase) GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error) {
	fulfillment := &models.Fulfillment{}
	err := d.db.QueryRowContext(ctx, `
		SELECT id, intent_id, tx_hash, status
		FROM fulfillments WHERE id = $1
	`, id).Scan(&fulfillment.ID, &fulfillment.IntentID, &fulfillment.TxHash, &fulfillment.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return fulfillment, err
}

// ListFulfillments retrieves all fulfillments
func (d *PostgresDatabase) ListFulfillments(ctx context.Context) ([]*models.Fulfillment, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, intent_id, tx_hash, status
		FROM fulfillments
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fulfillments []*models.Fulfillment
	for rows.Next() {
		fulfillment := &models.Fulfillment{}
		if err := rows.Scan(&fulfillment.ID, &fulfillment.IntentID, &fulfillment.TxHash, &fulfillment.Status); err != nil {
			return nil, err
		}
		fulfillments = append(fulfillments, fulfillment)
	}
	return fulfillments, nil
}

// GetTotalFulfilledAmount gets the total amount fulfilled for an intent
func (d *PostgresDatabase) GetTotalFulfilledAmount(ctx context.Context, intentID string) (string, error) {
	var amount string
	err := d.db.QueryRowContext(ctx, `
		SELECT i.amount
		FROM intents i
		JOIN fulfillments f ON i.id = f.intent_id
		WHERE i.id = $1 AND f.status = $2
		LIMIT 1
	`, intentID, models.FulfillmentStatusCompleted).Scan(&amount)
	if err == sql.ErrNoRows {
		return "0", nil
	}
	return amount, err
}
