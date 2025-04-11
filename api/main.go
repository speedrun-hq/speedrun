package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

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

		if strings.HasPrefix(chainConfig.RPCURL, "wss://") {
			rpcClient, err := rpc.DialWebsocket(context.Background(), chainConfig.RPCURL, "")
			if err != nil {
				return nil, fmt.Errorf("failed to create RPC client for chain %d: %v", chainID, err)
			}
			client = ethclient.NewClient(rpcClient)
		} else {
			client, err = ethclient.Dial(chainConfig.RPCURL)
			if err != nil {
				return nil, fmt.Errorf("failed to connect to chain %d: %v", chainID, err)
			}
		}
		clients[chainID] = client
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
	server := handlers.NewServer(fulfillmentService, intentService)
	if err := server.Start(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
