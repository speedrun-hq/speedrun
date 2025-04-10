package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/speedrun-hq/speedrun/api/db"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/speedrun-hq/speedrun/api/utils"
)

// FulfillmentServiceInterface defines the interface for fulfillment service operations
type FulfillmentServiceInterface interface {
	CreateFulfillment(ctx context.Context, id string, txHash string) error
}

var fulfillmentServices map[uint64]FulfillmentServiceInterface

// InitFulfillmentHandlers initializes the fulfillment handlers
func InitFulfillmentHandlers(db db.Database, services map[uint64]FulfillmentServiceInterface) {
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

	// Check if service exists for the chain
	service, ok := fulfillmentServices[req.ChainID]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("no fulfillment service for chain %d", req.ChainID)})
		return
	}

	// Create fulfillment in service
	err := service.CreateFulfillment(c.Request.Context(), req.ID, req.TxHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create fulfillment in database
	fulfillment := &models.Fulfillment{
		ID:     req.ID,
		TxHash: req.TxHash,
	}
	if err := database.CreateFulfillment(c.Request.Context(), fulfillment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Fulfillment created successfully",
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
		c.JSON(http.StatusNotFound, gin.H{"error": "fulfillment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fulfillment": fulfillment,
	})
}
