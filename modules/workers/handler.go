package workers

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
// @Summary      Create worker
// @Description  Create a new worker (store employee)
// @Tags         Workers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CreateRequest  true  "Worker data"
// @Success      201   {object}  response.APIResponse{data=Worker}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      403   {object}  response.APIResponse
// @Router       /workers [post]
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
// @Summary      List workers
// @Description  Get paginated list of workers. By default returns only active; set active_only=false to include inactive.
// @Tags         Workers
// @Produce      json
// @Security     BearerAuth
// @Param        page             query  int     false  "Page (default 1)"
// @Param        limit            query  int     false  "Limit (default 10, max 200)"
// @Param        skip             query  int     false  "Skip (legacy, default 0)"
// @Param        search           query  string  false  "Search by name, phone, or email"
// @Param        active_only      query  bool    false  "Only active workers (default true)"
// @Param        order_by         query  string  false  "Order by field (name, created_at)" Enums(name,created_at)
// @Param        order_direction  query  string  false  "Order direction (asc/desc)" Enums(asc,desc)
// @Success      200              {object}  response.APIResponse{data=ListResponse}
// @Failure      400              {object}  response.APIResponse
// @Failure      401              {object}  response.APIResponse
// @Failure      403              {object}  response.APIResponse
// @Router       /workers [get]
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

	search := c.Query("search")

	activeOnly := true
	if v := c.Query("active_only"); v != "" {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "false" || v == "0" {
			activeOnly = false
		}
	}

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

	out, err := h.svc.List(c.Request.Context(), limit, offset, orderBy, orderDir, search, activeOnly)
	if err != nil {
		c.Error(err)
		return
	}

	response.List(c, out, page, out.Limit, out.Offset, out.Total)
}

// Get godoc
// @Summary      Get worker by ID
// @Description  Get a single worker by ID (404 only if deleted)
// @Tags         Workers
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Worker ID"
// @Success      200  {object}  response.APIResponse{data=Worker}
// @Failure      401  {object}  response.APIResponse
// @Failure      403  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /workers/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	out, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// Update godoc
// @Summary      Update worker
// @Description  Update worker fields
// @Tags         Workers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int            true  "Worker ID"
// @Param        body  body      UpdateRequest  true  "Update data"
// @Success      200   {object}  response.APIResponse{data=Worker}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      403   {object}  response.APIResponse
// @Failure      404   {object}  response.APIResponse
// @Router       /workers/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
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
// @Summary      Delete worker (soft)
// @Description  Soft delete worker: sets deleted_at=now and is_active=false
// @Tags         Workers
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Worker ID"
// @Success      200  {object}  response.APIResponse{data=object}
// @Failure      401  {object}  response.APIResponse
// @Failure      403  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /workers/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
