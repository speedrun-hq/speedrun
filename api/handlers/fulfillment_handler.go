package handlers

import (
	"net/http"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/models"
	"github.com/zeta-chain/zetafast/api/services"
	"github.com/zeta-chain/zetafast/api/utils"
)

var fulfillmentService *services.FulfillmentService

// InitHandlers initializes the handlers with required dependencies
func InitHandlers(clients map[uint64]*ethclient.Client, contractAddresses map[uint64]string, database db.Database) error {
	var err error
	// TODO: Get contract ABI from a configuration or file
	contractABI := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":true,"internalType":"address","name":"receiver","type":"address"}],"name":"IntentFulfilled","type":"event"}]`
	fulfillmentService, err = services.NewFulfillmentService(clients, contractAddresses, database, contractABI)
	return err
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

	fulfillment, err := fulfillmentService.CreateFulfillment(c.Request.Context(), req.IntentID, req.Fulfiller, req.Amount)
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

	// Validate bytes32 format
	if !utils.IsValidBytes32(fulfillmentID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fulfillment ID format"})
		return
	}

	fulfillment, err := fulfillmentService.GetFulfillment(c.Request.Context(), fulfillmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fulfillment": fulfillment,
	})
}
