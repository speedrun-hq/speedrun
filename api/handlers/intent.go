package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/speedrun-hq/speedrun/api/db"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/speedrun-hq/speedrun/api/utils"
)

// IntentServiceInterface defines the interface for intent service operations
type IntentServiceInterface interface {
	GetIntent(ctx context.Context, id string) (*models.Intent, error)
	ListIntents(ctx context.Context) ([]*models.Intent, error)
	CreateIntent(ctx context.Context, id string, sourceChain uint64, destinationChain uint64, token, amount, recipient, sender, intentFee string) (*models.Intent, error)
	GetIntentsBySender(ctx context.Context, sender string) ([]*models.Intent, error)
	GetIntentsByRecipient(ctx context.Context, recipient string) ([]*models.Intent, error)
}

var (
	database       db.Database
	intentServices map[uint64]IntentServiceInterface
)

// InitIntentHandlers initializes the intent handlers with required dependencies
func InitIntentHandlers(db db.Database, services map[uint64]IntentServiceInterface) {
	database = db
	intentServices = services
}

// CreateIntent handles the creation of a new intent
func CreateIntent(c *gin.Context) {
	// Read raw request body
	rawBody, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to read request body: %v", err)})
		return
	}

	// Reset request body for binding
	c.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))

	// Bind request body
	var req models.CreateIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to bind request: %v", err)})
		return
	}

	// Validate request
	if err := utils.ValidateIntentRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to validate request: %v", err)})
		return
	}

	// When creating intents through the API, we use the current time
	// For blockchain events, the block timestamp will be used instead
	now := time.Now()

	// Create intent
	intent := &models.Intent{
		ID:               req.ID,
		SourceChain:      req.SourceChain,
		DestinationChain: req.DestinationChain,
		Token:            req.Token,
		Amount:           req.Amount,
		Recipient:        req.Recipient,
		Sender:           req.Sender,
		IntentFee:        req.IntentFee,
		Status:           models.IntentStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Store intent in database
	if err := database.CreateIntent(c.Request.Context(), intent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to store intent: %v", err)})
		return
	}

	// Return response
	c.JSON(http.StatusCreated, intent.ToResponse())
}

// GetIntent handles retrieving a specific intent
func GetIntent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "intent ID is required"})
		return
	}

	// Get the intent from the database first
	intent, err := database.GetIntent(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "intent not found"})
		return
	}

	// Get the intent service for the source chain
	service, ok := intentServices[intent.SourceChain]
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("no intent service for chain %d", intent.SourceChain)})
		return
	}

	// Get the intent from the service to get any updates
	updatedIntent, err := service.GetIntent(c.Request.Context(), id)
	if err != nil {
		// If not found in service, return the database version
		c.JSON(http.StatusOK, intent.ToResponse())
		return
	}

	c.JSON(http.StatusOK, updatedIntent.ToResponse())
}

// ListIntents handles retrieving all intents
func ListIntents(c *gin.Context) {
	// Get all intents from the database
	intents, err := database.ListIntents(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert intents to responses
	responses := make([]*models.IntentResponse, len(intents))
	for i, intent := range intents {
		// Try to get updated intent from service
		if service, ok := intentServices[intent.SourceChain]; ok {
			if updatedIntent, err := service.GetIntent(c.Request.Context(), intent.ID); err == nil {
				responses[i] = updatedIntent.ToResponse()
				continue
			}
		}
		// Fall back to database version if service not found or error
		responses[i] = intent.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}

// GetIntentsBySender handles retrieving all intents for a specific sender
func GetIntentsBySender(c *gin.Context) {
	sender := c.Param("sender")
	if sender == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sender address is required"})
		return
	}

	// Validate sender address format
	if err := utils.ValidateAddress(sender); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid sender address: %v", err)})
		return
	}

	// Get intents by sender from the database
	intents, err := database.ListIntentsBySender(c.Request.Context(), sender)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert intents to responses
	responses := make([]*models.IntentResponse, len(intents))
	for i, intent := range intents {
		responses[i] = intent.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}
