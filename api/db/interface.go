package db

import (
	"context"
	"database/sql"

	"github.com/zeta-chain/zetafast/api/models"
)

// Database interface defines the methods that a database implementation must provide
type Database interface {
	// Database connection management
	Close() error
	Ping() error
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

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

	// Settlement operations
	CreateSettlement(ctx context.Context, settlement *models.Settlement) error
	GetSettlement(ctx context.Context, id string) (*models.Settlement, error)
	ListSettlements(ctx context.Context) ([]*models.Settlement, error)

	// Block tracking operations
	GetLastProcessedBlock(ctx context.Context, chainID uint64) (uint64, error)
	UpdateLastProcessedBlock(ctx context.Context, chainID uint64, blockNumber uint64) error

	// Database initialization
	InitDB(ctx context.Context) error
}
