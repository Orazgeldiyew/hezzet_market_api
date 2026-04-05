package customerdebt

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ListAll — all open debts (manager/admin)
func (h *Handler) ListAll(c *gin.Context) {
	pg := c.MustGet("pagination").(middleware.Pagination)
	items, total, err := h.repo.ListAll(c.Request.Context(), pg.Limit, pg.Offset)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.List(c, items, pg.Page, pg.Limit, pg.Offset, total)
}

// DebtorsSummary — list of all customers with open debts
func (h *Handler) DebtorsSummary(c *gin.Context) {
	items, err := h.repo.DebtorsSummary(c.Request.Context())
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, items)
}

// ListByCustomer — debts for a specific customer
func (h *Handler) ListByCustomer(c *gin.Context) {
	customerID, _ := strconv.ParseInt(c.Param("customer_id"), 10, 64)
	if customerID <= 0 {
		c.Error(apperr.Validation("invalid customer_id"))
		return
	}
	onlyOpen := c.Query("status") != "all"
	pg := c.MustGet("pagination").(middleware.Pagination)

	items, total, err := h.repo.ListByCustomer(c.Request.Context(), customerID, onlyOpen, pg.Limit, pg.Offset)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.List(c, items, pg.Page, pg.Limit, pg.Offset, total)
}

// GetByID — single debt with details
func (h *Handler) GetByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.Error(apperr.Validation("invalid id"))
		return
	}
	debt, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.Error(apperr.NotFound("DEBT_NOT_FOUND", "debt not found"))
			return
		}
		c.Error(apperr.Internal(err))
		return
	}

	payments, err := h.repo.GetPayments(c.Request.Context(), id)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}

	response.OK(c, gin.H{
		"debt":     debt,
		"payments": payments,
	})
}

// Pay — make a payment on a debt
func (h *Handler) Pay(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.Error(apperr.Validation("invalid id"))
		return
	}
	var req PayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	var userID *int64
	if v, ok := c.Get("user_id"); ok {
		if uid, ok := v.(int64); ok {
			userID = &uid
		}
	}

	debt, payment, err := h.repo.Pay(c.Request.Context(), id, req, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.Error(apperr.NotFound("DEBT_NOT_FOUND", "debt not found or already settled"))
			return
		}
		if err == errOverpay {
			c.Error(apperr.Validation("payment amount exceeds remaining debt"))
			return
		}
		c.Error(apperr.Internal(err))
		return
	}

	response.OK(c, gin.H{
		"debt":    debt,
		"payment": payment,
	})
}

// GetCustomerTotal — total debt for a customer
func (h *Handler) GetCustomerTotal(c *gin.Context) {
	customerID, _ := strconv.ParseInt(c.Param("customer_id"), 10, 64)
	if customerID <= 0 {
		c.Error(apperr.Validation("invalid customer_id"))
		return
	}
	total, err := h.repo.GetCustomerTotalDebt(c.Request.Context(), customerID)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, gin.H{
		"customer_id":      customerID,
		"total_debt_cents": total,
	})
}
