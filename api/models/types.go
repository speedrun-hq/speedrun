package models

import "time"

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
	ID               string    `json:"id"`
	SourceChain      string    `json:"source_chain"`
	DestinationChain string    `json:"destination_chain"`
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
	ID        string    `json:"id"`
	IntentID  string    `json:"intent_id"`
	Fulfiller string    `json:"fulfiller"`
	Amount    string    `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IntentStatus represents the possible states of an intent
const (
	IntentStatusPending    = "pending"
	IntentStatusFulfilled  = "fulfilled"
	IntentStatusProcessing = "processing"
	IntentStatusCompleted  = "completed"
	IntentStatusFailed     = "failed"
)

// FulfillmentStatus represents the possible states of a fulfillment
const (
	FulfillmentStatusPending   = "pending"
	FulfillmentStatusAccepted  = "accepted"
	FulfillmentStatusRejected  = "rejected"
	FulfillmentStatusCompleted = "completed"
)
