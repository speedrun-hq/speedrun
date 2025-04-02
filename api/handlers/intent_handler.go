package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/models"
	"github.com/zeta-chain/zetafast/api/services"
	"github.com/zeta-chain/zetafast/api/utils"
)

var (
	database      db.Database
	intentService *services.IntentService
)

// InitIntentHandlers initializes the intent handlers with required dependencies
func InitIntentHandlers(db db.Database, service *services.IntentService) {
	database = db
	intentService = service
}

// CreateIntent handles the creation of a new intent
func CreateIntent(c *gin.Context) {
	// Read raw request body
	rawBody, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to read request body: %v", err)})
		return
	}
	fmt.Printf("Raw request body: %s\n", string(rawBody))

	// Reset request body for binding
	c.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))

	// Bind request body
	var req models.CreateIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to bind request: %v", err)})
		return
	}

	fmt.Printf("Bound request: %+v\n", req)

	// Validate request
	if err := utils.ValidateIntentRequest(&req); err != nil {
		fmt.Printf("Validation error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to validate request: %v", err)})
		return
	}

	// Create intent
	intent := &models.Intent{
		ID:               req.ID,
		SourceChain:      req.SourceChain,
		DestinationChain: req.DestinationChain,
		Token:            req.Token,
		Amount:           req.Amount,
		Recipient:        req.Recipient,
		IntentFee:        req.IntentFee,
		Status:           models.IntentStatusPending,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
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

	intent, err := intentService.GetIntent(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "intent not found"})
		return
	}

	c.JSON(http.StatusOK, intent.ToResponse())
}

// ListIntents handles retrieving all intents
func ListIntents(c *gin.Context) {
	intents, err := intentService.ListIntents(c.Request.Context())
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
