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
	DefaultBlock  uint64
}

// Config holds all configuration values
type Config struct {
	Port                    string
	DatabaseURL             string
	SupportedChains         []uint64
	ChainConfigs            map[uint64]*ChainConfig
	IntentFulfilledEventABI string
	IntentInitiatedEventABI string
	IntentSettledEventABI   string
}

// IntentInitiatedEventABI is the ABI for the IntentInitiated and IntentInitiatedWithCall events
const IntentInitiatedEventABI = `[
	{
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
	},
	{
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
			},
			{
				"indexed": false,
				"internalType": "bytes",
				"name": "data",
				"type": "bytes"
			}
		],
		"name": "IntentInitiatedWithCall",
		"type": "event"
	}
]`

// IntentFulfilledEventABI is the ABI for the IntentFulfilled and IntentFulfilledWithCall events
const IntentFulfilledEventABI = `[
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "internalType": "bytes32", "name": "intentId", "type": "bytes32"},
			{"indexed": true, "internalType": "address", "name": "asset", "type": "address"},
			{"indexed": false, "internalType": "uint256", "name": "amount", "type": "uint256"},
			{"indexed": true, "internalType": "address", "name": "receiver", "type": "address"}
		],
		"name": "IntentFulfilled",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "internalType": "bytes32", "name": "intentId", "type": "bytes32"},
			{"indexed": true, "internalType": "address", "name": "asset", "type": "address"},
			{"indexed": false, "internalType": "uint256", "name": "amount", "type": "uint256"},
			{"indexed": true, "internalType": "address", "name": "receiver", "type": "address"},
			{"indexed": false, "internalType": "bytes", "name": "data", "type": "bytes"}
		],
		"name": "IntentFulfilledWithCall",
		"type": "event"
	}
]`

// IntentSettledEventABI is the ABI for the IntentSettled and IntentSettledWithCall events
const IntentSettledEventABI = `[
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "internalType": "bytes32", "name": "intentId", "type": "bytes32"},
			{"indexed": true, "internalType": "address", "name": "asset", "type": "address"},
			{"indexed": false, "internalType": "uint256", "name": "amount", "type": "uint256"},
			{"indexed": true, "internalType": "address", "name": "receiver", "type": "address"},
			{"indexed": false, "internalType": "bool", "name": "fulfilled", "type": "bool"},
			{"indexed": false, "internalType": "address", "name": "fulfiller", "type": "address"},
			{"indexed": false, "internalType": "uint256", "name": "actualAmount", "type": "uint256"},
			{"indexed": false, "internalType": "uint256", "name": "paidTip", "type": "uint256"}
		],
		"name": "IntentSettled",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "internalType": "bytes32", "name": "intentId", "type": "bytes32"},
			{"indexed": true, "internalType": "address", "name": "asset", "type": "address"},
			{"indexed": false, "internalType": "uint256", "name": "amount", "type": "uint256"},
			{"indexed": true, "internalType": "address", "name": "receiver", "type": "address"},
			{"indexed": false, "internalType": "bool", "name": "fulfilled", "type": "bool"},
			{"indexed": false, "internalType": "address", "name": "fulfiller", "type": "address"},
			{"indexed": false, "internalType": "uint256", "name": "actualAmount", "type": "uint256"},
			{"indexed": false, "internalType": "uint256", "name": "paidTip", "type": "uint256"},
			{"indexed": false, "internalType": "bytes", "name": "data", "type": "bytes"}
		],
		"name": "IntentSettledWithCall",
		"type": "event"
	}
]`

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Get supported chains
	supportedChainsStr := strings.Split(getEnvOrDefault("SUPPORTED_CHAINS", "42161,8453,137,1,43114,56,7000"), ",")
	supportedChains := make([]uint64, len(supportedChainsStr))
	for i, chain := range supportedChainsStr {
		chainID, err := strconv.ParseUint(chain, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid chain ID for %s: %v", chain, err)
		}
		supportedChains[i] = chainID
	}

	// Create chain configs map
	chainConfigs := make(map[uint64]*ChainConfig)

	// Load configurations for each chain
	for _, chainID := range supportedChains {
		var prefix string
		switch chainID {
		case 42161:
			prefix = "ARBITRUM"
		case 8453:
			prefix = "BASE"
		case 7000:
			prefix = "ZETACHAIN"
		case 137:
			prefix = "POLYGON"
		case 1:
			prefix = "ETHEREUM"
		case 56:
			prefix = "BSC"
		case 43114:
			prefix = "AVALANCHE"
		default:
			return nil, fmt.Errorf("unsupported chain ID: %d", chainID)
		}

		chainConfigs[chainID] = &ChainConfig{
			RPCURL:        getEnvOrDefault(fmt.Sprintf("%s_RPC_URL", prefix), ""),
			ContractAddr:  getEnvOrDefault(fmt.Sprintf("%s_INTENT_ADDR", prefix), ""),
			ChainID:       chainID,
			BlockInterval: int64(getEnvIntOrDefault(fmt.Sprintf("%s_BLOCK_INTERVAL", prefix), 1)),
			MaxRetries:    getEnvIntOrDefault(fmt.Sprintf("%s_MAX_RETRIES", prefix), 3),
			RetryDelay:    getEnvIntOrDefault(fmt.Sprintf("%s_RETRY_DELAY", prefix), 5),
			Confirmations: getEnvIntOrDefault(fmt.Sprintf("%s_CONFIRMATIONS", prefix), 1),
			DefaultBlock:  getEnvUint64OrDefault(fmt.Sprintf("%s_DEFAULT_BLOCK", prefix), 0),
		}
	}

	return &Config{
		Port:                    getEnvOrDefault("PORT", "8080"),
		DatabaseURL:             getEnvOrDefault("DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/speedrun?sslmode=disable"),
		SupportedChains:         supportedChains,
		ChainConfigs:            chainConfigs,
		IntentFulfilledEventABI: getEnvOrDefault("INTENT_FULFILLED_EVENT_ABI", IntentFulfilledEventABI),
		IntentInitiatedEventABI: getEnvOrDefault("INTENT_INITIATED_EVENT_ABI", IntentInitiatedEventABI),
		IntentSettledEventABI:   getEnvOrDefault("INTENT_SETTLED_EVENT_ABI", IntentSettledEventABI),
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

// getEnvUint64OrDefault gets an environment variable as a uint64 or returns a default value
func getEnvUint64OrDefault(key string, defaultValue uint64) uint64 {
	if value := os.Getenv(key); value != "" {
		if uintValue, err := strconv.ParseUint(value, 10, 64); err == nil {
			return uintValue
		}
	}
	return defaultValue
}
