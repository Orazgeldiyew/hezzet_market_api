package supplierdebt

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

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
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
	debt, err := h.repo.CreateDebt(c.Request.Context(), req.SupplierID, nil, req.AmountCents, req.Note, userID)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.Created(c, debt)
}

func (h *Handler) ListAll(c *gin.Context) {
	pg := c.MustGet("pagination").(middleware.Pagination)
	items, total, err := h.repo.ListAll(c.Request.Context(), pg.Limit, pg.Offset)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.List(c, items, pg.Page, pg.Limit, pg.Offset, total)
}

func (h *Handler) DebtorsSummary(c *gin.Context) {
	items, err := h.repo.DebtorsSummary(c.Request.Context())
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, items)
}

func (h *Handler) ListBySupplier(c *gin.Context) {
	supplierID, _ := strconv.ParseInt(c.Param("supplier_id"), 10, 64)
	if supplierID <= 0 {
		c.Error(apperr.Validation("invalid supplier_id"))
		return
	}
	onlyOpen := c.Query("status") != "all"
	pg := c.MustGet("pagination").(middleware.Pagination)

	items, total, err := h.repo.ListBySupplier(c.Request.Context(), supplierID, onlyOpen, pg.Limit, pg.Offset)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.List(c, items, pg.Page, pg.Limit, pg.Offset, total)
}

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
	response.OK(c, gin.H{"debt": debt, "payments": payments})
}

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
	response.OK(c, gin.H{"debt": debt, "payment": payment})
}
