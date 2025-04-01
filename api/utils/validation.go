package utils

import (
	"errors"
	"math/big"
	"regexp"
	"strings"

	"github.com/zeta-chain/zetafast/api/config"
	"github.com/zeta-chain/zetafast/api/models"
)

var (
	// Address regex pattern (basic Ethereum address format)
	addressRegex = regexp.MustCompile(`^0x[a-fA-F0-9]{40}$`)

	// Amount regex pattern (positive decimal with up to 18 decimal places)
	amountRegex = regexp.MustCompile(`^[0-9]+\.?[0-9]{0,18}$`)

	// UUID regex pattern
	uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	// Config instance for validation
	cfg *config.Config
)

// Initialize sets up the validation package with configuration
func Initialize(c *config.Config) {
	cfg = c
}

// ValidateAddress checks if the address is in a valid format
func ValidateAddress(address string) error {
	if address == "" {
		return errors.New("address cannot be empty")
	}

	if !addressRegex.MatchString(address) {
		return errors.New("invalid address format")
	}

	return nil
}

// ValidateChain checks if the chain is supported
func ValidateChain(chain string) error {
	if chain == "" {
		return errors.New("chain cannot be empty")
	}

	chain = strings.ToLower(chain)

	// Check if chain is in supported chains from config
	for _, supported := range cfg.SupportedChains {
		if chain == strings.ToLower(supported) {
			return nil
		}
	}

	return errors.New("unsupported chain")
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
	value, success := new(big.Float).SetString(amount)
	if !success {
		return errors.New("invalid amount format")
	}

	// Check if amount is positive
	if value.Sign() < 0 {
		return errors.New("amount must be positive")
	}

	// Check maximum amount (e.g., 1 billion)
	maxAmount := new(big.Float).SetInt64(1_000_000_000)
	if value.Cmp(maxAmount) > 0 {
		return errors.New("amount exceeds maximum limit")
	}

	return nil
}

// ValidateFulfillmentAmount checks if the fulfillment amount is valid for the intent
func ValidateFulfillmentAmount(amount, intentAmount, totalFulfilled string) error {
	if err := ValidateAmount(amount); err != nil {
		return err
	}

	// Convert all amounts to big numbers for comparison
	fulfillmentAmount, _ := new(big.Float).SetString(amount)
	intentTotal, _ := new(big.Float).SetString(intentAmount)
	alreadyFulfilled, _ := new(big.Float).SetString(totalFulfilled)

	// Calculate remaining amount
	remaining := new(big.Float).Sub(intentTotal, alreadyFulfilled)

	// Check if fulfillment amount exceeds remaining amount
	if fulfillmentAmount.Cmp(remaining) > 0 {
		return errors.New("fulfillment amount exceeds remaining amount")
	}

	// Check minimum fulfillment amount (e.g., 0.000001)
	minAmount := new(big.Float).SetFloat64(0.000001)
	if fulfillmentAmount.Cmp(minAmount) < 0 {
		return errors.New("fulfillment amount below minimum limit")
	}

	return nil
}

// ValidateIntent checks if an intent is valid for fulfillment
func ValidateIntent(intent *models.Intent) error {
	if intent == nil {
		return errors.New("intent not found")
	}

	if intent.Status != models.IntentStatusPending {
		return errors.New("intent is not in pending status")
	}

	// Validate source chain
	if err := ValidateChain(intent.SourceChain); err != nil {
		return err
	}

	// Validate destination chain
	if err := ValidateChain(intent.DestinationChain); err != nil {
		return err
	}

	// Validate recipient address
	if err := ValidateAddress(intent.Recipient); err != nil {
		return err
	}

	// Validate amount
	if err := ValidateAmount(intent.Amount); err != nil {
		return err
	}

	// Validate intent fee
	if err := ValidateAmount(intent.IntentFee); err != nil {
		return err
	}

	// Validate source and destination chains are different
	if intent.SourceChain == intent.DestinationChain {
		return errors.New("source and destination chains must be different")
	}

	return nil
}

// IsValidUUID checks if a string is a valid UUID
func IsValidUUID(uuid string) bool {
	return uuidRegex.MatchString(strings.ToLower(uuid))
}

// ValidateFulfillmentRequest validates a fulfillment request
func ValidateFulfillmentRequest(req *models.CreateFulfillmentRequest) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}

	// Validate fulfiller address
	if err := ValidateAddress(req.Fulfiller); err != nil {
		return err
	}

	// Validate amount
	if err := ValidateAmount(req.Amount); err != nil {
		return err
	}

	// Validate intent ID format (UUID)
	if !IsValidUUID(req.IntentID) {
		return errors.New("invalid intent ID format")
	}

	return nil
}

// ValidateIntentRequest validates a create intent request
func ValidateIntentRequest(req *models.CreateIntentRequest) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}

	// Validate source chain
	if err := ValidateChain(req.SourceChain); err != nil {
		return err
	}

	// Validate destination chain
	if err := ValidateChain(req.DestinationChain); err != nil {
		return err
	}

	// Validate token
	if req.Token != "USDC" {
		return errors.New("only USDC token is supported")
	}

	// Validate amount
	if err := ValidateAmount(req.Amount); err != nil {
		return err
	}

	// Validate recipient address
	if err := ValidateAddress(req.Recipient); err != nil {
		return err
	}

	// Validate intent fee
	if err := ValidateAmount(req.IntentFee); err != nil {
		return err
	}

	// Validate source and destination chains are different
	if req.SourceChain == req.DestinationChain {
		return errors.New("source and destination chains must be different")
	}

	return nil
}
