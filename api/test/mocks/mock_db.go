package mocks

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/zeta-chain/zetafast/api/models"
)

// MockDB implements the Database interface for testing
type MockDB struct {
	mu           sync.RWMutex
	intents      map[string]*models.Intent
	fulfillments map[string]*models.Fulfillment
}

// NewMockDB creates a new MockDB instance
func NewMockDB() *MockDB {
	return &MockDB{
		intents:      make(map[string]*models.Intent),
		fulfillments: make(map[string]*models.Fulfillment),
	}
}

// Close implements the Database interface
func (db *MockDB) Close() error {
	return nil
}

// Ping implements the Database interface
func (db *MockDB) Ping() error {
	return nil
}

// Exec implements the Database interface
func (db *MockDB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

// QueryRow implements the Database interface
func (db *MockDB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

// Query implements the Database interface
func (db *MockDB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

// CreateIntent implements the Database interface
func (db *MockDB) CreateIntent(ctx context.Context, intent *models.Intent) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.intents[intent.ID] = intent
	return nil
}

// GetIntent implements the Database interface
func (db *MockDB) GetIntent(ctx context.Context, id string) (*models.Intent, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if intent, ok := db.intents[id]; ok {
		return intent, nil
	}
	return nil, errors.New("intent not found")
}

// ListIntents implements the Database interface
func (db *MockDB) ListIntents(ctx context.Context) ([]*models.Intent, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	intents := make([]*models.Intent, 0, len(db.intents))
	for _, intent := range db.intents {
		intents = append(intents, intent)
	}
	return intents, nil
}

// CreateFulfillment implements the Database interface
func (db *MockDB) CreateFulfillment(ctx context.Context, fulfillment *models.Fulfillment) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Store the fulfillment
	db.fulfillments[fulfillment.ID] = fulfillment

	// Check if intent exists
	if _, ok := db.intents[fulfillment.IntentID]; !ok {
		return errors.New("intent not found")
	}

	return nil
}

// GetFulfillment implements the Database interface
func (db *MockDB) GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if fulfillment, ok := db.fulfillments[id]; ok {
		return fulfillment, nil
	}
	return nil, errors.New("fulfillment not found")
}

// GetTotalFulfilledAmount implements the Database interface
func (db *MockDB) GetTotalFulfilledAmount(ctx context.Context, intentID string) (string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Get the intent
	intent, ok := db.intents[intentID]
	if !ok {
		return "0", errors.New("intent not found")
	}

	// Check if there's a completed fulfillment
	for _, f := range db.fulfillments {
		if f.IntentID == intentID && f.Status == models.FulfillmentStatusCompleted {
			return intent.Amount, nil
		}
	}

	return "0", nil
}

// UpdateIntentStatus implements the Database interface
func (db *MockDB) UpdateIntentStatus(ctx context.Context, id string, status models.IntentStatus) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if intent, ok := db.intents[id]; ok {
		intent.Status = status
		return nil
	}

	return errors.New("intent not found")
}

// ListFulfillments implements the Database interface
func (m *MockDB) ListFulfillments(ctx context.Context) ([]*models.Fulfillment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fulfillments := make([]*models.Fulfillment, 0, len(m.fulfillments))
	for _, fulfillment := range m.fulfillments {
		fulfillments = append(fulfillments, fulfillment)
	}

	return fulfillments, nil
}

// GetLastProcessedBlock implements DBInterface
func (db *MockDB) GetLastProcessedBlock(ctx context.Context, chainID uint64) (uint64, error) {
	// For testing purposes, return 0
	return 0, nil
}

// UpdateLastProcessedBlock implements DBInterface
func (db *MockDB) UpdateLastProcessedBlock(ctx context.Context, chainID uint64, blockNumber uint64) error {
	// For testing purposes, do nothing
	return nil
}
