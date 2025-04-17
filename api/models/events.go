package models

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
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
	Sender      string   `json:"sender"` // Sender address that initiated the intent
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

// fetchBlockTimestamp is a helper function to get a block timestamp with fallback methods
func fetchBlockTimestamp(ctx context.Context, client *ethclient.Client, blockNumber uint64, entityID string, entityType string) (time.Time, error) {
	if client == nil {
		return time.Time{}, fmt.Errorf("no client provided")
	}

	if blockNumber == 0 {
		return time.Time{}, fmt.Errorf("no block number provided")
	}

	// First try: Standard block fetching
	block, err := client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err == nil {
		return time.Unix(int64(block.Time()), 0), nil
	}

	// If we get a "transaction type not supported" error, or any other error, try a fallback
	log.Printf("Warning: Failed to get block timestamp for %s %s (block #%d): %v, trying fallback method",
		entityType, entityID, blockNumber, err)

	// Fallback: Use HeaderByNumber which should be more tolerant of different transaction types
	header, err := client.HeaderByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err == nil && header != nil {
		return time.Unix(int64(header.Time), 0), nil
	}

	// If that still fails, log it and return an error
	log.Printf("Warning: Fallback method also failed to get block timestamp: %v", err)
	return time.Time{}, fmt.Errorf("failed to get block timestamp: %v", err)
}

// ToIntent converts an IntentInitiatedEvent to an Intent
func (e *IntentInitiatedEvent) ToIntent(client *ethclient.Client, ctx ...context.Context) (*Intent, error) {
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

	// Get block timestamp
	var timestamp time.Time
	if client != nil {
		if e.BlockNumber == 0 {
			log.Printf("Warning: Intent %s has no block number, cannot fetch blockchain timestamp", e.IntentID)
			timestamp = time.Now()
		} else {
			// Use provided context if available, otherwise use background context
			requestCtx := context.Background()
			if len(ctx) > 0 && ctx[0] != nil {
				requestCtx = ctx[0]
			}

			ts, err := fetchBlockTimestamp(requestCtx, client, e.BlockNumber, e.IntentID, "intent")
			if err != nil {
				log.Printf("Warning: Could not fetch timestamp for intent %s (block #%d): %v, using current time instead",
					e.IntentID, e.BlockNumber, err)
				timestamp = time.Now()
			} else {
				timestamp = ts
				log.Printf("Using blockchain timestamp for intent %s: %s (block #%d)",
					e.IntentID, timestamp.Format(time.RFC3339), e.BlockNumber)
			}
		}
	} else {
		log.Printf("Warning: Intent %s has no client to fetch block timestamp, using current time", e.IntentID)
		timestamp = time.Now()
	}

	return &Intent{
		ID:               e.IntentID,
		SourceChain:      e.ChainID,
		DestinationChain: e.TargetChain,
		Token:            e.Asset,
		Amount:           amount,
		Recipient:        receiver,
		Sender:           e.Sender,
		IntentFee:        tip,
		Status:           IntentStatusPending,
		CreatedAt:        timestamp,
		UpdatedAt:        timestamp,
	}, nil
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

// ToFulfillment converts an IntentFulfilledEvent to a Fulfillment
func (e *IntentFulfilledEvent) ToFulfillment(client *ethclient.Client, ctx ...context.Context) (*Fulfillment, error) {
	amount := e.Amount.String()

	// Get block timestamp
	var timestamp time.Time
	if client != nil {
		if e.BlockNumber == 0 {
			log.Printf("Warning: Fulfillment for intent %s has no block number, cannot fetch blockchain timestamp", e.IntentID)
			timestamp = time.Now()
		} else {
			// Use provided context if available, otherwise use background context
			requestCtx := context.Background()
			if len(ctx) > 0 && ctx[0] != nil {
				requestCtx = ctx[0]
			}

			ts, err := fetchBlockTimestamp(requestCtx, client, e.BlockNumber, e.IntentID, "fulfillment")
			if err != nil {
				log.Printf("Warning: Could not fetch timestamp for fulfillment of intent %s (block #%d): %v, using current time instead",
					e.IntentID, e.BlockNumber, err)
				timestamp = time.Now()
			} else {
				timestamp = ts
				log.Printf("Using blockchain timestamp for fulfillment of intent %s: %s (block #%d)",
					e.IntentID, timestamp.Format(time.RFC3339), e.BlockNumber)
			}
		}
	} else {
		log.Printf("Warning: Fulfillment for intent %s has no client to fetch block timestamp, using current time", e.IntentID)
		timestamp = time.Now()
	}

	return &Fulfillment{
		ID:          e.IntentID,
		Asset:       e.Asset,
		Amount:      amount,
		Receiver:    e.Receiver,
		BlockNumber: e.BlockNumber,
		TxHash:      e.TxHash,
		CreatedAt:   timestamp,
		UpdatedAt:   timestamp,
	}, nil
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

// ToSettlement converts an IntentSettledEvent to a Settlement
func (e *IntentSettledEvent) ToSettlement(client *ethclient.Client, ctx ...context.Context) (*Settlement, error) {
	// Get block timestamp
	var timestamp time.Time
	if client != nil {
		if e.BlockNumber == 0 {
			log.Printf("Warning: Settlement for intent %s has no block number, cannot fetch blockchain timestamp", e.IntentID)
			timestamp = time.Now()
		} else {
			// Use provided context if available, otherwise use background context
			requestCtx := context.Background()
			if len(ctx) > 0 && ctx[0] != nil {
				requestCtx = ctx[0]
			}

			ts, err := fetchBlockTimestamp(requestCtx, client, e.BlockNumber, e.IntentID, "settlement")
			if err != nil {
				log.Printf("Warning: Could not fetch timestamp for settlement of intent %s (block #%d): %v, using current time instead",
					e.IntentID, e.BlockNumber, err)
				timestamp = time.Now()
			} else {
				timestamp = ts
				log.Printf("Using blockchain timestamp for settlement of intent %s: %s (block #%d)",
					e.IntentID, timestamp.Format(time.RFC3339), e.BlockNumber)
			}
		}
	} else {
		log.Printf("Warning: Settlement for intent %s has no client to fetch block timestamp, using current time", e.IntentID)
		timestamp = time.Now()
	}

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
		CreatedAt:    timestamp,
		UpdatedAt:    timestamp,
	}, nil
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
