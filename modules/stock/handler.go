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
// @Success 201 {object} response.APIResponse{data=MovementResult}
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

// BulkStockIn godoc
// @Summary Bulk Stock In
// @Description Add stock for multiple products at once in a single transaction. Each item needs its own idempotency_key.
// @Tags Stock
// @Accept json
// @Produce json
// @Security     BearerAuth
// @Param body body BulkInRequest true "Bulk Stock In Request"
// @Success 201 {object} response.APIResponse{data=BulkInResult}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/stock/in/bulk [post]
func (h *Handler) BulkStockIn(c *gin.Context) {
	var req BulkInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.BulkStockIn(c.Request.Context(), req, userID)
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
// @Success 201 {object} response.APIResponse{data=MovementResult}
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
// @Success 201 {object} response.APIResponse{data=TransferResult}
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
// @Summary Stock Move (adjustment/damaged/return_to_supplier/etc)
// @Description Universal move endpoint. For return_to_supplier: delta_milli must be negative, supplier_id is required. Optional note field for comments.
// @Tags Stock
// @Accept json
// @Produce json
// @Security     BearerAuth
// @Param body body MoveRequest true "Move Request"
// @Success 201 {object} response.APIResponse{data=MovementResult}
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
// @Success 200 {object} response.APIResponse{data=[]WarehouseItem}
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
// @Param type query string false "Movement type (in/out/transfer_in/transfer_out/damaged/adjustment/opening_balance/return_to_supplier)"
// @Param supplier_id query int false "Supplier ID (filter return_to_supplier movements)"
// @Param date_from query string false "RFC3339 datetime (inclusive)"
// @Param date_to query string false "RFC3339 datetime (inclusive)"
// @Param page query int false "page"
// @Param limit query int false "limit"
// @Success 200 {object} response.APIResponse{data=[]WarehouseItemDetail,meta=response.Meta}
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
	var supplierID *int64
	if v := c.Query("supplier_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			supplierID = &n
		}
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

	out, err := h.svc.GetDetails(c.Request.Context(), warehouseID, productID, mType, supplierID, dateFrom, dateTo, limit, offset)
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
// @Success 201 {object} response.APIResponse{data=MovementResult}
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

// NegativeStock godoc
// @Summary List products with negative stock (deficit)
// @Description Returns all warehouse items where qty_milli < 0. These are products oversold via force confirm.
// @Tags Stock
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=[]NegativeStockRow}
// @Failure 500 {object} response.APIResponse
// @Router /api/stock/negative [get]
func (h *Handler) NegativeStock(c *gin.Context) {
	out, err := h.svc.GetNegativeItems(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}
