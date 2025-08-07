package httpjson

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/speedrun-hq/speedrun/api/db"
	"github.com/speedrun-hq/speedrun/api/logging"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/speedrun-hq/speedrun/api/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"gopkg.in/h2non/gentleman.v2"
)

type testSuite struct {
	t *testing.T

	Ctx                 context.Context
	Client              *gentleman.Client
	Database            *db.MockDB
	IntentServices      map[uint64]*MockIntentService
	FulfillmentServices map[uint64]*MockFulfillmentService

	Logger zerolog.Logger
}

func newTestSuite(t *testing.T) *testSuite {
	gin.SetMode(gin.TestMode)

	var (
		ctx      = context.Background()
		logger   = logging.NewTesting(t)
		router   = gin.New()
		database = &db.MockDB{}

		ethIntentMock      = &MockIntentService{}
		ethFulfillmentMock = &MockFulfillmentService{}
	)

	cfg := Config{
		Logger:      logger,
		LogRequests: true,
		Dependencies: Dependencies{
			Database: database,
			IntentServices: map[uint64]IntentService{
				1: IntentService(ethIntentMock),
			},
			FulfillmentServices: map[uint64]FulfillmentService{
				1: FulfillmentService(ethFulfillmentMock),
			},
			Metrics: nil,
		},
	}

	// Create handler
	h := newHandler(cfg, router)
	// Run test server
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)

	client := gentleman.New()
	client.BaseURL(server.URL)

	// "Initialize validation with nil config to allow test chains" (c)
	utils.Initialize(nil)

	return &testSuite{
		t:        t,
		Ctx:      ctx,
		Client:   client,
		Logger:   logger,
		Database: database,
		IntentServices: map[uint64]*MockIntentService{
			1: ethIntentMock,
		},
		FulfillmentServices: map[uint64]*MockFulfillmentService{
			1: ethFulfillmentMock,
		},
	}
}

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

func (m *MockIntentService) CreateIntent(
	ctx context.Context,
	id string,
	sourceChain, destinationChain uint64,
	token, amount, recipient, sender, intentFee string,
	timestamp ...time.Time,
) (*models.Intent, error) {
	args := m.Called(ctx, id, sourceChain, destinationChain, token, amount, recipient, sender, intentFee, timestamp)
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

// MockFulfillmentService is a mock implementation of the FulfillmentService
type MockFulfillmentService struct {
	mock.Mock
}

func (m *MockFulfillmentService) CreateFulfillment(ctx context.Context, id string, txHash string) error {
	args := m.Called(ctx, id, txHash)
	return args.Error(0)
}

func TestHandler(t *testing.T) {
	t.Run("health check", func(t *testing.T) {
		// ARRANGE
		ts := newTestSuite(t)

		// ACT
		resp, err := ts.Client.Get().AddPath("/health").Do()

		// ASSERT
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assertResponseContainsJSON(t, resp, "status", "ok")
	})
}

func assertResponseContainsJSON(t *testing.T, res *gentleman.Response, path string, contains string) {
	r := gjson.GetBytes(res.Bytes(), path)

	assert.Contains(t, r.String(), contains, res.String())
}
