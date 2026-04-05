package inventory

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func extractUserID(c *gin.Context) int64 {
	v, _ := c.Get("user_id")
	uid, _ := v.(int64)
	return uid
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("warehouse_id is required"))
		return
	}
	detail, err := h.repo.Create(c.Request.Context(), req, extractUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, detail)
}

func (h *Handler) List(c *gin.Context) {
	pg, _ := c.Get("pagination")
	page := pg.(middleware.Pagination)

	var warehouseID *int64
	if v := c.Query("warehouse_id"); v != "" {
		id, _ := strconv.ParseInt(v, 10, 64)
		warehouseID = &id
	}

	items, total, err := h.repo.List(c.Request.Context(), warehouseID, page.Limit, page.Offset)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.List(c, items, page.Page, page.Limit, page.Offset, total)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid id"))
		return
	}
	detail, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, detail)
}

func (h *Handler) UpdateItems(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid id"))
		return
	}
	var req UpdateItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("items are required"))
		return
	}
	if err := h.repo.UpdateItems(c.Request.Context(), id, req); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"updated": true})
}

func (h *Handler) Confirm(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid id"))
		return
	}
	if err := h.repo.Confirm(c.Request.Context(), id, extractUserID(c)); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"confirmed": true})
}

func (h *Handler) Cancel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid id"))
		return
	}
	if err := h.repo.Cancel(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"cancelled": true})
}
