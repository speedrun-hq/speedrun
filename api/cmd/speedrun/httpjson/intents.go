package httpjson

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	web "github.com/speedrun-hq/speedrun/api/http"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/speedrun-hq/speedrun/api/utils"
)

func (h *handler) setupIntentRoutes(rg *gin.RouterGroup) {
	intents := rg.Group("/intents")

	intents.GET("", h.listIntents)
	intents.POST("", h.createIntent)
	intents.GET("/:id", h.getIntent)
	intents.GET("/sender/:sender", h.getIntentsBySender)
	intents.GET("/recipient/:recipient", h.getIntentsByRecipient)
}

func (h *handler) createIntent(c *gin.Context) {
	var req models.CreateIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		web.ErrBadRequest(c, errors.Wrap(err, "unable to parse json"))
		return
	}

	if err := utils.ValidateIntentRequest(&req); err != nil {
		web.ErrBadRequest(c, errors.Wrap(err, "invalid intent"))
		return
	}

	// When creating intents through the API, we use the current time
	// For blockchain events, the block timestamp will be used instead
	now := time.Now()

	// Create intent
	intent := &models.Intent{
		ID:               req.ID,
		SourceChain:      req.SourceChain,
		DestinationChain: req.DestinationChain,
		Token:            req.Token,
		Amount:           req.Amount,
		Recipient:        req.Recipient,
		Sender:           req.Sender,
		IntentFee:        req.IntentFee,
		Status:           models.IntentStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := h.deps.Database.CreateIntent(c.Request.Context(), intent); err != nil {
		h.logger.Error().Err(err).Any("intent", intent).Msg("Unable to create intent")

		// note: this error might expose sensitive information
		// ideally it should become `switch{} + errors.Is(...)` with predefined error messages
		web.ErrInternalServerError(c, errors.Wrap(err, "unable to store intent"))
		return
	}

	// Return response
	c.JSON(http.StatusCreated, intent.ToResponse())
}

func (h *handler) getIntent(c *gin.Context) {
	ctx := c.Request.Context()

	id := c.Param("id")
	if id == "" {
		web.ErrBadRequest(c, errors.Wrap(ErrParamRequired, "intent id"))
		return
	}

	// todo: refactor to have only ONE service call,
	// todo: the logic should be hidden under the service layer!

	// Get the intent from the database first
	intent, err := h.deps.Database.GetIntent(ctx, id)
	if err != nil {
		web.ErrNotFound(c, errors.Wrap(ErrNotFound, "intent"))
		return
	}

	service, err := h.resolveIntentService(intent.SourceChain)
	if err != nil {
		web.ErrBadRequest(c, err)
		return
	}

	// Get the intent from the service to get any updates
	updatedIntent, err := service.GetIntent(ctx, id)
	if err != nil {
		// if not found in service, return the database version
		c.JSON(http.StatusOK, intent.ToResponse())
		return
	}

	c.JSON(http.StatusOK, updatedIntent.ToResponse())
}

func (h *handler) listIntents(c *gin.Context) {
	ctx := c.Request.Context()

	// todo: refactor to have only ONE service call,
	// todo: the logic should be hidden under the service layer!

	// Get all intents from the database
	intents, err := h.deps.Database.ListIntents(ctx)
	if err != nil {
		web.ErrInternalServerError(c, errors.Wrap(err, "unable to list intents"))
		return
	}

	// Convert intents to responses
	responses := make([]*models.IntentResponse, len(intents))
	for i, intent := range intents {
		service, err := h.resolveIntentService(intent.SourceChain)
		if err != nil {
			// fallback 1
			responses[i] = intent.ToResponse()
			continue
		}

		updatedIntent, err := service.GetIntent(ctx, intent.ID)
		if err != nil {
			// fallback 2
			responses[i] = intent.ToResponse()
			continue
		}

		responses[i] = updatedIntent.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}

func (h *handler) getIntentsBySender(c *gin.Context) {
	ctx := c.Request.Context()

	sender := c.Param("sender")
	if sender == "" {
		web.ErrBadRequest(c, errors.Wrap(ErrParamRequired, "sender address"))
		return
	}

	if err := utils.ValidateAddress(sender); err != nil {
		web.ErrBadRequest(c, errors.Wrap(err, "invalid sender address"))
		return
	}

	// Get intents by sender from the database
	intents, err := h.deps.Database.ListIntentsBySender(ctx, sender)
	if err != nil {
		web.ErrInternalServerError(c, errors.Wrap(err, "unable to list intents by sender"))
		return
	}

	// Convert intents to responses
	responses := make([]*models.IntentResponse, len(intents))
	for i, intent := range intents {
		responses[i] = intent.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}

func (h *handler) getIntentsByRecipient(c *gin.Context) {
	ctx := c.Request.Context()

	recipient := c.Param("recipient")
	if recipient == "" {
		web.ErrBadRequest(c, errors.Wrap(ErrParamRequired, "recipient address"))
		return
	}

	// Get pagination parameters
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	// Convert to integers
	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		web.ErrBadRequest(c, errors.New("invalid page parameter"))
		return
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil || pageSizeInt < 1 || pageSizeInt > 100 {
		web.ErrBadRequest(c, errors.New("invalid page_size parameter (must be between 1 and 100)"))
		return
	}

	// Validate address format
	if !utils.IsValidAddress(recipient) {
		web.ErrBadRequest(c, errors.New("invalid recipient address format"))
		return
	}

	// Get intents with pagination using optimized method
	intents, totalCount, err := h.deps.Database.ListIntentsByRecipientPaginatedOptimized(
		ctx,
		recipient,
		pageInt,
		pageSizeInt,
	)

	if err != nil {
		web.ErrInternalServerError(c, errors.Wrap(err, "unable to list intents by recipient"))
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

func (h *handler) resolveIntentService(chainID uint64) (IntentService, error) {
	s, ok := h.deps.IntentServices[chainID]
	if !ok {
		return nil, errors.Wrapf(ErrNotFound, "intent service for chain %d", chainID)
	}

	return s, nil
}
