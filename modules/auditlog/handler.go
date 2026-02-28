package auditlog

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

// List godoc
// @Summary      List audit logs
// @Description  Returns paginated audit log entries. Filter by user, entity type, action, or date range.
// @Tags         AuditLog
// @Produce      json
// @Security     BearerAuth
// @Param        user_id      query  int     false  "Filter by user ID"
// @Param        entity_type  query  string  false  "Filter by entity type (product, worker, payroll…)"
// @Param        action       query  string  false  "Filter by action (CREATE, UPDATE, DELETE, PAY…)"
// @Param        from         query  string  false  "Filter from datetime (RFC3339)"
// @Param        to           query  string  false  "Filter to datetime (RFC3339)"
// @Success      200  {object}  response.APIResponse
// @Failure      400  {object}  response.APIResponse
// @Router       /api/audit-logs [get]
func (h *Handler) List(c *gin.Context) {
	pg, _ := c.Get("pagination")
	page := pg.(middleware.Pagination)

	var f AuditFilter

	if s := c.Query("user_id"); s != "" {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			c.Error(apperr.Validation("user_id must be an integer"))
			return
		}
		f.UserID = &id
	}
	f.EntityType = c.Query("entity_type")
	f.Action = c.Query("action")

	if s := c.Query("from"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			c.Error(apperr.Validation("from must be RFC3339 datetime"))
			return
		}
		f.From = &t
	}
	if s := c.Query("to"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			c.Error(apperr.Validation("to must be RFC3339 datetime"))
			return
		}
		f.To = &t
	}

	items, total, err := h.repo.List(c.Request.Context(), f, page.Limit, page.Offset)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}

	response.List(c, items, page.Page, page.Limit, page.Offset, total)
}
