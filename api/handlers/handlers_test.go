package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/zeta-chain/zetafast/api/config"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/models"
	"github.com/zeta-chain/zetafast/api/services"
	"github.com/zeta-chain/zetafast/api/test/mocks"
	"github.com/zeta-chain/zetafast/api/utils"
)

func setupTestRouter(t *testing.T) (*gin.Engine, *mocks.MockDB, *services.IntentService, *services.FulfillmentService) {
	// Create a mock database and services
	mockDB := mocks.NewMockDB()
	mockEthClient := &ethclient.Client{}

	// Initialize validation package with test config
	testConfig := &config.Config{
		SupportedChains: []uint64{7001, 42161, 8453},
		ChainConfigs: map[uint64]*config.ChainConfig{
			42161: {
				DefaultBlock: 322207320, // Arbitrum
			},
			8453: {
				DefaultBlock: 28411000, // Base
			},
			7001: {
				DefaultBlock: 1000000, // ZetaChain
			},
		},
	}
	utils.Initialize(testConfig)

	// Create intent service
	intentService, err := services.NewIntentService(mockEthClient, mockDB, `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":false,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"targetChain","type":"uint256"},{"indexed":false,"internalType":"address","name":"receiver","type":"address"},{"indexed":false,"internalType":"uint256","name":"tip","type":"uint256"},{"indexed":false,"internalType":"bytes32","name":"salt","type":"bytes32"}],"name":"IntentInitiated","type":"event"}]`, 7001)
	assert.NoError(t, err)

	// Create fulfillment service
	clients := map[uint64]*ethclient.Client{
		42161: mockEthClient, // Arbitrum
		8453:  mockEthClient, // Base
		7001:  mockEthClient, // ZetaChain
	}
	contractAddresses := map[uint64]string{
		42161: "0x1234567890123456789012345678901234567890",
		8453:  "0x0987654321098765432109876543210987654321",
		7001:  "0x1234567890123456789012345678901234567890",
	}

	// Create default blocks map from test config
	defaultBlocks := make(map[uint64]uint64)
	for chainID, chainConfig := range testConfig.ChainConfigs {
		defaultBlocks[chainID] = chainConfig.DefaultBlock
	}

	fulfillmentService, err := services.NewFulfillmentService(clients, contractAddresses, db.DBInterface(mockDB), `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":false,"internalType":"address","name":"fulfiller","type":"address"}],"name":"IntentFulfilled","type":"event"}]`, defaultBlocks)
	assert.NoError(t, err)

	// Set up Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	server := NewServer(fulfillmentService, intentService)

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		// Intent routes
		intents := v1.Group("/intents")
		{
			intents.POST("", server.CreateIntent)
			intents.GET("/:id", server.GetIntent)
			intents.GET("", server.ListIntents)
		}
	}

	return router, mockDB, intentService, fulfillmentService
}

func TestIntentHandlers(t *testing.T) {
	router, mockDB, _, _ := setupTestRouter(t)

	// Test CreateIntent with valid data
	t.Run("CreateIntent_Valid", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"id":                "0x1234567890123456789012345678901234567890123456789012345678901234",
			"source_chain":      uint64(7001),
			"destination_chain": uint64(42161),
			"token":             "0x1234567890123456789012345678901234567890",
			"amount":            "1000000000000000000",
			"recipient":         "0x0987654321098765432109876543210987654321",
			"intent_fee":        "100000000000000000",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/intents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		var response models.IntentResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, reqBody["id"], response.ID)
		assert.Equal(t, reqBody["source_chain"], response.SourceChain)
		assert.Equal(t, reqBody["destination_chain"], response.DestinationChain)
	})

	// Test CreateIntent with invalid chain
	t.Run("CreateIntent_InvalidChain", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"id":                "0x1234567890123456789012345678901234567890123456789012345678901234",
			"source_chain":      uint64(9999), // Invalid chain ID
			"destination_chain": uint64(42161),
			"token":             "0x1234567890123456789012345678901234567890",
			"amount":            "1000000000000000000",
			"recipient":         "0x0987654321098765432109876543210987654321",
			"intent_fee":        "100000000000000000",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/intents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	// Test CreateIntent with invalid amount
	t.Run("CreateIntent_InvalidAmount", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"id":                "0x1234567890123456789012345678901234567890123456789012345678901234",
			"source_chain":      uint64(7001),
			"destination_chain": uint64(42161),
			"token":             "0x1234567890123456789012345678901234567890",
			"amount":            "-1000000000000000000", // negative amount
			"recipient":         "0x0987654321098765432109876543210987654321",
			"intent_fee":        "100000000000000000",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/intents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	// Test CreateIntent with invalid token address
	t.Run("CreateIntent_InvalidToken", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"id":                "0x1234567890123456789012345678901234567890123456789012345678901234",
			"source_chain":      uint64(7001),
			"destination_chain": uint64(42161),
			"token":             "invalid_token",
			"amount":            "1000000000000000000",
			"recipient":         "0x0987654321098765432109876543210987654321",
			"intent_fee":        "100000000000000000",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/intents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	// Test GetIntent with valid ID
	t.Run("GetIntent_Valid", func(t *testing.T) {
		// Create test intent
		now := time.Now()
		intent := &models.Intent{
			ID:               "0x1234567890123456789012345678901234567890123456789012345678901234",
			SourceChain:      7001,
			DestinationChain: 42161,
			Token:            "0x1234567890123456789012345678901234567890",
			Amount:           "1000000000000000000",
			Recipient:        "0x0987654321098765432109876543210987654321",
			IntentFee:        "100000000000000000",
			Status:           models.IntentStatusPending,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		err := mockDB.CreateIntent(context.Background(), intent)
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/intents/"+intent.ID, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var response models.IntentResponse
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, intent.ID, response.ID)
	})

	// Test GetIntent with invalid ID format
	t.Run("GetIntent_InvalidFormat", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/intents/invalid_id", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	// Test ListIntents with pagination
	t.Run("ListIntents_Pagination", func(t *testing.T) {
		// Create multiple test intents
		for i := 0; i < 3; i++ {
			intent := &models.Intent{
				ID:               fmt.Sprintf("0x%064d", i),
				SourceChain:      7001,
				DestinationChain: 42161,
				Token:            "0x1234567890123456789012345678901234567890",
				Amount:           "1000000000000000000",
				Recipient:        "0x0987654321098765432109876543210987654321",
				IntentFee:        "100000000000000000",
				Status:           models.IntentStatusPending,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}
			err := mockDB.CreateIntent(context.Background(), intent)
			assert.NoError(t, err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/intents?limit=2&offset=0", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var response []models.IntentResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 2)
	})

	// Test ListIntents with status filter
	t.Run("ListIntents_StatusFilter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/intents?status=pending", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var response []models.IntentResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		for _, intent := range response {
			assert.Equal(t, "pending", intent.Status)
		}
	})
}
