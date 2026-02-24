package finance

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

// ListPaymentTypes godoc
// @Summary List payment types
// @Description Returns all active payment types.
// @Tags Finance
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=[]PaymentType}
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/payment-types [get]
func (h *Handler) ListPaymentTypes(c *gin.Context) {
	out, err := h.svc.ListPaymentTypes(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// ListTransactions godoc
// @Summary List transactions
// @Description List financial transactions with filters + pagination.
// @Tags Finance
// @Produce json
// @Security BearerAuth
// @Param type query string false "income or expense"
// @Param status query string false "pending, partial, paid, canceled"
// @Param related_table query string false "sale, purchase, manual, adjustment"
// @Param related_id query int false "Related entity ID"
// @Param date_from query string false "RFC3339 datetime (inclusive)"
// @Param date_to query string false "RFC3339 datetime (inclusive)"
// @Param payment_type_id query int false "Payment type ID"
// @Param page query int false "page"
// @Param limit query int false "limit"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/transactions [get]
func (h *Handler) ListTransactions(c *gin.Context) {
	var f TransactionFilter

	if v := c.Query("type"); v != "" {
		f.Type = &v
	}
	if v := c.Query("status"); v != "" {
		f.Status = &v
	}
	if v := c.Query("related_table"); v != "" {
		f.RelatedTable = &v
	}
	if v := c.Query("related_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.RelatedID = &n
		}
	}
	if v := c.Query("date_from"); v != "" {
		tm, err := time.Parse(time.RFC3339, v)
		if err != nil {
			c.Error(apperr.Validation("date_from must be RFC3339"))
			return
		}
		f.DateFrom = &tm
	}
	if v := c.Query("date_to"); v != "" {
		tm, err := time.Parse(time.RFC3339, v)
		if err != nil {
			c.Error(apperr.Validation("date_to must be RFC3339"))
			return
		}
		f.DateTo = &tm
	}
	if v := c.Query("payment_type_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.PaymentTypeID = &n
		}
	}

	page := 1
	limit := 10
	offset := 0
	if pRaw, ok := c.Get("pagination"); ok {
		if p, ok := pRaw.(middleware.Pagination); ok {
			page = p.Page
			limit = p.Limit
			offset = p.Offset
		}
	}

	out, err := h.svc.ListTransactions(c.Request.Context(), f, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}
	response.List(c, out.Items, page, out.Limit, out.Offset, out.Total)
}

// GetTransaction godoc
// @Summary Get transaction detail
// @Description Returns a transaction with all its payments.
// @Tags Finance
// @Produce json
// @Security BearerAuth
// @Param id path int true "Transaction ID"
// @Success 200 {object} response.APIResponse{data=TransactionDetail}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/transactions/{id} [get]
func (h *Handler) GetTransaction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid transaction id"))
		return
	}

	out, err := h.svc.GetTransaction(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// CreateManual godoc
// @Summary Create manual transaction
// @Description Create a manual income or expense transaction with optional initial payment.
// @Tags Finance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateManualRequest true "Manual transaction request"
// @Success 201 {object} response.APIResponse{data=TransactionDetail}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/transactions/manual [post]
func (h *Handler) CreateManual(c *gin.Context) {
	var req CreateManualRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.CreateManual(c.Request.Context(), req, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// AddPayment godoc
// @Summary Add payment to transaction
// @Description Add a partial or full payment to an existing transaction.
// @Tags Finance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Transaction ID"
// @Param body body AddPaymentRequest true "Payment request"
// @Success 201 {object} response.APIResponse{data=TransactionDetail}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/transactions/{id}/payments [post]
func (h *Handler) AddPayment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid transaction id"))
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

// CancelTransaction godoc
// @Summary Cancel transaction
// @Description Cancel a transaction. Idempotent if already canceled.
// @Tags Finance
// @Produce json
// @Security BearerAuth
// @Param id path int true "Transaction ID"
// @Success 200 {object} response.APIResponse{data=Transaction}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/transactions/{id}/cancel [post]
func (h *Handler) CancelTransaction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid transaction id"))
		return
	}

	out, err := h.svc.CancelTransaction(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}
