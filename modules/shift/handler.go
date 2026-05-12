package shift

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func extractUserID(c *gin.Context) int64 {
	v, _ := c.Get("user_id")
	uid, _ := v.(int64)
	return uid
}

// OpenShift godoc
// @Summary      Open a new shift
// @Tags         Shifts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body OpenRequest true "Register ID and opening cash"
// @Success      201 {object} response.APIResponse{data=Shift}
// @Failure      409 {object} response.APIResponse
// @Router       /api/shifts/open [post]
func (h *Handler) OpenShift(c *gin.Context) {
	var req OpenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("register_id is required"))
		return
	}
	s, err := h.repo.OpenShift(c.Request.Context(), extractUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, s)
}

// CloseShift godoc
// @Summary      Close a shift
// @Tags         Shifts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int          true "Shift ID"
// @Param        body body CloseRequest true "Closing cash and note"
// @Success      200 {object} response.APIResponse{data=Shift}
// @Failure      404 {object} response.APIResponse
// @Failure      409 {object} response.APIResponse
// @Router       /api/shifts/{id}/close [post]
func (h *Handler) CloseShift(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid shift id"))
		return
	}
	var req CloseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("invalid request"))
		return
	}
	rolesVal, _ := c.Get("roles")
	roles, _ := rolesVal.([]string)
	_, err = h.repo.CloseShift(c.Request.Context(), id, extractUserID(c), roles, req)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"closed": true})
}

// GetCurrent godoc
// @Summary      Get current open shift
// @Tags         Shifts
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.APIResponse{data=Shift}
// @Failure      404 {object} response.APIResponse
// @Router       /api/shifts/current [get]
func (h *Handler) GetCurrent(c *gin.Context) {
	s, err := h.repo.GetCurrent(c.Request.Context(), extractUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, s)
}

// GetShift godoc
// @Summary      Get shift details (Z-report)
// @Tags         Shifts
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Shift ID"
// @Success      200 {object} response.APIResponse{data=Shift}
// @Failure      404 {object} response.APIResponse
// @Router       /api/shifts/{id} [get]
func (h *Handler) GetShift(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid shift id"))
		return
	}
	s, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, s)
}

// ListShifts godoc
// @Summary      List shifts
// @Tags         Shifts
// @Produce      json
// @Security     BearerAuth
// @Param        user_id     query int    false "Filter by user ID"
// @Param        register_id query int    false "Filter by register ID"
// @Param        status      query string false "Filter by status (open, closed)"
// @Success      200 {object} response.APIResponse
// @Router       /api/shifts [get]
func (h *Handler) ListShifts(c *gin.Context) {
	pg, _ := c.Get("pagination")
	page := pg.(middleware.Pagination)

	var userID, registerID *int64
	var status *string

	if v := c.Query("user_id"); v != "" {
		id, _ := strconv.ParseInt(v, 10, 64)
		userID = &id
	}
	if v := c.Query("register_id"); v != "" {
		id, _ := strconv.ParseInt(v, 10, 64)
		registerID = &id
	}
	if v := c.Query("status"); v != "" {
		status = &v
	}

	items, total, err := h.repo.List(c.Request.Context(), userID, registerID, status, page.Limit, page.Offset)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.List(c, items, page.Page, page.Limit, page.Offset, total)
}

// ListRegisters godoc
// @Summary      List cash registers
// @Tags         Shifts
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.APIResponse{data=[]CashRegister}
// @Router       /api/registers [get]
func (h *Handler) ListRegisters(c *gin.Context) {
	regs, err := h.repo.ListRegisters(c.Request.Context())
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, regs)
}
