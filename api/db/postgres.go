package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"time"

	//nolint:revive // uses PG init() internally
	_ "github.com/lib/pq"
	"github.com/speedrun-hq/speedrun/api/models"
)

//go:embed schema.sql
var schemaFS embed.FS

// PostgresDB implements the Database interface using PostgreSQL
type PostgresDB struct {
	db *sql.DB

	// Prepared statements
	listIntentsStmt            *sql.Stmt
	listIntentsWithStatusStmt  *sql.Stmt
	listIntentsBySenderStmt    *sql.Stmt
	listIntentsByRecipientStmt *sql.Stmt
	listFulfillmentsStmt       *sql.Stmt
	listSettlementsStmt        *sql.Stmt
	getIntentStmt              *sql.Stmt
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

	// Set optimized connection pool settings
	db.SetMaxOpenConns(50)                  // Increased from 25 to handle more concurrent requests
	db.SetMaxIdleConns(10)                  // Increased from 5 to maintain more idle connections
	db.SetConnMaxLifetime(15 * time.Minute) // Increased from 5 minutes for longer connection reuse
	db.SetConnMaxIdleTime(5 * time.Minute)  // Set idle timeout to clean up unused connections

	postgresDB := &PostgresDB{db: db}

	// Initialize the database schema
	if err := postgresDB.InitDB(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	// Prepare statements for improved performance
	if err := postgresDB.PrepareStatements(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to prepare statements: %v", err)
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
		SELECT id, source_chain, destination_chain, token, amount, recipient, sender, intent_fee, status, created_at, updated_at
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
		&intent.Sender,
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
			id, source_chain, destination_chain, token, amount, recipient, sender, intent_fee, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	// Ensure created_at and updated_at are set
	// For blockchain events, these should already be set to the block timestamp
	// This is a fallback for API-created intents or testing
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
		intent.Sender,
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
	// For blockchain events, these should already be set to the block timestamp
	// This is a fallback for API-created fulfillments or testing
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
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListFulfillments: failed to close: %v", err)
		}
	}()

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
		SELECT id, asset, amount, receiver, fulfilled, fulfiller, actual_amount, paid_tip, tx_hash, is_call, call_data, created_at, updated_at
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
		&settlement.IsCall,
		&settlement.CallData,
		&settlement.CreatedAt,
		&settlement.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
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
		SELECT id, asset, amount, receiver, fulfilled, fulfiller, actual_amount, paid_tip, tx_hash, is_call, call_data, created_at, updated_at
		FROM settlements
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query settlements: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListSettlements: failed to close: %v", err)
		}
	}()

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
			&s.IsCall,
			&s.CallData,
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
			id, asset, amount, receiver, fulfilled, fulfiller, actual_amount, paid_tip, tx_hash, is_call, call_data, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	// Ensure timestamps are set
	// For blockchain events, these should already be set to the block timestamp
	// This is a fallback for API-created settlements or testing
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
		settlement.IsCall,
		settlement.CallData,
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
		SELECT id, source_chain, destination_chain, token, amount, recipient, sender, intent_fee, status, created_at, updated_at
		FROM intents
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query intents: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListIntents: failed to close: %v", err)
		}
	}()

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
			&intent.Sender,
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
	if errors.Is(err, sql.ErrNoRows) {
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

// ListIntentsBySender retrieves all intents for a specific sender address
func (p *PostgresDB) ListIntentsBySender(ctx context.Context, sender string) ([]*models.Intent, error) {
	query := `
		SELECT id, source_chain, destination_chain, token, amount, recipient, sender, intent_fee, status, created_at, updated_at
		FROM intents
		WHERE sender = $1
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, sender)
	if err != nil {
		return nil, fmt.Errorf("failed to query intents for sender %s: %v", sender, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListIntentsBySender: failed to close: %v", err)
		}
	}()

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
			&intent.Sender,
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

// ListIntentsByRecipient retrieves all intents for a specific recipient address
func (p *PostgresDB) ListIntentsByRecipient(ctx context.Context, recipient string) ([]*models.Intent, error) {
	query := `
		SELECT id, source_chain, destination_chain, token, amount, recipient, sender, intent_fee, status, created_at, updated_at
		FROM intents
		WHERE recipient = $1
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, recipient)
	if err != nil {
		return nil, fmt.Errorf("failed to query intents for recipient %s: %v", recipient, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListIntentsByRecipient: failed to close: %v", err)
		}
	}()

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
			&intent.Sender,
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

// ListIntentsPaginated retrieves intents with pagination
func (p *PostgresDB) ListIntentsPaginated(
	ctx context.Context,
	page, pageSize int,
	status string,
) ([]*models.Intent, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	// Get total count first
	countQuery := `SELECT COUNT(*) FROM intents`
	countArgs := []interface{}{}
	if status != "" {
		countQuery += ` WHERE status = $1`
		countArgs = append(countArgs, status)
	}
	var totalCount int
	err := p.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count intents: %v", err)
	}

	// Get paginated results
	query := `
		SELECT id, source_chain, destination_chain, token, amount, recipient, sender, intent_fee, status, created_at, updated_at
		FROM intents
	`
	args := []interface{}{}
	if status != "" {
		query += ` WHERE status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`
	if status != "" {
		query += ` LIMIT $2 OFFSET $3`
		args = append(args, pageSize, offset)
	} else {
		query += ` LIMIT $1 OFFSET $2`
		args = append(args, pageSize, offset)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query intents: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListIntentsPaginated: failed to close: %v", err)
		}
	}()

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
			&intent.Sender,
			&intent.IntentFee,
			&intent.Status,
			&intent.CreatedAt,
			&intent.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan intent: %v", err)
		}
		intents = append(intents, &intent)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating intents: %v", err)
	}

	return intents, totalCount, nil
}

// ListIntentsPaginatedOptimized retrieves intents with pagination using a single query with window functions
func (p *PostgresDB) ListIntentsPaginatedOptimized(
	ctx context.Context,
	page, pageSize int,
	status string,
) ([]*models.Intent, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	var rows *sql.Rows
	var err error

	// Use prepared statements for better performance
	if status == "" {
		// Use the statement without status filter
		rows, err = p.listIntentsStmt.QueryContext(ctx, pageSize, offset)
	} else {
		// Use the statement with status filter
		rows, err = p.listIntentsWithStatusStmt.QueryContext(ctx, status, pageSize, offset)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to query intents: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListIntentsPaginatedOptimized: failed to close: %v", err)
		}
	}()

	var intents []*models.Intent
	var totalCount int

	for rows.Next() {
		var intent models.Intent
		err := rows.Scan(
			&intent.ID,
			&intent.SourceChain,
			&intent.DestinationChain,
			&intent.Token,
			&intent.Amount,
			&intent.Recipient,
			&intent.Sender,
			&intent.IntentFee,
			&intent.Status,
			&intent.CreatedAt,
			&intent.UpdatedAt,
			&totalCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan intent: %v", err)
		}
		intents = append(intents, &intent)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating intents: %v", err)
	}

	return intents, totalCount, nil
}

// ListIntentsBySenderPaginated retrieves intents by sender with pagination
func (p *PostgresDB) ListIntentsBySenderPaginated(
	ctx context.Context,
	sender string,
	page, pageSize int,
) ([]*models.Intent, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	// Get total count first
	countQuery := `SELECT COUNT(*) FROM intents WHERE sender = $1`
	var totalCount int
	err := p.db.QueryRowContext(ctx, countQuery, sender).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count intents: %v", err)
	}

	// Get paginated results
	query := `
		SELECT id, source_chain, destination_chain, token, amount, recipient, sender, intent_fee, status, created_at, updated_at
		FROM intents
		WHERE sender = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := p.db.QueryContext(ctx, query, sender, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query intents: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListIntentsBySenderPaginated: failed to close: %v", err)
		}
	}()

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
			&intent.Sender,
			&intent.IntentFee,
			&intent.Status,
			&intent.CreatedAt,
			&intent.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan intent: %v", err)
		}
		intents = append(intents, &intent)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating intents: %v", err)
	}

	return intents, totalCount, nil
}

// ListIntentsBySenderPaginatedOptimized retrieves intents by sender with pagination using a single query
func (p *PostgresDB) ListIntentsBySenderPaginatedOptimized(
	ctx context.Context,
	sender string,
	page, pageSize int,
) ([]*models.Intent, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	// Use prepared statement
	rows, err := p.listIntentsBySenderStmt.QueryContext(ctx, sender, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query intents: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListIntentsBySenderPaginatedOptimized: failed to close: %v", err)
		}
	}()

	var intents []*models.Intent
	var totalCount int

	for rows.Next() {
		var intent models.Intent
		err := rows.Scan(
			&intent.ID,
			&intent.SourceChain,
			&intent.DestinationChain,
			&intent.Token,
			&intent.Amount,
			&intent.Recipient,
			&intent.Sender,
			&intent.IntentFee,
			&intent.Status,
			&intent.CreatedAt,
			&intent.UpdatedAt,
			&totalCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan intent: %v", err)
		}
		intents = append(intents, &intent)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating intents: %v", err)
	}

	return intents, totalCount, nil
}

// ListIntentsByRecipientPaginated retrieves intents by recipient with pagination
func (p *PostgresDB) ListIntentsByRecipientPaginated(
	ctx context.Context,
	recipient string,
	page, pageSize int,
) ([]*models.Intent, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	// Get total count first
	countQuery := `SELECT COUNT(*) FROM intents WHERE recipient = $1`
	var totalCount int
	err := p.db.QueryRowContext(ctx, countQuery, recipient).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count intents: %v", err)
	}

	// Get paginated results
	query := `
		SELECT id, source_chain, destination_chain, token, amount, recipient, sender, intent_fee, status, created_at, updated_at
		FROM intents
		WHERE recipient = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := p.db.QueryContext(ctx, query, recipient, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query intents: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListIntentsByRecipientPaginated: failed to close: %v", err)
		}
	}()

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
			&intent.Sender,
			&intent.IntentFee,
			&intent.Status,
			&intent.CreatedAt,
			&intent.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan intent: %v", err)
		}
		intents = append(intents, &intent)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating intents: %v", err)
	}

	return intents, totalCount, nil
}

// ListIntentsByRecipientPaginatedOptimized retrieves intents by recipient with pagination using a single query
func (p *PostgresDB) ListIntentsByRecipientPaginatedOptimized(
	ctx context.Context,
	recipient string,
	page, pageSize int,
) ([]*models.Intent, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	// Use prepared statement
	rows, err := p.listIntentsByRecipientStmt.QueryContext(ctx, recipient, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query intents: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListIntentsByRecipientPaginatedOptimized: failed to close: %v", err)
		}
	}()

	var intents []*models.Intent
	var totalCount int

	for rows.Next() {
		var intent models.Intent
		err := rows.Scan(
			&intent.ID,
			&intent.SourceChain,
			&intent.DestinationChain,
			&intent.Token,
			&intent.Amount,
			&intent.Recipient,
			&intent.Sender,
			&intent.IntentFee,
			&intent.Status,
			&intent.CreatedAt,
			&intent.UpdatedAt,
			&totalCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan intent: %v", err)
		}
		intents = append(intents, &intent)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating intents: %v", err)
	}

	return intents, totalCount, nil
}

// ListFulfillmentsPaginated retrieves fulfillments with pagination
func (p *PostgresDB) ListFulfillmentsPaginated(
	ctx context.Context,
	page, pageSize int,
) ([]*models.Fulfillment, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	// Get total count first
	countQuery := `SELECT COUNT(*) FROM fulfillments`
	var totalCount int
	err := p.db.QueryRowContext(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count fulfillments: %v", err)
	}

	// Get paginated results
	query := `
		SELECT id, asset, amount, receiver, tx_hash, created_at, updated_at
		FROM fulfillments
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := p.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query fulfillments: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListFulfillmentsPaginated: failed to close: %v", err)
		}
	}()

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
			return nil, 0, fmt.Errorf("failed to scan fulfillment: %v", err)
		}
		fulfillments = append(fulfillments, &f)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating fulfillments: %v", err)
	}

	return fulfillments, totalCount, nil
}

// ListSettlementsPaginated retrieves settlements with pagination
func (p *PostgresDB) ListSettlementsPaginated(
	ctx context.Context,
	page, pageSize int,
) ([]*models.Settlement, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	// Get total count first
	countQuery := `SELECT COUNT(*) FROM settlements`
	var totalCount int
	err := p.db.QueryRowContext(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count settlements: %v", err)
	}

	// Get paginated results
	query := `
		SELECT id, asset, amount, receiver, fulfilled, fulfiller, actual_amount, paid_tip, tx_hash, is_call, call_data, created_at, updated_at
		FROM settlements
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := p.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query settlements: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListSettlementsPaginated: failed to close: %v", err)
		}
	}()

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
			&s.IsCall,
			&s.CallData,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan settlement: %v", err)
		}
		settlements = append(settlements, &s)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating settlements: %v", err)
	}

	return settlements, totalCount, nil
}

// ListFulfillmentsPaginatedOptimized retrieves fulfillments with pagination using a single query
func (p *PostgresDB) ListFulfillmentsPaginatedOptimized(
	ctx context.Context,
	page, pageSize int,
) ([]*models.Fulfillment, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	// Use prepared statement
	rows, err := p.listFulfillmentsStmt.QueryContext(ctx, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query fulfillments: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListFulfillmentsPaginatedOptimized: failed to close: %v", err)
		}
	}()

	var fulfillments []*models.Fulfillment
	var totalCount int

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
			&totalCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan fulfillment: %v", err)
		}
		fulfillments = append(fulfillments, &f)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating fulfillments: %v", err)
	}

	return fulfillments, totalCount, nil
}

// ListSettlementsPaginatedOptimized retrieves settlements with pagination using a single query
func (p *PostgresDB) ListSettlementsPaginatedOptimized(
	ctx context.Context,
	page, pageSize int,
) ([]*models.Settlement, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	// Use prepared statement
	rows, err := p.listSettlementsStmt.QueryContext(ctx, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query settlements: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListSettlementsPaginatedOptimized: failed to close: %v", err)
		}
	}()

	var settlements []*models.Settlement
	var totalCount int

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
			&s.IsCall,
			&s.CallData,
			&s.CreatedAt,
			&s.UpdatedAt,
			&totalCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan settlement: %v", err)
		}
		settlements = append(settlements, &s)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating settlements: %v", err)
	}

	return settlements, totalCount, nil
}

// PrepareStatements prepares SQL statements for reuse
func (p *PostgresDB) PrepareStatements(ctx context.Context) error {
	var err error

	// Prepare statement for listing intents with window function for count
	p.listIntentsStmt, err = p.db.PrepareContext(ctx, `
		WITH data AS (
			SELECT id, source_chain, destination_chain, token, amount, recipient, sender, 
				   intent_fee, status, created_at, updated_at,
				   COUNT(*) OVER() AS total_count
			FROM intents
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		)
		SELECT id, source_chain, destination_chain, token, amount, recipient, sender, 
			   intent_fee, status, created_at, updated_at, 
			   total_count 
		FROM data
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare listIntentsStmt: %v", err)
	}

	// Prepare statement for listing intents with status filter
	p.listIntentsWithStatusStmt, err = p.db.PrepareContext(ctx, `
		WITH data AS (
			SELECT id, source_chain, destination_chain, token, amount, recipient, sender, 
				   intent_fee, status, created_at, updated_at,
				   COUNT(*) OVER() AS total_count
			FROM intents
			WHERE status = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		)
		SELECT id, source_chain, destination_chain, token, amount, recipient, sender, 
			   intent_fee, status, created_at, updated_at, 
			   total_count
		FROM data
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare listIntentsWithStatusStmt: %v", err)
	}

	// Prepare statement for listing intents by sender
	p.listIntentsBySenderStmt, err = p.db.PrepareContext(ctx, `
		WITH data AS (
			SELECT id, source_chain, destination_chain, token, amount, recipient, sender, 
				   intent_fee, status, created_at, updated_at,
				   COUNT(*) OVER() AS total_count
			FROM intents
			WHERE sender = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		)
		SELECT id, source_chain, destination_chain, token, amount, recipient, sender, 
			   intent_fee, status, created_at, updated_at, 
			   total_count
		FROM data
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare listIntentsBySenderStmt: %v", err)
	}

	// Prepare statement for listing intents by recipient
	p.listIntentsByRecipientStmt, err = p.db.PrepareContext(ctx, `
		WITH data AS (
			SELECT id, source_chain, destination_chain, token, amount, recipient, sender, 
				   intent_fee, status, created_at, updated_at,
				   COUNT(*) OVER() AS total_count
			FROM intents
			WHERE recipient = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		)
		SELECT id, source_chain, destination_chain, token, amount, recipient, sender, 
			   intent_fee, status, created_at, updated_at, 
			   total_count
		FROM data
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare listIntentsByRecipientStmt: %v", err)
	}

	// Prepare statement for listing fulfillments
	p.listFulfillmentsStmt, err = p.db.PrepareContext(ctx, `
		WITH data AS (
			SELECT id, asset, amount, receiver, tx_hash, created_at, updated_at,
				   COUNT(*) OVER() AS total_count
			FROM fulfillments
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		)
		SELECT id, asset, amount, receiver, tx_hash, created_at, updated_at,
			   total_count
		FROM data
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare listFulfillmentsStmt: %v", err)
	}

	// Prepare statement for listing settlements
	p.listSettlementsStmt, err = p.db.PrepareContext(ctx, `
		WITH data AS (
			SELECT id, asset, amount, receiver, fulfilled, fulfiller, actual_amount, 
				   paid_tip, tx_hash, is_call, call_data, created_at, updated_at,
				   COUNT(*) OVER() AS total_count
			FROM settlements
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		)
		SELECT id, asset, amount, receiver, fulfilled, fulfiller, actual_amount,
			   paid_tip, tx_hash, is_call, call_data, created_at, updated_at,
			   total_count
		FROM data
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare listSettlementsStmt: %v", err)
	}

	// Prepare statement for getting a single intent
	p.getIntentStmt, err = p.db.PrepareContext(ctx, `
		SELECT id, source_chain, destination_chain, token, amount, recipient, sender, 
			   intent_fee, status, created_at, updated_at
		FROM intents
		WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare getIntentStmt: %v", err)
	}

	return nil
}

// ListIntentsKeysetPaginated retrieves intents using keyset pagination (more efficient for large datasets)
func (p *PostgresDB) ListIntentsKeysetPaginated(
	ctx context.Context,
	lastTimestamp time.Time,
	lastID string,
	pageSize int,
	status string,
) ([]*models.Intent, bool, error) {
	whereClause := ""
	var args []interface{}
	argIndex := 1

	// Add status filter if provided
	if status != "" {
		whereClause = "WHERE status = $1"
		args = append(args, status)
		argIndex++
	}

	// Add keyset condition if not first page
	if !lastTimestamp.IsZero() {
		if whereClause == "" {
			whereClause = "WHERE "
		} else {
			whereClause += " AND "
		}
		whereClause += fmt.Sprintf("(created_at, id) < ($%d, $%d)", argIndex, argIndex+1)
		args = append(args, lastTimestamp, lastID)
		argIndex += 2
	}

	// Build the query
	query := fmt.Sprintf(`
		SELECT id, source_chain, destination_chain, token, amount, recipient, sender, 
			   intent_fee, status, created_at, updated_at
		FROM intents
		%s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d
	`, whereClause, argIndex)

	// Request one extra record to determine if there are more pages
	args = append(args, pageSize+1)

	// Execute the query
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("failed to query intents: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("ListIntentsKeysetPaginated: failed to close: %v", err)
		}
	}()

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
			&intent.Sender,
			&intent.IntentFee,
			&intent.Status,
			&intent.CreatedAt,
			&intent.UpdatedAt,
		)
		if err != nil {
			return nil, false, fmt.Errorf("failed to scan intent: %v", err)
		}
		intents = append(intents, &intent)
	}

	if err = rows.Err(); err != nil {
		return nil, false, fmt.Errorf("error iterating intents: %v", err)
	}

	// Determine if there are more pages by checking if we got more records than requested
	hasMore := false
	if len(intents) > pageSize {
		intents = intents[:pageSize] // Remove the extra record
		hasMore = true
	}

	return intents, hasMore, nil
}
