package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/models"
	"github.com/zeta-chain/zetafast/api/services"
	"github.com/zeta-chain/zetafast/api/utils"
)

var (
	fulfillmentServices map[uint64]*services.FulfillmentService
)

// InitFulfillmentHandlers initializes the fulfillment handlers
func InitFulfillmentHandlers(db db.Database, services map[uint64]*services.FulfillmentService) {
	database = db
	fulfillmentServices = services
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

	err := fulfillmentServices[req.ChainID].CreateFulfillment(c.Request.Context(), req.ID, req.TxHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Fulfillment created successfully",
	})
}

// GetFulfillment retrieves a fulfillment by ID
func GetFulfillment(c *gin.Context) {
	fulfillmentID := c.Param("id")
	if fulfillmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fulfillment ID is required"})
		return
	}

	// Validate bytes32 format
	if !utils.IsValidBytes32(fulfillmentID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fulfillment ID format"})
		return
	}

	fulfillment, err := database.GetFulfillment(c.Request.Context(), fulfillmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fulfillment": fulfillment,
	})
}