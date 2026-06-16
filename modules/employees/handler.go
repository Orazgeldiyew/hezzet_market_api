package employees

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List
//
//	@Summary	List employees
//	@Tags		Employees
//	@Produce	json
//	@Security	BearerAuth
//	@Param		search        query  string  false  "Search by name/username/phone/email"
//	@Param		is_worker     query  bool    false  "Filter by is_worker"
//	@Param		has_account   query  bool    false  "Filter by has_account"
//	@Param		active_only   query  bool    false  "Only is_active=true"
//	@Param		page          query  int     false  "Page number"
//	@Param		limit         query  int     false  "Page size"
//	@Success	200  {object}  response.APIResponse
//	@Router		/api/employees [get]
func (h *Handler) List(c *gin.Context) {
	f := ListFilter{Search: c.Query("search")}
	if v := c.Query("is_worker"); v != "" {
		b := v == "true"
		f.IsWorker = &b
	}
	if v := c.Query("has_account"); v != "" {
		b := v == "true"
		f.HasAccount = &b
	}
	if c.Query("active_only") == "true" {
		f.ActiveOnly = true
	}

	page := 1
	limit := 50
	offset := 0
	if pgVal, ok := c.Get("pagination"); ok {
		if p, ok := pgVal.(middleware.Pagination); ok {
			page = p.Page
			limit = p.Limit
			offset = p.Offset
		}
	}

	out, err := h.svc.List(c.Request.Context(), f, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}
	response.List(c, out.Items, page, out.Limit, out.Offset, out.Total)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid employee id"))
		return
	}
	out, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	uid := extractUserID(c)
	out, err := h.svc.Create(c.Request.Context(), req, uid)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid employee id"))
		return
	}
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	uid := extractUserID(c)
	out, err := h.svc.Update(c.Request.Context(), id, req, uid)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid employee id"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func extractUserID(c *gin.Context) *int64 {
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(int64); ok {
			return &id
		}
	}
	return nil
}
