package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/zeta-chain/zetafast/api/config"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/handlers"
	"github.com/zeta-chain/zetafast/api/services"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	var database db.Database
	if os.Getenv("GO_ENV") == "development" {
		log.Println("Using mock database for development")
		database = services.NewMockDB()
	} else {
		database, err = db.NewDB(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		// Initialize schema
		if err := database.(*db.DB).InitSchema(); err != nil {
			log.Fatalf("Failed to initialize database schema: %v", err)
		}
	}
	defer database.Close()

	// Initialize handlers
	handlers.InitIntentHandlers(database)

	// Initialize router with trailing slash handling
	router := gin.Default()
	router.RemoveExtraSlash = true // Prevent automatic trailing slash redirects

	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60, // 12 hours
	}))

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
			intents.POST("", handlers.CreateIntent) // No trailing slash
			intents.GET("/:id", handlers.GetIntent) // Keep trailing slash for parameterized routes
			intents.GET("", handlers.ListIntents)   // No trailing slash
		}

		// Fulfillment routes
		fulfillments := v1.Group("/fulfillments")
		{
			fulfillments.POST("", handlers.CreateFulfillment) // No trailing slash
			fulfillments.GET("/:id", handlers.GetFulfillment) // Keep trailing slash for parameterized routes
		}
	}
}
