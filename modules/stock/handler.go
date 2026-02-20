package stock

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

// StockIn godoc
// @Summary Stock In
// @Description Add stock to a warehouse (qty_milli, SCALE=1000). Strict idempotency.
// @Tags Stock
// @Accept json
// @Produce json
// @Security     BearerAuth
// @Param body body InRequest true "Stock In Request"
// @Success 201 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/stock/in [post]
func (h *Handler) StockIn(c *gin.Context) {
	var req InRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.StockIn(c.Request.Context(), req, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// StockOut godoc
// @Summary Stock Out
// @Description Remove stock from a warehouse (qty_milli, SCALE=1000). Strict idempotency.
// @Tags Stock
// @Accept json
// @Produce json
// @Security     BearerAuth
// @Param body body OutRequest true "Stock Out Request"
// @Success 201 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/stock/out [post]
func (h *Handler) StockOut(c *gin.Context) {
	var req OutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.StockOut(c.Request.Context(), req, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// Transfer godoc
// @Summary Transfer Stock
// @Description Transfer stock between warehouses. Strict idempotency.
// @Tags Stock
// @Accept json
// @Produce json
// @Security     BearerAuth
// @Param body body TransferRequest true "Transfer Request"
// @Success 201 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/stock/transfer [post]
func (h *Handler) Transfer(c *gin.Context) {
	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.Transfer(c.Request.Context(), req, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// Move godoc
// @Summary Stock Move (adjustment/damaged/etc)
// @Description Universal move endpoint writing to warehouse_item_details + applying delta to warehouse_items.
// @Tags Stock
// @Accept json
// @Produce json
// @Security     BearerAuth
// @Param body body MoveRequest true "Move Request"
// @Success 201 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/stock/move [post]
func (h *Handler) Move(c *gin.Context) {
	var req MoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.Move(c.Request.Context(), req, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// GetItems godoc
// @Summary Get Warehouse Items (balances)
// @Tags Stock
// @Produce json
// @Security     BearerAuth
// @Param warehouse_id query int false "Warehouse ID"
// @Param product_id query int false "Product ID"
// @Success 200 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/stock/balance [get]
func (h *Handler) GetItems(c *gin.Context) {
	var warehouseID, productID *int64

	if v := c.Query("warehouse_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			warehouseID = &n
		}
	}
	if v := c.Query("product_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			productID = &n
		}
	}

	out, err := h.svc.GetItems(c.Request.Context(), warehouseID, productID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

func extractUserID(c *gin.Context) int64 {
	v, _ := c.Get("user_id")
	uid, _ := v.(int64)
	return uid
}

// GetDetails godoc
// @Summary Get Stock Details (ledger)
// @Description List warehouse_item_details with filters + pagination.
// @Tags Stock
// @Produce json
// @Security     BearerAuth
// @Param warehouse_id query int false "Warehouse ID"
// @Param product_id query int false "Product ID"
// @Param type query string false "Movement type (in/out/transfer_in/transfer_out/damaged/adjustment)"
// @Param date_from query string false "RFC3339 datetime (inclusive)"
// @Param date_to query string false "RFC3339 datetime (inclusive)"
// @Param page query int false "page"
// @Param limit query int false "limit"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/stock/details [get]
func (h *Handler) GetDetails(c *gin.Context) {
	var warehouseID, productID *int64
	var mType *string
	var dateFrom *time.Time
	var dateTo *time.Time

	if v := c.Query("warehouse_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			warehouseID = &n
		}
	}
	if v := c.Query("product_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			productID = &n
		}
	}
	if v := c.Query("type"); v != "" {
		s := v
		mType = &s
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

	// pagination from middleware
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

	out, err := h.svc.GetDetails(c.Request.Context(), warehouseID, productID, mType, dateFrom, dateTo, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}

	response.List(c, out.Items, page, out.Limit, out.Offset, out.Total)
}

// OpeningBalance godoc
// @Summary Opening Balance
// @Description Set opening balance for stock (qty_milli, SCALE=1000). Strict idempotency.
// @Tags Stock
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body OpeningBalanceRequest true "Opening Balance Request"
// @Success 201 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/stock/opening-balance [post]
func (h *Handler) OpeningBalance(c *gin.Context) {
	var req OpeningBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.OpeningBalance(c.Request.Context(), req, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}
