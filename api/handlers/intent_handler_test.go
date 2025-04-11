package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/speedrun-hq/speedrun/api/db"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockIntentService is a mock implementation of the IntentService
type MockIntentService struct {
	mock.Mock
}

func (m *MockIntentService) GetIntent(ctx context.Context, id string) (*models.Intent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Intent), args.Error(1)
}

func (m *MockIntentService) ListIntents(ctx context.Context) ([]*models.Intent, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func (m *MockIntentService) CreateIntent(ctx context.Context, id string, sourceChain uint64, destinationChain uint64, token, amount, recipient, sender, intentFee string) (*models.Intent, error) {
	args := m.Called(ctx, id, sourceChain, destinationChain, token, amount, recipient, sender, intentFee)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Intent), args.Error(1)
}

func (m *MockIntentService) GetIntentsByUser(ctx context.Context, userAddress string) ([]*models.Intent, error) {
	args := m.Called(ctx, userAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func (m *MockIntentService) GetIntentsByRecipient(ctx context.Context, recipientAddress string) ([]*models.Intent, error) {
	args := m.Called(ctx, recipientAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func (m *MockIntentService) GetIntentsBySender(ctx context.Context, senderAddress string) ([]*models.Intent, error) {
	args := m.Called(ctx, senderAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func setupTestRouter() (*gin.Engine, db.Database, map[uint64]IntentServiceInterface) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	mockDB := &MockDatabase{}
	mockServices := make(map[uint64]IntentServiceInterface)
	mockService := &MockIntentService{}
	mockServices[1] = mockService

	InitIntentHandlers(mockDB, mockServices)

	router.POST("/intents", CreateIntent)
	router.GET("/intents/:id", GetIntent)
	router.GET("/intents", ListIntents)
	router.GET("/intents/sender/:sender", GetIntentsBySender)

	return router, mockDB, mockServices
}

func TestCreateIntent(t *testing.T) {
	router, mockDB, _ := setupTestRouter()
	mockDBTyped := mockDB.(*MockDatabase)

	validID := "0x1234567890123456789012345678901234567890123456789012345678901234"
	validRecipient := "0x1234567890123456789012345678901234567890"
	validSender := "0x0987654321098765432109876543210987654321"

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		setupMock      func()
	}{
		{
			name: "Valid Intent Creation",
			requestBody: models.CreateIntentRequest{
				ID:               validID,
				SourceChain:      1,
				DestinationChain: 2,
				Token:            "ETH",
				Amount:           "1.0",
				Recipient:        validRecipient,
				Sender:           validSender,
				IntentFee:        "0.1",
			},
			expectedStatus: http.StatusCreated,
			setupMock: func() {
				mockDBTyped.On("CreateIntent", mock.Anything, mock.MatchedBy(func(i *models.Intent) bool {
					return i.ID == validID &&
						i.SourceChain == 1 &&
						i.DestinationChain == 2 &&
						i.Token == "ETH" &&
						i.Amount == "1.0" &&
						i.Recipient == validRecipient &&
						i.Sender == validSender &&
						i.IntentFee == "0.1"
				})).Return(nil)
			},
		},
		{
			name:           "Invalid Request Body",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			setupMock:      func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/intents", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusCreated {
				mockDBTyped.AssertExpectations(t)
			}
		})
	}
}

func TestGetIntent(t *testing.T) {
	router, mockDB, mockServices := setupTestRouter()

	mockIntent := &models.Intent{
		ID:               "test-id",
		SourceChain:      1,
		DestinationChain: 2,
		Token:            "ETH",
		Amount:           "1.0",
		Recipient:        "0x123",
		IntentFee:        "0.1",
		Status:           models.IntentStatusPending,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	tests := []struct {
		name           string
		intentID       string
		expectedStatus int
		setupMock      func()
	}{
		{
			name:           "Valid Intent Retrieval",
			intentID:       "test-id",
			expectedStatus: http.StatusOK,
			setupMock: func() {
				mockDB.(*MockDatabase).On("GetIntent", mock.Anything, "test-id").Return(mockIntent, nil)
				mockServices[1].(*MockIntentService).On("GetIntent", mock.Anything, "test-id").Return(mockIntent, nil)
			},
		},
		{
			name:           "Intent Not Found",
			intentID:       "non-existent",
			expectedStatus: http.StatusNotFound,
			setupMock: func() {
				mockDB.(*MockDatabase).On("GetIntent", mock.Anything, "non-existent").Return(nil, assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			req := httptest.NewRequest("GET", "/intents/"+tt.intentID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockDB.(*MockDatabase).AssertExpectations(t)
		})
	}
}

func TestListIntents(t *testing.T) {
	var mockDBTyped *MockDatabase
	var mockServices map[uint64]IntentServiceInterface

	tests := []struct {
		name           string
		expectedStatus int
		setupMock      func()
	}{
		{
			name:           "Successful List",
			expectedStatus: http.StatusOK,
			setupMock: func() {
				mockDBTyped.On("ListIntents", mock.Anything).Return([]*models.Intent{
					{
						ID:               "0x1234567890123456789012345678901234567890123456789012345678901234",
						SourceChain:      1,
						DestinationChain: 2,
						Token:            "ETH",
						Amount:           "1.0",
						Recipient:        "0x1234567890123456789012345678901234567890",
						IntentFee:        "0.1",
					},
				}, nil)
				mockServices[1].(*MockIntentService).On("GetIntent", mock.Anything, mock.Anything).Return(&models.Intent{}, nil)
			},
		},
		{
			name:           "Database Error",
			expectedStatus: http.StatusInternalServerError,
			setupMock: func() {
				mockDBTyped.On("ListIntents", mock.Anything).Return(nil, assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fresh mock and router for each test
			mockDBTyped = new(MockDatabase)
			mockServices = make(map[uint64]IntentServiceInterface)
			mockService := new(MockIntentService)
			mockServices[1] = mockService

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(gin.Recovery())

			InitIntentHandlers(mockDBTyped, mockServices)
			router.GET("/intents", ListIntents)

			tt.setupMock()

			req := httptest.NewRequest("GET", "/intents", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockDBTyped.AssertExpectations(t)
			if tt.expectedStatus == http.StatusOK {
				mockServices[1].(*MockIntentService).AssertExpectations(t)
			}
		})
	}
}

func TestGetIntentsBySender(t *testing.T) {
	router, mockDB, _ := setupTestRouter()
	mockDBTyped := mockDB.(*MockDatabase)

	mockIntents := []*models.Intent{
		{
			ID:               "test-id-1",
			SourceChain:      1,
			DestinationChain: 2,
			Token:            "ETH",
			Amount:           "1.0",
			Recipient:        "0x123",
			Sender:           "0x456",
			IntentFee:        "0.1",
			Status:           models.IntentStatusPending,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               "test-id-2",
			SourceChain:      1,
			DestinationChain: 2,
			Token:            "ETH",
			Amount:           "2.0",
			Recipient:        "0x789",
			Sender:           "0x456",
			IntentFee:        "0.2",
			Status:           models.IntentStatusPending,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	validAddress := "0x1234567890123456789012345678901234567890"

	tests := []struct {
		name           string
		senderAddress  string
		expectedStatus int
		setupMock      func()
	}{
		{
			name:           "Valid Sender Address",
			senderAddress:  validAddress,
			expectedStatus: http.StatusOK,
			setupMock: func() {
				mockDBTyped.On("ListIntentsBySender", mock.Anything, validAddress).Return(mockIntents, nil)
			},
		},
		{
			name:           "Invalid Sender Address",
			senderAddress:  "invalid-address",
			expectedStatus: http.StatusBadRequest,
			setupMock: func() {
				// No mock calls expected - validation will fail before database is called
			},
		},
		{
			name:           "Database Error",
			senderAddress:  validAddress,
			expectedStatus: http.StatusInternalServerError,
			setupMock: func() {
				mockDBTyped.On("ListIntentsBySender", mock.Anything, validAddress).Return(nil, assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock expectations for each test case
			mockDBTyped.ExpectedCalls = nil
			mockDBTyped.Calls = nil

			tt.setupMock()

			req := httptest.NewRequest("GET", "/intents/sender/"+tt.senderAddress, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				var response []*models.IntentResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, len(mockIntents), len(response))
			}
			mockDBTyped.AssertExpectations(t)
		})
	}
}
