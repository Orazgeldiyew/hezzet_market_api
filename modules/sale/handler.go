package sale

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/receiptsettings"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	svc         *Service
	receiptRepo *receiptsettings.Repository
	baseURL     string
}

func NewHandler(svc *Service, receiptRepo *receiptsettings.Repository, baseURL string) *Handler {
	return &Handler{svc: svc, receiptRepo: receiptRepo, baseURL: baseURL}
}

func extractUserID(c *gin.Context) int64 {
	v, _ := c.Get("user_id")
	uid, _ := v.(int64)
	return uid
}

// CreateSale godoc
// @Summary Create sale (draft)
// @Description Create a draft sale that reserves stock. Call POST /:id/confirm to deduct stock and create a finance transaction.
// @Tags Sales
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateSaleRequest true "Sale request"
// @Success 201 {object} response.APIResponse{data=SaleDetail}
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
// @Param status query string false "Filter by status: draft, confirmed, cancelled"
// @Param date_from query string false "RFC3339 datetime (inclusive)"
// @Param date_to query string false "RFC3339 datetime (inclusive)"
// @Param page query int false "page"
// @Param limit query int false "limit"
// @Success 200 {object} response.APIResponse{data=[]SaleListItem,meta=response.Meta}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/sales [get]
func (h *Handler) ListSales(c *gin.Context) {
	var warehouseID, customerID, createdBy *int64
	var status *string
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
	limit := 10
	offset := 0
	if pRaw, ok := c.Get("pagination"); ok {
		if p, ok := pRaw.(middleware.Pagination); ok {
			page = p.Page
			limit = p.Limit
			offset = p.Offset
		}
	}

	out, err := h.svc.ListSales(c.Request.Context(), warehouseID, customerID, createdBy, status, dateFrom, dateTo, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}
	response.List(c, out.Items, page, out.Limit, out.Offset, out.Total)
}

// ConfirmSale godoc
// @Summary Confirm sale
// @Description Transitions a draft sale to confirmed: deducts stock, records COGS, creates a finance transaction. Payment is optional.
// @Tags Sales
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Sale ID"
// @Param body body ConfirmSaleRequest false "Optional payment details"
// @Success 200 {object} response.APIResponse{data=SaleDetail}
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/sales/{id}/confirm [post]
func (h *Handler) ConfirmSale(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid sale id"))
		return
	}
	var req ConfirmSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.Error(err)
		return
	}

	userID := extractUserID(c)

	out, err := h.svc.ConfirmSale(c.Request.Context(), id, req, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// CancelSale godoc
// @Summary Cancel sale
// @Description Cancels a draft or confirmed sale. Draft: releases reservation. Confirmed: restores stock and voids the finance transaction.
// @Tags Sales
// @Produce json
// @Security BearerAuth
// @Param id path int true "Sale ID"
// @Success 200 {object} response.APIResponse{data=object}
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/sales/{id}/cancel [post]
func (h *Handler) CancelSale(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid sale id"))
		return
	}
	userID := extractUserID(c)

	if err := h.svc.CancelSale(c.Request.Context(), id, userID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"cancelled": true})
}

// GetSale godoc
// @Summary Get sale detail
// @Description Returns a sale with all its items and transaction ID.
// @Tags Sales
// @Produce json
// @Security BearerAuth
// @Param id path int true "Sale ID"
// @Success 200 {object} response.APIResponse{data=SaleDetail}
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

// DeleteSaleItem godoc
// @Summary      Delete item from draft sale
// @Description  Removes a sale item from a draft sale, releases stock reservation, and recalculates totals.
// @Tags         Sales
// @Produce      json
// @Security     BearerAuth
// @Param        id       path int true "Sale ID"
// @Param        item_id  path int true "Sale Item ID"
// @Success      200 {object} response.APIResponse{data=SaleDetail}
// @Failure      400 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      409 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Param        body     body object{delete_code=string} true "Confirmation code"
// @Router       /api/sales/{id}/items/{item_id} [delete]
func (h *Handler) DeleteSaleItem(c *gin.Context) {
	saleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || saleID <= 0 {
		c.Error(apperr.Validation("invalid sale id"))
		return
	}
	itemID, err := strconv.ParseInt(c.Param("item_id"), 10, 64)
	if err != nil || itemID <= 0 {
		c.Error(apperr.Validation("invalid item id"))
		return
	}

	var body struct {
		DeleteCode string `json:"delete_code"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.DeleteCode == "" {
		c.Error(apperr.Validation("delete_code is required"))
		return
	}

	storedCode, err := h.receiptRepo.GetDeleteCode(c.Request.Context())
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	if body.DeleteCode != storedCode {
		c.Error(apperr.Forbidden("invalid delete code"))
		return
	}

	out, err := h.svc.DeleteSaleItem(c.Request.Context(), saleID, itemID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// GetReceipt godoc
// @Summary Render sale receipt as HTML
// TransferDraft godoc
// @Summary      Transfer draft sale to another cashier
// @Description  Changes the owner of a draft sale so another cashier can continue it
// @Tags         Sales
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  int                  true  "Sale ID"
// @Param        body body  TransferSaleRequest   true  "New cashier ID"
// @Success      200  {object} response.APIResponse
// @Failure      400  {object} response.APIResponse
// @Failure      403  {object} response.APIResponse
// @Failure      404  {object} response.APIResponse
// @Router       /api/sales/{id}/transfer [post]
func (h *Handler) TransferDraft(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid sale id"))
		return
	}

	var req TransferSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("cashier_id is required"))
		return
	}

	userID := extractUserID(c)
	rolesVal, _ := c.Get("roles")
	roles, _ := rolesVal.([]string)

	if err := h.svc.TransferDraft(c.Request.Context(), id, userID, req.CashierID, roles); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"transferred": true, "new_cashier_id": req.CashierID})
}

// @Description Returns an HTML page formatted for an 80mm thermal printer. Uses custom template from receipt settings if configured.
// @Tags Sales
// @Produce html
// @Security BearerAuth
// @Param id path int true "Sale ID"
// @Success 200 {string} string "HTML receipt"
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/sales/{id}/receipt [get]
func (h *Handler) GetReceipt(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid sale id"))
		return
	}

	ctx := c.Request.Context()

	saleRow, items, err := h.svc.repo.GetReceiptData(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.Error(apperr.NotFound("SALE_NOT_FOUND", "sale not found"))
			return
		}
		c.Error(apperr.Internal(err))
		return
	}

	settings, err := h.receiptRepo.Get(ctx)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}

	var logoURL string
	if settings.LogoPath != "" && h.baseURL != "" {
		logoURL = h.baseURL + "/uploads/" + settings.LogoPath
	}

	var note string
	if saleRow.Note != nil {
		note = *saleRow.Note
	}

	receiptItems := make([]ReceiptItem, len(items))
	var subtotalCents int64
	for i, it := range items {
		receiptItems[i] = ReceiptItem{
			ProductName:     it.ProductName,
			QtyMilli:        it.QtyMilli,
			UnitType:        it.UnitType,
			UnitPriceCents:  it.UnitPriceCents,
			LineTotalCents:  it.LineTotalCents,
			DiscountPercent: it.DiscountPercent,
		}
		subtotalCents += (it.QtyMilli*it.UnitPriceCents + 500) / 1000
	}
	totalDiscountCents := subtotalCents - saleRow.TotalCents - saleRow.BonusUsedCents
	if totalDiscountCents < 0 {
		totalDiscountCents = 0
	}
	totalDiscountPercent := 0
	if subtotalCents > 0 {
		totalDiscountPercent = int(totalDiscountCents * 100 / subtotalCents)
	}

	receiptNumber := saleRow.CreatedAt.Format("20060102") + "-" + fmt.Sprintf("%06d", saleRow.ID)

	data := ReceiptData{
		ShopName:             settings.ShopName,
		ShopAddress:          settings.ShopAddress,
		ShopPhone:            settings.ShopPhone,
		LogoURL:              logoURL,
		LogoWidth:            settings.LogoWidth,
		LogoHeight:           settings.LogoHeight,
		Footer:               settings.Footer,
		SaleID:               saleRow.ID,
		ReceiptNumber:        receiptNumber,
		Date:                 saleRow.CreatedAt.Format("02.01.2006"),
		Time:                 saleRow.CreatedAt.Format("15:04"),
		Status:               string(saleRow.Status),
		CashierName:          saleRow.CashierName,
		WarehouseName:        saleRow.WarehouseName,
		CustomerName:         saleRow.CustomerName,
		TotalCents:           saleRow.TotalCents,
		BonusUsedCents:       saleRow.BonusUsedCents,
		DiscountPercent:      saleRow.DiscountPercent,
		DiscountCents:        saleRow.DiscountCents,
		SubtotalCents:        subtotalCents,
		TotalDiscountCents:   totalDiscountCents,
		TotalDiscountPercent: totalDiscountPercent,
		PaymentMethod:        saleRow.PaymentMethod,
		PaidCents:            saleRow.PaidCents,
		ChangeCents:     max(saleRow.PaidCents-saleRow.TotalCents, 0),
		WorkerName:      saleRow.WorkerName,
		Note:            note,
		Items:           receiptItems,
	}

	html, err := RenderReceipt(data, settings.Template)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// Reprint godoc
// @Summary Reprint receipt to thermal printer
// @Description Re-fires the same auto-print that runs on sale confirm. The cashier
// @Description can call this as many times as they like — there is no per-sale limit.
// @Tags Sales
// @Produce json
// @Security BearerAuth
// @Param id path int true "Sale ID"
// @Success 200 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 503 {object} response.APIResponse "printer not configured"
// @Router /api/sales/{id}/print [post]
func (h *Handler) Reprint(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid sale id"))
		return
	}

	printerSv := h.svc.repo.PrinterService()
	if printerSv == nil {
		c.Error(apperr.Internal(fmt.Errorf("printer not configured")))
		return
	}

	// Verify the sale exists and pull the cashier's open-shift register so the
	// reprint targets the same physical printer the original print used.
	ctx := c.Request.Context()
	if _, _, err := h.svc.repo.GetReceiptData(ctx, id); err != nil {
		if err == pgx.ErrNoRows {
			c.Error(apperr.NotFound("SALE_NOT_FOUND", "sale not found"))
			return
		}
		c.Error(apperr.Internal(err))
		return
	}

	userID := extractUserID(c)
	var regID *int64
	var rid int64
	if err := h.svc.repo.DB().QueryRow(ctx, `
		SELECT register_id FROM shifts
		WHERE user_id = $1 AND status = 'open'
		ORDER BY opened_at DESC LIMIT 1
	`, userID).Scan(&rid); err == nil {
		regID = &rid
	}

	if err := printerSv.PrintSale(ctx, id, regID); err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, gin.H{"printed": true})
}

// ReturnSale godoc
// @Summary      Return sale items (partial or full)
// @Description  Returns items from a confirmed sale. If items array is empty, returns the entire sale.
// @Tags         Sales
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  int               true  "Sale ID"
// @Param        body body  ReturnSaleRequest  true  "Items to return and reason"
// @Success      200  {object}  response.APIResponse
// @Failure      400  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      409  {object}  response.APIResponse
// @Router       /api/sales/{id}/return [post]
func (h *Handler) ReturnSale(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid sale id"))
		return
	}

	var req ReturnSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("invalid request body"))
		return
	}

	userID := extractUserID(c)
	if err := h.svc.ReturnSale(c.Request.Context(), id, req, userID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"returned": true})
}
// Decrease item qty
// @Router /api/sales/{id}/items/{item_id}/decrease [patch]
func (h *Handler) DecreaseItem(c *gin.Context) {
	saleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(apperr.Validation("invalid sale id"))
		return
	}

	itemID, err := strconv.ParseInt(c.Param("item_id"), 10, 64)
	if err != nil {
		c.Error(apperr.Validation("invalid item id"))
		return
	}

	out, err := h.svc.DecreaseItem(c.Request.Context(), saleID, itemID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, out)
}