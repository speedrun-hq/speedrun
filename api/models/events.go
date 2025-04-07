package models

import (
	"log"
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

// IntentFulfilledEvent represents the event emitted when an intent is fulfilled
type IntentFulfilledEvent struct {
	IntentID    string
	Asset       string
	Amount      *big.Int
	Receiver    string
	BlockNumber uint64
	TxHash      string
}

type IntentSettledEvent struct {
	IntentID     string
	Asset        string
	Amount       *big.Int
	Receiver     string
	Fulfilled    bool
	Fulfiller    string
	ActualAmount *big.Int
	PaidTip      *big.Int
	BlockNumber  uint64
	TxHash       string
}

// ToIntent converts an IntentInitiatedEvent to an Intent
func (e *IntentInitiatedEvent) ToIntent() *Intent {
	// Convert big.Int to string for amount and tip
	amount := e.Amount.String()
	tip := e.Tip.String()

	// Convert receiver bytes to hex string
	receiver := common.BytesToAddress(e.Receiver).Hex()

	// Validate target chain
	if e.TargetChain == 0 {
		log.Printf("Warning: Target chain is 0, using source chain as target")
		e.TargetChain = e.ChainID
	}

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

func (e *IntentFulfilledEvent) ToFulfillment() *Fulfillment {
	amount := e.Amount.String()

	return &Fulfillment{
		ID:          e.IntentID,
		Asset:       e.Asset,
		Amount:      amount,
		Receiver:    e.Receiver,
		BlockNumber: e.BlockNumber,
		TxHash:      e.TxHash,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func FromFulfillment(fulfillment *Fulfillment) *IntentFulfilledEvent {
	amount := new(big.Int)
	amount.SetString(fulfillment.Amount, 10)

	return &IntentFulfilledEvent{
		IntentID:    fulfillment.ID,
		Asset:       fulfillment.Asset,
		Amount:      amount,
		Receiver:    fulfillment.Receiver,
		BlockNumber: fulfillment.BlockNumber,
		TxHash:      fulfillment.TxHash,
	}
}

func (e *IntentSettledEvent) ToSettlement() *Settlement {
	return &Settlement{
		ID:           e.IntentID,
		Asset:        e.Asset,
		Amount:       e.Amount.String(),
		Receiver:     e.Receiver,
		Fulfilled:    e.Fulfilled,
		Fulfiller:    e.Fulfiller,
		ActualAmount: e.ActualAmount.String(),
		PaidTip:      e.PaidTip.String(),
		BlockNumber:  e.BlockNumber,
		TxHash:       e.TxHash,
	}
}

func FromSettlement(settlement *Settlement) *IntentSettledEvent {
	amount := new(big.Int)
	amount.SetString(settlement.Amount, 10)

	paidTip := new(big.Int)
	paidTip.SetString(settlement.PaidTip, 10)

	return &IntentSettledEvent{
		IntentID:     settlement.ID,
		Asset:        settlement.Asset,
		Amount:       amount,
		Receiver:     settlement.Receiver,
		Fulfilled:    settlement.Fulfilled,
		Fulfiller:    settlement.Fulfiller,
		ActualAmount: amount,
		PaidTip:      paidTip,
		TxHash:       settlement.TxHash,
	}
}
