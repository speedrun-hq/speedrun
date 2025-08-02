package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/speedrun-hq/speedrun/api/models"
)

// Database interface defines the methods that a database implementation must provide
type Database interface {
	// Database connection management
	Close() error
	Ping() error
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	// Prepared statements
	PrepareStatements(ctx context.Context) error

	// Intent operations
	CreateIntent(ctx context.Context, intent *models.Intent) error
	GetIntent(ctx context.Context, id string) (*models.Intent, error)
	ListIntents(ctx context.Context) ([]*models.Intent, error)
	ListIntentsPaginated(ctx context.Context, page, pageSize int, status string) ([]*models.Intent, int, error)
	ListIntentsBySender(ctx context.Context, sender string) ([]*models.Intent, error)
	ListIntentsBySenderPaginated(ctx context.Context, sender string, page, pageSize int) ([]*models.Intent, int, error)
	ListIntentsByRecipient(ctx context.Context, recipient string) ([]*models.Intent, error)
	ListIntentsByRecipientPaginated(
		ctx context.Context,
		recipient string,
		page, pageSize int,
	) ([]*models.Intent, int, error)
	UpdateIntentStatus(ctx context.Context, id string, status models.IntentStatus) error

	// Optimized intent operations
	ListIntentsPaginatedOptimized(ctx context.Context, page, pageSize int, status string) ([]*models.Intent, int, error)
	ListIntentsBySenderPaginatedOptimized(
		ctx context.Context,
		sender string,
		page, pageSize int,
	) ([]*models.Intent, int, error)
	ListIntentsByRecipientPaginatedOptimized(
		ctx context.Context,
		recipient string,
		page, pageSize int,
	) ([]*models.Intent, int, error)

	// Keyset pagination
	ListIntentsKeysetPaginated(
		ctx context.Context,
		lastTimestamp time.Time,
		lastID string,
		pageSize int,
		status string,
	) ([]*models.Intent, bool, error)

	// Fulfillment operations
	CreateFulfillment(ctx context.Context, fulfillment *models.Fulfillment) error
	GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error)
	ListFulfillments(ctx context.Context) ([]*models.Fulfillment, error)
	ListFulfillmentsPaginated(ctx context.Context, page, pageSize int) ([]*models.Fulfillment, int, error)
	ListFulfillmentsPaginatedOptimized(ctx context.Context, page, pageSize int) ([]*models.Fulfillment, int, error)
	GetTotalFulfilledAmount(ctx context.Context, intentID string) (string, error)

	// Settlement operations
	CreateSettlement(ctx context.Context, settlement *models.Settlement) error
	GetSettlement(ctx context.Context, id string) (*models.Settlement, error)
	ListSettlements(ctx context.Context) ([]*models.Settlement, error)
	ListSettlementsPaginated(ctx context.Context, page, pageSize int) ([]*models.Settlement, int, error)
	ListSettlementsPaginatedOptimized(ctx context.Context, page, pageSize int) ([]*models.Settlement, int, error)

	// Block tracking operations
	GetLastProcessedBlock(ctx context.Context, chainID uint64) (uint64, error)
	UpdateLastProcessedBlock(ctx context.Context, chainID uint64, blockNumber uint64) error

	// Database initialization
	InitDB(ctx context.Context) error
}
