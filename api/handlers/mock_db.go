package handlers

import (
	"context"
	"database/sql"

	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/stretchr/testify/mock"
)

// MockDatabase is a mock implementation of the Database interface
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDatabase) Ping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDatabase) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0).(sql.Result), callArgs.Error(1)
}

func (m *MockDatabase) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0).(*sql.Row)
}

func (m *MockDatabase) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0).(*sql.Rows), callArgs.Error(1)
}

func (m *MockDatabase) CreateIntent(ctx context.Context, intent *models.Intent) error {
	args := m.Called(ctx, intent)
	return args.Error(0)
}

func (m *MockDatabase) GetIntent(ctx context.Context, id string) (*models.Intent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Intent), args.Error(1)
}

func (m *MockDatabase) ListIntents(ctx context.Context) ([]*models.Intent, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func (m *MockDatabase) UpdateIntentStatus(ctx context.Context, id string, status models.IntentStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockDatabase) CreateFulfillment(ctx context.Context, fulfillment *models.Fulfillment) error {
	args := m.Called(ctx, fulfillment)
	return args.Error(0)
}

func (m *MockDatabase) ListFulfillments(ctx context.Context) ([]*models.Fulfillment, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Fulfillment), args.Error(1)
}

func (m *MockDatabase) GetTotalFulfilledAmount(ctx context.Context, intentID string) (string, error) {
	args := m.Called(ctx, intentID)
	return args.String(0), args.Error(1)
}

func (m *MockDatabase) CreateSettlement(ctx context.Context, settlement *models.Settlement) error {
	args := m.Called(ctx, settlement)
	return args.Error(0)
}

func (m *MockDatabase) GetSettlement(ctx context.Context, id string) (*models.Settlement, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Settlement), args.Error(1)
}

func (m *MockDatabase) ListSettlements(ctx context.Context) ([]*models.Settlement, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Settlement), args.Error(1)
}

func (m *MockDatabase) GetLastProcessedBlock(ctx context.Context, chainID uint64) (uint64, error) {
	args := m.Called(ctx, chainID)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockDatabase) UpdateLastProcessedBlock(ctx context.Context, chainID uint64, blockNumber uint64) error {
	args := m.Called(ctx, chainID, blockNumber)
	return args.Error(0)
}

func (m *MockDatabase) InitDB(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDatabase) GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Fulfillment), args.Error(1)
}

func (m *MockDatabase) ListIntentsByRecipient(ctx context.Context, recipientAddress string) ([]*models.Intent, error) {
	args := m.Called(ctx, recipientAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func (m *MockDatabase) ListIntentsBySender(ctx context.Context, senderAddress string) ([]*models.Intent, error) {
	args := m.Called(ctx, senderAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

// Add pagination methods
func (m *MockDatabase) ListIntentsPaginated(ctx context.Context, page, pageSize int) ([]*models.Intent, int, error) {
	args := m.Called(ctx, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}

func (m *MockDatabase) ListFulfillmentsPaginated(ctx context.Context, page, pageSize int) ([]*models.Fulfillment, int, error) {
	args := m.Called(ctx, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Fulfillment), args.Int(1), args.Error(2)
}

func (m *MockDatabase) ListSettlementsPaginated(ctx context.Context, page, pageSize int) ([]*models.Settlement, int, error) {
	args := m.Called(ctx, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Settlement), args.Int(1), args.Error(2)
}

func (m *MockDatabase) ListIntentsBySenderPaginated(ctx context.Context, senderAddress string, page, pageSize int) ([]*models.Intent, int, error) {
	args := m.Called(ctx, senderAddress, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}

func (m *MockDatabase) ListIntentsByRecipientPaginated(ctx context.Context, recipientAddress string, page, pageSize int) ([]*models.Intent, int, error) {
	args := m.Called(ctx, recipientAddress, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Intent), args.Int(1), args.Error(2)
}
