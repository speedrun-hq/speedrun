package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/speedrun-hq/speedrun/api/services"
	"github.com/speedrun-hq/speedrun/api/utils"
)

// Server handles HTTP requests
type Server struct {
	fulfillmentService *services.FulfillmentService
	intentService      *services.IntentService
}

// NewServer creates a new HTTP server
func NewServer(fulfillmentService *services.FulfillmentService, intentService *services.IntentService) *Server {
	return &Server{
		fulfillmentService: fulfillmentService,
		intentService:      intentService,
	}
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	router := gin.Default()

	// Configure CORS
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	config := cors.DefaultConfig()
	config.AllowOrigins = strings.Split(allowedOrigins, ",")
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	router.Use(cors.New(config))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		// Intent routes
		intents := v1.Group("/intents")
		{
			intents.POST("", s.CreateIntent)
			intents.GET("/:id", s.GetIntent)
			intents.GET("", s.ListIntents)
			intents.GET("/sender/:sender", s.GetIntentsBySender)
			intents.GET("/recipient/:recipient", s.GetIntentsByRecipient)
		}

		// Fulfillment routes
		fulfillments := v1.Group("/fulfillments")
		{
			fulfillments.POST("", s.CreateFulfillment)
			fulfillments.GET("/:id", s.GetFulfillment)
			fulfillments.GET("", s.ListFulfillments)
		}
	}

	return router.Run(addr)
}

// CreateIntent handles the creation of a new intent
func (s *Server) CreateIntent(c *gin.Context) {
	var req struct {
		ID               string `json:"id" binding:"required"`
		SourceChain      uint64 `json:"source_chain" binding:"required"`
		DestinationChain uint64 `json:"destination_chain" binding:"required"`
		Token            string `json:"token" binding:"required"`
		Amount           string `json:"amount" binding:"required"`
		Recipient        string `json:"recipient" binding:"required"`
		Sender           string `json:"sender" binding:"required"`
		IntentFee        string `json:"intent_fee" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	intent, err := s.intentService.CreateIntent(c.Request.Context(), req.ID, req.SourceChain, req.DestinationChain, req.Token, req.Amount, req.Recipient, req.Sender, req.IntentFee)
	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "invalid") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, intent)
}

// GetIntent handles retrieving an intent by ID
func (s *Server) GetIntent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "intent ID is required"})
		return
	}

	// Log the request for debugging
	log.Printf("GetIntent request received for ID: %s", id)

	// Validate ID format
	if !utils.ValidateBytes32(id) {
		log.Printf("Invalid intent ID format: %s", id)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid intent ID format"})
		return
	}

	intent, err := s.intentService.GetIntent(c.Request.Context(), id)
	if err != nil {
		// Log the error for debugging
		log.Printf("Error getting intent %s: %v", id, err)

		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("intent not found: %s", id)})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Successfully retrieved intent %s", id)
	c.JSON(http.StatusOK, intent)
}

// ListIntents handles retrieving all intents
func (s *Server) ListIntents(c *gin.Context) {
	// Get pagination parameters
	limit := c.DefaultQuery("limit", "100")
	offset := c.DefaultQuery("offset", "0")

	// Convert to integers
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
		return
	}
	offsetInt, err := strconv.Atoi(offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset parameter"})
		return
	}

	// Get status filter
	status := c.Query("status")

	// Get intents
	intents, err := s.intentService.ListIntents(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Filter by status if provided
	if status != "" {
		var filtered []*models.Intent
		for _, intent := range intents {
			if string(intent.Status) == status {
				filtered = append(filtered, intent)
			}
		}
		intents = filtered
	}

	// Apply pagination
	start := offsetInt
	end := offsetInt + limitInt
	if start >= len(intents) {
		c.JSON(http.StatusOK, []*models.Intent{})
		return
	}
	if end > len(intents) {
		end = len(intents)
	}
	intents = intents[start:end]

	// Convert to response format
	var response []*models.IntentResponse
	for _, intent := range intents {
		response = append(response, intent.ToResponse())
	}

	c.JSON(http.StatusOK, response)
}

// CreateFulfillmentRequest represents the request body for creating a fulfillment
type CreateFulfillmentRequest struct {
	IntentID string `json:"intent_id" binding:"required"`
	TxHash   string `json:"tx_hash" binding:"required"`
}

// CreateFulfillment handles the creation of a new fulfillment
func (s *Server) CreateFulfillment(c *gin.Context) {
	var req CreateFulfillmentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := s.fulfillmentService.CreateFulfillment(c.Request.Context(), req.IntentID, req.TxHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "fulfillment created successfully"})
}

// GetFulfillment handles retrieving a fulfillment by ID
func (s *Server) GetFulfillment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fulfillment ID is required"})
		return
	}

	fulfillment, err := s.fulfillmentService.GetFulfillment(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, fulfillment)
}

// ListFulfillments handles retrieving all fulfillments
func (s *Server) ListFulfillments(c *gin.Context) {
	fulfillments, err := s.fulfillmentService.ListFulfillments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, fulfillments)
}

// GetIntentsBySender handles retrieving all intents for a specific sender
func (s *Server) GetIntentsBySender(c *gin.Context) {
	sender := c.Param("sender")
	if sender == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sender address is required"})
		return
	}

	intents, err := s.intentService.GetIntentsBySender(c.Request.Context(), sender)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, intents)
}

// GetIntentsByRecipient handles retrieving all intents for a specific recipient
func (s *Server) GetIntentsByRecipient(c *gin.Context) {
	recipient := c.Param("recipient")
	if recipient == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipient address is required"})
		return
	}

	intents, err := s.intentService.GetIntentsByRecipient(c.Request.Context(), recipient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, intents)
}
