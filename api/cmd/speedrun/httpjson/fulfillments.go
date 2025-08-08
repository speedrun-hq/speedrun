package httpjson

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	web "github.com/speedrun-hq/speedrun/api/http"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/speedrun-hq/speedrun/api/utils"
)

func (h *handler) setupFulfillmentRoutes(rg *gin.RouterGroup) {
	ff := rg.Group("/fulfillments")

	ff.POST("", h.createFulfillment)
	ff.GET("/:id", h.getFulfillment)
	ff.GET("", h.listFulfillments)
}

func (h *handler) createFulfillment(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.CreateFulfillmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		web.ErrBadRequest(c, errors.Wrap(err, "invalid request"))
		return
	}

	// Validate request
	if err := utils.ValidateFulfillmentRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if service exists for the chain
	service, err := h.resolveFulfillmentService(req.ChainID)
	if err != nil {
		web.ErrBadRequest(c, err)
		return
	}

	// Create fulfillment in service
	err = service.CreateFulfillment(ctx, req.ID, req.TxHash)
	if err != nil {
		web.ErrInternalServerError(c, errors.Wrap(err, "unable to create fulfillment"))
		return
	}

	// todo: refactor to have only ONE service call,
	// todo: the logic should be hidden under the service layer!

	// Create fulfillment in database
	fulfillment := &models.Fulfillment{
		ID:     req.ID,
		TxHash: req.TxHash,
	}

	if err := h.deps.Database.CreateFulfillment(ctx, fulfillment); err != nil {
		web.ErrInternalServerError(c, errors.Wrap(err, "unable to create fulfillment"))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Fulfillment created successfully",
	})
}

func (h *handler) getFulfillment(c *gin.Context) {
	ctx := c.Request.Context()

	id := c.Param("id")
	if id == "" {
		web.ErrBadRequest(c, errors.Wrap(ErrParamRequired, "fulfillment id"))
		return
	}

	// Validate bytes32 format
	if !utils.IsValidBytes32(id) {
		web.ErrBadRequest(c, errors.New("invalid fulfillment ID"))
		return
	}

	fulfillment, err := h.deps.Database.GetFulfillment(ctx, id)
	if err != nil {
		web.ErrNotFound(c, errors.Wrap(ErrNotFound, "fulfillment"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"fulfillment": fulfillment})
}

func (h *handler) listFulfillments(c *gin.Context) {
	ctx := c.Request.Context()

	pag, err := resolvePagination(c)
	if err != nil {
		web.ErrBadRequest(c, err)
		return
	}

	// Get fulfillments with pagination using optimized method
	fulfillments, totalCount, err := h.deps.Database.ListFulfillmentsPaginatedOptimized(ctx, pag.Page, pag.PageSize)
	if err != nil {
		web.ErrInternalServerError(c, err)
		return
	}

	res := models.NewPaginatedResponse(fulfillments, pag.Page, pag.PageSize, totalCount)

	c.JSON(http.StatusOK, res)
}

func (h *handler) resolveFulfillmentService(chainID uint64) (FulfillmentService, error) {
	s, ok := h.deps.FulfillmentServices[chainID]
	if !ok {
		return nil, errors.Wrapf(ErrNotFound, "fulfillment service for chain %d", chainID)
	}

	return s, nil
}
