package supplierreturn

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func extractUserID(c *gin.Context) int64 {
	v, _ := c.Get("user_id")
	uid, _ := v.(int64)
	return uid
}

// Create godoc
// @Summary Create supplier return (draft)
// @Description Create a draft supplier return. Call POST /:id/confirm to confirm and deduct stock.
// @Tags SupplierReturns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateRequest true "Return request"
// @Success 201 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/supplier-returns [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.Create(c.Request.Context(), req, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// List godoc
// @Summary List supplier returns
// @Tags SupplierReturns
// @Produce json
// @Security BearerAuth
// @Param supplier_id query int false "Supplier ID"
// @Param warehouse_id query int false "Warehouse ID"
// @Param status query string false "Status (draft|confirmed|cancelled)"
// @Param date_from query string false "RFC3339"
// @Param date_to query string false "RFC3339"
// @Param page query int false "page"
// @Param limit query int false "limit"
// @Success 200 {object} response.APIResponse
// @Router /api/supplier-returns [get]
func (h *Handler) List(c *gin.Context) {
	var supplierID, warehouseID *int64
	var status *string
	var dateFrom, dateTo *time.Time

	if v := c.Query("supplier_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			supplierID = &n
		}
	}
	if v := c.Query("warehouse_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			warehouseID = &n
		}
	}
	if v := c.Query("status"); v != "" {
		status = &v
	}
	if v := c.Query("date_from"); v != "" {
		tm, err := time.Parse(time.RFC3339, v)
		if err != nil {
			c.Error(apperr.Validation("date_from must be RFC3339"))
			return
		}
		dateFrom = &tm
	}
	if v := c.Query("date_to"); v != "" {
		tm, err := time.Parse(time.RFC3339, v)
		if err != nil {
			c.Error(apperr.Validation("date_to must be RFC3339"))
			return
		}
		dateTo = &tm
	}

	page := 1
	limit := 20
	offset := 0
	if pRaw, ok := c.Get("pagination"); ok {
		if p, ok := pRaw.(middleware.Pagination); ok {
			page = p.Page
			limit = p.Limit
			offset = p.Offset
		}
	}

	items, total, err := h.svc.List(c.Request.Context(), supplierID, warehouseID, status, dateFrom, dateTo, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}
	response.List(c, items, page, limit, offset, total)
}

// GetByID godoc
// @Summary Get supplier return detail
// @Tags SupplierReturns
// @Produce json
// @Security BearerAuth
// @Param id path int true "Return ID"
// @Success 200 {object} response.APIResponse
// @Router /api/supplier-returns/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid return id"))
		return
	}

	out, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// Confirm godoc
// @Summary Confirm supplier return
// @Description Deducts stock from warehouse and reduces supplier debt.
// @Tags SupplierReturns
// @Produce json
// @Security BearerAuth
// @Param id path int true "Return ID"
// @Success 200 {object} response.APIResponse
// @Router /api/supplier-returns/{id}/confirm [post]
func (h *Handler) Confirm(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid return id"))
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.Confirm(c.Request.Context(), id, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// Cancel godoc
// @Summary Cancel supplier return
// @Tags SupplierReturns
// @Produce json
// @Security BearerAuth
// @Param id path int true "Return ID"
// @Success 200 {object} response.APIResponse
// @Router /api/supplier-returns/{id}/cancel [post]
func (h *Handler) Cancel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid return id"))
		return
	}

	if err := h.svc.Cancel(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"cancelled": true})
}
