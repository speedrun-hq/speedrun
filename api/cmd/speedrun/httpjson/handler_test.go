package httpjson

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/speedrun-hq/speedrun/api/logging"
	"github.com/speedrun-hq/speedrun/api/testing/mocks"
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
	Database            *mocks.DatabaseMock
	IntentServices      map[uint64]*mocks.IntentServiceMock
	FulfillmentServices map[uint64]*mocks.FulfillmentServiceMock

	Logger zerolog.Logger
}

var once sync.Once

func newTestSuite(t *testing.T) *testSuite {
	once.Do(func() {
		gin.SetMode(gin.TestMode)
	})

	var (
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		logger      = logging.NewTesting(t)
		router      = gin.New()

		database           = mocks.NewDatabaseMock(t)
		ethIntentMock      = mocks.NewIntentServiceMock(t)
		ethFulfillmentMock = mocks.NewFulfillmentServiceMock(t)
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
	t.Cleanup(cancel)

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
		IntentServices: map[uint64]*mocks.IntentServiceMock{
			1: ethIntentMock,
		},
		FulfillmentServices: map[uint64]*mocks.FulfillmentServiceMock{
			1: ethFulfillmentMock,
		},
	}
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

func numOfArgs(v uint) []any {
	vals := make([]any, 0, v)

	for v > 0 {
		vals = append(vals, mock.Anything)
		v--
	}

	return vals
}
