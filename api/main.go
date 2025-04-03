package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gin-gonic/gin"
	"github.com/zeta-chain/zetafast/api/config"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/handlers"
	"github.com/zeta-chain/zetafast/api/services"
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

// startEventListeners initializes and starts the event listeners for intents and fulfillments
func startEventListeners(ctx context.Context, clients map[uint64]*ethclient.Client, db db.Database, cfg *config.Config) error {
	// Create services for each chain
	intentServices := make(map[uint64]*services.IntentService)
	for chainID, client := range clients {
		// Create intent service
		intentService, err := services.NewIntentService(client, db, cfg.IntentInitiatedEventABI, chainID)
		if err != nil {
			return fmt.Errorf("failed to create intent service for chain %d: %v", chainID, err)
		}
		intentServices[chainID] = intentService

		// Create fulfillment service
		fulfillmentService, err := services.NewFulfillmentService(client, db, cfg.IntentFulfilledEventABI, chainID)
		if err != nil {
			return fmt.Errorf("failed to create fulfillment service for chain %d: %v", chainID, err)
		}

		// Create event catchup service
		eventCatchupService := services.NewEventCatchupService(map[uint64]*services.IntentService{chainID: intentService}, fulfillmentService, db)

		// Start coordinated event listening
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)
		if err := eventCatchupService.StartListening(ctx, contractAddress); err != nil {
			return fmt.Errorf("failed to start event listening for chain %d: %v", chainID, err)
		}
	}

	return nil
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

	// Start event listeners
	ctx := context.Background()
	err = startEventListeners(ctx, clients, database, cfg)
	if err != nil {
		log.Fatalf("Failed to start event listeners: %v", err)
	}

	// Create services for the server
	intentService, err := services.NewIntentService(clients[1], database, cfg.IntentInitiatedEventABI, 1)
	if err != nil {
		log.Fatalf("Failed to create intent service: %v", err)
	}

	fulfillmentService, err := services.NewFulfillmentService(clients[1], database, cfg.IntentFulfilledEventABI, 1)
	if err != nil {
		log.Fatalf("Failed to create fulfillment service: %v", err)
	}

	// Create and start the server
	server := handlers.NewServer(fulfillmentService, intentService)
	if err := server.Start(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
