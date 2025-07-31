package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
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
	// Periodic catchup configuration
	PeriodicCatchupInterval       int64 // in minutes
	PeriodicCatchupTimeout        int64 // in minutes
	PeriodicCatchupLookbackBlocks uint64
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Get supported chains
	supportedChainsStr := strings.Split(getEnvOrDefault("SUPPORTED_CHAINS", mainnetDefaultChains), ",")
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
		prefix, err := chainNameFromID(chainID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get chain name for chain ID %d", chainID)
		}

		intentAddr, ok := intentAddressByChain[chainID]
		if !ok {
			return nil, fmt.Errorf("no intent address configured for chain ID %d", chainID)
		}

		chainConfigs[chainID] = &ChainConfig{
			RPCURL:        getEnvOrDefault(fmt.Sprintf("%s_RPC_URL", prefix), ""),
			ContractAddr:  intentAddr,
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
		IntentFulfilledEventABI: IntentFulfilledEventABI,
		IntentInitiatedEventABI: IntentInitiatedEventABI,
		IntentSettledEventABI:   IntentSettledEventABI,
		// Periodic catchup configuration with defaults
		PeriodicCatchupInterval:       int64(getEnvIntOrDefault("PERIODIC_CATCHUP_INTERVAL_MINUTES", 30)),
		PeriodicCatchupTimeout:        int64(getEnvIntOrDefault("PERIODIC_CATCHUP_TIMEOUT_MINUTES", 15)),
		PeriodicCatchupLookbackBlocks: getEnvUint64OrDefault("PERIODIC_CATCHUP_LOOKBACK_BLOCKS", 1000),
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
