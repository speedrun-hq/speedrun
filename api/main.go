package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/zeta-chain/zetafast/api/config"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/handlers"
	"github.com/zeta-chain/zetafast/api/services"
)

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

// createContractAddresses creates a map of contract addresses for each chain
func createContractAddresses(cfg *config.Config) map[uint64]string {
	contractAddresses := make(map[uint64]string)
	for chainID, chainConfig := range cfg.ChainConfigs {
		contractAddresses[chainID] = chainConfig.ContractAddr
	}
	return contractAddresses
}

// startEventListeners initializes and starts the event listeners for intents and fulfillments
func startEventListeners(ctx context.Context, clients map[uint64]*ethclient.Client, contractAddresses map[uint64]string, db db.DBInterface, cfg *config.Config) (*services.FulfillmentService, *services.IntentService, error) {
	// Create default blocks map
	defaultBlocks := make(map[uint64]uint64)
	for chainID, chainConfig := range cfg.ChainConfigs {
		defaultBlocks[chainID] = chainConfig.DefaultBlock
	}

	// Create fulfillment service
	fulfillmentService, err := services.NewFulfillmentService(clients, contractAddresses, db, cfg.ContractABI, defaultBlocks)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create fulfillment service: %v", err)
	}

	// Create intent service for each chain
	var intentService *services.IntentService
	for chainID, client := range clients {
		intentService, err = services.NewIntentService(client, db, cfg.IntentInitiatedEventABI, chainID)
		if err != nil {
			log.Fatalf("Failed to create intent service for chain %d: %v", chainID, err)
		}

		// Start listening for intent events
		contractAddress := common.HexToAddress(cfg.ChainConfigs[chainID].ContractAddr)
		if err := intentService.StartListening(ctx, contractAddress); err != nil {
			log.Fatalf("Failed to start listening for intent events on chain %d: %v", chainID, err)
		}
	}

	// Start listening for fulfillment events
	if err := fulfillmentService.StartListening(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to start listening for fulfillment events: %v", err)
	}

	return fulfillmentService, intentService, nil
}

func main() {
	log.Println("Starting ZetaFast application...")

	// Load configuration
	log.Println("Loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Configuration loaded successfully. Supported chains: %v", cfg.SupportedChains)

	// Initialize database
	log.Println("Initializing database connection...")
	db, err := db.NewPostgresDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Println("Database connection established successfully")

	// Create Ethereum clients
	log.Println("Creating Ethereum clients...")
	clients, err := createEthereumClients(cfg)
	if err != nil {
		log.Fatalf("Failed to create Ethereum clients: %v", err)
	}
	defer func() {
		for _, client := range clients {
			client.Close()
		}
	}()
	log.Printf("Successfully created Ethereum clients for chains: %v", cfg.SupportedChains)

	// Create contract addresses map
	log.Println("Creating contract addresses map...")
	contractAddresses := createContractAddresses(cfg)
	log.Printf("Contract addresses map created: %v", contractAddresses)

	// Initialize handlers
	log.Println("Initializing HTTP handlers...")
	if err := handlers.InitHandlers(clients, contractAddresses, db); err != nil {
		log.Fatalf("Failed to initialize handlers: %v", err)
	}
	log.Println("HTTP handlers initialized successfully")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start event listeners
	log.Println("Starting event listeners...")
	fulfillmentService, intentService, err := startEventListeners(ctx, clients, contractAddresses, db, cfg)
	if err != nil {
		log.Fatalf("Failed to start event listeners: %v", err)
	}
	log.Println("Event listeners started successfully")

	// Initialize HTTP server
	log.Printf("Starting HTTP server on port %s...", cfg.Port)
	server := handlers.NewServer(fulfillmentService, intentService)
	go func() {
		if err := server.Start(fmt.Sprintf(":%s", cfg.Port)); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	log.Println("HTTP server started successfully")

	// Wait for interrupt signal
	log.Println("Application is running. Press Ctrl+C to stop.")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Println("Shutting down...")
	cancel()
	fulfillmentService.Stop()
	log.Println("Application shutdown complete")
}
