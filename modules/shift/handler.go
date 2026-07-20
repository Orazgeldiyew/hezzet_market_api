package shift

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

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
// @Param        all  query  bool  false  "Include inactive registers (management screen)"
// @Success      200 {object} response.APIResponse{data=[]CashRegister}
// @Router       /api/registers [get]
func (h *Handler) ListRegisters(c *gin.Context) {
	var regs []CashRegister
	var err error
	if c.Query("all") == "true" {
		regs, err = h.repo.ListAllRegisters(c.Request.Context())
	} else {
		regs, err = h.repo.ListRegisters(c.Request.Context())
	}
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, regs)
}

// CreateRegister godoc
// @Summary      Create cash register
// @Tags         Shifts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      201 {object} response.APIResponse{data=CashRegister}
// @Router       /api/registers [post]
func (h *Handler) CreateRegister(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	cr, err := h.repo.CreateRegister(c.Request.Context(), req.Name)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.Created(c, cr)
}

// UpdateRegister godoc
// @Summary      Update cash register (rename / enable / disable)
// @Tags         Shifts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  int  true  "Register ID"
// @Success      200 {object} response.APIResponse{data=CashRegister}
// @Router       /api/registers/{id} [patch]
func (h *Handler) UpdateRegister(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid register id"))
		return
	}
	var req struct {
		Name     *string `json:"name" binding:"omitempty,min=1,max=100"`
		IsActive *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	// Deactivating a till someone is mid-shift on would strand the cashier.
	if req.IsActive != nil && !*req.IsActive {
		busy, err := h.repo.RegisterHasOpenShift(c.Request.Context(), id)
		if err != nil {
			c.Error(apperr.Internal(err))
			return
		}
		if busy {
			c.Error(apperr.Conflict("REGISTER_IN_USE", "register has an open shift — close it first"))
			return
		}
	}

	cr, err := h.repo.UpdateRegister(c.Request.Context(), id, req.Name, req.IsActive)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.Error(apperr.NotFound("REGISTER_NOT_FOUND", "register not found"))
			return
		}
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, cr)
}

// DeleteRegister godoc
// @Summary      Delete cash register
// @Description  Hard delete. Fails when shifts or printers still reference the register — deactivate instead.
// @Tags         Shifts
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  int  true  "Register ID"
// @Success      200 {object} response.APIResponse
// @Router       /api/registers/{id} [delete]
func (h *Handler) DeleteRegister(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid register id"))
		return
	}
	busy, err := h.repo.RegisterHasOpenShift(c.Request.Context(), id)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	if busy {
		c.Error(apperr.Conflict("REGISTER_IN_USE", "register has an open shift — close it first"))
		return
	}
	if err := h.repo.DeleteRegister(c.Request.Context(), id); err != nil {
		if err == pgx.ErrNoRows {
			c.Error(apperr.NotFound("REGISTER_NOT_FOUND", "register not found"))
			return
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			c.Error(apperr.Conflict("REGISTER_REFERENCED",
				"register has shift history or a bound printer — deactivate it instead of deleting"))
			return
		}
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
