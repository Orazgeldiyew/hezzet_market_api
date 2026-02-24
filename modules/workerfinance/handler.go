package workerfinance

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

func extractUserID(c *gin.Context) int64 {
	v, _ := c.Get("user_id")
	uid, _ := v.(int64)
	return uid
}

// ── compensation ────────────────────────────────────────────────────────────

// SetCompensation godoc
// @Summary Set worker compensation
// @Description Set or update base salary configuration for a worker (upsert).
// @Tags Worker Finance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body SetCompensationRequest true "Compensation request"
// @Success 201 {object} response.APIResponse{data=WorkerCompensation}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/workers/compensation [post]
func (h *Handler) SetCompensation(c *gin.Context) {
	var req SetCompensationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	out, err := h.svc.SetCompensation(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// GetCompensation godoc
// @Summary Get worker compensation
// @Description Get base salary configuration for a worker.
// @Tags Worker Finance
// @Produce json
// @Security BearerAuth
// @Param id path int true "Worker ID"
// @Success 200 {object} response.APIResponse{data=WorkerCompensation}
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/workers/{id}/compensation [get]
func (h *Handler) GetCompensation(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid worker id"))
		return
	}
	out, err := h.svc.GetCompensation(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// ── fines ───────────────────────────────────────────────────────────────────

// CreateFine godoc
// @Summary Create worker fine
// @Description Create a fine (penalty) for a worker.
// @Tags Worker Finance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Worker ID"
// @Param body body CreateFineRequest true "Fine request"
// @Success 201 {object} response.APIResponse{data=WorkerFine}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/workers/{id}/fines [post]
func (h *Handler) CreateFine(c *gin.Context) {
	workerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || workerID <= 0 {
		c.Error(apperr.Validation("invalid worker id"))
		return
	}
	var req CreateFineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.CreateFine(c.Request.Context(), workerID, req, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// ListFines godoc
// @Summary List worker fines
// @Description List all fines for a worker with pagination.
// @Tags Worker Finance
// @Produce json
// @Security BearerAuth
// @Param id path int true "Worker ID"
// @Param page query int false "page"
// @Param limit query int false "limit"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/workers/{id}/fines [get]
func (h *Handler) ListFines(c *gin.Context) {
	workerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || workerID <= 0 {
		c.Error(apperr.Validation("invalid worker id"))
		return
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
	out, err := h.svc.ListFines(c.Request.Context(), workerID, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}
	response.List(c, out.Items, page, out.Limit, out.Offset, out.Total)
}

// ── debts ───────────────────────────────────────────────────────────────────

// CreateDebt godoc
// @Summary Create worker debt (advance/loan)
// @Description Give an advance or loan to a worker. Creates an expense transaction automatically.
// @Tags Worker Finance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Worker ID"
// @Param body body CreateDebtRequest true "Debt request"
// @Success 201 {object} response.APIResponse{data=WorkerDebt}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/workers/{id}/debts [post]
func (h *Handler) CreateDebt(c *gin.Context) {
	workerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || workerID <= 0 {
		c.Error(apperr.Validation("invalid worker id"))
		return
	}
	var req CreateDebtRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.CreateDebt(c.Request.Context(), workerID, req, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// ListDebts godoc
// @Summary List worker debts
// @Description List all debts (advances/loans) for a worker with pagination.
// @Tags Worker Finance
// @Produce json
// @Security BearerAuth
// @Param id path int true "Worker ID"
// @Param page query int false "page"
// @Param limit query int false "limit"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/workers/{id}/debts [get]
func (h *Handler) ListDebts(c *gin.Context) {
	workerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || workerID <= 0 {
		c.Error(apperr.Validation("invalid worker id"))
		return
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
	out, err := h.svc.ListDebts(c.Request.Context(), workerID, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}
	response.List(c, out.Items, page, out.Limit, out.Offset, out.Total)
}
