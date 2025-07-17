package services

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/speedrun-hq/speedrun/api/logger"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/speedrun-hq/speedrun/api/db"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/speedrun-hq/speedrun/api/utils"
)

// Constants for event processing
const (
	// IntentInitiatedEventName is the name of the intent initiated event
	IntentInitiatedEventName = "IntentInitiated"

	// IntentInitiatedWithCallEventName is the name of the intent initiated with call event
	IntentInitiatedWithCallEventName = "IntentInitiatedWithCall"

	// IntentInitiatedRequiredTopics is the minimum number of topics required in a log
	IntentInitiatedRequiredTopics = 3

	// IntentInitiatedRequiredFields is the number of fields expected in the event data
	IntentInitiatedRequiredFields = 5

	// IntentInitiatedWithCallRequiredFields is the number of fields expected in the event data for call intents
	IntentInitiatedWithCallRequiredFields = 7

	// Buffer sizes for channels
	DefaultErrorChannelBuffer = 100 // Increased from 10 for better handling of multiple chains
	DefaultLogsChannelBuffer  = 200 // Increased from 100 for high-throughput scenarios

	// Timeout configurations
	DefaultDBTimeout  = 10 * time.Second // Increased from 5s for complex DB operations
	DefaultRPCTimeout = 15 * time.Second // For RPC calls that might be slow
	DefaultLogTimeout = 45 * time.Second // Increased from 30s for slow log processing

	// Ticker intervals for reduced CPU usage
	HealthCheckInterval  = 5 * time.Minute // Health check every 5 minutes
	DebugLogInterval     = 2 * time.Minute // Debug logs every 2 minutes (reduced from 30s)
	HealthTickerInterval = 1 * time.Minute // Health ticker every 1 minute
)

// IntentService handles monitoring and processing of intent events
type IntentService struct {
	client           *ethclient.Client
	clientResolver   ClientResolver
	db               db.Database
	abi              abi.ABI
	chainID          uint64
	subs             map[string]ethereum.Subscription
	activeGoroutines int32      // Counter for active goroutines
	errChannel       chan error // Channel for collecting errors from goroutines
	mu               sync.Mutex // Mutex for thread-safe operations
	logger           logger.Logger

	// Metrics tracking
	eventsProcessed   int64     // Total events processed
	eventsSkipped     int64     // Total events skipped (duplicates)
	processingErrors  int64     // Total processing errors
	lastEventTime     time.Time // Time of last processed event
	lastHealthCheck   time.Time // Time of last health check
	reconnectionCount int64     // Number of times reconnected
	startTime         time.Time // When the service was started

	// ZetaChain polling health tracking
	isZetaChain      bool      // Whether this is ZetaChain (uses polling instead of subscriptions)
	lastPollingCheck time.Time // Last time polling health was verified
	pollingHealthy   bool      // Whether HTTP polling is working

	// Goroutine cleanup management
	cleanupCtx    context.Context    // Context for cleanup operations
	cleanupCancel context.CancelFunc // Cancel function for cleanup context
	goroutineWg   sync.WaitGroup     // WaitGroup to track all goroutines
	isShutdown    bool               // Flag to prevent new goroutines after shutdown
	shutdownMu    sync.RWMutex       // Mutex for shutdown operations
}

// NewIntentService creates a new IntentService instance
func NewIntentService(
	client *ethclient.Client,
	clientResolver ClientResolver,
	db db.Database,
	intentInitiatedEventABI string,
	chainID uint64,
	logger logger.Logger,
) (*IntentService, error) {
	// Parse the contract ABI
	parsedABI, err := abi.JSON(strings.NewReader(intentInitiatedEventABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %v", err)
	}

	errChan := make(chan error, DefaultErrorChannelBuffer) // Buffer for errors to avoid blocking

	// Detect if this is ZetaChain (uses polling instead of subscriptions)
	isZetaChain := chainID == 7000

	// Create cleanup context
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())

	return &IntentService{
		client:         client,
		clientResolver: clientResolver,
		db:             db,
		abi:            parsedABI,
		chainID:        chainID,
		subs:           make(map[string]ethereum.Subscription),
		errChannel:     errChan,
		logger:         logger,
		startTime:      time.Now(),
		isZetaChain:    isZetaChain,
		pollingHealthy: isZetaChain, // ZetaChain starts as healthy (polling assumed working)
		cleanupCtx:     cleanupCtx,
		cleanupCancel:  cleanupCancel,
	}, nil
}

// ActiveGoroutines returns the current count of active goroutines
func (s *IntentService) ActiveGoroutines() int32 {
	return atomic.LoadInt32(&s.activeGoroutines)
}

// IsHealthy checks if the service is healthy and processing events
func (s *IntentService) IsHealthy() bool {
	activeGoroutines := atomic.LoadInt32(&s.activeGoroutines)
	subscriptionCount := s.GetSubscriptionCount()

	// Update last health check time
	s.mu.Lock()
	s.lastHealthCheck = time.Now()
	s.mu.Unlock()

	// Provide a 30-second grace period for newly started services
	gracePeriod := 30 * time.Second
	isStartingUp := time.Since(s.startTime) < gracePeriod

	var isHealthy bool

	// ZetaChain uses HTTP polling instead of WebSocket subscriptions
	if s.isZetaChain {
		// For ZetaChain, health depends on HTTP client connectivity and polling status
		// We don't expect subscriptions or the normal goroutines since polling happens in catchup service

		// During startup, be lenient
		if isStartingUp {
			s.logger.DebugWithChain(s.chainID, "ZetaChain service starting up (grace period): HTTP polling assumed healthy")
			isHealthy = true
		} else {
			// Check if polling health has been verified recently (within 10 minutes)
			pollingStale := time.Since(s.lastPollingCheck) > 10*time.Minute
			isHealthy = s.pollingHealthy && !pollingStale

			if !isHealthy {
				if pollingStale {
					s.logger.DebugWithChain(s.chainID, "ZetaChain polling health stale (last check: %v ago)", time.Since(s.lastPollingCheck))
				} else {
					s.logger.DebugWithChain(s.chainID, "ZetaChain polling unhealthy")
				}
			}
		}
	} else {
		// For other chains, use the normal subscription-based health check
		// Service is healthy if it has:
		// 1. At least 3 goroutines (error monitor + health monitor + subscription goroutine)
		// 2. At least 1 active subscription
		isHealthy = activeGoroutines >= 3 && subscriptionCount >= 1

		// During startup, be more lenient
		if isStartingUp {
			s.logger.DebugWithChain(s.chainID, "Service starting up (grace period): activeGoroutines=%d, subscriptions=%d",
				activeGoroutines, subscriptionCount)
			isHealthy = true
		}

		// Add debug logging when service is unhealthy
		if !isHealthy {
			timeSinceStart := time.Since(s.startTime)
			s.logger.DebugWithChain(s.chainID, "Service unhealthy (running %v): activeGoroutines=%d (need >=3), subscriptions=%d (need >=1)",
				timeSinceStart, activeGoroutines, subscriptionCount)
		}
	}

	return isHealthy
}

// GetSubscriptionCount returns the number of active subscriptions
func (s *IntentService) GetSubscriptionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.subs)
}

// UpdatePollingHealth updates the polling health status for ZetaChain
func (s *IntentService) UpdatePollingHealth(healthy bool) {
	if !s.isZetaChain {
		return // Only applicable to ZetaChain
	}

	s.mu.Lock()
	s.pollingHealthy = healthy
	s.lastPollingCheck = time.Now()
	s.mu.Unlock()

	if healthy {
		s.logger.DebugWithChain(s.chainID, "ZetaChain polling health updated: healthy")
	} else {
		s.logger.DebugWithChain(s.chainID, "ZetaChain polling health updated: unhealthy")
	}
}

// IsZetaChain returns whether this service is for ZetaChain
func (s *IntentService) IsZetaChain() bool {
	return s.isZetaChain
}

// ServiceMetrics represents detailed metrics for the service
type ServiceMetrics struct {
	ChainID            uint64    `json:"chain_id"`
	ActiveGoroutines   int32     `json:"active_goroutines"`
	SubscriptionCount  int       `json:"subscription_count"`
	EventsProcessed    int64     `json:"events_processed"`
	EventsSkipped      int64     `json:"events_skipped"`
	ProcessingErrors   int64     `json:"processing_errors"`
	LastEventTime      time.Time `json:"last_event_time"`
	LastHealthCheck    time.Time `json:"last_health_check"`
	ReconnectionCount  int64     `json:"reconnection_count"`
	TimeSinceLastEvent string    `json:"time_since_last_event"`
	IsHealthy          bool      `json:"is_healthy"`

	// ZetaChain-specific metrics
	IsZetaChain           bool      `json:"is_zetachain"`
	PollingHealthy        bool      `json:"polling_healthy,omitempty"`
	LastPollingCheck      time.Time `json:"last_polling_check,omitempty"`
	TimeSincePollingCheck string    `json:"time_since_polling_check,omitempty"`
}

// GetMetrics returns detailed metrics about the service
func (s *IntentService) GetMetrics() ServiceMetrics {
	s.mu.Lock()
	subscriptionCount := len(s.subs)
	eventsProcessed := atomic.LoadInt64(&s.eventsProcessed)
	eventsSkipped := atomic.LoadInt64(&s.eventsSkipped)
	processingErrors := atomic.LoadInt64(&s.processingErrors)
	lastEventTime := s.lastEventTime
	lastHealthCheck := s.lastHealthCheck
	reconnectionCount := atomic.LoadInt64(&s.reconnectionCount)

	// ZetaChain-specific metrics
	isZetaChain := s.isZetaChain
	pollingHealthy := s.pollingHealthy
	lastPollingCheck := s.lastPollingCheck
	s.mu.Unlock()

	activeGoroutines := atomic.LoadInt32(&s.activeGoroutines)
	isHealthy := s.IsHealthy()

	var timeSinceLastEvent string
	if !lastEventTime.IsZero() {
		timeSinceLastEvent = time.Since(lastEventTime).String()
	} else {
		timeSinceLastEvent = "never"
	}

	var timeSincePollingCheck string
	if isZetaChain && !lastPollingCheck.IsZero() {
		timeSincePollingCheck = time.Since(lastPollingCheck).String()
	} else if isZetaChain {
		timeSincePollingCheck = "never"
	}

	return ServiceMetrics{
		ChainID:               s.chainID,
		ActiveGoroutines:      activeGoroutines,
		SubscriptionCount:     subscriptionCount,
		EventsProcessed:       eventsProcessed,
		EventsSkipped:         eventsSkipped,
		ProcessingErrors:      processingErrors,
		LastEventTime:         lastEventTime,
		LastHealthCheck:       lastHealthCheck,
		ReconnectionCount:     reconnectionCount,
		TimeSinceLastEvent:    timeSinceLastEvent,
		IsHealthy:             isHealthy,
		IsZetaChain:           isZetaChain,
		PollingHealthy:        pollingHealthy,
		LastPollingCheck:      lastPollingCheck,
		TimeSincePollingCheck: timeSincePollingCheck,
	}
}

// RestartSubscription restarts a subscription for a given contract address
func (s *IntentService) RestartSubscription(ctx context.Context, contractAddress common.Address) error {
	subID := contractAddress.Hex()

	// Unsubscribe existing subscription if it exists
	s.mu.Lock()
	if existingSub, exists := s.subs[subID]; exists {
		existingSub.Unsubscribe()
		delete(s.subs, subID)
		s.logger.InfoWithChain(s.chainID, "Unsubscribed existing subscription for restart: %s", subID)
	}
	s.mu.Unlock()

	// Get the current block to avoid reprocessing old events
	blockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	currentBlock, err := s.client.BlockNumber(blockCtx)
	cancel()

	startBlock := uint64(0)
	if err != nil {
		s.logger.InfoWithChain(s.chainID, "WARNING: Failed to get current block for restart: %v, starting from block 0", err)
	} else {
		startBlock = currentBlock
		s.logger.InfoWithChain(s.chainID, "Restarting subscription from current block %d", startBlock)
	}

	// Track reconnection
	atomic.AddInt64(&s.reconnectionCount, 1)

	// Start a new subscription with reconnection
	go s.startSubscriptionWithReconnection(ctx, contractAddress, startBlock)

	s.logger.InfoWithChain(s.chainID, "Restarted subscription for contract %s (reconnection #%d)",
		contractAddress.Hex(), atomic.LoadInt64(&s.reconnectionCount))
	return nil
}

// StartHealthMonitor starts a goroutine that monitors the health of the service
func (s *IntentService) StartHealthMonitor(ctx context.Context, contractAddress common.Address) {
	ticker := time.NewTicker(HealthCheckInterval) // Use constant for reduced CPU usage
	defer ticker.Stop()

	consecutiveFailures := 0
	maxConsecutiveFailures := 3

	for {
		select {
		case <-ticker.C:
			// Check if service is healthy
			if !s.IsHealthy() {
				consecutiveFailures++
				s.logger.InfoWithChain(s.chainID, "Health check failed (%d/%d): activeGoroutines=%d, subscriptions=%d",
					consecutiveFailures, maxConsecutiveFailures, s.ActiveGoroutines(), s.GetSubscriptionCount())

				if consecutiveFailures >= maxConsecutiveFailures {
					s.logger.InfoWithChain(s.chainID, "Service appears unhealthy, attempting restart")

					// Attempt to restart the subscription
					if err := s.RestartSubscription(ctx, contractAddress); err != nil {
						s.logger.ErrorWithChain(s.chainID, "Failed to restart subscription: %v", err)
					} else {
						consecutiveFailures = 0 // Reset counter on successful restart
					}
				}
			} else {
				// Service is healthy, reset failure counter
				if consecutiveFailures > 0 {
					s.logger.InfoWithChain(s.chainID, "Service health restored")
					consecutiveFailures = 0
				}
			}
		case <-ctx.Done():
			s.logger.DebugWithChain(s.chainID, "Health monitor shutting down")
			return
		}
	}
}

// StartListening starts a goroutine to listen for intent events from the specified contract address.
// It sets up a subscription to the blockchain and processes events as they arrive.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - contractAddress: The address of the contract to listen to
//
// Returns:
//   - error: Any error that occurred during setup
func (s *IntentService) StartListening(ctx context.Context, contractAddress common.Address) error {
	// Check if service is shutdown
	if s.IsShutdown() {
		return fmt.Errorf("cannot start listening: service is shutdown")
	}

	// Check if the client is using a websocket connection, which is needed for subscriptions
	clientType := reflect.TypeOf(s.client).String()
	isWebsocket := strings.Contains(strings.ToLower(clientType), "websocket")
	s.logger.Info("Intent service using client type: %s, is websocket: %v", clientType, isWebsocket)

	if !isWebsocket {
		s.logger.Info("WARNING: Intent service may not receive real-time events because client type is %s, not websocket", clientType)
	}

	// Get current block number as a starting point to avoid processing old events
	startBlockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	latestBlock, err := s.client.BlockNumber(startBlockCtx)
	cancel()
	if err != nil {
		s.logger.Info("WARNING: Failed to get current block number: %v, will listen to all new blocks", err)
	} else {
		s.logger.Notice("Starting intent event subscription from block %d", latestBlock)
	}

	// Start a goroutine to monitor the error channel
	s.startGoroutine("error-monitor", func() {
		s.monitorErrors(s.cleanupCtx)
	})

	// Start the subscription listener with automatic reconnection
	s.startGoroutine("subscription-reconnection", func() {
		s.startSubscriptionWithReconnection(s.cleanupCtx, contractAddress, latestBlock)
	})

	// Start the health monitor
	s.startGoroutine("health-monitor", func() {
		s.StartHealthMonitor(s.cleanupCtx, contractAddress)
	})

	return nil
}

// monitorErrors processes errors from goroutines
func (s *IntentService) monitorErrors(ctx context.Context) {
	for {
		select {
		case err := <-s.errChannel:
			s.logger.Error("Error in IntentService goroutine: %v", err)
		case <-ctx.Done():
			s.logger.DebugWithChain(s.chainID, "Error monitor shutting down")
			return
		}
	}
}

// startSubscriptionWithReconnection handles the subscription lifecycle with automatic reconnection
func (s *IntentService) startSubscriptionWithReconnection(ctx context.Context, contractAddress common.Address, startBlock uint64) {
	subID := contractAddress.Hex()

	// Retry configuration
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 5 * time.Minute

	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			s.logger.DebugWithChain(s.chainID, "Context cancelled, stopping subscription attempts")
			return
		default:
		}

		// Calculate delay with exponential backoff
		delay := time.Duration(1<<attempt) * baseDelay
		if delay > maxDelay {
			delay = maxDelay
		}

		if attempt > 0 {
			s.logger.InfoWithChain(s.chainID, "Retrying subscription attempt %d/%d after %v", attempt+1, maxRetries, delay)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}

		// Create subscription
		err := s.createAndRunSubscription(ctx, contractAddress, subID, startBlock)
		if err == nil {
			// Subscription ended normally (context cancelled)
			return
		}

		// Log the error
		s.logger.ErrorWithChain(s.chainID, "Subscription failed (attempt %d/%d): %v", attempt+1, maxRetries, err)

		// If this is a context cancellation, don't retry
		if ctx.Err() != nil {
			return
		}
	}

	// If we get here, all retries failed
	s.errChannel <- fmt.Errorf("failed to establish stable subscription after %d attempts", maxRetries)
	s.logger.ErrorWithChain(s.chainID, "CRITICAL: Unable to establish stable subscription after %d attempts", maxRetries)
}

// createAndRunSubscription creates a new subscription and runs the event processing loop
func (s *IntentService) createAndRunSubscription(ctx context.Context, contractAddress common.Address, subID string, startBlock uint64) error {
	// Configure the filter query for events
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics: [][]common.Hash{
			{
				s.abi.Events[IntentInitiatedEventName].ID,
				s.abi.Events[IntentInitiatedWithCallEventName].ID,
			},
		},
	}

	// Set FromBlock if we have a start block
	if startBlock > 0 {
		query.FromBlock = big.NewInt(int64(startBlock))
	}

	// Log the full query details for debugging
	s.logger.Debug("Intent subscription filter query: Addresses=%v, Topics=%v, FromBlock=%v",
		query.Addresses, query.Topics, query.FromBlock)

	// Create a new logs channel for this subscription
	logs := make(chan types.Log, DefaultLogsChannelBuffer) // Buffer to prevent blocking

	// Create the subscription
	sub, err := s.client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return fmt.Errorf("failed to subscribe to logs: %v", err)
	}

	// Store the subscription
	s.mu.Lock()
	s.subs[subID] = sub
	s.mu.Unlock()

	s.logger.InfoWithChain(s.chainID, "Successfully subscribed to intent events for contract %s", contractAddress.Hex())

	// Run the event processing loop
	err = s.runEventProcessingLoop(ctx, sub, logs, subID)

	// Clean up subscription
	s.mu.Lock()
	if storedSub, exists := s.subs[subID]; exists && storedSub == sub {
		delete(s.subs, subID)
	}
	s.mu.Unlock()

	sub.Unsubscribe()
	close(logs) // Close the logs channel to prevent goroutine leaks

	return err
}

// runEventProcessingLoop runs the main event processing loop for a subscription
func (s *IntentService) runEventProcessingLoop(ctx context.Context, sub ethereum.Subscription, logs chan types.Log, subID string) error {
	s.logger.InfoWithChain(s.chainID, "Starting event log processing, subscription %s", subID)

	// Use a ticker to periodically check system health
	healthTicker := time.NewTicker(HealthTickerInterval)
	defer healthTicker.Stop()

	// Add a ticker for debugging to periodically log subscription status
	debugTicker := time.NewTicker(DebugLogInterval)
	defer debugTicker.Stop()

	// Track the number of events processed for debugging
	eventCount := 0

	for {
		select {
		case err := <-sub.Err():
			if err != nil {
				s.logger.ErrorWithChain(s.chainID, "Subscription %s error: %v", subID, err)
				return fmt.Errorf("subscription error: %v", err)
			}
		case vLog, ok := <-logs:
			if !ok {
				s.logger.ErrorWithChain(s.chainID, "Log channel closed for subscription %s", subID)
				return fmt.Errorf("log channel closed")
			}

			eventCount++
			s.logger.InfoWithChain(s.chainID, "EVENT RECEIVED: Block %d, TxHash %s, Topics: %v", vLog.BlockNumber, vLog.TxHash.Hex(), len(vLog.Topics))

			// Process the log with timeout to prevent processing for too long
			logCtx, logCancel := context.WithTimeout(ctx, DefaultLogTimeout)
			startTime := time.Now()
			err := s.processLog(logCtx, vLog)
			processingTime := time.Since(startTime)
			logCancel()

			if err != nil {
				atomic.AddInt64(&s.processingErrors, 1)
				s.errChannel <- fmt.Errorf("error processing log: %v", err)
				s.logger.ErrorWithChain(s.chainID, "Failed to process log, subscription %s: %v", subID, err)
			} else {
				s.logger.InfoWithChain(s.chainID, "Successfully processed event from block %d, tx %s (took %v)", vLog.BlockNumber, vLog.TxHash.Hex(), processingTime)
			}
		case <-healthTicker.C:
			// Log system health information
			s.logger.DebugWithChain(s.chainID, "IntentService health: activeGoroutines=%d, events_processed=%d",
				s.ActiveGoroutines(), eventCount)
		case <-debugTicker.C:
			// Extra debugging info
			s.logger.DebugWithChain(s.chainID, "Intent subscription %s still active, processed %d events so far",
				subID, eventCount)
		case <-ctx.Done():
			s.logger.DebugWithChain(s.chainID, "Context cancelled, stopping event processing, subscription %s", subID)
			return nil // Normal termination
		}
	}
}

// processLog processes a single log entry from the blockchain.
// It validates the log, extracts event data, and stores the intent in the database.
func (s *IntentService) processLog(ctx context.Context, vLog types.Log) error {
	// Check for context cancellation
	if ctx.Err() != nil {
		return ctx.Err()
	}

	logStart := time.Now()
	defer func() {
		logLatency := time.Since(logStart)
		if logLatency > 1*time.Second {
			s.logger.Debug("SLOW LOG PROCESSING: Chain %d, Block %d, TxHash %s took %v",
				s.chainID, vLog.BlockNumber, vLog.TxHash.Hex(), logLatency)
		}
	}()

	if err := s.validateLog(vLog); err != nil {
		return err
	}

	// Set a timeout for event data extraction
	extractCtx, extractCancel := context.WithTimeout(ctx, DefaultRPCTimeout)
	event, err := s.extractEventData(extractCtx, vLog)
	extractCancel()

	if err != nil {
		return err
	}

	// Use the target chain from the event data
	event.ChainID = s.chainID

	// Important: Use the correct chain client for intent events
	// Intent events happen on the source chain, so we need to use the source chain client
	client := s.client
	if s.clientResolver != nil {
		// Try to get the source chain client
		sourceClient, err := s.clientResolver.GetClient(event.ChainID)
		if err == nil {
			client = sourceClient
		} else {
			s.logger.Debug("Warning: Failed to get source chain client: %v, using default client", err)
		}
	}

	// Set a timeout for intent conversion
	intentCtx, intentCancel := context.WithTimeout(ctx, DefaultRPCTimeout)
	intent, err := event.ToIntent(client, intentCtx)
	intentCancel()

	if err != nil {
		s.logger.Debug("Warning: Failed to get block timestamp: %v", err)
		// Continue with what we have
	}

	// Add a warning log if the chain IDs don't match and we're using the default client
	if intent.SourceChain != s.chainID && client == s.client {
		s.logger.Debug("Warning: Using client for chain %d to fetch timestamp for intent event on chain %d",
			s.chainID, intent.SourceChain)
	}

	// Check if intent already exists - set a timeout
	dbCtx, dbCancel := context.WithTimeout(ctx, DefaultDBTimeout)
	existingIntent, err := s.db.GetIntent(dbCtx, intent.ID)
	dbCancel()

	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("failed to check for existing intent: %v", err)
	}

	// Skip if intent already exists
	if existingIntent != nil {
		atomic.AddInt64(&s.eventsSkipped, 1)
		s.logger.Debug("Skipped duplicate intent: %s", intent.ID)
		return nil
	}

	// Create the intent with a timeout
	createCtx, createCancel := context.WithTimeout(ctx, DefaultDBTimeout)
	err = s.db.CreateIntent(createCtx, intent)
	createCancel()

	if err != nil {
		// Skip if intent already exists
		if strings.Contains(err.Error(), "duplicate key") {
			atomic.AddInt64(&s.eventsSkipped, 1)
			s.logger.Debug("Skipped duplicate intent during creation: %s", intent.ID)
			return nil
		}
		atomic.AddInt64(&s.processingErrors, 1)
		return fmt.Errorf("failed to store intent in database: %v", err)
	}

	// Update metrics
	atomic.AddInt64(&s.eventsProcessed, 1)
	s.mu.Lock()
	s.lastEventTime = time.Now()
	s.mu.Unlock()

	s.logger.Info("Successfully processed and stored intent: %s", intent.ID)
	return nil
}

// validateLog checks if the log has the required structure and data.
func (s *IntentService) validateLog(vLog types.Log) error {
	s.logger.Debug("Validating log: BlockNum=%d, TxHash=%s, Address=%s, Topics=%d, DataSize=%d bytes",
		vLog.BlockNumber, vLog.TxHash.Hex(), vLog.Address.Hex(), len(vLog.Topics), len(vLog.Data))

	if len(vLog.Topics) == 0 {
		return fmt.Errorf("invalid log: no topics found")
	}

	// Log the first topic which should be the event signature
	if len(vLog.Topics) > 0 {
		expectedSig := s.abi.Events[IntentInitiatedEventName].ID.Hex()
		expectedCallSig := s.abi.Events[IntentInitiatedWithCallEventName].ID.Hex()
		actualSig := vLog.Topics[0].Hex()

		isStandard := expectedSig == actualSig
		isCall := expectedCallSig == actualSig

		s.logger.Debug("Event signature check - Expected Standard: %s, Expected Call: %s, Got: %s, Match Standard: %v, Match Call: %v",
			expectedSig, expectedCallSig, actualSig, isStandard, isCall)
	}

	if len(vLog.Topics) < IntentInitiatedRequiredTopics {
		s.logger.Error("Invalid log: expected at least %d topics, got %d",
			IntentInitiatedRequiredTopics, len(vLog.Topics))
		return fmt.Errorf("invalid log: expected at least %d topics, got %d", IntentInitiatedRequiredTopics, len(vLog.Topics))
	}

	// Validate event signature - now check for both event types
	expectedStandardSig := s.abi.Events[IntentInitiatedEventName].ID
	expectedCallSig := s.abi.Events[IntentInitiatedWithCallEventName].ID

	if vLog.Topics[0] != expectedStandardSig && vLog.Topics[0] != expectedCallSig {
		s.logger.Error("Invalid event signature - Expected Standard: %s, Expected Call: %s, Got: %s",
			expectedStandardSig.Hex(), expectedCallSig.Hex(), vLog.Topics[0].Hex())
		return fmt.Errorf("invalid event signature: expected %s or %s, got %s",
			expectedStandardSig.Hex(), expectedCallSig.Hex(), vLog.Topics[0].Hex())
	}

	s.logger.Debug("Log validation passed - BlockNum=%d, TxHash=%s",
		vLog.BlockNumber, vLog.TxHash.Hex())
	return nil
}

// extractEventData extracts and validates the event data from the log.
func (s *IntentService) extractEventData(ctx context.Context, vLog types.Log) (*models.IntentInitiatedEvent, error) {
	s.logger.Debug("Extracting event data from log: BlockNum=%d, TxHash=%s",
		vLog.BlockNumber, vLog.TxHash.Hex())

	event := &models.IntentInitiatedEvent{
		BlockNumber: vLog.BlockNumber,
		TxHash:      vLog.TxHash.Hex(),
	}

	// Parse indexed parameters from topics
	if len(vLog.Topics) < 3 {
		s.logger.Error("Invalid log: expected at least 3 topics, got %d", len(vLog.Topics))
		return nil, fmt.Errorf("invalid log: expected at least 3 topics, got %d", len(vLog.Topics))
	}

	// Topic[0] is the event signature
	// Topic[1] is the indexed intentId
	// Topic[2] is the indexed asset address
	event.IntentID = vLog.Topics[1].Hex()
	event.Asset = common.HexToAddress(vLog.Topics[2].Hex()).Hex()

	s.logger.Debug("Extracted indexed parameters - IntentID: %s, Asset: %s",
		event.IntentID, event.Asset)

	// Parse non-indexed parameters from data
	if len(vLog.Data) == 0 {
		s.logger.Error("Log data is empty, cannot unpack parameters")
		return nil, fmt.Errorf("event data is empty")
	}

	// Determine if this is a standard intent or a call intent based on the event signature
	var eventName string
	switch eventTopic := vLog.Topics[0]; eventTopic {
	case s.abi.Events[IntentInitiatedEventName].ID:
		eventName = IntentInitiatedEventName
		s.logger.Debug("Processing standard intent event")
	case s.abi.Events[IntentInitiatedWithCallEventName].ID:
		eventName = IntentInitiatedWithCallEventName
		s.logger.Debug("Processing intent with call event")
		event.IsCall = true
	default:
		s.logger.Error("Unknown event signature: %s", eventTopic.Hex())
		return nil, fmt.Errorf("unknown event signature: %s", eventTopic.Hex())
	}

	s.logger.Debug("Unpacking event data (%d bytes) using ABI for %s",
		len(vLog.Data), eventName)

	unpacked, err := s.abi.Unpack(eventName, vLog.Data)
	if err != nil {
		s.logger.Error("Failed to unpack event data: %v", err)
		return nil, fmt.Errorf("failed to unpack event data: %v", err)
	}

	s.logger.Debug("Unpacked %d fields from event data", len(unpacked))

	// Check minimum field requirements based on event type
	requiredFields := IntentInitiatedRequiredFields
	if event.IsCall {
		requiredFields = IntentInitiatedWithCallRequiredFields
	}

	if len(unpacked) < requiredFields {
		s.logger.Error("Invalid event data: expected %d fields, got %d",
			requiredFields, len(unpacked))
		return nil, fmt.Errorf("invalid event data: expected %d fields, got %d", requiredFields, len(unpacked))
	}

	if err := s.validateEventFields(unpacked, event); err != nil {
		s.logger.Error("Failed to validate event fields: %v", err)
		return nil, err
	}

	// Get the sender address from the transaction - add timeout
	txCtx, txCancel := context.WithTimeout(ctx, DefaultRPCTimeout)
	defer txCancel()

	s.logger.Debug("Fetching transaction %s to extract sender", vLog.TxHash.Hex())
	tx, _, err := s.client.TransactionByHash(txCtx, vLog.TxHash)
	if err != nil {
		s.logger.Error("Failed to get transaction: %v", err)
		return nil, fmt.Errorf("failed to get transaction: %v", err)
	}

	// Get the sender address from the transaction
	signer := types.LatestSignerForChainID(big.NewInt(int64(s.chainID)))
	sender, err := signer.Sender(tx)
	if err != nil {
		s.logger.Error("Failed to get sender address: %v", err)
		return nil, fmt.Errorf("failed to get sender address: %v", err)
	}

	event.Sender = sender.Hex()
	s.logger.Debug("Extracted sender: %s", event.Sender)

	s.logger.Debug("Successfully extracted all event data for intent %s", event.IntentID)
	return event, nil
}

// validateEventFields validates each field of the event data.
func (s *IntentService) validateEventFields(unpacked []interface{}, event *models.IntentInitiatedEvent) error {
	var ok bool

	s.logger.Debug("Validating event fields (%d values)", len(unpacked))

	// Log the types of unpacked values for debugging
	for i, val := range unpacked {
		if val == nil {
			s.logger.Debug("Field %d is nil", i)
		} else {
			s.logger.Debug("Field %d type: %T, value: %v", i, val, val)
		}
	}

	// Determine if this is a standard intent or a call intent based on the unpacked data length
	isCallIntent := len(unpacked) >= IntentInitiatedWithCallRequiredFields

	event.Amount, ok = unpacked[0].(*big.Int)
	if !ok || event.Amount == nil {
		return fmt.Errorf("invalid amount in event data")
	}

	targetChainBig, ok := unpacked[1].(*big.Int)
	if !ok || targetChainBig == nil {
		return fmt.Errorf("invalid target chain in event data")
	}
	event.TargetChain = targetChainBig.Uint64()

	event.Receiver, ok = unpacked[2].([]byte)
	if !ok || len(event.Receiver) == 0 {
		return fmt.Errorf("invalid receiver in event data")
	}

	event.Tip, ok = unpacked[3].(*big.Int)
	if !ok || event.Tip == nil {
		return fmt.Errorf("invalid tip in event data")
	}

	event.Salt, ok = unpacked[4].(*big.Int)
	if !ok || event.Salt == nil {
		return fmt.Errorf("invalid salt in event data")
	}

	// If this is a call intent, extract the data field
	if isCallIntent {
		event.IsCall = true

		if len(unpacked) > 5 {
			event.Data, ok = unpacked[5].([]byte)
			if !ok {
				return fmt.Errorf("invalid data in event data")
			}
		}
	} else {
		event.IsCall = false
	}

	return nil
}

// GetIntent retrieves an intent from the database
func (s *IntentService) GetIntent(ctx context.Context, id string) (*models.Intent, error) {
	// First, check if the intent exists in the database
	intent, err := s.db.GetIntent(ctx, id)
	if err != nil {
		// Check if the error is "not found"
		if strings.Contains(err.Error(), "not found") {
			// Try to check on-chain via RPC if this intent exists
			s.logger.Error("Intent not found in database, attempting to check on-chain for intent %s", id)

			// Here you would typically query the blockchain or other sources
			// For now, we're just improving error logging
			return nil, fmt.Errorf("intent not found: %s (not in database)", id)
		}

		// Log detailed error for debugging
		s.logger.Error("ERROR: Failed to get intent %s from database: %v", id, err)

		return nil, fmt.Errorf("error retrieving intent: %v", err)
	}

	// Log success
	s.logger.Debug("Successfully retrieved intent %s from database", id)
	return intent, nil
}

// ListIntents retrieves all intents
func (s *IntentService) ListIntents(ctx context.Context) ([]*models.Intent, error) {
	intents, err := s.db.ListIntents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list intents: %v", err)
	}

	return intents, nil
}

// GetIntentsBySender retrieves all intents for a specific sender address
func (s *IntentService) GetIntentsBySender(ctx context.Context, sender string) ([]*models.Intent, error) {
	intents, err := s.db.ListIntentsBySender(ctx, sender)
	if err != nil {
		return nil, fmt.Errorf("failed to list intents by sender: %v", err)
	}
	return intents, nil
}

// GetIntentsByRecipient retrieves all intents for a specific recipient address
func (s *IntentService) GetIntentsByRecipient(ctx context.Context, recipient string) ([]*models.Intent, error) {
	intents, err := s.db.ListIntentsByRecipient(ctx, recipient)
	if err != nil {
		return nil, fmt.Errorf("failed to list intents by recipient: %v", err)
	}
	return intents, nil
}

// CreateIntent creates a new intent
func (s *IntentService) CreateIntent(ctx context.Context, id string, sourceChain uint64, destinationChain uint64, token, amount, recipient, sender, intentFee string, timestamp ...time.Time) (*models.Intent, error) {
	// Validate chain IDs
	if err := utils.ValidateChain(sourceChain); err != nil {
		return nil, fmt.Errorf("invalid source chain: %v", err)
	}
	if err := utils.ValidateChain(destinationChain); err != nil {
		return nil, fmt.Errorf("invalid destination chain: %v", err)
	}

	// Validate token address
	if err := utils.ValidateAddress(token); err != nil {
		return nil, fmt.Errorf("invalid token address: %v", err)
	}

	// Validate amount
	if err := utils.ValidateAmount(amount); err != nil {
		return nil, fmt.Errorf("invalid amount: %v", err)
	}

	// Validate recipient address
	if err := utils.ValidateAddress(recipient); err != nil {
		return nil, fmt.Errorf("invalid recipient address: %v", err)
	}

	// Validate sender address
	if err := utils.ValidateAddress(sender); err != nil {
		return nil, fmt.Errorf("invalid sender address: %v", err)
	}

	// Validate intent fee
	if err := utils.ValidateAmount(intentFee); err != nil {
		return nil, fmt.Errorf("invalid intent fee: %v", err)
	}

	// For API-created intents, we use the current time
	// For blockchain events, the block timestamp should be used and passed as a parameter
	var now time.Time
	if len(timestamp) > 0 && !timestamp[0].IsZero() {
		now = timestamp[0]
	} else {
		now = time.Now()
	}

	intent := &models.Intent{
		ID:               id,
		SourceChain:      sourceChain,
		DestinationChain: destinationChain,
		Token:            token,
		Amount:           amount,
		Recipient:        recipient,
		Sender:           sender,
		IntentFee:        intentFee,
		Status:           models.IntentStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.db.CreateIntent(ctx, intent); err != nil {
		return nil, err
	}

	return intent, nil
}

// CreateCallIntent creates a new intent with call data
func (s *IntentService) CreateCallIntent(ctx context.Context, id string, sourceChain uint64, destinationChain uint64, token, amount, recipient, sender, intentFee string, callData string, timestamp ...time.Time) (*models.Intent, error) {
	// Validate chain IDs
	if err := utils.ValidateChain(sourceChain); err != nil {
		return nil, fmt.Errorf("invalid source chain: %v", err)
	}
	if err := utils.ValidateChain(destinationChain); err != nil {
		return nil, fmt.Errorf("invalid destination chain: %v", err)
	}

	// Validate token address
	if err := utils.ValidateAddress(token); err != nil {
		return nil, fmt.Errorf("invalid token address: %v", err)
	}

	// Validate amount
	if err := utils.ValidateAmount(amount); err != nil {
		return nil, fmt.Errorf("invalid amount: %v", err)
	}

	// Validate recipient address
	if err := utils.ValidateAddress(recipient); err != nil {
		return nil, fmt.Errorf("invalid recipient address: %v", err)
	}

	// Validate sender address
	if err := utils.ValidateAddress(sender); err != nil {
		return nil, fmt.Errorf("invalid sender address: %v", err)
	}

	// Validate intent fee
	if err := utils.ValidateAmount(intentFee); err != nil {
		return nil, fmt.Errorf("invalid intent fee: %v", err)
	}

	// For API-created intents, we use the current time
	// For blockchain events, the block timestamp should be used and passed as a parameter
	var now time.Time
	if len(timestamp) > 0 && !timestamp[0].IsZero() {
		now = timestamp[0]
	} else {
		now = time.Now()
	}

	intent := &models.Intent{
		ID:               id,
		SourceChain:      sourceChain,
		DestinationChain: destinationChain,
		Token:            token,
		Amount:           amount,
		Recipient:        recipient,
		Sender:           sender,
		IntentFee:        intentFee,
		Status:           models.IntentStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
		IsCall:           true,
		CallData:         callData,
	}

	if err := s.db.CreateIntent(ctx, intent); err != nil {
		return nil, err
	}

	return intent, nil
}

// UnsubscribeAll unsubscribes from all active subscriptions
func (s *IntentService) UnsubscribeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug("Unsubscribing from all intent subscriptions for chain %d (%d active subscriptions)",
		s.chainID, len(s.subs))

	for id, sub := range s.subs {
		sub.Unsubscribe()
		s.logger.Debug("Unsubscribed from intent subscription %s on chain %d", id, s.chainID)
		delete(s.subs, id)
	}
}

// drainErrorChannel drains the error channel to prevent goroutine leaks during shutdown
func (s *IntentService) drainErrorChannel() {
	// Drain the error channel with a shorter timeout to prevent blocking
	drainCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	for {
		select {
		case <-s.errChannel:
			// Consume and discard errors during shutdown
		case <-drainCtx.Done():
			return
		}
	}
}

// Shutdown gracefully shuts down the service and waits for all goroutines to complete
func (s *IntentService) Shutdown(timeout time.Duration) error {
	s.shutdownMu.Lock()
	if s.isShutdown {
		s.shutdownMu.Unlock()
		return nil // Already shutdown
	}
	s.isShutdown = true
	s.shutdownMu.Unlock()

	s.logger.InfoWithChain(s.chainID, "Shutting down IntentService...")

	// Cancel the cleanup context to signal all goroutines to stop
	s.cleanupCancel()

	// Unsubscribe from all subscriptions
	s.UnsubscribeAll()

	// Drain the error channel to prevent goroutine leaks (run in current goroutine)
	s.drainErrorChannel()

	// Wait for all goroutines to complete with timeout
	done := make(chan struct{})
	go func() {
		s.goroutineWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.InfoWithChain(s.chainID, "IntentService shutdown completed successfully")
		return nil
	case <-time.After(timeout):
		s.logger.ErrorWithChain(s.chainID, "IntentService shutdown timed out after %v", timeout)
		return fmt.Errorf("shutdown timed out after %v", timeout)
	}
}

// IsShutdown returns whether the service is in shutdown state
func (s *IntentService) IsShutdown() bool {
	s.shutdownMu.RLock()
	defer s.shutdownMu.RUnlock()
	return s.isShutdown
}

// startGoroutine safely starts a goroutine with proper cleanup tracking
func (s *IntentService) startGoroutine(name string, fn func()) {
	s.shutdownMu.RLock()
	if s.isShutdown {
		s.shutdownMu.RUnlock()
		s.logger.DebugWithChain(s.chainID, "Cannot start goroutine %s: service is shutdown", name)
		return
	}
	s.shutdownMu.RUnlock()

	s.goroutineWg.Add(1)
	atomic.AddInt32(&s.activeGoroutines, 1)

	go func() {
		defer func() {
			s.goroutineWg.Done()
			atomic.AddInt32(&s.activeGoroutines, -1)

			// Recover from panics
			if r := recover(); r != nil {
				err := fmt.Errorf("panic in goroutine %s: %v\nstack: %s", name, r, debug.Stack())
				s.errChannel <- err
				s.logger.Error("CRITICAL: %v", err)
			}
		}()

		fn()
	}()
}
