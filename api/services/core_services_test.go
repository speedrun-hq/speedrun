package services

import (
	"testing"
	"time"

	"github.com/speedrun-hq/speedrun/api/logging"
	"github.com/stretchr/testify/assert"
)

func TestCoreServices_GoroutineTracking(t *testing.T) {
	// Create logger
	logger := logging.NewTesting(t)

	// Create mock database
	mockDB := &mockDB{}

	// Test IntentService
	t.Run("IntentService", func(t *testing.T) {
		intentService, err := NewIntentService(
			nil, // client will be nil for this test
			nil, // clientResolver will be nil for this test
			mockDB,
			`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"targetChain","type":"uint256"},{"indexed":false,"internalType":"bytes","name":"receiver","type":"bytes"},{"indexed":false,"internalType":"uint256","name":"tip","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"salt","type":"uint256"}],"name":"IntentInitiated","type":"event"}]`,
			1, // chainID
			logger,
		)
		assert.NoError(t, err)

		// Initially should have 0 goroutines
		assert.Equal(t, int32(0), intentService.ActiveGoroutines())

		// Start some test goroutines
		intentService.startGoroutine("test-1", func() {
			time.Sleep(50 * time.Millisecond)
		})
		intentService.startGoroutine("test-2", func() {
			time.Sleep(50 * time.Millisecond)
		})

		// Wait a bit for goroutines to start
		time.Sleep(10 * time.Millisecond)

		// Should have 2 goroutines
		assert.Equal(t, int32(2), intentService.ActiveGoroutines())

		// Wait for goroutines to complete
		time.Sleep(100 * time.Millisecond)

		// Should be back to 0
		assert.Equal(t, int32(0), intentService.ActiveGoroutines())

		// Test shutdown
		err = intentService.Shutdown(5 * time.Second)
		assert.NoError(t, err)
		assert.Equal(t, int32(0), intentService.ActiveGoroutines())
	})

	// Test FulfillmentService
	t.Run("FulfillmentService", func(t *testing.T) {
		fulfillmentService, err := NewFulfillmentService(
			nil, // client will be nil for this test
			nil, // clientResolver will be nil for this test
			mockDB,
			`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"address","name":"receiver","type":"address"}],"name":"IntentFulfilled","type":"event"}]`,
			1, // chainID
			logger,
		)
		assert.NoError(t, err)

		// Initially should have 0 goroutines
		assert.Equal(t, int32(0), fulfillmentService.ActiveGoroutines())

		// Start some test goroutines
		fulfillmentService.startGoroutine("test-1", func() {
			time.Sleep(50 * time.Millisecond)
		})
		fulfillmentService.startGoroutine("test-2", func() {
			time.Sleep(50 * time.Millisecond)
		})

		// Wait a bit for goroutines to start
		time.Sleep(10 * time.Millisecond)

		// Should have 2 goroutines
		assert.Equal(t, int32(2), fulfillmentService.ActiveGoroutines())

		// Wait for goroutines to complete
		time.Sleep(100 * time.Millisecond)

		// Should be back to 0
		assert.Equal(t, int32(0), fulfillmentService.ActiveGoroutines())

		// Test shutdown
		err = fulfillmentService.Shutdown(5 * time.Second)
		assert.NoError(t, err)
		assert.Equal(t, int32(0), fulfillmentService.ActiveGoroutines())
	})

	// Test SettlementService
	t.Run("SettlementService", func(t *testing.T) {
		settlementService, err := NewSettlementService(
			nil, // client will be nil for this test
			nil, // clientResolver will be nil for this test
			mockDB,
			`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"bool","name":"fulfilled","type":"bool"}],"name":"IntentSettled","type":"event"}]`,
			1, // chainID
			logger,
		)
		assert.NoError(t, err)

		// Initially should have 0 goroutines
		assert.Equal(t, int32(0), settlementService.ActiveGoroutines())

		// Start some test goroutines
		settlementService.startGoroutine("test-1", func() {
			time.Sleep(50 * time.Millisecond)
		})
		settlementService.startGoroutine("test-2", func() {
			time.Sleep(50 * time.Millisecond)
		})

		// Wait a bit for goroutines to start
		time.Sleep(10 * time.Millisecond)

		// Should have 2 goroutines
		assert.Equal(t, int32(2), settlementService.ActiveGoroutines())

		// Wait for goroutines to complete
		time.Sleep(100 * time.Millisecond)

		// Should be back to 0
		assert.Equal(t, int32(0), settlementService.ActiveGoroutines())

		// Test shutdown
		err = settlementService.Shutdown(5 * time.Second)
		assert.NoError(t, err)
		assert.Equal(t, int32(0), settlementService.ActiveGoroutines())
	})
}

func TestCoreServices_NoGoroutineLeaks(t *testing.T) {
	// Create logger
	logger := logging.NewTesting(t)

	// Create mock database
	mockDB := &mockDB{}

	// Create all services
	intentService, err := NewIntentService(
		nil, nil, mockDB,
		`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"targetChain","type":"uint256"},{"indexed":false,"internalType":"bytes","name":"receiver","type":"bytes"},{"indexed":false,"internalType":"uint256","name":"tip","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"salt","type":"uint256"}],"name":"IntentInitiated","type":"event"}]`,
		1, logger,
	)
	assert.NoError(t, err)

	fulfillmentService, err := NewFulfillmentService(
		nil, nil, mockDB,
		`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"address","name":"receiver","type":"address"}],"name":"IntentFulfilled","type":"event"}]`,
		1, logger,
	)
	assert.NoError(t, err)

	settlementService, err := NewSettlementService(
		nil, nil, mockDB,
		`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"bool","name":"fulfilled","type":"bool"}],"name":"IntentSettled","type":"event"}]`,
		1, logger,
	)
	assert.NoError(t, err)

	// All services should start with 0 goroutines
	assert.Equal(t, int32(0), intentService.ActiveGoroutines())
	assert.Equal(t, int32(0), fulfillmentService.ActiveGoroutines())
	assert.Equal(t, int32(0), settlementService.ActiveGoroutines())

	// Start multiple goroutines in each service
	for i := 0; i < 5; i++ {
		intentService.startGoroutine("test-intent", func() {
			time.Sleep(20 * time.Millisecond)
		})
		fulfillmentService.startGoroutine("test-fulfillment", func() {
			time.Sleep(20 * time.Millisecond)
		})
		settlementService.startGoroutine("test-settlement", func() {
			time.Sleep(20 * time.Millisecond)
		})
	}

	// Wait a bit for goroutines to start
	time.Sleep(10 * time.Millisecond)

	// Each service should have 5 goroutines
	assert.Equal(t, int32(5), intentService.ActiveGoroutines())
	assert.Equal(t, int32(5), fulfillmentService.ActiveGoroutines())
	assert.Equal(t, int32(5), settlementService.ActiveGoroutines())

	// Wait for all goroutines to complete
	time.Sleep(100 * time.Millisecond)

	// All services should be back to 0 goroutines
	assert.Equal(t, int32(0), intentService.ActiveGoroutines())
	assert.Equal(t, int32(0), fulfillmentService.ActiveGoroutines())
	assert.Equal(t, int32(0), settlementService.ActiveGoroutines())

	// Test shutdown of all services
	err = intentService.Shutdown(5 * time.Second)
	assert.NoError(t, err)
	err = fulfillmentService.Shutdown(5 * time.Second)
	assert.NoError(t, err)
	err = settlementService.Shutdown(5 * time.Second)
	assert.NoError(t, err)

	// All services should still have 0 goroutines after shutdown
	assert.Equal(t, int32(0), intentService.ActiveGoroutines())
	assert.Equal(t, int32(0), fulfillmentService.ActiveGoroutines())
	assert.Equal(t, int32(0), settlementService.ActiveGoroutines())
}
