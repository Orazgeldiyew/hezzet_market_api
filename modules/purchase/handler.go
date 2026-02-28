package purchase

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

// CreatePO godoc
// @Summary Create purchase order (draft)
// @Description Create a draft purchase order. Call POST /:id/receive to receive goods and deduct supplier debt.
// @Tags PurchaseOrders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreatePORequest true "Purchase order request"
// @Success 201 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/purchases [post]
func (h *Handler) CreatePO(c *gin.Context) {
	var req CreatePORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.CreatePO(c.Request.Context(), req, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// ListPOs godoc
// @Summary List purchase orders
// @Description List purchase orders with optional filters and pagination.
// @Tags PurchaseOrders
// @Produce json
// @Security BearerAuth
// @Param supplier_id query int false "Supplier ID"
// @Param warehouse_id query int false "Warehouse ID"
// @Param status query string false "Status (draft|received|cancelled)"
// @Param date_from query string false "RFC3339 datetime (inclusive)"
// @Param date_to query string false "RFC3339 datetime (inclusive)"
// @Param page query int false "page"
// @Param limit query int false "limit"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/purchases [get]
func (h *Handler) ListPOs(c *gin.Context) {
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

	items, total, err := h.svc.ListPOs(c.Request.Context(), supplierID, warehouseID, status, dateFrom, dateTo, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}
	response.List(c, items, page, limit, offset, total)
}

// GetPO godoc
// @Summary Get purchase order detail
// @Description Returns a purchase order with all its items and transaction ID.
// @Tags PurchaseOrders
// @Produce json
// @Security BearerAuth
// @Param id path int true "Purchase Order ID"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/purchases/{id} [get]
func (h *Handler) GetPO(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid purchase order id"))
		return
	}

	out, err := h.svc.GetPO(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// ReceivePO godoc
// @Summary Receive purchase order
// @Description Transitions a draft PO to received: adds stock, creates an expense finance transaction (supplier debt).
// @Tags PurchaseOrders
// @Produce json
// @Security BearerAuth
// @Param id path int true "Purchase Order ID"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/purchases/{id}/receive [post]
func (h *Handler) ReceivePO(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid purchase order id"))
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.ReceivePO(c.Request.Context(), id, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// CancelPO godoc
// @Summary Cancel purchase order
// @Description Cancels a draft purchase order. Received orders cannot be cancelled.
// @Tags PurchaseOrders
// @Produce json
// @Security BearerAuth
// @Param id path int true "Purchase Order ID"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/purchases/{id}/cancel [post]
func (h *Handler) CancelPO(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid purchase order id"))
		return
	}

	if err := h.svc.CancelPO(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"cancelled": true})
}

// AddPayment godoc
// @Summary Add supplier payment
// @Description Record a payment to the supplier for a received purchase order. Reduces outstanding debt.
// @Tags PurchaseOrders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Purchase Order ID"
// @Param body body AddPaymentRequest true "Payment details"
// @Success 201 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/purchases/{id}/payments [post]
func (h *Handler) AddPayment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid purchase order id"))
		return
	}
	var req AddPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.AddPayment(c.Request.Context(), id, req, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// DebtSummary godoc
// @Summary Supplier debt summary
// @Description Returns all suppliers with outstanding unpaid balances, ordered by debt descending.
// @Tags PurchaseOrders
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/purchases/debt [get]
func (h *Handler) DebtSummary(c *gin.Context) {
	out, err := h.svc.DebtSummary(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}
