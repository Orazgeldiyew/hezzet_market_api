package printer

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Create godoc
// @Summary Create printer
// @Tags Printers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateRequest true "Printer data"
// @Success 201 {object} response.APIResponse
// @Router /api/printers [post]
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
// @Summary List printers
// @Tags Printers
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Router /api/printers [get]
func (h *Handler) List(c *gin.Context) {
	out, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// Update godoc
// @Summary Update printer
// @Tags Printers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Printer ID"
// @Param body body UpdateRequest true "Printer data"
// @Success 200 {object} response.APIResponse
// @Router /api/printers/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid printer id"))
		return
	}

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
// @Summary Delete printer
// @Tags Printers
// @Produce json
// @Security BearerAuth
// @Param id path int true "Printer ID"
// @Success 200 {object} response.APIResponse
// @Router /api/printers/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid printer id"))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

// TestPrint godoc
// @Summary Send test print
// @Tags Printers
// @Produce json
// @Security BearerAuth
// @Param id path int true "Printer ID"
// @Success 200 {object} response.APIResponse
// @Router /api/printers/{id}/test [post]
func (h *Handler) TestPrint(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid printer id"))
		return
	}

	if err := h.svc.TestPrint(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"printed": true})
}
