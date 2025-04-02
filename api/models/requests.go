package models

// CreateIntentRequest represents the request body for creating a new intent
type CreateIntentRequest struct {
	ID               string `json:"id" binding:"required"`
	SourceChain      uint64 `json:"source_chain" binding:"required"`
	DestinationChain uint64 `json:"destination_chain" binding:"required"`
	Token            string `json:"token" binding:"required"`
	Amount           string `json:"amount" binding:"required"`
	Recipient        string `json:"recipient" binding:"required"`
	IntentFee        string `json:"intent_fee" binding:"required"`
}

// CreateFulfillmentRequest represents the request body for creating a new fulfillment
type CreateFulfillmentRequest struct {
	IntentID  string `json:"intent_id" binding:"required"`
	Fulfiller string `json:"fulfiller" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
}
