package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/models"
	"github.com/zeta-chain/zetafast/api/utils"
)

var database db.Database

// InitIntentHandlers initializes the intent handlers with required dependencies
func InitIntentHandlers(db db.Database) {
	database = db
}

// CreateIntent handles the creation of a new transfer intent
func CreateIntent(c *gin.Context) {
	var req models.CreateIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate request
	if err := utils.ValidateIntentRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create intent
	intent := &models.Intent{
		ID:               utils.GenerateID(),
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
	if err := database.CreateIntent(intent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Intent created successfully",
		"intent":  intent,
	})
}

// GetIntent retrieves an intent by ID
func GetIntent(c *gin.Context) {
	intentID := c.Param("id")
	if intentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "intent ID is required"})
		return
	}

	intent, err := database.GetIntent(intentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"intent": intent,
	})
}

// ListIntents retrieves a list of intents with optional filtering
func ListIntents(c *gin.Context) {
	page := 1
	limit := 10

	intents, total, err := database.ListIntents(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"intents": intents,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}
