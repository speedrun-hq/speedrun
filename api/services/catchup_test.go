package services

import (
	"testing"
	"time"

	"github.com/speedrun-hq/speedrun/api/logger"
	"github.com/stretchr/testify/assert"
)

func TestEventCatchupService_GoroutineTracking(t *testing.T) {
	// Create logger
	logger := logger.NewStdLogger(false, logger.InfoLevel)

	// Create mock database
	mockDB := &mockDB{}

	// Create empty service maps
	intentServices := make(map[uint64]*IntentService)
	fulfillmentServices := make(map[uint64]*FulfillmentService)
	settlementServices := make(map[uint64]*SettlementService)

	// Create EventCatchupService
	eventCatchupService := NewEventCatchupService(
		intentServices,
		fulfillmentServices,
		settlementServices,
		mockDB,
		logger,
	)

	// Initially should have 0 goroutines
	assert.Equal(t, int32(0), eventCatchupService.ActiveGoroutines())

	// Start some test goroutines
	eventCatchupService.StartGoroutine("test-1", func() {
		time.Sleep(50 * time.Millisecond)
	})
	eventCatchupService.StartGoroutine("test-2", func() {
		time.Sleep(50 * time.Millisecond)
	})

	// Wait a bit for goroutines to start
	time.Sleep(10 * time.Millisecond)

	// Should have 2 goroutines
	assert.Equal(t, int32(2), eventCatchupService.ActiveGoroutines())

	// Wait for goroutines to complete
	time.Sleep(100 * time.Millisecond)

	// Should be back to 0
	assert.Equal(t, int32(0), eventCatchupService.ActiveGoroutines())

	// Test shutdown
	err := eventCatchupService.Shutdown(5 * time.Second)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), eventCatchupService.ActiveGoroutines())
}

func TestEventCatchupService_ShutdownPreventsNewGoroutines(t *testing.T) {
	// Create logger
	logger := logger.NewStdLogger(false, logger.InfoLevel)

	// Create mock database
	mockDB := &mockDB{}

	// Create empty service maps
	intentServices := make(map[uint64]*IntentService)
	fulfillmentServices := make(map[uint64]*FulfillmentService)
	settlementServices := make(map[uint64]*SettlementService)

	// Create EventCatchupService
	eventCatchupService := NewEventCatchupService(
		intentServices,
		fulfillmentServices,
		settlementServices,
		mockDB,
		logger,
	)

	// Shutdown the service
	err := eventCatchupService.Shutdown(5 * time.Second)
	assert.NoError(t, err)

	// Try to start a new goroutine after shutdown
	eventCatchupService.StartGoroutine("test-after-shutdown", func() {
		time.Sleep(10 * time.Millisecond)
	})

	// Should still have 0 goroutines (new ones shouldn't start)
	assert.Equal(t, int32(0), eventCatchupService.ActiveGoroutines())
}
