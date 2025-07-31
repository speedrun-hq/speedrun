package services

import (
	"context"
	"testing"
	"time"

	"github.com/speedrun-hq/speedrun/api/config"
	"github.com/speedrun-hq/speedrun/api/db"
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
		nil, // metricsService
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
		nil, // metricsService
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

func TestPeriodicCatchupService(t *testing.T) {
	// Create a mock logger
	logger := logger.NewStdLogger(false, logger.InfoLevel)

	// Create a mock database
	mockDB := &db.MockDB{}

	// Create mock services
	intentServices := make(map[uint64]*IntentService)
	fulfillmentServices := make(map[uint64]*FulfillmentService)
	settlementServices := make(map[uint64]*SettlementService)

	// Create the event catchup service
	eventCatchupService := NewEventCatchupService(
		intentServices,
		fulfillmentServices,
		settlementServices,
		mockDB,
		logger,
		nil, // metricsService
	)

	// Create a test configuration with short intervals for testing
	cfg := &config.Config{
		PeriodicCatchupInterval:       1, // 1 minute for testing
		PeriodicCatchupTimeout:        2, // 2 minutes for testing
		PeriodicCatchupLookbackBlocks: 100,
		ChainConfigs:                  make(map[uint64]*config.ChainConfig),
	}

	// Create a context with timeout for the test
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start the periodic catchup service
	eventCatchupService.StartPeriodicCatchup(ctx, cfg)

	// Wait a bit to let the service start
	time.Sleep(1 * time.Second)

	// Verify the service is running by checking if it's not shutdown
	if eventCatchupService.IsShutdown() {
		t.Error("Periodic catchup service should not be shutdown")
	}

	// Test shutdown with a longer timeout to account for the periodic catchup goroutine
	shutdownTimeout := 10 * time.Second
	err := eventCatchupService.Shutdown(shutdownTimeout)
	if err != nil {
		t.Errorf("Failed to shutdown periodic catchup service: %v", err)
	}

	// Verify the service is now shutdown
	if !eventCatchupService.IsShutdown() {
		t.Error("Periodic catchup service should be shutdown after Shutdown() call")
	}
}
