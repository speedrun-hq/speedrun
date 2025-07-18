package services

import (
	"testing"
	"time"

	"github.com/speedrun-hq/speedrun/api/logger"
	"github.com/stretchr/testify/assert"
)

func TestMetricsService_GoroutineTracking(t *testing.T) {
	// Create logger
	logger := logger.NewStdLogger(false, logger.InfoLevel)

	// Create metrics service
	metricsService := NewMetricsService(logger)

	// Create mock services
	mockDB := &mockDB{}

	// Create intent service
	intentService, err := NewIntentService(
		nil, // client will be nil for this test
		nil, // clientResolver will be nil for this test
		mockDB,
		`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"targetChain","type":"uint256"},{"indexed":false,"internalType":"bytes","name":"receiver","type":"bytes"},{"indexed":false,"internalType":"uint256","name":"tip","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"salt","type":"uint256"}],"name":"IntentInitiated","type":"event"}]`,
		1, // chainID
		logger,
	)
	assert.NoError(t, err)

	// Create fulfillment service
	fulfillmentService, err := NewFulfillmentService(
		nil, // client will be nil for this test
		nil, // clientResolver will be nil for this test
		mockDB,
		`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"address","name":"receiver","type":"address"}],"name":"IntentFulfilled","type":"event"}]`,
		1, // chainID
		logger,
	)
	assert.NoError(t, err)

	// Create settlement service
	settlementService, err := NewSettlementService(
		nil, // client will be nil for this test
		nil, // clientResolver will be nil for this test
		mockDB,
		`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"bool","name":"fulfilled","type":"bool"}],"name":"IntentSettled","type":"event"}]`,
		1, // chainID
		logger,
	)
	assert.NoError(t, err)

	// Register all services
	metricsService.RegisterIntentService(1, intentService)
	metricsService.RegisterFulfillmentService(1, fulfillmentService)
	metricsService.RegisterSettlementService(1, settlementService)

	// Initially, all services should have 0 goroutines
	assert.Equal(t, int32(0), intentService.ActiveGoroutines())
	assert.Equal(t, int32(0), fulfillmentService.ActiveGoroutines())
	assert.Equal(t, int32(0), settlementService.ActiveGoroutines())

	// Start some test goroutines in each service
	intentService.startGoroutine("test-intent", func() {
		time.Sleep(100 * time.Millisecond)
	})

	fulfillmentService.startGoroutine("test-fulfillment", func() {
		time.Sleep(100 * time.Millisecond)
	})

	settlementService.startGoroutine("test-settlement", func() {
		time.Sleep(100 * time.Millisecond)
	})

	// Wait a bit for goroutines to start
	time.Sleep(50 * time.Millisecond)

	// Check that each service has 1 goroutine
	assert.Equal(t, int32(1), intentService.ActiveGoroutines())
	assert.Equal(t, int32(1), fulfillmentService.ActiveGoroutines())
	assert.Equal(t, int32(1), settlementService.ActiveGoroutines())

	// Update metrics
	metricsService.UpdateMetrics()

	// Get the metrics summary to verify total goroutines
	summary := metricsService.GetMetricsSummary()
	chains := summary["chains"].(map[string]interface{})
	ethereum := chains["ethereum"].(map[string]interface{})
	activeGoroutines := ethereum["active_goroutines"].(int32)

	// Total should be 3 (1 from each service)
	assert.Equal(t, int32(3), activeGoroutines)

	// Wait for goroutines to complete
	time.Sleep(100 * time.Millisecond)

	// Shutdown all services
	err = intentService.Shutdown(5 * time.Second)
	assert.NoError(t, err)

	err = fulfillmentService.Shutdown(5 * time.Second)
	assert.NoError(t, err)

	err = settlementService.Shutdown(5 * time.Second)
	assert.NoError(t, err)

	// All services should have 0 goroutines after shutdown
	assert.Equal(t, int32(0), intentService.ActiveGoroutines())
	assert.Equal(t, int32(0), fulfillmentService.ActiveGoroutines())
	assert.Equal(t, int32(0), settlementService.ActiveGoroutines())
}
