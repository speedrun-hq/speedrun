package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/models"
	"github.com/zeta-chain/zetafast/api/services"
	"github.com/zeta-chain/zetafast/api/utils"
)

var fulfillmentService *services.FulfillmentService

// InitHandlers initializes the handlers with required dependencies
func InitHandlers(database db.Database) {
	fulfillmentService = services.NewFulfillmentService(database)
}

// CreateFulfillment handles the creation of a new fulfillment
func CreateFulfillment(c *gin.Context) {
	var req models.CreateFulfillmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate request
	if err := utils.ValidateFulfillmentRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fulfillment, err := fulfillmentService.CreateFulfillment(req.IntentID, req.Fulfiller, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Fulfillment created successfully",
		"fulfillment": fulfillment,
	})
}

// GetFulfillment retrieves a fulfillment by ID
func GetFulfillment(c *gin.Context) {
	fulfillmentID := c.Param("id")
	if fulfillmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fulfillment ID is required"})
		return
	}

	// Validate UUID format
	if !utils.IsValidUUID(fulfillmentID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fulfillment ID format"})
		return
	}

	fulfillment, err := fulfillmentService.GetFulfillment(fulfillmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fulfillment": fulfillment,
	})
}
