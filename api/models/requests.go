package models

// CreateIntentRequest represents the request body for creating a new intent
type CreateIntentRequest struct {
	ID               string `json:"id"                binding:"required"`
	SourceChain      uint64 `json:"source_chain"      binding:"required"`
	DestinationChain uint64 `json:"destination_chain" binding:"required"`
	Token            string `json:"token"             binding:"required"`
	Amount           string `json:"amount"            binding:"required"`
	Recipient        string `json:"recipient"         binding:"required"`
	Sender           string `json:"sender"            binding:"required"`
	IntentFee        string `json:"intent_fee"        binding:"required"`
}

// CreateFulfillmentRequest represents the request body for creating a new fulfillment
type CreateFulfillmentRequest struct {
	ID       string `json:"id"       binding:"required"`
	Asset    string `json:"asset"    binding:"required"`
	Amount   string `json:"amount"   binding:"required"`
	Receiver string `json:"receiver" binding:"required"`
	ChainID  uint64 `json:"chain_id" binding:"required"`
	TxHash   string `json:"tx_hash"  binding:"required"`
}

// PaginationRequest represents common pagination parameters
type PaginationRequest struct {
	Page     int `form:"page"      json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalCount int         `json:"total_count"`
	TotalPages int         `json:"total_pages"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse(data interface{}, page, pageSize, totalCount int) *PaginatedResponse {
	totalPages := totalCount / pageSize
	if totalCount%pageSize > 0 {
		totalPages++
	}

	return &PaginatedResponse{
		Data:       data,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}
}
