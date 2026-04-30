package notification

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

// InboxHandler serves the in-app notification bell endpoints.
type InboxHandler struct {
	repo *InboxRepository
}

func NewInboxHandler(repo *InboxRepository) *InboxHandler {
	return &InboxHandler{repo: repo}
}

func inboxUserID(c *gin.Context) (int64, bool) {
	v, ok := c.Get("user_id")
	if !ok {
		return 0, false
	}
	uid, ok := v.(int64)
	return uid, ok
}

// List returns the current user's inbox (newest first, with read flags).
func (h *InboxHandler) List(c *gin.Context) {
	uid, ok := inboxUserID(c)
	if !ok {
		c.Error(apperr.Unauthorized("unauthorized"))
		return
	}

	limit := 20
	offset := 0
	page := 1
	if pRaw, ok := c.Get("pagination"); ok {
		if p, ok := pRaw.(middleware.Pagination); ok {
			limit = p.Limit
			offset = p.Offset
			page = p.Page
		}
	}

	out, err := h.repo.List(c.Request.Context(), uid, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}
	// Pass unread_count + total via response metadata so the frontend can
	// render the badge and paginator from a single request.
	response.List(c, gin.H{
		"items":        out.Items,
		"unread_count": out.UnreadCount,
	}, page, out.Limit, out.Offset, out.Total)
}

// UnreadCount returns { unread_count: N } — polled by the bell for the badge.
func (h *InboxHandler) UnreadCount(c *gin.Context) {
	uid, ok := inboxUserID(c)
	if !ok {
		c.Error(apperr.Unauthorized("unauthorized"))
		return
	}
	n, err := h.repo.UnreadCount(c.Request.Context(), uid)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"unread_count": n})
}

// MarkRead marks a single sms_log as read for the current user.
func (h *InboxHandler) MarkRead(c *gin.Context) {
	uid, ok := inboxUserID(c)
	if !ok {
		c.Error(apperr.Unauthorized("unauthorized"))
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(apperr.Validation("id must be an integer"))
		return
	}
	if err := h.repo.MarkRead(c.Request.Context(), uid, id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"read": true})
}

// MarkAllRead marks every inbox-eligible log as read for the current user.
func (h *InboxHandler) MarkAllRead(c *gin.Context) {
	uid, ok := inboxUserID(c)
	if !ok {
		c.Error(apperr.Unauthorized("unauthorized"))
		return
	}
	n, err := h.repo.MarkAllRead(c.Request.Context(), uid)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"marked": n})
}
