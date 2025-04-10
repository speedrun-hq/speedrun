package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/speedrun-hq/speedrun/api/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	utils.Initialize(nil) // Initialize validation with nil config to allow test chains
}

// MockFulfillmentService is a mock implementation of the FulfillmentService
type MockFulfillmentService struct {
	mock.Mock
}

func (m *MockFulfillmentService) CreateFulfillment(ctx context.Context, id string, txHash string) error {
	args := m.Called(ctx, id, txHash)
	return args.Error(0)
}

func setupFulfillmentTestRouter() (*gin.Engine, *MockDatabase, map[uint64]FulfillmentServiceInterface) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	mockDB := new(MockDatabase)
	mockServices := make(map[uint64]FulfillmentServiceInterface)
	mockService := new(MockFulfillmentService)
	mockServices[1] = mockService

	InitFulfillmentHandlers(mockDB, mockServices)

	router.POST("/fulfillments", CreateFulfillment)
	router.GET("/fulfillments/:id", GetFulfillment)

	return router, mockDB, mockServices
}

func TestCreateFulfillment(t *testing.T) {
	router, mockDB, mockServices := setupFulfillmentTestRouter()

	validID := "0x1234567890123456789012345678901234567890123456789012345678901234"
	validTxHash := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	validAsset := "ETH"
	validAmount := "1.0"
	validReceiver := "0x1234567890123456789012345678901234567890"

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		setupMock      func()
	}{
		{
			name: "Valid Fulfillment Creation",
			requestBody: models.CreateFulfillmentRequest{
				ID:       validID,
				Asset:    validAsset,
				Amount:   validAmount,
				Receiver: validReceiver,
				ChainID:  1,
				TxHash:   validTxHash,
			},
			expectedStatus: http.StatusCreated,
			setupMock: func() {
				mockServices[1].(*MockFulfillmentService).On(
					"CreateFulfillment",
					mock.Anything,
					validID,
					validTxHash,
				).Return(nil)
				mockDB.On("CreateFulfillment", mock.Anything, mock.MatchedBy(func(f *models.Fulfillment) bool {
					return f.ID == validID && f.TxHash == validTxHash
				})).Return(nil)
			},
		},
		{
			name:           "Invalid Request Body",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			setupMock:      func() {},
		},
		{
			name: "Invalid Chain ID",
			requestBody: models.CreateFulfillmentRequest{
				ID:       validID,
				Asset:    validAsset,
				Amount:   validAmount,
				Receiver: validReceiver,
				ChainID:  999,
				TxHash:   validTxHash,
			},
			expectedStatus: http.StatusBadRequest,
			setupMock:      func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/fulfillments", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusCreated {
				mockDB.AssertExpectations(t)
				mockServices[1].(*MockFulfillmentService).AssertExpectations(t)
			}
		})
	}
}

func TestGetFulfillment(t *testing.T) {
	validID := "0x1234567890123456789012345678901234567890123456789012345678901234"
	mockFulfillment := &models.Fulfillment{
		ID:     validID,
		TxHash: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
	}

	var mockDB *MockDatabase

	tests := []struct {
		name           string
		fulfillmentID  string
		expectedStatus int
		setupMock      func()
	}{
		{
			name:           "Valid Fulfillment Retrieval",
			fulfillmentID:  validID,
			expectedStatus: http.StatusOK,
			setupMock: func() {
				mockDB.On("GetFulfillment", mock.Anything, validID).Return(mockFulfillment, nil)
			},
		},
		{
			name:           "Invalid Fulfillment ID Format",
			fulfillmentID:  "invalid-id",
			expectedStatus: http.StatusBadRequest,
			setupMock:      func() {},
		},
		{
			name:           "Fulfillment Not Found",
			fulfillmentID:  validID,
			expectedStatus: http.StatusNotFound,
			setupMock: func() {
				mockDB.On("GetFulfillment", mock.Anything, validID).Return(nil, assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fresh mock and router for each test
			mockDB = new(MockDatabase)
			mockServices := make(map[uint64]FulfillmentServiceInterface)
			mockService := new(MockFulfillmentService)
			mockServices[1] = mockService

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(gin.Recovery())

			InitFulfillmentHandlers(mockDB, mockServices)
			router.GET("/fulfillments/:id", GetFulfillment)

			tt.setupMock()

			req := httptest.NewRequest("GET", "/fulfillments/"+tt.fulfillmentID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus != http.StatusBadRequest {
				mockDB.AssertExpectations(t)
			}
		})
	}
}
