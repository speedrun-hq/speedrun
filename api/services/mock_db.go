package services

import (
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zeta-chain/zetafast/api/models"
)

// MockDB implements the Database interface for testing
type MockDB struct {
	intents      map[string]*models.Intent
	fulfillments map[string]*models.Fulfillment
	mu           sync.RWMutex
}

// NewMockDB creates a new mock database instance
func NewMockDB() *MockDB {
	return &MockDB{
		intents:      make(map[string]*models.Intent),
		fulfillments: make(map[string]*models.Fulfillment),
	}
}

// Close implements the Database interface
func (m *MockDB) Close() error {
	return nil
}

// Ping implements the Database interface
func (m *MockDB) Ping() error {
	return nil
}

// Exec implements the Database interface
func (m *MockDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

// QueryRow implements the Database interface
func (m *MockDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return nil
}

// Query implements the Database interface
func (m *MockDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

// CreateIntent implements the Database interface
func (m *MockDB) CreateIntent(intent *models.Intent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	intent.ID = uuid.New().String()
	intent.CreatedAt = time.Now()
	intent.UpdatedAt = time.Now()
	intent.Status = "pending"
	intent.Token = "USDC"

	m.intents[intent.ID] = intent
	return nil
}

// GetIntent implements the Database interface
func (m *MockDB) GetIntent(id string) (*models.Intent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	intent, exists := m.intents[id]
	if !exists {
		return nil, errors.New("intent not found")
	}
	return intent, nil
}

// ListIntents implements the Database interface
func (m *MockDB) ListIntents(page, limit int) ([]*models.Intent, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Convert map to slice
	intents := make([]*models.Intent, 0, len(m.intents))
	for _, intent := range m.intents {
		intents = append(intents, intent)
	}

	// Simple pagination
	start := (page - 1) * limit
	end := start + limit
	if start >= len(intents) {
		return []*models.Intent{}, len(intents), nil
	}
	if end > len(intents) {
		end = len(intents)
	}

	return intents[start:end], len(intents), nil
}

// CreateFulfillment implements the Database interface
func (m *MockDB) CreateFulfillment(fulfillment *models.Fulfillment) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fulfillment.ID = uuid.New().String()
	fulfillment.CreatedAt = time.Now()
	fulfillment.UpdatedAt = time.Now()

	// Update intent status
	intent, exists := m.intents[fulfillment.IntentID]
	if !exists {
		return errors.New("intent not found")
	}
	intent.Status = "completed"
	intent.UpdatedAt = time.Now()

	m.fulfillments[fulfillment.ID] = fulfillment
	return nil
}

// GetFulfillment implements the Database interface
func (m *MockDB) GetFulfillment(id string) (*models.Fulfillment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fulfillment, exists := m.fulfillments[id]
	if !exists {
		return nil, errors.New("fulfillment not found")
	}
	return fulfillment, nil
}

// GetTotalFulfilledAmount implements the Database interface
func (m *MockDB) GetTotalFulfilledAmount(intentID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total string = "0"
	for _, fulfillment := range m.fulfillments {
		if fulfillment.IntentID == intentID && fulfillment.Status == models.FulfillmentStatusAccepted {
			// In a real implementation, we would add the amounts as big numbers
			// For mock purposes, we'll just return the last accepted amount
			total = fulfillment.Amount
		}
	}
	return total, nil
}

// UpdateIntentStatus implements the Database interface
func (m *MockDB) UpdateIntentStatus(id, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	intent, exists := m.intents[id]
	if !exists {
		return errors.New("intent not found")
	}

	intent.Status = status
	intent.UpdatedAt = time.Now()
	return nil
}
