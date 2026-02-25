package sale

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

// CreateSale godoc
// @Summary Create sale
// @Description Create a sale from the POS. Prices are automatically from product sale_price. Stock is automatically deducted. Finance transaction (income) is created. Payment is optional.
// @Tags Sales
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateSaleRequest true "Sale request"
// @Success 201 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/sales [post]
func (h *Handler) CreateSale(c *gin.Context) {
	var req CreateSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.CreateSale(c.Request.Context(), req, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// ListSales godoc
// @Summary List sales
// @Description List sales with optional filters and pagination.
// @Tags Sales
// @Produce json
// @Security BearerAuth
// @Param warehouse_id query int false "Warehouse ID"
// @Param customer_id query int false "Customer ID"
// @Param created_by query int false "Cashier user ID"
// @Param date_from query string false "RFC3339 datetime (inclusive)"
// @Param date_to query string false "RFC3339 datetime (inclusive)"
// @Param page query int false "page"
// @Param limit query int false "limit"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/sales [get]
func (h *Handler) ListSales(c *gin.Context) {
	var warehouseID, customerID, createdBy *int64
	var dateFrom, dateTo *time.Time

	if v := c.Query("warehouse_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			warehouseID = &n
		}
	}
	if v := c.Query("customer_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			customerID = &n
		}
	}
	if v := c.Query("created_by"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			createdBy = &n
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

	out, err := h.svc.ListSales(c.Request.Context(), warehouseID, customerID, createdBy, dateFrom, dateTo, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}
	response.List(c, out.Items, page, out.Limit, out.Offset, out.Total)
}

// GetSale godoc
// @Summary Get sale detail
// @Description Returns a sale with all its items and transaction ID.
// @Tags Sales
// @Produce json
// @Security BearerAuth
// @Param id path int true "Sale ID"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/sales/{id} [get]
func (h *Handler) GetSale(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid sale id"))
		return
	}

	out, err := h.svc.GetSale(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}
