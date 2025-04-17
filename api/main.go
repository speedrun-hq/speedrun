package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
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

		// For historical operations like catch-up, HTTP is more reliable than WebSocket
		// This avoids WebSocket connection drops during long-running operations
		httpURL := strings.Replace(chainConfig.RPCURL, "wss://", "https://", 1)
		httpURL = strings.Replace(httpURL, "ws://", "http://", 1)

		// Log the RPC URL for debugging
		log.Printf("Connecting to chain %d using RPC URL: %s", chainID, httpURL)

		// Try to establish connection with retries
		maxRetries := 3
		var lastErr error

		for attempt := 0; attempt < maxRetries; attempt++ {
			client, err = ethclient.Dial(httpURL)
			if err == nil {
				// Successfully connected
				break
			}

			lastErr = err
			if attempt < maxRetries-1 {
				// Wait before retrying with exponential backoff
				retryDelay := time.Duration(5*(attempt+1)) * time.Second
				log.Printf("Failed to connect to chain %d (attempt %d/%d): %v. Retrying in %v...",
					chainID, attempt+1, maxRetries, err, retryDelay)
				time.Sleep(retryDelay)
			}
		}

		if err != nil {
			return nil, fmt.Errorf("failed to connect to chain %d after %d attempts: %v",
				chainID, maxRetries, lastErr)
		}

		clients[chainID] = client
		log.Printf("Successfully connected to chain %d", chainID)
	}
	return clients, nil
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
