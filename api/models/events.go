package models

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// IntentInitiatedEvent represents the event emitted when a new intent is created
type IntentInitiatedEvent struct {
	IntentID    string   `json:"intentId" gorm:"primaryKey"`
	Asset       string   `json:"asset"`  // ERC20 token address
	Amount      *big.Int `json:"amount"` // Amount to receive on target chain
	TargetChain uint64   `json:"targetChain"`
	Receiver    []byte   `json:"receiver"` // Receiver address in bytes format
	Tip         *big.Int `json:"tip"`      // Tip for the fulfiller
	Salt        *big.Int `json:"salt"`     // Salt used for intent ID generation
	ChainID     uint64   `json:"chainId"`  // Source chain ID
	BlockNumber uint64   `json:"blockNumber"`
	TxHash      string   `json:"txHash"`
}

// FulfillmentEvent represents the event emitted when a fulfillment occurs
type FulfillmentEvent struct {
	IntentID    string
	TargetChain uint64
	Receiver    string
	Amount      string
	TxHash      string
	BlockNumber uint64
}

// ToIntent converts an IntentInitiatedEvent to an Intent
func (e *IntentInitiatedEvent) ToIntent() *Intent {
	// Convert big.Int to string for amount and tip
	amount := e.Amount.String()
	tip := e.Tip.String()

	// Convert receiver bytes to hex string
	receiver := common.BytesToAddress(e.Receiver).Hex()

	return &Intent{
		ID:               e.IntentID,
		SourceChain:      e.ChainID,
		DestinationChain: e.TargetChain,
		Token:            e.Asset,
		Amount:           amount,
		Recipient:        receiver,
		IntentFee:        tip,
		Status:           IntentStatusPending,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// FromIntent converts an Intent to an IntentInitiatedEvent
func FromIntent(intent *Intent) *IntentInitiatedEvent {
	// Convert string to big.Int for amount and tip
	amount := new(big.Int)
	amount.SetString(intent.Amount, 10)

	tip := new(big.Int)
	tip.SetString(intent.IntentFee, 10)

	return &IntentInitiatedEvent{
		IntentID:    intent.ID,
		Asset:       intent.Token,
		Amount:      amount,
		TargetChain: intent.DestinationChain,
		Receiver:    common.FromHex(intent.Recipient),
		Tip:         tip,
		ChainID:     intent.SourceChain,
	}
}

// ToFulfillment converts a FulfillmentEvent to a Fulfillment
func (e *FulfillmentEvent) ToFulfillment() *Fulfillment {
	now := time.Now()
	return &Fulfillment{
		ID:          e.TxHash, // Using tx hash as ID for uniqueness
		IntentID:    e.IntentID,
		Fulfiller:   e.Receiver,
		TargetChain: e.TargetChain,
		Amount:      e.Amount,
		Status:      FulfillmentStatusPending,
		TxHash:      e.TxHash,
		BlockNumber: e.BlockNumber,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
