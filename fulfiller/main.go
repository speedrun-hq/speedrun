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
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/zeta-chain/zetafast/fulfiller/contracts"
)

// ChainConfig holds the configuration for a specific chain
type ChainConfig struct {
	RPCURL        string
	IntentAddress string
	Client        *ethclient.Client
	Contract      *contracts.Intent
	Auth          *bind.TransactOpts
}

// Config holds the configuration for the fulfiller service
type Config struct {
	APIEndpoint     string               `json:"apiEndpoint"`
	PollingInterval time.Duration        `json:"pollingInterval"`
	PrivateKey      string               `json:"privateKey"`
	Chains          map[int]*ChainConfig `json:"chains"`
}

// Intent represents an intent from the API
type Intent struct {
	ID               string    `json:"id"`
	SourceChain      int       `json:"source_chain"`
	DestinationChain int       `json:"destination_chain"`
	Token            string    `json:"token"`
	Amount           string    `json:"amount"`
	Recipient        string    `json:"recipient"`
	IntentFee        string    `json:"intent_fee"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
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
	for chainID, chainConfig := range config.Chains {
		// Connect to Ethereum client
		client, err := ethclient.Dial(chainConfig.RPCURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to %d client: %v", chainID, err)
		}

		// Create auth from private key
		privateKey, err := crypto.HexToECDSA(config.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}

		chainID, err := client.ChainID(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to get chain ID for %d: %v", chainID, err)
		}

		auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create transactor for %d: %v", chainID, err)
		}

		// Initialize contract binding
		contract, err := contracts.NewIntent(common.HexToAddress(chainConfig.IntentAddress), client)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize contract for %d: %v", chainID, err)
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
	resp, err := s.httpClient.Get(s.config.APIEndpoint + "/api/v1/intents")
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

	// Only return intents that are pending
	intents = filterPendingIntents(intents)

	return intents, nil
}

// filterPendingIntents filters intents that are pending
func filterPendingIntents(intents []Intent) []Intent {
	pendingIntents := []Intent{}
	for _, intent := range intents {
		if intent.Status == "pending" {
			pendingIntents = append(pendingIntents, intent)
		}
	}
	log.Printf("Found %d pending intents", len(pendingIntents))
	return pendingIntents
}

// fulfillIntent attempts to fulfill a single intent
func (s *FulfillerService) fulfillIntent(intent Intent) error {
	s.mu.Lock()
	chainConfig, exists := s.config.Chains[intent.DestinationChain]
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("destination chain configuration not found for: %d", intent.DestinationChain)
	}

	// Convert intent ID to bytes32
	intentID := common.HexToHash(intent.ID)

	// Convert amount to big.Int
	amount, ok := new(big.Int).SetString(intent.Amount, 10)
	if !ok {
		return fmt.Errorf("invalid amount: %s", intent.Amount)
	}

	// Add intent fee to amount
	intentFee, ok := new(big.Int).SetString(intent.IntentFee, 10)
	if !ok {
		return fmt.Errorf("invalid intent fee: %s", intent.IntentFee)
	}
	amount.Add(amount, intentFee)
	approveAmount := new(big.Int).Add(amount, big.NewInt(1000000))

	log.Printf("Fulfilling intent %s on chain %d with amount %s", intent.ID, intent.DestinationChain, amount.String())

	// Convert addresses
	receiver := common.HexToAddress(intent.Recipient)

	// Get the Intent contract address
	intentAddress := common.HexToAddress(chainConfig.IntentAddress)

	// First, approve the token transfer
	// We need to approve the Intent contract to spend our tokens
	erc20ABI, err := abi.JSON(strings.NewReader(`[
		{
			"constant": true,
			"inputs": [
				{
					"name": "_owner",
					"type": "address"
				},
				{
					"name": "_spender",
					"type": "address"
				}
			],
			"name": "allowance",
			"outputs": [
				{
					"name": "",
					"type": "uint256"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": false,
			"inputs": [
				{
					"name": "_spender",
					"type": "address"
				},
				{
					"name": "_value",
					"type": "uint256"
				}
			],
			"name": "approve",
			"outputs": [
				{
					"name": "",
					"type": "bool"
				}
			],
			"payable": false,
			"stateMutability": "nonpayable",
			"type": "function"
		}
	]`))
	if err != nil {
		return fmt.Errorf("failed to parse ERC20 ABI: %v", err)
	}

	var tokenAddress common.Address
	if intent.DestinationChain == 8453 {
		tokenAddress = common.HexToAddress(os.Getenv("BASE_USDC_ADDRESS"))
	} else if intent.DestinationChain == 42161 {
		tokenAddress = common.HexToAddress(os.Getenv("ARBITRUM_USDC_ADDRESS"))
	} else {
		return fmt.Errorf("unsupported chain: %d", intent.DestinationChain)
	}
	// Create ERC20 contract binding
	erc20Contract := bind.NewBoundContract(
		tokenAddress,
		erc20ABI,
		chainConfig.Client,
		chainConfig.Client,
		chainConfig.Client,
	)

	// Send the approve transaction
	approveTx, err := erc20Contract.Transact(chainConfig.Auth, "approve", intentAddress, approveAmount)
	if err != nil {
		return fmt.Errorf("failed to approve token transfer: %v", err)
	}

	// Wait for the approve transaction to be mined
	approveReceipt, err := bind.WaitMined(context.Background(), chainConfig.Client, approveTx)
	if err != nil {
		return fmt.Errorf("failed to wait for approve transaction: %v", err)
	}

	if approveReceipt.Status == 0 {
		return fmt.Errorf("approve transaction failed")
	}

	log.Printf("Approved token transfer for intent %s on chain %d", intent.ID, intent.DestinationChain)

	// Now call the contract's fulfill function
	tx, err := chainConfig.Contract.Fulfill(chainConfig.Auth, intentID, tokenAddress, amount, receiver)
	if err != nil {
		return fmt.Errorf("failed to fulfill intent on %d: %v", intent.DestinationChain, err)
	}

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), chainConfig.Client, tx)
	if err != nil {
		return fmt.Errorf("failed to wait for transaction on %d: %v", intent.DestinationChain, err)
	}

	if receipt.Status == 0 {
		return fmt.Errorf("transaction failed on %d", intent.DestinationChain)
	}

	log.Printf("Successfully fulfilled intent %s on %d with transaction %s",
		intent.ID, intent.DestinationChain, tx.Hash().Hex())
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
	chains := make(map[int]*ChainConfig)

	// BASE chain
	if rpcURL := os.Getenv("BASE_RPC_URL"); rpcURL != "" {
		chains[8453] = &ChainConfig{
			RPCURL:        rpcURL,
			IntentAddress: os.Getenv("BASE_INTENT_ADDRESS"),
		}
	}

	// Arbitrum chain
	if rpcURL := os.Getenv("ARBITRUM_RPC_URL"); rpcURL != "" {
		chains[42161] = &ChainConfig{
			RPCURL:        rpcURL,
			IntentAddress: os.Getenv("ARBITRUM_INTENT_ADDRESS"),
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
	for chainID, chainConfig := range config.Chains {
		if chainConfig.IntentAddress == "" {
			log.Fatalf("%d_INTENT_ADDRESS for %d chain is required", chainID, chainID)
		}
	}

	// Set default API endpoint if not provided
	if config.APIEndpoint == "" {
		config.APIEndpoint = "http://localhost:8080"
	}

	service, err := NewFulfillerService(config)
	if err != nil {
		log.Fatalf("Failed to create fulfiller service: %v", err)
	}

	ctx := context.Background()
	service.Start(ctx)
}
