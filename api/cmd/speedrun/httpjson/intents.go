package httpjson

import (
	"net/http"
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

	status := c.Query("status")

	pag, err := resolvePagination(c)
	if err != nil {
		web.ErrBadRequest(c, err)
		return
	}

	// Get intents with pagination and status filter using optimized method
	intents, totalCount, err := h.deps.Database.ListIntentsPaginatedOptimized(ctx, pag.Page, pag.PageSize, status)
	if err != nil {
		web.ErrInternalServerError(c, err)
		return
	}

	response := make([]*models.IntentResponse, 0, len(intents))
	for _, intent := range intents {
		response = append(response, intent.ToResponse())
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponse(response, pag.Page, pag.PageSize, totalCount))
}

// GetIntentsBySender handles retrieving intents by sender
func (h *handler) getIntentsBySender(c *gin.Context) {
	ctx := c.Request.Context()

	sender := c.Param("sender")
	if sender == "" {
		web.ErrBadRequest(c, errors.Wrap(ErrParamRequired, "sender address"))
		return
	}

	pag, err := resolvePagination(c)
	if err != nil {
		web.ErrBadRequest(c, err)
		return
	}

	// Validate address format
	if !utils.IsValidAddress(sender) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sender address format"})
		return
	}

	intents, totalCount, err := h.deps.Database.ListIntentsBySenderPaginatedOptimized(
		ctx,
		sender,
		pag.Page,
		pag.PageSize,
	)

	if err != nil {
		web.ErrInternalServerError(c, err)
		return
	}

	// Convert to response format
	response := make([]*models.IntentResponse, 0, len(intents))
	for _, intent := range intents {
		response = append(response, intent.ToResponse())
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponse(response, pag.Page, pag.PageSize, totalCount))
}

func (h *handler) getIntentsByRecipient(c *gin.Context) {
	ctx := c.Request.Context()

	recipient := c.Param("recipient")
	if recipient == "" {
		web.ErrBadRequest(c, errors.Wrap(ErrParamRequired, "recipient address"))
		return
	}

	pag, err := resolvePagination(c)
	if err != nil {
		web.ErrBadRequest(c, err)
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
		pag.Page,
		pag.PageSize,
	)

	if err != nil {
		web.ErrInternalServerError(c, errors.Wrap(err, "unable to list intents by recipient"))
		return
	}

	// Convert to response format
	response := make([]*models.IntentResponse, 0, len(intents))
	for _, intent := range intents {
		response = append(response, intent.ToResponse())
	}

	paginatedResponse := models.NewPaginatedResponse(response, pag.Page, pag.PageSize, totalCount)

	c.JSON(http.StatusOK, paginatedResponse)
}

func (h *handler) resolveIntentService(chainID uint64) (IntentService, error) {
	s, ok := h.deps.IntentServices[chainID]
	if !ok {
		return nil, errors.Wrapf(ErrNotFound, "intent service for chain %d", chainID)
	}

	return s, nil
}
