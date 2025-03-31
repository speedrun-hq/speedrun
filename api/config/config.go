package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Port string

	// Database configuration
	DatabaseURL string

	// ZetaChain configuration
	ZetaChainRPCURL  string
	ZetaChainChainID string

	// Supported chains configuration
	SupportedChains []string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		Port:             getEnvOrDefault("PORT", "8080"),
		DatabaseURL:      getEnvOrDefault("DATABASE_URL", "postgresql://localhost:5432/zetafast?sslmode=disable"),
		ZetaChainRPCURL:  getEnvOrDefault("ZETACHAIN_RPC_URL", "http://localhost:1317"),
		ZetaChainChainID: getEnvOrDefault("ZETACHAIN_CHAIN_ID", "testnet-1"),
		SupportedChains:  []string{"base", "polygon"}, // Default supported chains
	}

	return config, nil
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
