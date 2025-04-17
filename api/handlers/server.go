package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/speedrun-hq/speedrun/api/db"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/speedrun-hq/speedrun/api/services"
	"github.com/speedrun-hq/speedrun/api/utils"
)

// Server handles HTTP requests
type Server struct {
	fulfillmentService *services.FulfillmentService
	intentService      *services.IntentService
	db                 db.Database
}

// NewServer creates a new HTTP server
func NewServer(fulfillmentService *services.FulfillmentService, intentService *services.IntentService, database db.Database) *Server {
	return &Server{
		fulfillmentService: fulfillmentService,
		intentService:      intentService,
		db:                 database,
	}
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	// Create a new router with default middleware
	router := gin.Default()

	// Configure timeouts for the server
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second, // Time to read the request headers/body
		WriteTimeout: 15 * time.Second, // Time to write the response
		IdleTimeout:  60 * time.Second, // Time to keep connections alive
	}

	// Add custom middleware to set request timeouts
	router.Use(func(c *gin.Context) {
		// Create a timeout context for each request
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		// Update the request with the timeout context
		c.Request = c.Request.WithContext(ctx)

		// Create a channel to signal when the request is complete
		done := make(chan struct{})

		go func() {
			// Continue processing the request chain
			c.Next()
			close(done)
		}()

		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				// Log timeout and send an error response
				log.Printf("Request timeout: %s %s", c.Request.Method, c.Request.URL.Path)
				c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
					"error": "Request timeout",
				})
			}
		case <-done:
			// Request completed before timeout
		}
	})

	// Add recovery middleware to catch panics in request handling goroutines
	router.Use(gin.Recovery())

	// Configure CORS
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	config := cors.DefaultConfig()
	config.AllowOrigins = strings.Split(allowedOrigins, ",")
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	router.Use(cors.New(config))

	// Add request logging with execution time
	router.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Process request
		c.Next()

		// Log information after request is processed
		latency := time.Since(start)
		log.Printf("%s %s [%d] %v", c.Request.Method, path, c.Writer.Status(), latency)

		// Log slow requests
		if latency > 500*time.Millisecond {
			log.Printf("SLOW REQUEST: %s %s took %v", c.Request.Method, path, latency)
		}
	})

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

	// Start server in a separate goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Set up a channel to listen for OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a timeout context for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
	return nil
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
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	// Convert to integers
	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page parameter"})
		return
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil || pageSizeInt < 1 || pageSizeInt > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page_size parameter (must be between 1 and 100)"})
		return
	}

	// Get status filter
	status := c.Query("status")

	// Get intents with pagination and status filter using optimized method
	intents, totalCount, err := s.db.ListIntentsPaginatedOptimized(c.Request.Context(), pageInt, pageSizeInt, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to response format
	var response []*models.IntentResponse
	for _, intent := range intents {
		response = append(response, intent.ToResponse())
	}

	// Create paginated response
	paginatedResponse := models.NewPaginatedResponse(response, pageInt, pageSizeInt, totalCount)

	c.JSON(http.StatusOK, paginatedResponse)
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
	// Get pagination parameters
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	// Convert to integers
	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page parameter"})
		return
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil || pageSizeInt < 1 || pageSizeInt > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page_size parameter (must be between 1 and 100)"})
		return
	}

	// Get fulfillments with pagination using optimized method
	fulfillments, totalCount, err := s.db.ListFulfillmentsPaginatedOptimized(c.Request.Context(), pageInt, pageSizeInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create paginated response
	paginatedResponse := models.NewPaginatedResponse(fulfillments, pageInt, pageSizeInt, totalCount)

	c.JSON(http.StatusOK, paginatedResponse)
}

// GetIntentsBySender handles retrieving intents by sender
func (s *Server) GetIntentsBySender(c *gin.Context) {
	sender := c.Param("sender")
	if sender == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sender address is required"})
		return
	}

	// Get pagination parameters
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	// Convert to integers
	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page parameter"})
		return
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil || pageSizeInt < 1 || pageSizeInt > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page_size parameter (must be between 1 and 100)"})
		return
	}

	// Validate address format
	if !utils.IsValidAddress(sender) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sender address format"})
		return
	}

	// Get intents with pagination using optimized method
	intents, totalCount, err := s.db.ListIntentsBySenderPaginatedOptimized(c.Request.Context(), sender, pageInt, pageSizeInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to response format
	var response []*models.IntentResponse
	for _, intent := range intents {
		response = append(response, intent.ToResponse())
	}

	// Create paginated response
	paginatedResponse := models.NewPaginatedResponse(response, pageInt, pageSizeInt, totalCount)

	c.JSON(http.StatusOK, paginatedResponse)
}

// GetIntentsByRecipient handles retrieving intents by recipient
func (s *Server) GetIntentsByRecipient(c *gin.Context) {
	recipient := c.Param("recipient")
	if recipient == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipient address is required"})
		return
	}

	// Get pagination parameters
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	// Convert to integers
	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page parameter"})
		return
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil || pageSizeInt < 1 || pageSizeInt > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page_size parameter (must be between 1 and 100)"})
		return
	}

	// Validate address format
	if !utils.IsValidAddress(recipient) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient address format"})
		return
	}

	// Get intents with pagination using optimized method
	intents, totalCount, err := s.db.ListIntentsByRecipientPaginatedOptimized(c.Request.Context(), recipient, pageInt, pageSizeInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to response format
	var response []*models.IntentResponse
	for _, intent := range intents {
		response = append(response, intent.ToResponse())
	}

	// Create paginated response
	paginatedResponse := models.NewPaginatedResponse(response, pageInt, pageSizeInt, totalCount)

	c.JSON(http.StatusOK, paginatedResponse)
}
