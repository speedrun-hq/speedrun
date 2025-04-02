package models

import (
	"time"
)

// Chain represents a supported blockchain network
type Chain struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	NetworkID   string `json:"network_id"`
	RPCURL      string `json:"rpc_url"`
	ExplorerURL string `json:"explorer_url"`
}

// Token represents a supported token on a chain
type Token struct {
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	ChainID  string `json:"chain_id"`
	LogoURL  string `json:"logo_url,omitempty"`
}

// Intent represents a cross-chain transfer intent
type Intent struct {
	ID               string       `json:"id"`
	SourceChain      uint64       `json:"source_chain"`
	DestinationChain uint64       `json:"destination_chain"`
	Token            string       `json:"token"`
	Amount           string       `json:"amount"`
	Recipient        string       `json:"recipient"`
	IntentFee        string       `json:"intent_fee"`
	Status           IntentStatus `json:"status"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

// IntentStatus represents the possible states of an intent
// Note: These statuses are tracked in the API only, not in the contract.
// The contract only emits events for intent initiation and fulfillment.
// The API maintains additional state to track the full lifecycle of an intent.
type IntentStatus string

const (
	// IntentStatusPending indicates the intent has been initiated but not yet fulfilled
	IntentStatusPending IntentStatus = "pending"

	// IntentStatusFulfilled indicates the intent has been fulfilled on the target chain
	IntentStatusFulfilled IntentStatus = "fulfilled"

	// IntentStatusProcessing indicates the intent is being processed by the gateway
	IntentStatusProcessing IntentStatus = "processing"

	// IntentStatusCompleted indicates the intent has been fully completed
	IntentStatusCompleted IntentStatus = "completed"

	// IntentStatusFailed indicates the intent failed to complete
	IntentStatusFailed IntentStatus = "failed"
)

// ToResponse converts an Intent to an IntentResponse
func (e *Intent) ToResponse() *IntentResponse {
	return &IntentResponse{
		ID:               e.ID,
		SourceChain:      e.SourceChain,
		DestinationChain: e.DestinationChain,
		Token:            e.Token,
		Amount:           e.Amount,
		Recipient:        e.Recipient,
		IntentFee:        e.IntentFee,
		Status:           string(e.Status),
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}
}

// IntentResponse represents the response format for an intent
type IntentResponse struct {
	ID               string    `json:"id"`
	SourceChain      uint64    `json:"source_chain"`
	DestinationChain uint64    `json:"destination_chain"`
	Token            string    `json:"token"`
	Amount           string    `json:"amount"`
	Recipient        string    `json:"recipient"`
	IntentFee        string    `json:"intent_fee"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Fulfillment represents a partial or complete fulfillment of an intent
type Fulfillment struct {
	ID          string            `json:"id"`
	IntentID    string            `json:"intent_id"`
	Fulfiller   string            `json:"fulfiller"`
	TargetChain uint64            `json:"target_chain"`
	Amount      string            `json:"amount"`
	Status      FulfillmentStatus `json:"status"`
	TxHash      string            `json:"tx_hash"`
	BlockNumber uint64            `json:"block_number"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// FulfillmentStatus represents the possible states of a fulfillment
type FulfillmentStatus string

const (
	FulfillmentStatusPending   FulfillmentStatus = "pending"
	FulfillmentStatusAccepted  FulfillmentStatus = "accepted"
	FulfillmentStatusRejected  FulfillmentStatus = "rejected"
	FulfillmentStatusCompleted FulfillmentStatus = "completed"
)
