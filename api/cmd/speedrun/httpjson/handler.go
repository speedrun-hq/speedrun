package httpjson

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/speedrun-hq/speedrun/api/db"
	web "github.com/speedrun-hq/speedrun/api/http"
	"github.com/speedrun-hq/speedrun/api/logging"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/speedrun-hq/speedrun/api/services"
)

type handler struct {
	*gin.Engine

	deps   Dependencies
	logger zerolog.Logger
}

type Config struct {
	Dependencies

	Addr           string
	AllowedOrigins string
	LogRequests    bool

	Logger zerolog.Logger
}

type Dependencies struct {
	Database            db.Database
	IntentServices      map[uint64]IntentService
	FulfillmentServices map[uint64]FulfillmentService
	Metrics             *services.MetricsService
}

// IntentService defines the interface for intent service operations
type IntentService interface {
	GetIntent(ctx context.Context, id string) (*models.Intent, error)
	ListIntents(ctx context.Context) ([]*models.Intent, error)
	CreateIntent(
		ctx context.Context,
		id string,
		sourceChain uint64,
		destinationChain uint64,
		token, amount, recipient, sender, intentFee string,
		timestamp ...time.Time,
	) (*models.Intent, error)
	GetIntentsBySender(ctx context.Context, sender string) ([]*models.Intent, error)
	GetIntentsByRecipient(ctx context.Context, recipient string) ([]*models.Intent, error)
}

type FulfillmentService interface {
	CreateFulfillment(ctx context.Context, id, txHash string) error
}

const (
	RequestTimeout = 10 * time.Second
)

var (
	ErrNotFound      = errors.New("not found")
	ErrParamRequired = errors.New("param required")
)

func New(cfg Config) *http.Server {
	cfg.Logger = cfg.Logger.With().Str(logging.FieldModule, "api").Logger()

	h := newHandler(gin.New(), cfg)

	return &http.Server{
		Addr:    cfg.Addr,
		Handler: h,

		// Time to read the request headers/body
		ReadTimeout: 15 * time.Second,

		// Time to write the response
		WriteTimeout: 15 * time.Second,

		// Time to keep connections alive
		IdleTimeout: 60 * time.Second,
	}
}

func newHandler(router *gin.Engine, cfg Config) *handler {
	h := &handler{
		Engine: router,
		deps:   cfg.Dependencies,
		logger: cfg.Logger,
	}

	logLevel := zerolog.DebugLevel
	if cfg.LogRequests {
		logLevel = zerolog.InfoLevel
	}

	h.Use(
		gin.Recovery(),
		web.Zerolog(cfg.Logger, logLevel),
		web.Timeout(RequestTimeout, cfg.Logger),
		web.CORS(cfg.AllowedOrigins),
	)

	h.setupAPIRoutes()
	h.setupObservabilityRoutes()

	return h
}

func (h *handler) setupAPIRoutes() {
	v1 := h.Group("/api/v1")

	h.setupIntentRoutes(v1)
	h.setupFulfillmentRoutes(v1)
}

func (h *handler) setupObservabilityRoutes() {
	h.GET("/health", h.getHealthCheck)

	if h.deps.Metrics != nil {
		h.GET("/metrics", gin.WrapH(h.deps.Metrics.GetHandler()))

		// "Metrics summary endpoint for debugging" (c)
		h.GET("/api/v1/metrics", h.getMetricsSummary)
	}
}

func (h *handler) getHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *handler) getMetricsSummary(c *gin.Context) {
	summary := h.deps.Metrics.GetMetricsSummary()
	c.JSON(http.StatusOK, summary)
}
