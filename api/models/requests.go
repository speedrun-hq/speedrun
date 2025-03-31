package models

// CreateFulfillmentRequest represents the request body for creating a new fulfillment
type CreateFulfillmentRequest struct {
	IntentID  string `json:"intent_id" binding:"required"`
	Fulfiller string `json:"fulfiller" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
}
