package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/zeta-chain/zetafast/fulfiller/contracts"
)

// Config holds the configuration for the fulfiller service
type Config struct {
	APIEndpoint     string        `json:"apiEndpoint"`
	PollingInterval time.Duration `json:"pollingInterval"`
	PrivateKey      string        `json:"privateKey"`
	RPCURL          string        `json:"rpcUrl"`
	ContractAddress string        `json:"contractAddress"`
}

// Intent represents an intent from the API
type Intent struct {
	ID       string `json:"id"`
	Asset    string `json:"asset"`
	Amount   string `json:"amount"`
	Receiver string `json:"receiver"`
	Tip      string `json:"tip"`
}

// FulfillerService handles the intent fulfillment process
type FulfillerService struct {
	config     *Config
	client     *ethclient.Client
	auth       *bind.TransactOpts
	contract   *contracts.Intent
	httpClient *http.Client
}

// NewFulfillerService creates a new fulfiller service
func NewFulfillerService(config *Config) (*FulfillerService, error) {
	// Connect to Ethereum client
	client, err := ethclient.Dial(config.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %v", err)
	}

	// Create auth from private key
	privateKey, err := crypto.HexToECDSA(config.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %v", err)
	}

	// Initialize contract binding
	contract, err := contracts.NewIntent(common.HexToAddress(config.ContractAddress), client)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize contract: %v", err)
	}

	return &FulfillerService{
		config:     config,
		client:     client,
		auth:       auth,
		contract:   contract,
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
	tx, err := s.contract.Fulfill(s.auth, intentID, asset, amount, receiver)
	if err != nil {
		return fmt.Errorf("failed to fulfill intent: %v", err)
	}

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), s.client, tx)
	if err != nil {
		return fmt.Errorf("failed to wait for transaction: %v", err)
	}

	if receipt.Status == 0 {
		return fmt.Errorf("transaction failed")
	}

	log.Printf("Successfully fulfilled intent %s with transaction %s", intent.ID, tx.Hash().Hex())
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
	// Load configuration
	config := &Config{
		APIEndpoint:     "http://localhost:8080/api/v1/intents",
		PollingInterval: 5 * time.Second,
		PrivateKey:      os.Getenv("PRIVATE_KEY"),
		RPCURL:          os.Getenv("RPC_URL"),
		ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
	}

	if config.PrivateKey == "" || config.RPCURL == "" || config.ContractAddress == "" {
		log.Fatal("Missing required environment variables")
	}

	service, err := NewFulfillerService(config)
	if err != nil {
		log.Fatalf("Failed to create fulfiller service: %v", err)
	}

	ctx := context.Background()
	service.Start(ctx)
}
