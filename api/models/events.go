package models

import (
	"context"
	"fmt"
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
	IsCall      bool     `json:"isCall"` // Whether this is a call intent
	Data        []byte   `json:"data"`   // Call data if this is a call intent
}

// IntentFulfilledEvent represents the event emitted when an intent is fulfilled
type IntentFulfilledEvent struct {
	IntentID    string
	Asset       string
	Amount      *big.Int
	Receiver    string
	BlockNumber uint64
	TxHash      string
	IsCall      bool   // Whether this is a call intent
	Data        []byte // Call data if this is a call intent
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
	IsCall       bool   // Whether this is a call intent
	Data         []byte // Call data if this is a call intent
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
		// TODO: consider return error here
		//log.Printf("Warning: Target chain is 0, using source chain as target")
		e.TargetChain = e.ChainID
	}

	// Get block timestamp
	var timestamp time.Time
	if client != nil {
		if e.BlockNumber == 0 {
			// TODO: consider return error here
			//log.Printf("Warning: Intent %s has no block number, cannot fetch blockchain timestamp", e.IntentID)
			timestamp = time.Now()
		} else {
			// Use provided context if available, otherwise use background context
			requestCtx := context.Background()
			if len(ctx) > 0 && ctx[0] != nil {
				requestCtx = ctx[0]
			}

			ts, err := fetchBlockTimestamp(requestCtx, client, e.BlockNumber)
			if err != nil {
				// TODO: consider return error here
				//log.Printf("Warning: Could not fetch timestamp for intent %s (block #%d): %v, using current time instead",
				//	e.IntentID, e.BlockNumber, err)
				timestamp = time.Now()
			} else {
				timestamp = ts
			}
		}
	} else {
		// TODO: consider return error here
		//log.Printf("Warning: Intent %s has no client to fetch block timestamp, using current time", e.IntentID)
		timestamp = time.Now()
	}

	// Create intent with call data if this is a call intent
	intent := &Intent{
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
		IsCall:           e.IsCall,
	}

	// Set call data if present
	if e.IsCall && len(e.Data) > 0 {
		intent.CallData = common.Bytes2Hex(e.Data)
	}

	return intent, nil
}

// ToFulfillment converts an IntentFulfilledEvent to a Fulfillment
func (e *IntentFulfilledEvent) ToFulfillment(client *ethclient.Client, ctx ...context.Context) (*Fulfillment, error) {
	amount := e.Amount.String()

	// Get block timestamp
	var timestamp time.Time
	if client != nil {
		if e.BlockNumber == 0 {
			// TODO: consider return error here
			//log.Printf("Warning: Fulfillment for intent %s has no block number, cannot fetch blockchain timestamp", e.IntentID)
			timestamp = time.Now()
		} else {
			// Use provided context if available, otherwise use background context
			requestCtx := context.Background()
			if len(ctx) > 0 && ctx[0] != nil {
				requestCtx = ctx[0]
			}

			ts, err := fetchBlockTimestamp(requestCtx, client, e.BlockNumber)
			if err != nil {
				// TODO: consider return error here
				//log.Printf("Warning: Could not fetch timestamp for fulfillment of intent %s (block #%d): %v, using current time instead",
				//	e.IntentID, e.BlockNumber, err)
				timestamp = time.Now()
			} else {
				timestamp = ts
			}
		}
	} else {
		// TODO: consider return error here
		//log.Printf("Warning: Fulfillment for intent %s has no client to fetch block timestamp, using current time", e.IntentID)
		timestamp = time.Now()
	}

	// Create fulfillment with call information
	fulfillment := &Fulfillment{
		ID:          e.IntentID,
		Asset:       e.Asset,
		Amount:      amount,
		Receiver:    e.Receiver,
		BlockNumber: e.BlockNumber,
		TxHash:      e.TxHash,
		CreatedAt:   timestamp,
		UpdatedAt:   timestamp,
		IsCall:      e.IsCall,
	}

	// Set call data if present
	if e.IsCall && len(e.Data) > 0 {
		fulfillment.CallData = common.Bytes2Hex(e.Data)
	}

	return fulfillment, nil
}

// ToSettlement converts an IntentSettledEvent to a Settlement
func (e *IntentSettledEvent) ToSettlement(client *ethclient.Client, ctx ...context.Context) (*Settlement, error) {
	// Get block timestamp
	var timestamp time.Time
	if client != nil {
		if e.BlockNumber == 0 {
			// TODO: consider return error here
			//log.Printf("Warning: Settlement for intent %s has no block number, cannot fetch blockchain timestamp", e.IntentID)
			timestamp = time.Now()
		} else {
			// Use provided context if available, otherwise use background context
			requestCtx := context.Background()
			if len(ctx) > 0 && ctx[0] != nil {
				requestCtx = ctx[0]
			}

			ts, err := fetchBlockTimestamp(requestCtx, client, e.BlockNumber)
			if err != nil {
				// TODO: consider return error here
				//log.Printf("Warning: Could not fetch timestamp for settlement of intent %s (block #%d): %v, using current time instead",
				//	e.IntentID, e.BlockNumber, err)
				timestamp = time.Now()
			} else {
				timestamp = ts
			}
		}
	} else {
		// TODO: consider return error here
		//log.Printf("Warning: Settlement for intent %s has no client to fetch block timestamp, using current time", e.IntentID)
		timestamp = time.Now()
	}

	// Create settlement with call information
	settlement := &Settlement{
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
		IsCall:       e.IsCall,
	}

	// Set call data if present
	if e.IsCall && len(e.Data) > 0 {
		settlement.CallData = common.Bytes2Hex(e.Data)
	}

	return settlement, nil
}

// fetchBlockTimestamp is a helper function to get a block timestamp with fallback methods
func fetchBlockTimestamp(ctx context.Context, client *ethclient.Client, blockNumber uint64) (time.Time, error) {
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

	// Fallback: Use HeaderByNumber which should be more tolerant of different transaction types
	header, err := client.HeaderByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err == nil && header != nil {
		return time.Unix(int64(header.Time), 0), nil
	}

	// If that still fails, log it and return an error
	return time.Time{}, fmt.Errorf("failed to get block timestamp: %v", err)
}
