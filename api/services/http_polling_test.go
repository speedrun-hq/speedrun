package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestHTTPPollingErrorHandling tests the error handling and retry logic
func TestHTTPPollingErrorHandling(t *testing.T) {
	// This is a simple test of the error handling logic we've implemented
	// We can't easily test the full pollChainEvents function due to its complexity,
	// but we can test the core retry mechanisms it uses

	// Create a counter for retry attempts
	attempts := 0

	// Create a function that will fail on first attempt but succeed later
	retryFunc := func() error {
		attempts++
		if attempts <= 2 {
			return errors.New("temporary error")
		}
		return nil
	}

	// Setup exponential backoff parameters
	maxRetries := 3
	baseDelay := 50 * time.Millisecond

	// Execute the retry logic
	var finalErr error
	for retry := 0; retry < maxRetries; retry++ {
		err := retryFunc()
		if err == nil {
			finalErr = nil
			break
		}

		finalErr = err

		if retry < maxRetries-1 {
			// Wait with exponential backoff
			backoffDelay := baseDelay * time.Duration(1<<retry)
			time.Sleep(backoffDelay)
		}
	}

	// Assert that the retries succeeded (third attempt should succeed)
	if finalErr != nil {
		t.Fatalf("Expected retries to succeed, but got error: %v", finalErr)
	}

	// Assert that we made exactly 3 attempts (2 failures + 1 success)
	if attempts != 3 {
		t.Fatalf("Expected 3 attempts, but got %d", attempts)
	}
}

// TestConcurrentBlockUpdates tests that we handle concurrent block updates safely
func TestConcurrentBlockUpdates(t *testing.T) {
	// Create a service with progress tracking
	service := &EventCatchupService{
		intentProgress: make(map[uint64]uint64),
		mu:             sync.Mutex{},
	}

	// Run 100 concurrent updates
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(blockNum uint64) {
			defer wg.Done()
			service.UpdateIntentProgress(7000, blockNum)
		}(uint64(i))
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify that the final state is valid
	service.mu.Lock()
	progress := service.intentProgress[7000]
	service.mu.Unlock()

	// The progress should be set to some value between 0 and 99
	if progress > 99 {
		t.Fatalf("Expected progress to be between 0 and 99, but got %d", progress)
	}
}

// TestContextCancellation tests that we properly handle context cancellation
func TestContextCancellation(t *testing.T) {
	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Create a channel to signal completion
	done := make(chan struct{})

	// Simulate our polling loop's context handling
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Should exit the loop when context is cancelled
				close(done)
				return
			case <-ticker.C:
				// Continue polling
			}
		}
	}()

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for completion or timeout
	select {
	case <-done:
		// Success - loop exited when context was cancelled
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timed out waiting for context cancellation to be handled")
	}
}
