package permissions

import (
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

// List godoc
// @Summary      List module permissions
// @Description  Returns all role-module permissions (enabled/disabled).
// @Tags         Permissions
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.APIResponse{data=[]Permission}
// @Failure      500 {object} response.APIResponse
// @Router       /api/permissions [get]
func (h *Handler) List(c *gin.Context) {
	perms, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, perms)
}

// Update godoc
// @Summary      Update module permission
// @Description  Enable or disable a module for a specific role. Takes effect immediately (cache invalidated).
// @Tags         Permissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        role    path string true "Role (cashier, operator, manager)"
// @Param        module  path string true "Module name (sales, stock, products, etc.)"
// @Param        body    body UpdatePermissionRequest true "enabled flag"
// @Success      200 {object} response.APIResponse{data=Permission}
// @Failure      400 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Router       /api/permissions/{role}/{module} [put]
func (h *Handler) Update(c *gin.Context) {
	role := c.Param("role")
	module := c.Param("module")

	if role == "" || module == "" {
		c.Error(apperr.Validation("role and module are required"))
		return
	}

	var req UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("invalid request body"))
		return
	}

	callerIsAdmin := middleware.HasAnyRole(c, "admin")

	p, err := h.svc.Update(c.Request.Context(), role, module, req.Enabled, callerIsAdmin)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, p)
}
