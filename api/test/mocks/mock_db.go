package mocks

import (
	"context"
	"database/sql"
	"errors"
	"math/big"
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

	// Get the intent
	intent, ok := db.intents[fulfillment.IntentID]
	if !ok {
		return errors.New("intent not found")
	}

	// Calculate total fulfilled amount
	totalFulfilled := "0"
	for _, f := range db.fulfillments {
		if f.IntentID == fulfillment.IntentID {
			// Add the amount to total
			totalFulfilled = addBigIntStrings(totalFulfilled, f.Amount)
		}
	}

	// Update intent status only if total fulfilled amount equals intent amount
	if totalFulfilled == intent.Amount {
		intent.Status = models.IntentStatusFulfilled
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

	// Calculate total fulfilled amount
	totalFulfilled := "0"
	for _, f := range db.fulfillments {
		if f.IntentID == intentID {
			// Add the amount to total
			totalFulfilled = addBigIntStrings(totalFulfilled, f.Amount)
		}
	}

	return totalFulfilled, nil
}

// addBigIntStrings adds two big integer strings and returns the result as a string
func addBigIntStrings(a, b string) string {
	bigA := new(big.Int)
	bigA.SetString(a, 10)
	bigB := new(big.Int)
	bigB.SetString(b, 10)
	result := new(big.Int).Add(bigA, bigB)
	return result.String()
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
