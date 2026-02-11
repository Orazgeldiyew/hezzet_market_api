package supplier

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

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Create godoc
// @Summary      Create supplier
// @Description  Create a new supplier
// @Tags         Suppliers
// @Accept       json
// @Produce      json
// @Param        body  body      CreateRequest  true  "Supplier data"
// @Success      201   {object}  response.APIResponse{data=Supplier}
// @Failure      400   {object}  response.APIResponse
// @Router       /suppliers [post]
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
// @Summary      List suppliers
// @Description  Get paginated list of active suppliers with optional search by name/phone/email
// @Tags         Suppliers
// @Produce      json
// @Param        page             query  int     false  "Page (default 1)"
// @Param        limit            query  int     false  "Limit (default 10, max 100)"
// @Param        skip             query  int     false  "Skip (legacy, default 0). If provided, overrides page/offset."
// @Param        search           query  string  false  "Search by name, phone, or email"
// @Param        order_by         query  string  false  "Order by field (name, created_at)" Enums(name,created_at)
// @Param        order_direction  query  string  false  "Order direction (asc/desc)" Enums(asc,desc)
// @Success      200              {object}  response.APIResponse{data=ListResponse}
// @Failure      400              {object}  response.APIResponse
// @Failure      500              {object}  response.APIResponse
// @Router       /suppliers [get]
func (h *Handler) List(c *gin.Context) {
	// Defaults from pagination middleware (page+limit -> offset)
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

	// legacy overrides
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

	// Validate limit/offset hard
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	q := c.Query("search")

	orderBy := c.Query("order_by")
	if orderBy == "" {
		orderBy = "created_at"
	}
	switch orderBy {
	case "name", "created_at":
	default:
		c.Error(apperr.Validation("order_by must be 'name' or 'created_at'"))
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

	out, err := h.svc.List(c.Request.Context(), limit, offset, orderBy, orderDir, q)
	if err != nil {
		c.Error(err)
		return
	}

	response.List(c, out, page, out.Limit, out.Offset, out.Total)
}

// Get godoc
// @Summary      Get supplier by ID
// @Description  Get a single supplier by its ID
// @Tags         Suppliers
// @Produce      json
// @Param        id   path      int  true  "Supplier ID"
// @Success      200  {object}  response.APIResponse{data=Supplier}
// @Failure      404  {object}  response.APIResponse
// @Router       /suppliers/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	out, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// Update godoc
// @Summary      Update supplier
// @Description  Update supplier fields (name, phone, email, address, is_active)
// @Tags         Suppliers
// @Accept       json
// @Produce      json
// @Param        id    path      int            true  "Supplier ID"
// @Param        body  body      UpdateRequest  true  "Update data"
// @Success      200   {object}  response.APIResponse{data=Supplier}
// @Failure      400   {object}  response.APIResponse
// @Failure      404   {object}  response.APIResponse
// @Router       /suppliers/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	out, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// Delete godoc
// @Summary      Delete supplier (soft)
// @Description  Soft delete supplier by setting is_active=false
// @Tags         Suppliers
// @Produce      json
// @Param        id   path      int  true  "Supplier ID"
// @Success      200  {object}  response.APIResponse{data=object}
// @Failure      404  {object}  response.APIResponse
// @Router       /suppliers/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
