package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/zeta-chain/zetafast/api/config"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/handlers"
	"github.com/zeta-chain/zetafast/api/services"
)

// createEthereumClients creates and returns a map of Ethereum clients for each chain
func createEthereumClients(cfg *config.Config) (map[uint64]*ethclient.Client, error) {
	clients := make(map[uint64]*ethclient.Client)
	for chainID, chainConfig := range cfg.ChainConfigs {
		client, err := ethclient.Dial(chainConfig.RPCURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to chain %d: %v", chainID, err)
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
func startEventListeners(ctx context.Context, clients map[uint64]*ethclient.Client, contractAddresses map[uint64]string, db *db.PostgresDB, cfg *config.Config) (*services.FulfillmentService, *services.IntentService, error) {
	// Create fulfillment service
	fulfillmentService, err := services.NewFulfillmentService(clients, contractAddresses, db, cfg.ContractABI)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create fulfillment service: %v", err)
	}

	// Create intent service
	intentService, err := services.NewIntentService(clients[cfg.ChainConfigs[0].ChainID], db, cfg.IntentInitiatedEventABI)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create intent service: %v", err)
	}

	// Start listening for intent events
	for chainID, chainConfig := range cfg.ChainConfigs {
		addr := common.HexToAddress(chainConfig.ContractAddr)
		if err := intentService.StartListening(ctx, addr); err != nil {
			log.Printf("Error starting intent service for chain %d: %v", chainID, err)
			continue
		}
		log.Printf("Started intent service for chain %d", chainID)
	}

	// Start listening for fulfillment events
	if err := fulfillmentService.StartListening(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to start listening for fulfillment events: %v", err)
	}

	return fulfillmentService, intentService, nil
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := db.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create Ethereum clients
	clients, err := createEthereumClients(cfg)
	if err != nil {
		log.Fatalf("Failed to create Ethereum clients: %v", err)
	}
	defer func() {
		for _, client := range clients {
			client.Close()
		}
	}()

	// Create contract addresses map
	contractAddresses := createContractAddresses(cfg)

	// Initialize handlers
	if err := handlers.InitHandlers(clients, contractAddresses, db); err != nil {
		log.Fatalf("Failed to initialize handlers: %v", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start event listeners
	fulfillmentService, intentService, err := startEventListeners(ctx, clients, contractAddresses, db, cfg)
	if err != nil {
		log.Fatalf("Failed to start event listeners: %v", err)
	}

	// Initialize HTTP server
	server := handlers.NewServer(fulfillmentService, intentService)
	go func() {
		if err := server.Start(fmt.Sprintf(":%s", cfg.Port)); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Println("Shutting down...")
	cancel()
	fulfillmentService.Stop()
}
