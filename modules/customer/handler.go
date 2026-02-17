// modules/customer/handler.go
package customer

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Create godoc
// @Summary      Create customer
// @Tags         Customers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CreateRequest  true  "Customer data"
// @Success      201   {object}  response.APIResponse{data=Customer}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      403   {object}  response.APIResponse
// @Failure      409   {object}  response.APIResponse
// @Router       /customers [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	out, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// List godoc
// @Summary      List customers
// @Description  Get paginated list of customers (not deleted). By default returns all not deleted; set active_only=true to return only active.
// @Tags         Customers
// @Produce      json
// @Security     BearerAuth
// @Param        page             query  int     false  "Page (default 1)"
// @Param        limit            query  int     false  "Limit (default 10, max 200)"
// @Param        skip             query  int     false  "Skip (legacy, default 0). If provided, overrides page/offset."
// @Param        search           query  string  false  "Search by name, phone, or email"
// @Param        active_only      query  bool    false  "Only active customers (default false)"
// @Param        order_by         query  string  false  "Order by field (name, created_at, total_spent)" Enums(name,created_at,total_spent)
// @Param        order_direction  query  string  false  "Order direction (asc/desc)" Enums(asc,desc)
// @Success      200              {object}  response.APIResponse{data=ListResponse}
// @Failure      400              {object}  response.APIResponse
// @Failure      401              {object}  response.APIResponse
// @Failure      403              {object}  response.APIResponse
// @Failure      500              {object}  response.APIResponse
// @Router       /customers [get]
func (h *Handler) List(c *gin.Context) {
	page := 1
	limit := 10
	offset := 0

	if pRaw, ok := c.Get("pagination"); ok {
		if p, ok := pRaw.(middleware.Pagination); ok {
			if p.Page > 0 {
				page = p.Page
			}
			if p.Limit > 0 {
				limit = p.Limit
			}
			if p.Offset >= 0 {
				offset = p.Offset
			}
		}
	}

	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if v := c.Query("skip"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
			if limit > 0 {
				page = (offset / limit) + 1
			}
		}
	}

	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	q := c.Query("search")

	// activeOnly := true
	// if v := c.Query("active_only"); v != "" {
	// 	// accept: true/false/1/0
	// 	v = strings.ToLower(strings.TrimSpace(v))
	// 	if v == "false" || v == "0" {
	// 		activeOnly = false
	// 	}
	// }
	activeOnly := false // default: show all not deleted
	if v := c.Query("active_only"); v != "" {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "true" || v == "1" {
			activeOnly = true
		}
	}

	orderBy := c.Query("order_by")
	if orderBy == "" {
		orderBy = "created_at"
	}
	switch orderBy {
	case "name", "created_at", "total_spent":
	default:
		c.Error(apperr.Validation("order_by must be 'name', 'created_at', or 'total_spent'"))
		return
	}

	orderDir := strings.ToLower(c.Query("order_direction"))
	if orderDir == "" {
		orderDir = "desc"
	}
	if orderDir != "asc" && orderDir != "desc" {
		c.Error(apperr.Validation("order_direction must be 'asc' or 'desc'"))
		return
	}

	out, err := h.svc.List(c.Request.Context(), limit, offset, orderBy, orderDir, q, activeOnly)
	if err != nil {
		c.Error(err)
		return
	}

	response.List(c, out, page, out.Limit, out.Offset, out.Total)
}

// Get godoc
// @Summary      Get customer by ID
// @Description  Get a single customer by its ID (404 only if deleted). If inactive, still returns object with is_active=false.
// @Tags         Customers
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Customer ID"
// @Success      200  {object}  response.APIResponse{data=Customer}
// @Failure      401  {object}  response.APIResponse
// @Failure      403  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /customers/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	out, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// Delete godoc
// @Summary      Delete customer (soft)
// @Description  Soft delete customer: sets deleted_at=now and is_active=false (admin only)
// @Tags         Customers
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Customer ID"
// @Success      200  {object}  response.APIResponse{data=object}
// @Failure      401  {object}  response.APIResponse
// @Failure      403  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /customers/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

// AddSpent godoc
// @Summary      Add spent amount
// @Description  Add spent amount and calculate bonus points (1% of amount)
// @Tags         Customers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int              true  "Customer ID"
// @Param        body  body      AddSpentRequest  true  "Amount"
// @Success      200   {object}  response.APIResponse{data=Customer}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      403   {object}  response.APIResponse
// @Failure      404   {object}  response.APIResponse
// @Router       /customers/{id}/spent [post]
func (h *Handler) AddSpent(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req AddSpentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	out, err := h.svc.AddSpent(c.Request.Context(), id, req.AmountCents)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

type AddSpentRequest struct {
	AmountCents int64 `json:"amount_cents" binding:"required,gt=0"`
}

// UpdateContact godoc
// @Summary      Update customer contact fields
// @Description  Update name/phone/email/notes (cashier/operator/manager)
// @Tags         Customers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                  true  "Customer ID"
// @Param        body  body      UpdateContactRequest  true  "Contact update"
// @Success      200   {object}  response.APIResponse{data=Customer}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      403   {object}  response.APIResponse
// @Failure      404   {object}  response.APIResponse
// @Router       /customers/{id}/contact [patch]
func (h *Handler) UpdateContact(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpdateContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	out, err := h.svc.UpdateContact(c.Request.Context(), id, req)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// UpdateAdmin godoc
// @Summary      Update customer business fields
// @Description  Update only type and is_active (manager/admin)
// @Tags         Customers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int               true  "Customer ID"
// @Param        body  body      UpdateAdminRequest true  "Business update"
// @Success      200   {object}  response.APIResponse{data=Customer}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      403   {object}  response.APIResponse
// @Failure      404   {object}  response.APIResponse
// @Router       /customers/{id}/admin [patch]
func (h *Handler) UpdateAdmin(c *gin.Context) {

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req UpdateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	out, err := h.svc.UpdateAdmin(c.Request.Context(), id, req)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}
