package utils

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/zeta-chain/zetafast/api/config"
	"github.com/zeta-chain/zetafast/api/models"
)

var (
	// Address regex pattern (basic Ethereum address format)
	addressRegex = regexp.MustCompile(`^0x[a-fA-F0-9]{40}$`)

	// Amount regex pattern (positive integer)
	amountRegex = regexp.MustCompile(`^[0-9]+$`)

	// Bytes32 regex pattern (for intent IDs)
	bytes32Regex = regexp.MustCompile(`^0x[a-fA-F0-9]{64}$`)

	// Config instance for validation
	cfg *config.Config
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
	if chainID == 0 {
		return errors.New("chain ID cannot be zero")
	}

	fmt.Printf("Validating chain: %d, supported chains: %v\n", chainID, cfg.SupportedChains)

	// Check if chain is in supported chains from config
	for _, supported := range cfg.SupportedChains {
		if chainID == supported {
			return nil
		}
	}

	return fmt.Errorf("unsupported chain: %d", chainID)
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

	// Parse the amount as a big number
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
	if !IsValidBytes32(req.IntentID) {
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
	if err := ValidateAddress(req.Token); err != nil {
		fmt.Printf("Invalid token address: %s, error: %v\n", req.Token, err)
		return err
	}

	// Validate amount
	if err := ValidateAmount(req.Amount); err != nil {
		fmt.Printf("Invalid amount: %s, error: %v\n", req.Amount, err)
		return err
	}

	// Validate recipient address
	if err := ValidateAddress(req.Recipient); err != nil {
		fmt.Printf("Invalid recipient address: %s, error: %v\n", req.Recipient, err)
		return err
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

// IsValidBytes32 checks if a string is a valid bytes32 hash
func IsValidBytes32(hash string) bool {
	return bytes32Regex.MatchString(strings.ToLower(hash))
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
