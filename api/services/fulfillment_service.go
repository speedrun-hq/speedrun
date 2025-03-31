package services

import (
	"time"

	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/models"
	"github.com/zeta-chain/zetafast/api/utils"
)

// FulfillmentService handles business logic for fulfillments
type FulfillmentService struct {
	db db.Database
}

// NewFulfillmentService creates a new fulfillment service
func NewFulfillmentService(db db.Database) *FulfillmentService {
	return &FulfillmentService{db: db}
}

// CreateFulfillment creates a new fulfillment for an intent
func (s *FulfillmentService) CreateFulfillment(intentID, fulfiller, amount string) (*models.Fulfillment, error) {
	// Get the intent
	intent, err := s.db.GetIntent(intentID)
	if err != nil {
		return nil, err
	}

	// Validate intent
	if err := utils.ValidateIntent(intent); err != nil {
		return nil, err
	}

	// Get total amount already fulfilled
	totalFulfilled, err := s.db.GetTotalFulfilledAmount(intentID)
	if err != nil {
		return nil, err
	}

	// Validate fulfillment amount
	if err := utils.ValidateFulfillmentAmount(amount, intent.Amount, totalFulfilled); err != nil {
		return nil, err
	}

	// Create fulfillment
	fulfillment := &models.Fulfillment{
		ID:        utils.GenerateID(),
		IntentID:  intentID,
		Fulfiller: fulfiller,
		Amount:    amount,
		Status:    models.FulfillmentStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to database
	if err := s.db.CreateFulfillment(fulfillment); err != nil {
		return nil, err
	}

	// Check if intent is fully fulfilled
	newTotal, err := s.db.GetTotalFulfilledAmount(intentID)
	if err != nil {
		return nil, err
	}

	// Update intent status if fully fulfilled
	if newTotal == intent.Amount {
		if err := s.db.UpdateIntentStatus(intentID, models.IntentStatusFulfilled); err != nil {
			return nil, err
		}
	}

	return fulfillment, nil
}

// GetFulfillment retrieves a fulfillment by ID
func (s *FulfillmentService) GetFulfillment(id string) (*models.Fulfillment, error) {
	return s.db.GetFulfillment(id)
}
