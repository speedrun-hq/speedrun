package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/zeta-chain/zetafast/fulfiller/contracts"
)

// ChainConfig holds the configuration for a specific chain
type ChainConfig struct {
	RPCURL          string
	ContractAddress string
	Client          *ethclient.Client
	Contract        *contracts.Intent
	Auth            *bind.TransactOpts
}

// Config holds the configuration for the fulfiller service
type Config struct {
	APIEndpoint     string                  `json:"apiEndpoint"`
	PollingInterval time.Duration           `json:"pollingInterval"`
	PrivateKey      string                  `json:"privateKey"`
	Chains          map[string]*ChainConfig `json:"chains"`
}

// Intent represents an intent from the API
type Intent struct {
	ID       string `json:"id"`
	Asset    string `json:"asset"`
	Amount   string `json:"amount"`
	Receiver string `json:"receiver"`
	Tip      string `json:"tip"`
	Chain    string `json:"chain"` // Added chain field
}

// FulfillerService handles the intent fulfillment process
type FulfillerService struct {
	config     *Config
	httpClient *http.Client
	mu         sync.Mutex
}

// NewFulfillerService creates a new fulfiller service
func NewFulfillerService(config *Config) (*FulfillerService, error) {
	// Initialize chain configurations
	for chainName, chainConfig := range config.Chains {
		// Connect to Ethereum client
		client, err := ethclient.Dial(chainConfig.RPCURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to %s client: %v", chainName, err)
		}

		// Create auth from private key
		privateKey, err := crypto.HexToECDSA(config.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}

		chainID, err := client.ChainID(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to get chain ID for %s: %v", chainName, err)
		}

		auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create transactor for %s: %v", chainName, err)
		}

		// Initialize contract binding
		contract, err := contracts.NewIntent(common.HexToAddress(chainConfig.ContractAddress), client)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize contract for %s: %v", chainName, err)
		}

		chainConfig.Client = client
		chainConfig.Contract = contract
		chainConfig.Auth = auth
	}

	return &FulfillerService{
		config:     config,
		httpClient: &http.Client{},
	}, nil
}

// fetchPendingIntents gets pending intents from the API
func (s *FulfillerService) fetchPendingIntents() ([]Intent, error) {
	resp, err := s.httpClient.Get(s.config.APIEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch intents: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var intents []Intent
	if err := json.NewDecoder(resp.Body).Decode(&intents); err != nil {
		return nil, fmt.Errorf("failed to decode intents: %v", err)
	}

	return intents, nil
}

// fulfillIntent attempts to fulfill a single intent
func (s *FulfillerService) fulfillIntent(intent Intent) error {
	s.mu.Lock()
	chainConfig, exists := s.config.Chains[intent.Chain]
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("unsupported chain: %s", intent.Chain)
	}

	// Convert intent ID to bytes32
	intentID := common.HexToHash(intent.ID)

	// Convert amount to big.Int
	amount, ok := new(big.Int).SetString(intent.Amount, 10)
	if !ok {
		return fmt.Errorf("invalid amount: %s", intent.Amount)
	}

	// Convert addresses
	asset := common.HexToAddress(intent.Asset)
	receiver := common.HexToAddress(intent.Receiver)

	// Call the contract's fulfill function
	tx, err := chainConfig.Contract.Fulfill(chainConfig.Auth, intentID, asset, amount, receiver)
	if err != nil {
		return fmt.Errorf("failed to fulfill intent on %s: %v", intent.Chain, err)
	}

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), chainConfig.Client, tx)
	if err != nil {
		return fmt.Errorf("failed to wait for transaction on %s: %v", intent.Chain, err)
	}

	if receipt.Status == 0 {
		return fmt.Errorf("transaction failed on %s", intent.Chain)
	}

	log.Printf("Successfully fulfilled intent %s on %s with transaction %s",
		intent.ID, intent.Chain, tx.Hash().Hex())
	return nil
}

// Start begins the fulfiller service
func (s *FulfillerService) Start(ctx context.Context) {
	ticker := time.NewTicker(s.config.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			intents, err := s.fetchPendingIntents()
			if err != nil {
				log.Printf("Error fetching intents: %v", err)
				continue
			}

			for _, intent := range intents {
				if err := s.fulfillIntent(intent); err != nil {
					log.Printf("Error fulfilling intent %s: %v", intent.ID, err)
				}
			}
		}
	}
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

	// Load configuration from environment variables
	pollingInterval, err := strconv.Atoi(os.Getenv("POLLING_INTERVAL"))
	if err != nil {
		pollingInterval = 5 // default value
	}

	// Initialize chain configurations
	chains := make(map[string]*ChainConfig)

	// BASE chain
	if rpcURL := os.Getenv("BASE_RPC_URL"); rpcURL != "" {
		chains["base"] = &ChainConfig{
			RPCURL:          rpcURL,
			ContractAddress: os.Getenv("BASE_CONTRACT_ADDRESS"),
		}
	}

	// Arbitrum chain
	if rpcURL := os.Getenv("ARBITRUM_RPC_URL"); rpcURL != "" {
		chains["arbitrum"] = &ChainConfig{
			RPCURL:          rpcURL,
			ContractAddress: os.Getenv("ARBITRUM_CONTRACT_ADDRESS"),
		}
	}

	config := &Config{
		APIEndpoint:     os.Getenv("API_ENDPOINT"),
		PollingInterval: time.Duration(pollingInterval) * time.Second,
		PrivateKey:      os.Getenv("PRIVATE_KEY"),
		Chains:          chains,
	}

	// Validate required environment variables
	if config.PrivateKey == "" {
		log.Fatal("PRIVATE_KEY environment variable is required")
	}
	if len(config.Chains) == 0 {
		log.Fatal("At least one chain configuration is required")
	}
	for chainName, chainConfig := range config.Chains {
		if chainConfig.ContractAddress == "" {
			log.Fatalf("CONTRACT_ADDRESS for %s chain is required", chainName)
		}
	}

	// Set default API endpoint if not provided
	if config.APIEndpoint == "" {
		config.APIEndpoint = "http://localhost:8080/api/v1/intents"
	}

	service, err := NewFulfillerService(config)
	if err != nil {
		log.Fatalf("Failed to create fulfiller service: %v", err)
	}

	ctx := context.Background()
	service.Start(ctx)
}
