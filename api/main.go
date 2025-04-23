package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gin-gonic/gin"
	"github.com/speedrun-hq/speedrun/api/config"
	"github.com/speedrun-hq/speedrun/api/db"
	"github.com/speedrun-hq/speedrun/api/handlers"
	"github.com/speedrun-hq/speedrun/api/services"
)

// HealthCheck handler
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// createEthereumClients creates and returns a map of Ethereum clients for each chain
func createEthereumClients(cfg *config.Config) (map[uint64]*ethclient.Client, error) {
	clients := make(map[uint64]*ethclient.Client)
	for chainID, chainConfig := range cfg.ChainConfigs {
		var client *ethclient.Client
		var err error

		// Check if this is a WebSocket URL or HTTP URL
		isWebSocket := strings.HasPrefix(chainConfig.RPCURL, "wss://") || strings.HasPrefix(chainConfig.RPCURL, "ws://")

		log.Printf("Creating client for chain %d with RPC URL %s (WebSocket: %v)",
			chainID, maskRPCURL(chainConfig.RPCURL), isWebSocket)

		if isWebSocket {
			// Use WebSocket connection for subscriptions
			rpcClient, err := rpc.DialWebsocket(context.Background(), chainConfig.RPCURL, "")
			if err != nil {
				return nil, fmt.Errorf("failed to create WebSocket RPC client for chain %d: %v", chainID, err)
			}
			client = ethclient.NewClient(rpcClient)
			log.Printf("Successfully created WebSocket client for chain %d", chainID)

			// Verify that the websocket connection supports subscriptions
			if err := verifyWebsocketSubscription(client, chainID); err != nil {
				log.Printf("WARNING: WebSocket connection for chain %d failed subscription test: %v", chainID, err)
				log.Printf("CRITICAL: Check your RPC provider. Some 'WebSocket' endpoints do not properly support subscriptions!")
			} else {
				log.Printf("SUCCESS: WebSocket connection for chain %d verified - subscriptions are working", chainID)
			}
		} else {
			// For HTTP connections, emit a warning that subscriptions might not work
			log.Printf("WARNING: Using HTTP RPC URL for chain %d. Real-time subscriptions may not work. Consider using a WebSocket URL instead.", chainID)
			client, err = ethclient.Dial(chainConfig.RPCURL)
			if err != nil {
				return nil, fmt.Errorf("failed to connect to chain %d: %v", chainID, err)
			}
		}

		// Verify client is working by getting the current block number
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		blockNumber, err := client.BlockNumber(ctx)
		cancel()

		if err != nil {
			log.Printf("WARNING: Could not get block number for chain %d: %v", chainID, err)
		} else {
			log.Printf("Client for chain %d connected successfully. Current block: %d", chainID, blockNumber)
		}

		clients[chainID] = client
	}
	return clients, nil
}

// verifyWebsocketSubscription tests if a client supports subscriptions by attempting to subscribe to new heads
func verifyWebsocketSubscription(client *ethclient.Client, chainID uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create a channel to receive headers
	headers := make(chan *types.Header)

	// Try to subscribe to new heads - this only works with websocket connections
	sub, err := client.SubscribeNewHead(ctx, headers)
	if err != nil {
		return fmt.Errorf("subscription test failed: %v", err)
	}

	// Create a channel to signal when we've received a header or timed out
	received := make(chan bool, 1)

	// Set up a timeout for receiving the first header
	timeout := time.After(10 * time.Second)

	// Start a goroutine to receive headers
	go func() {
		select {
		case header := <-headers:
			log.Printf("Received new block header for chain %d: number=%d, hash=%s",
				chainID, header.Number.Uint64(), header.Hash().Hex())
			received <- true
		case err := <-sub.Err():
			log.Printf("Subscription error for chain %d: %v", chainID, err)
			received <- false
		case <-timeout:
			log.Printf("Timed out waiting for header from chain %d", chainID)
			received <- false
		}
	}()

	// Wait for the result
	result := <-received

	// Clean up
	sub.Unsubscribe()

	if !result {
		return fmt.Errorf("did not receive block header within timeout")
	}

	return nil
}

// maskRPCURL masks an RPC URL to avoid logging sensitive information
func maskRPCURL(url string) string {
	// If URL contains API key as query parameter or path segment, mask it
	if strings.Contains(url, "api-key=") {
		return strings.Split(url, "api-key=")[0] + "api-key=***"
	}
	if strings.Contains(url, "apikey=") {
		return strings.Split(url, "apikey=")[0] + "apikey=***"
	}

	// If URL contains API key as part of the path, mask that too
	parts := strings.Split(url, "/")
	if len(parts) > 3 {
		// Keep protocol and domain, mask the rest
		return parts[0] + "//" + parts[2] + "/***"
	}

	return url
}

// createServices creates and returns the intent and fulfillment services for each chain
func createServices(clients map[uint64]*ethclient.Client, db db.Database, cfg *config.Config) (map[uint64]*services.IntentService, map[uint64]*services.FulfillmentService, map[uint64]*services.SettlementService, error) {
	intentServices := make(map[uint64]*services.IntentService)
	fulfillmentServices := make(map[uint64]*services.FulfillmentService)
	settlementServices := make(map[uint64]*services.SettlementService)

	// Create a client resolver for cross-chain operations
	clientResolver := services.NewSimpleClientResolver(clients)

	for chainID, client := range clients {
		// Create intent service
		intentService, err := services.NewIntentService(client, clientResolver, db, cfg.IntentInitiatedEventABI, chainID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create intent service for chain %d: %v", chainID, err)
		}
		intentServices[chainID] = intentService

		// Create fulfillment service
		fulfillmentService, err := services.NewFulfillmentService(client, clientResolver, db, cfg.IntentFulfilledEventABI, chainID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create fulfillment service for chain %d: %v", chainID, err)
		}
		fulfillmentServices[chainID] = fulfillmentService

		// Create settlement service
		settlementService, err := services.NewSettlementService(client, clientResolver, db, cfg.IntentSettledEventABI, chainID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create settlement service for chain %d: %v", chainID, err)
		}
		settlementServices[chainID] = settlementService
	}

	return intentServices, fulfillmentServices, settlementServices, nil
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	log.Println("Initializing database connection...")
	database, err := db.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()
	log.Println("Database connection established successfully")

	// Initialize Ethereum clients
	clients, err := createEthereumClients(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Ethereum clients: %v", err)
	}

	// Create services for all chains
	intentServices, fulfillmentServices, settlementServices, err := createServices(clients, database, cfg)
	if err != nil {
		log.Fatalf("Failed to create services: %v", err)
	}

	// Start event listeners for each chain
	ctx := context.Background()

	// Create event catchup service for this chain
	eventCatchupService := services.NewEventCatchupService(
		intentServices,
		fulfillmentServices,
		settlementServices,
		database,
	)
	err = eventCatchupService.StartListening(ctx)
	if err != nil {
		log.Printf("Failed to catchup on events: %v", err)
	}

	// Perform a simple diagnostic check on clients
	log.Println("Performing basic diagnostic checks on clients...")
	for chainID, client := range clients {
		// Test getting the latest block number as a diagnostic
		ctxTest, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		blockNum, err := client.BlockNumber(ctxTest)
		if err != nil {
			log.Printf("ERROR: Client for chain %d failed basic diagnosis: %v", chainID, err)
		} else {
			log.Printf("SUCCESS: Client for chain %d is functioning - current block: %d", chainID, blockNum)
		}
		cancel()
	}

	// Get the first chain's services for the HTTP server
	firstChainID := uint64(0)
	for chainID := range intentServices {
		firstChainID = chainID
		break
	}
	intentService := intentServices[firstChainID]
	fulfillmentService := fulfillmentServices[firstChainID]

	// Create and start the server
	server := handlers.NewServer(fulfillmentService, intentService, database)
	if err := server.Start(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
