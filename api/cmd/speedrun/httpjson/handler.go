package httpjson

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/speedrun-hq/speedrun/api/db"
	web "github.com/speedrun-hq/speedrun/api/http"
	"github.com/speedrun-hq/speedrun/api/http/timeout"
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
	GetFulfillment(ctx context.Context, id string) (*models.Fulfillment, error)
}

const (
	requestTimeout = 10 * time.Second
	rwTimeout      = 15 * time.Second
	maxPageSize    = 100
)

var (
	ErrNotFound      = errors.New("not found")
	ErrParamRequired = errors.New("param required")
)

func New(cfg Config) *http.Server {
	return &http.Server{
		Addr:    cfg.Addr,
		Handler: newHandler(cfg, gin.New()),

		// Time to read the request headers/body
		ReadTimeout: rwTimeout,

		// Time to write the response
		WriteTimeout: rwTimeout,

		// Time to keep connections alive
		IdleTimeout: 60 * time.Second,

		// Max header bytes (1MB)
		MaxHeaderBytes: 1024 * 1024,
	}
}

func newHandler(cfg Config, router *gin.Engine) *handler {
	h := &handler{
		Engine: router,
		deps:   cfg.Dependencies,
		logger: cfg.Logger.With().Str(logging.FieldModule, "api").Logger(),
	}

	logLevel := zerolog.DebugLevel
	if cfg.LogRequests {
		logLevel = zerolog.InfoLevel
	}

	h.Use(
		gin.Recovery(),
		web.Zerolog(cfg.Logger, logLevel),
		timeout.New(requestTimeout, cfg.Logger),
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

type paginationParams struct {
	Page     int
	PageSize int
}

var errPageSize = errors.Errorf("invalid page_size parameter (must be between 1 and %d)", maxPageSize)

func resolvePagination(c *gin.Context) (paginationParams, error) {
	var (
		pageRaw     = c.DefaultQuery("page", "1")
		pageSizeRaw = c.DefaultQuery("page_size", "20")
	)

	page, err := strconv.Atoi(pageRaw)
	if err != nil || page < 1 {
		return paginationParams{}, errors.New("invalid page parameter")
	}

	pageSize, err := strconv.Atoi(pageSizeRaw)
	if err != nil || pageSize < 1 || pageSize > maxPageSize {
		return paginationParams{}, errPageSize
	}

	return paginationParams{
		Page:     page,
		PageSize: pageSize,
	}, nil
}
