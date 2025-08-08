package httpjson

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	web "github.com/speedrun-hq/speedrun/api/http"
	"github.com/speedrun-hq/speedrun/api/models"
)

func (h *handler) setupFulfillmentRoutes(rg *gin.RouterGroup) {
	ff := rg.Group("/fulfillments")

	ff.POST("", h.createFulfillment)
	ff.GET("/:id", h.getFulfillment)
	ff.GET("", h.listFulfillments)
}

// CreateFulfillmentRequest represents the request body for creating a fulfillment
type CreateFulfillmentRequest struct {
	IntentID string `json:"intent_id" binding:"required"`
	TxHash   string `json:"tx_hash"   binding:"required"`
}

func (h *handler) createFulfillment(c *gin.Context) {
	ctx := c.Request.Context()

	var req CreateFulfillmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		web.ErrBadRequest(c, errors.Wrap(err, "invalid request"))
		return
	}

	service, err := h.resolveFirstFulfillmentService()
	if err != nil {
		web.ErrBadRequest(c, err)
	}

	err = service.CreateFulfillment(ctx, req.IntentID, req.TxHash)
	if err != nil {
		web.ErrInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "fulfillment created successfully"})
}

func (h *handler) getFulfillment(c *gin.Context) {
	ctx := c.Request.Context()

	id := c.Param("id")
	if id == "" {
		web.ErrBadRequest(c, errors.Wrap(ErrParamRequired, "fulfillment id"))
		return
	}

	service, err := h.resolveFirstFulfillmentService()
	if err != nil {
		web.ErrBadRequest(c, err)
		return
	}

	fulfillment, err := service.GetFulfillment(ctx, id)
	if err != nil {
		web.ErrInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, fulfillment)
}

func (h *handler) listFulfillments(c *gin.Context) {
	ctx := c.Request.Context()

	pag, err := resolvePagination(c)
	if err != nil {
		web.ErrBadRequest(c, err)
		return
	}

	fulfillments, totalCount, err := h.deps.Database.ListFulfillmentsPaginatedOptimized(ctx, pag.Page, pag.PageSize)
	if err != nil {
		web.ErrInternalServerError(c, err)
		return
	}

	res := models.NewPaginatedResponse(fulfillments, pag.Page, pag.PageSize, totalCount)

	c.JSON(http.StatusOK, res)
}

// just resolve any fulfillment service.
func (h *handler) resolveFirstFulfillmentService() (FulfillmentService, error) {
	for _, s := range h.deps.FulfillmentServices {
		return s, nil
	}

	return nil, errors.Wrap(ErrNotFound, "fulfillment service")
}
