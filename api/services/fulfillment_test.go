package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/speedrun-hq/speedrun/api/logger"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/stretchr/testify/assert"
)

// Simple mock database for testing
type mockDB struct{}

func (m *mockDB) Close() error { return nil }
func (m *mockDB) Ping() error  { return nil }
func (m *mockDB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

func (m *mockDB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

func (m *mockDB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}
func (m *mockDB) CreateIntent(ctx context.Context, intent *models.Intent) error    { return nil }
func (m *mockDB) GetIntent(ctx context.Context, id string) (*models.Intent, error) { return nil, nil }
func (m *mockDB) ListIntents(ctx context.Context) ([]*models.Intent, error)        { return nil, nil }
func (m *mockDB) UpdateIntentStatus(ctx context.Context, id string, status models.IntentStatus) error {
	return nil
}

func (m *mockDB) CreateFulfillment(ctx context.Context, fulfillment *models.Fulfillment) error {
	return nil
}

func (m *mockDB) ListFulfillments(ctx context.Context) ([]*models.Fulfillment, error) {
	return nil, nil
}

func (m *mockDB) GetTotalFulfilledAmount(ctx context.Context, intentID string) (string, error) {
	return "", nil
}

func (m *mockDB) CreateSettlement(ctx context.Context, settlement *models.Settlement) error {
	return nil
}

func (m *mockDB) GetSettlement(ctx context.Context, id string) (*models.Settlement, error) {
	return nil, nil
}
func (m *mockDB) ListSettlements(ctx context.Context) ([]*models.Settlement, error) { return nil, nil }
func (m *mockDB) GetLastProcessedBlock(ctx context.Context, chainID uint64) (uint64, error) {
	return 0, nil
}

func (m *mockDB) UpdateLastProcessedBlock(ctx context.Context, chainID uint64, blockNumber uint64) error {
	return nil
}
func (m *mockDB) InitDB(ctx context.Context) error { return nil }
func (m *mockDB) GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error) {
	return nil, nil
}

func (m *mockDB) ListIntentsByRecipient(ctx context.Context, recipientAddress string) ([]*models.Intent, error) {
	return nil, nil
}

func (m *mockDB) ListIntentsBySender(ctx context.Context, senderAddress string) ([]*models.Intent, error) {
	return nil, nil
}

func (m *mockDB) ListIntentsPaginated(ctx context.Context, page, pageSize int, status string) ([]*models.Intent, int, error) {
	return nil, 0, nil
}

func (m *mockDB) ListFulfillmentsPaginated(ctx context.Context, page, pageSize int) ([]*models.Fulfillment, int, error) {
	return nil, 0, nil
}

func (m *mockDB) ListSettlementsPaginated(ctx context.Context, page, pageSize int) ([]*models.Settlement, int, error) {
	return nil, 0, nil
}

func (m *mockDB) ListIntentsBySenderPaginated(ctx context.Context, senderAddress string, page, pageSize int) ([]*models.Intent, int, error) {
	return nil, 0, nil
}

func (m *mockDB) ListIntentsByRecipientPaginated(ctx context.Context, recipientAddress string, page, pageSize int) ([]*models.Intent, int, error) {
	return nil, 0, nil
}

func (m *mockDB) ListIntentsPaginatedOptimized(ctx context.Context, page, pageSize int, status string) ([]*models.Intent, int, error) {
	return nil, 0, nil
}

func (m *mockDB) ListIntentsBySenderPaginatedOptimized(ctx context.Context, sender string, page, pageSize int) ([]*models.Intent, int, error) {
	return nil, 0, nil
}

func (m *mockDB) ListIntentsByRecipientPaginatedOptimized(ctx context.Context, recipient string, page, pageSize int) ([]*models.Intent, int, error) {
	return nil, 0, nil
}

func (m *mockDB) ListFulfillmentsPaginatedOptimized(ctx context.Context, page, pageSize int) ([]*models.Fulfillment, int, error) {
	return nil, 0, nil
}

func (m *mockDB) ListSettlementsPaginatedOptimized(ctx context.Context, page, pageSize int) ([]*models.Settlement, int, error) {
	return nil, 0, nil
}

func (m *mockDB) ListIntentsKeysetPaginated(ctx context.Context, lastTimestamp time.Time, lastID string, pageSize int, status string) ([]*models.Intent, bool, error) {
	return nil, false, nil
}
func (m *mockDB) PrepareStatements(ctx context.Context) error { return nil }

func TestFulfillmentService_Shutdown(t *testing.T) {
	// Create mock database
	mockDB := &mockDB{}

	// Create logger
	logger := logger.NewStdLogger(false, logger.InfoLevel)

	// Create service
	service, err := NewFulfillmentService(
		nil, // client will be nil for this test
		nil, // clientResolver will be nil for this test
		mockDB,
		`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"address","name":"receiver","type":"address"}],"name":"IntentFulfilled","type":"event"}]`,
		1, // chainID
		logger,
	)
	assert.NoError(t, err)
	assert.NotNil(t, service)

	// Test that service is not shutdown initially
	assert.False(t, service.IsShutdown())

	// Test shutdown
	err = service.Shutdown(5 * time.Second)
	assert.NoError(t, err)

	// Test that service is now shutdown
	assert.True(t, service.IsShutdown())

	// Test that calling shutdown again returns nil (idempotent)
	err = service.Shutdown(5 * time.Second)
	assert.NoError(t, err)
}

func TestFulfillmentService_StartListening_Shutdown(t *testing.T) {
	// Create mock database
	mockDB := &mockDB{}

	// Create logger
	logger := logger.NewStdLogger(false, logger.InfoLevel)

	// Create service
	service, err := NewFulfillmentService(
		nil, // client will be nil for this test
		nil, // clientResolver will be nil for this test
		mockDB,
		`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"address","name":"receiver","type":"address"}],"name":"IntentFulfilled","type":"event"}]`,
		1, // chainID
		logger,
	)
	assert.NoError(t, err)

	// Shutdown the service first
	err = service.Shutdown(5 * time.Second)
	assert.NoError(t, err)

	// Try to start listening after shutdown - should fail
	err = service.StartListening(context.Background(), common.HexToAddress("0x1234567890123456789012345678901234567890"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service is shutdown")
}

func TestFulfillmentService_GoroutineCleanup(t *testing.T) {
	// Create mock database
	mockDB := &mockDB{}

	// Create logger
	logger := logger.NewStdLogger(false, logger.InfoLevel)

	// Create service
	service, err := NewFulfillmentService(
		nil, // client will be nil for this test
		nil, // clientResolver will be nil for this test
		mockDB,
		`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"address","name":"receiver","type":"address"}],"name":"IntentFulfilled","type":"event"}]`,
		1, // chainID
		logger,
	)
	assert.NoError(t, err)

	// Start a test goroutine
	done := make(chan bool)
	service.startGoroutine("test-goroutine", func() {
		time.Sleep(100 * time.Millisecond)
		done <- true
	})

	// Wait for goroutine to complete
	select {
	case <-done:
		// Goroutine completed successfully
	case <-time.After(1 * time.Second):
		t.Fatal("Goroutine did not complete within timeout")
	}

	// Shutdown should complete quickly since no goroutines are running
	err = service.Shutdown(1 * time.Second)
	assert.NoError(t, err)
}

func TestFulfillmentService_ShutdownTimeout(t *testing.T) {
	// Create mock database
	mockDB := &mockDB{}

	// Create logger
	logger := logger.NewStdLogger(false, logger.InfoLevel)

	// Create service
	service, err := NewFulfillmentService(
		nil, // client will be nil for this test
		nil, // clientResolver will be nil for this test
		mockDB,
		`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"address","name":"receiver","type":"address"}],"name":"IntentFulfilled","type":"event"}]`,
		1, // chainID
		logger,
	)
	assert.NoError(t, err)

	// Start a long-running goroutine
	service.startGoroutine("long-running", func() {
		time.Sleep(2 * time.Second)
	})

	// Try to shutdown with a short timeout - should timeout
	err = service.Shutdown(100 * time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "shutdown timed out")
}
