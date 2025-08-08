package httpjson

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	web "github.com/speedrun-hq/speedrun/api/http"
	"github.com/speedrun-hq/speedrun/api/logging"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/speedrun-hq/speedrun/api/utils"
)

func (h *handler) setupIntentRoutes(rg *gin.RouterGroup) {
	intents := rg.Group("/intents")

	intents.GET("", h.listIntents)
	intents.POST("", h.createIntent)
	intents.GET(":id", h.getIntent)
	intents.GET("/sender/:sender", h.getIntentsBySender)
	intents.GET("/recipient/:recipient", h.getIntentsByRecipient)
}

func (h *handler) createIntent(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.CreateIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		web.ErrBadRequest(c, errors.Wrap(err, "invalid request"))
		return
	}

	service, err := h.resolveIntentService(req.SourceChain)
	if err != nil {
		web.ErrBadRequest(c, err)
		return
	}

	intent, err := service.CreateIntent(
		ctx,
		req.ID,
		req.SourceChain,
		req.DestinationChain,
		req.Token,
		req.Amount,
		req.Recipient,
		req.Sender,
		req.IntentFee,
	)

	if err != nil {
		// "check if it's a validation error" (ported code)
		if strings.Contains(err.Error(), "invalid") {
			web.ErrBadRequest(c, err)
			return
		}

		web.ErrInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusCreated, intent)
}

func (h *handler) getIntent(c *gin.Context) {
	ctx := c.Request.Context()

	id := c.Param("id")
	if id == "" {
		web.ErrBadRequest(c, errors.Wrap(ErrParamRequired, "intent id"))
		return
	}

	h.logger.Debug().Str(logging.FieldIntent, id).Msg("GetIntent request received")

	if !utils.ValidateBytes32(id) {
		h.logger.Debug().Str(logging.FieldIntent, id).Msg("Invalid intent ID format")
		web.ErrBadRequest(c, errors.Wrap(ErrParamRequired, "intent id"))
		return
	}

	service, err := h.resolveFirstIntentService()
	if err != nil {
		web.ErrBadRequest(c, err)
		return
	}

	intent, err := service.GetIntent(ctx, id)
	if err != nil {
		h.logger.Debug().Err(err).Str(logging.FieldIntent, id).Msgf("Error getting intent")

		// "check if it's a not found error" (ported code)
		if strings.Contains(err.Error(), "not found") {
			web.ErrNotFound(c, errors.Wrap(ErrNotFound, "intent"))
			return
		}

		web.ErrInternalServerError(c, err)
		return
	}

	h.logger.Debug().Str(logging.FieldIntent, id).Msg("Successfully retrieved intent")

	c.JSON(http.StatusOK, intent)
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

	if !utils.IsValidAddress(sender) {
		web.ErrBadRequest(c, errors.New("invalid sender address format"))
		return
	}

	pag, err := resolvePagination(c)
	if err != nil {
		web.ErrBadRequest(c, err)
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

	if !utils.IsValidAddress(recipient) {
		web.ErrBadRequest(c, errors.New("invalid recipient address format"))
		return
	}

	pag, err := resolvePagination(c)
	if err != nil {
		web.ErrBadRequest(c, err)
		return
	}

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

// just resolve any intent service.
func (h *handler) resolveFirstIntentService() (IntentService, error) {
	for _, s := range h.deps.IntentServices {
		return s, nil
	}

	return nil, errors.Wrap(ErrNotFound, "intent service")
}
