package auditlog

import (
	"fmt"
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
	pgVal, _ := c.Get("pagination")
	page, ok := pgVal.(middleware.Pagination)
	if !ok {
		c.Error(apperr.Internal(fmt.Errorf("pagination middleware missing")))
		return
	}

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

type DeleteRangeRequest struct {
	// From / To are RFC3339 datetimes. At least one is required.
	// Inclusive on created_at. Example: delete everything from 2026-09-01 to 2026-09-11.
	From *time.Time `json:"from"`
	To   *time.Time `json:"to"`
}

// Stats godoc
// @Summary      Audit-log health stats
// @Description  Returns counters useful for spotting silent audit loss.
// @Tags         AuditLog
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.APIResponse
// @Router       /api/audit-logs/stats [get]
func (h *Handler) Stats(c *gin.Context) {
	response.OK(c, gin.H{
		"failed_writes": h.repo.FailedWrites(),
	})
}

// DeleteRange godoc
// @Summary      Delete audit logs by date range
// @Description  Hard-deletes audit_logs where created_at is between from and to (inclusive).
// @Tags         AuditLog
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  DeleteRangeRequest  true  "Date range"
// @Success      200  {object}  response.APIResponse
// @Failure      400  {object}  response.APIResponse
// @Router       /api/audit-logs/delete-range [post]
func (h *Handler) DeleteRange(c *gin.Context) {
	var req DeleteRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	if req.From == nil && req.To == nil {
		c.Error(apperr.Validation("from or to date is required"))
		return
	}

	deleted, err := h.repo.DeleteByDateRange(c.Request.Context(), req.From, req.To)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}

	out := gin.H{"deleted": deleted}
	if req.From != nil {
		out["from"] = req.From.Format(time.RFC3339)
	}
	if req.To != nil {
		out["to"] = req.To.Format(time.RFC3339)
	}
	response.OK(c, out)
}
