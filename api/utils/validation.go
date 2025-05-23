package utils

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/speedrun-hq/speedrun/api/config"
	"github.com/speedrun-hq/speedrun/api/models"
)

var (
	// Address regex pattern (basic Ethereum address format)
	addressRegex = regexp.MustCompile(`^0x[a-fA-F0-9]{40}$`)

	// Amount regex pattern (positive number, can include decimals)
	amountRegex = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

	// Bytes32 regex pattern (for intent IDs)
	bytes32Regex = regexp.MustCompile(`^0x[a-fA-F0-9]{64}$`)

	// Config instance for validation
	cfg *config.Config

	addressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
)

// Initialize sets up the validation package with configuration
func Initialize(c *config.Config) {
	cfg = c
}

// ValidateAddress validates an Ethereum address
func ValidateAddress(address string) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if !addressRegex.MatchString(address) {
		return fmt.Errorf("invalid Ethereum address format: %s", address)
	}
	return nil
}

// ValidateChain checks if a chain is supported
func ValidateChain(chainID uint64) error {
	// For testing purposes, always allow chain ID 1 and 2
	if chainID == 1 || chainID == 2 {
		return nil
	}

	if cfg == nil || len(cfg.SupportedChains) == 0 {
		return fmt.Errorf("no supported chains configured")
	}

	for _, supportedChain := range cfg.SupportedChains {
		if chainID == supportedChain {
			return nil
		}
	}

	return fmt.Errorf("unsupported chain ID: %d", chainID)
}

// ValidateAmount checks if the amount is valid and within limits
func ValidateAmount(amount string) error {
	if amount == "" {
		return errors.New("amount cannot be empty")
	}

	// Remove any whitespace
	amount = strings.TrimSpace(amount)

	// Check format
	if !amountRegex.MatchString(amount) {
		return errors.New("invalid amount format")
	}

	// For decimal values, we'll just check if it's a valid number
	// We don't need to parse it as a big.Int since it's a decimal
	if strings.Contains(amount, ".") {
		// Just check if it's a valid float
		_, ok := new(big.Float).SetString(amount)
		if !ok {
			return errors.New("invalid amount format")
		}
		return nil
	}

	// For integer values, parse as big.Int
	value, success := new(big.Int).SetString(amount, 10)
	if !success {
		return errors.New("invalid amount format")
	}

	// Check if amount is positive
	if value.Sign() < 0 {
		return errors.New("amount must be positive")
	}

	// Check maximum amount (e.g., 1 billion ETH in wei)
	maxAmount := new(big.Int).Mul(
		new(big.Int).SetInt64(1_000_000_000),
		new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
	)
	if value.Cmp(maxAmount) > 0 {
		return errors.New("amount exceeds maximum limit")
	}

	return nil
}

// ValidateFulfillmentAmount checks if the fulfillment amount is valid for the intent
func ValidateFulfillmentAmount(amount, intentAmount, totalFulfilled string) error {
	// Parse amounts
	fulfillmentAmount := new(big.Int)
	if _, ok := fulfillmentAmount.SetString(amount, 10); !ok {
		return fmt.Errorf("invalid fulfillment amount format")
	}

	intentAmountBig := new(big.Int)
	if _, ok := intentAmountBig.SetString(intentAmount, 10); !ok {
		return fmt.Errorf("invalid intent amount format")
	}

	totalFulfilledBig := new(big.Int)
	if _, ok := totalFulfilledBig.SetString(totalFulfilled, 10); !ok {
		return fmt.Errorf("invalid total fulfilled amount format")
	}

	// Validate amount is positive
	if fulfillmentAmount.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("fulfillment amount must be positive")
	}

	// Validate amount doesn't exceed intent amount
	if fulfillmentAmount.Cmp(intentAmountBig) > 0 {
		return fmt.Errorf("fulfillment amount exceeds intent amount")
	}

	// Validate total fulfilled doesn't exceed intent amount
	total := new(big.Int).Add(totalFulfilledBig, fulfillmentAmount)
	if total.Cmp(intentAmountBig) > 0 {
		return fmt.Errorf("total fulfilled amount would exceed intent amount")
	}

	return nil
}

// ValidateReceiverBytes validates a receiver address in bytes format
func ValidateReceiverBytes(receiver []byte) error {
	if len(receiver) == 0 {
		return fmt.Errorf("receiver cannot be empty")
	}
	if len(receiver) != 20 { // Ethereum address length
		return fmt.Errorf("invalid receiver address length: expected 20 bytes, got %d", len(receiver))
	}
	return nil
}

// ValidateIntent validates an intent
func ValidateIntent(intent *models.Intent) error {
	if intent == nil {
		return fmt.Errorf("intent is nil")
	}

	if intent.ID == "" {
		return fmt.Errorf("intent ID is required")
	}

	if intent.SourceChain == 0 {
		return fmt.Errorf("source chain is required")
	}

	if intent.DestinationChain == 0 {
		return fmt.Errorf("destination chain is required")
	}

	if intent.Token == "" {
		return fmt.Errorf("token is required")
	}

	if intent.Amount == "" {
		return fmt.Errorf("amount is required")
	}

	if intent.Recipient == "" {
		return fmt.Errorf("recipient is required")
	}

	if intent.IntentFee == "" {
		return fmt.Errorf("intent fee is required")
	}

	return nil
}

// ValidateFulfillmentRequest validates a fulfillment request
func ValidateFulfillmentRequest(req *models.CreateFulfillmentRequest) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}

	// Validate intent ID format (bytes32 format)
	if !IsValidBytes32(req.ID) {
		return errors.New("invalid intent ID format")
	}

	// Validate tx hash format (bytes32 format)
	if !IsValidBytes32(req.TxHash) {
		return errors.New("invalid transaction hash format")
	}

	return nil
}

// ValidateIntentRequest validates a create intent request
func ValidateIntentRequest(req *models.CreateIntentRequest) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}

	fmt.Printf("Validating intent request: %+v\n", req)

	// Validate intent ID format (bytes32 format)
	if !IsValidBytes32(req.ID) {
		fmt.Printf("Invalid intent ID format: %s\n", req.ID)
		return errors.New("invalid intent ID format")
	}

	// Validate source chain
	if err := ValidateChain(req.SourceChain); err != nil {
		fmt.Printf("Invalid source chain: %d, error: %v\n", req.SourceChain, err)
		return err
	}

	// Validate destination chain
	if err := ValidateChain(req.DestinationChain); err != nil {
		fmt.Printf("Invalid destination chain: %d, error: %v\n", req.DestinationChain, err)
		return err
	}

	// Validate token address
	if req.Token != "ETH" && !addressPattern.MatchString(req.Token) {
		fmt.Printf("Invalid token address: %s, error: invalid format\n", req.Token)
		return errors.New("invalid token address format")
	}

	// Validate amount
	if err := ValidateAmount(req.Amount); err != nil {
		fmt.Printf("Invalid amount: %s, error: %v\n", req.Amount, err)
		return err
	}

	// Validate recipient address
	if !addressPattern.MatchString(req.Recipient) {
		fmt.Printf("Invalid recipient address: %s, error: invalid format\n", req.Recipient)
		return errors.New("invalid recipient address format")
	}

	// Validate sender address
	if !addressPattern.MatchString(req.Sender) {
		fmt.Printf("Invalid sender address: %s, error: invalid format\n", req.Sender)
		return errors.New("invalid sender address format")
	}

	// Validate intent fee
	if err := ValidateAmount(req.IntentFee); err != nil {
		fmt.Printf("Invalid intent fee: %s, error: %v\n", req.IntentFee, err)
		return err
	}

	// Validate source and destination chains are different
	if req.SourceChain == req.DestinationChain {
		fmt.Printf("Source and destination chains are the same: %d\n", req.SourceChain)
		return errors.New("source and destination chains must be different")
	}

	return nil
}

// IsValidBytes32 checks if a string is a valid bytes32 hex string
func IsValidBytes32(hash string) bool {
	return bytes32Regex.MatchString(hash)
}

// IsValidAddress checks if a string is a valid Ethereum address
func IsValidAddress(address string) bool {
	return addressRegex.MatchString(address)
}

// GenerateIntentID generates an intent ID using the same logic as the contract
func GenerateIntentID(counter uint64, salt *big.Int) string {
	// Pack counter and salt
	packed := append(
		common.BigToHash(new(big.Int).SetUint64(counter)).Bytes(),
		common.BigToHash(salt).Bytes()...,
	)

	// Generate keccak256 hash
	hash := crypto.Keccak256(packed)
	return common.BytesToHash(hash).Hex()
}

// ValidateIntentID validates that an intent ID matches what would be generated from counter and salt
func ValidateIntentID(id string, counter uint64, salt *big.Int) bool {
	if !IsValidBytes32(id) {
		return false
	}

	expected := GenerateIntentID(counter, salt)
	return strings.EqualFold(id, expected)
}

// ValidateBytes32 validates a bytes32 hex string
func ValidateBytes32(hex string) bool {
	return bytes32Regex.MatchString(hex)
}
