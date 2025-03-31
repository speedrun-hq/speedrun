package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zeta-chain/zetafast/api/config"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/handlers"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	database, err := db.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize handlers
	handlers.InitHandlers(database)

	// Initialize router
	router := gin.Default()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Initialize routes
	initializeRoutes(router)

	// Start server
	log.Printf("Starting server on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func initializeRoutes(router *gin.Engine) {
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
			intents.POST("/", handlers.CreateIntent)
			intents.GET("/:id", handlers.GetIntent)
			intents.GET("/", handlers.ListIntents)
		}

		// Fulfillment routes
		fulfillments := v1.Group("/fulfillments")
		{
			fulfillments.POST("/", handlers.CreateFulfillment)
			fulfillments.GET("/:id", handlers.GetFulfillment)
		}
	}
}
