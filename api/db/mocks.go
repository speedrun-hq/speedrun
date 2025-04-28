package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/stretchr/testify/mock"
)

// MockDB is a mock implementation of the Database interface for testing
type MockDB struct {
	mock.Mock
}

func (m *MockDB) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDB) Ping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	mockArgs := m.Called(ctx, query, args)
	return mockArgs.Get(0).(sql.Result), mockArgs.Error(1)
}

func (m *MockDB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	mockArgs := m.Called(ctx, query, args)
	return mockArgs.Get(0).(*sql.Row)
}

func (m *MockDB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	mockArgs := m.Called(ctx, query, args)
	return mockArgs.Get(0).(*sql.Rows), mockArgs.Error(1)
}

func (m *MockDB) PrepareStatements(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDB) CreateIntent(ctx context.Context, intent *models.Intent) error {
	args := m.Called(ctx, intent)
	return args.Error(0)
}

func (m *MockDB) GetIntent(ctx context.Context, id string) (*models.Intent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Intent), args.Error(1)
}

func (m *MockDB) ListIntents(ctx context.Context) ([]*models.Intent, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func (m *MockDB) ListIntentsPaginated(ctx context.Context, page, pageSize int, status string) ([]*models.Intent, int, error) {
	args := m.Called(ctx, page, pageSize, status)
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}

func (m *MockDB) ListIntentsPaginatedOptimized(ctx context.Context, page, pageSize int, status string) ([]*models.Intent, int, error) {
	args := m.Called(ctx, page, pageSize, status)
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}

func (m *MockDB) ListIntentsBySender(ctx context.Context, sender string) ([]*models.Intent, error) {
	args := m.Called(ctx, sender)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func (m *MockDB) ListIntentsBySenderPaginated(ctx context.Context, sender string, page, pageSize int) ([]*models.Intent, int, error) {
	args := m.Called(ctx, sender, page, pageSize)
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}

func (m *MockDB) ListIntentsBySenderPaginatedOptimized(ctx context.Context, sender string, page, pageSize int) ([]*models.Intent, int, error) {
	args := m.Called(ctx, sender, page, pageSize)
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}

func (m *MockDB) ListIntentsByRecipient(ctx context.Context, recipient string) ([]*models.Intent, error) {
	args := m.Called(ctx, recipient)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func (m *MockDB) ListIntentsByRecipientPaginated(ctx context.Context, recipient string, page, pageSize int) ([]*models.Intent, int, error) {
	args := m.Called(ctx, recipient, page, pageSize)
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}

func (m *MockDB) ListIntentsByRecipientPaginatedOptimized(ctx context.Context, recipient string, page, pageSize int) ([]*models.Intent, int, error) {
	args := m.Called(ctx, recipient, page, pageSize)
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}

func (m *MockDB) ListIntentsKeysetPaginated(ctx context.Context, lastTimestamp time.Time, lastID string, pageSize int, status string) ([]*models.Intent, bool, error) {
	args := m.Called(ctx, lastTimestamp, lastID, pageSize, status)
	return args.Get(0).([]*models.Intent), args.Bool(1), args.Error(2)
}

func (m *MockDB) UpdateIntentStatus(ctx context.Context, id string, status models.IntentStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockDB) CreateFulfillment(ctx context.Context, fulfillment *models.Fulfillment) error {
	args := m.Called(ctx, fulfillment)
	return args.Error(0)
}

func (m *MockDB) GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Fulfillment), args.Error(1)
}

func (m *MockDB) ListFulfillments(ctx context.Context) ([]*models.Fulfillment, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Fulfillment), args.Error(1)
}

func (m *MockDB) ListFulfillmentsPaginated(ctx context.Context, page, pageSize int) ([]*models.Fulfillment, int, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]*models.Fulfillment), args.Int(1), args.Error(2)
}

func (m *MockDB) ListFulfillmentsPaginatedOptimized(ctx context.Context, page, pageSize int) ([]*models.Fulfillment, int, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]*models.Fulfillment), args.Int(1), args.Error(2)
}

func (m *MockDB) GetTotalFulfilledAmount(ctx context.Context, intentID string) (string, error) {
	args := m.Called(ctx, intentID)
	return args.String(0), args.Error(1)
}

func (m *MockDB) CreateSettlement(ctx context.Context, settlement *models.Settlement) error {
	args := m.Called(ctx, settlement)
	return args.Error(0)
}

func (m *MockDB) GetSettlement(ctx context.Context, id string) (*models.Settlement, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Settlement), args.Error(1)
}

func (m *MockDB) ListSettlements(ctx context.Context) ([]*models.Settlement, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Settlement), args.Error(1)
}

func (m *MockDB) ListSettlementsPaginated(ctx context.Context, page, pageSize int) ([]*models.Settlement, int, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]*models.Settlement), args.Int(1), args.Error(2)
}

func (m *MockDB) ListSettlementsPaginatedOptimized(ctx context.Context, page, pageSize int) ([]*models.Settlement, int, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]*models.Settlement), args.Int(1), args.Error(2)
}

func (m *MockDB) GetLastProcessedBlock(ctx context.Context, chainID uint64) (uint64, error) {
	args := m.Called(ctx, chainID)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockDB) UpdateLastProcessedBlock(ctx context.Context, chainID uint64, blockNumber uint64) error {
	args := m.Called(ctx, chainID, blockNumber)
	return args.Error(0)
}

func (m *MockDB) InitDB(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
