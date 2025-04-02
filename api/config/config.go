package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// ChainConfig represents configuration for a specific chain
type ChainConfig struct {
	RPCURL        string
	ContractAddr  string
	ChainID       uint64
	BlockInterval int64
	MaxRetries    int
	RetryDelay    int
	Confirmations int
}

// Config holds all configuration values
type Config struct {
	Port                    string
	DatabaseURL             string
	SupportedChains         []string
	ChainConfigs            map[uint64]*ChainConfig
	ContractABI             string
	IntentInitiatedEventABI string
}

// IntentInitiatedEventABI is the ABI for the IntentInitiated event
const IntentInitiatedEventABI = `[{
	"anonymous": false,
	"inputs": [
		{
			"indexed": true,
			"internalType": "bytes32",
			"name": "intentId",
			"type": "bytes32"
		},
		{
			"indexed": true,
			"internalType": "address",
			"name": "asset",
			"type": "address"
		},
		{
			"indexed": false,
			"internalType": "uint256",
			"name": "amount",
			"type": "uint256"
		},
		{
			"indexed": false,
			"internalType": "uint256",
			"name": "targetChain",
			"type": "uint256"
		},
		{
			"indexed": false,
			"internalType": "bytes",
			"name": "receiver",
			"type": "bytes"
		},
		{
			"indexed": false,
			"internalType": "uint256",
			"name": "tip",
			"type": "uint256"
		},
		{
			"indexed": false,
			"internalType": "uint256",
			"name": "salt",
			"type": "uint256"
		}
	],
	"name": "IntentInitiated",
	"type": "event"
}]`

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Get supported chains
	supportedChains := strings.Split(getEnvOrDefault("SUPPORTED_CHAINS", "zeta,arbitrum,base"), ",")

	// Create chain configs map
	chainConfigs := make(map[uint64]*ChainConfig)

	// Load configurations for each chain
	for _, chain := range supportedChains {
		chainID, err := strconv.ParseUint(getEnvOrDefault(fmt.Sprintf("%s_CHAIN_ID", strings.ToUpper(chain)), "0"), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid chain ID for %s: %v", chain, err)
		}

		chainConfigs[chainID] = &ChainConfig{
			RPCURL:        getEnvOrDefault(fmt.Sprintf("%s_RPC_URL", strings.ToUpper(chain)), ""),
			ContractAddr:  getEnvOrDefault(fmt.Sprintf("%s_CONTRACT_ADDR", strings.ToUpper(chain)), ""),
			ChainID:       chainID,
			BlockInterval: int64(getEnvIntOrDefault(fmt.Sprintf("%s_BLOCK_INTERVAL", strings.ToUpper(chain)), 1)),
			MaxRetries:    getEnvIntOrDefault(fmt.Sprintf("%s_MAX_RETRIES", strings.ToUpper(chain)), 3),
			RetryDelay:    getEnvIntOrDefault(fmt.Sprintf("%s_RETRY_DELAY", strings.ToUpper(chain)), 5),
			Confirmations: getEnvIntOrDefault(fmt.Sprintf("%s_CONFIRMATIONS", strings.ToUpper(chain)), 1),
		}
	}

	// Default contract ABI
	defaultABI := `[{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "bytes32",
				"name": "intentId",
				"type": "bytes32"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "asset",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "receiver",
				"type": "address"
			}
		],
		"name": "IntentFulfilled",
		"type": "event"
	}]`

	return &Config{
		Port:                    getEnvOrDefault("PORT", "8080"),
		DatabaseURL:             getEnvOrDefault("DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/zetafast?sslmode=disable"),
		SupportedChains:         supportedChains,
		ChainConfigs:            chainConfigs,
		ContractABI:             getEnvOrDefault("CONTRACT_ABI", defaultABI),
		IntentInitiatedEventABI: getEnvOrDefault("INTENT_INITIATED_EVENT_ABI", IntentInitiatedEventABI),
	}, nil
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntOrDefault gets an environment variable as an integer or returns a default value
func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
