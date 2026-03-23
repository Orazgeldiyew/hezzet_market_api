package permissions

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ── Roles ──

// ListRoles godoc
// @Summary      List all roles
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.APIResponse{data=[]Role}
// @Router       /api/roles [get]
func (h *Handler) ListRoles(c *gin.Context) {
	roles, err := h.svc.ListRoles(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, roles)
}

// GetRole godoc
// @Summary      Get role with permissions
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Role ID"
// @Success      200 {object} response.APIResponse
// @Router       /api/roles/{id} [get]
func (h *Handler) GetRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Error(apperr.Validation("invalid role id"))
		return
	}
	role, perms, err := h.svc.GetRole(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"role": role, "permissions": perms})
}

// CreateRole godoc
// @Summary      Create a new role
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body CreateRoleRequest true "Role data"
// @Success      201 {object} response.APIResponse{data=Role}
// @Router       /api/roles [post]
func (h *Handler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("invalid request body"))
		return
	}
	role, err := h.svc.CreateRole(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, role)
}

// UpdateRole godoc
// @Summary      Update role name/description
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int              true "Role ID"
// @Param        body body UpdateRoleRequest true "Fields to update"
// @Success      200 {object} response.APIResponse{data=Role}
// @Router       /api/roles/{id} [patch]
func (h *Handler) UpdateRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Error(apperr.Validation("invalid role id"))
		return
	}
	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("invalid request body"))
		return
	}
	role, err := h.svc.UpdateRole(c.Request.Context(), id, req)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, role)
}

// DeleteRole godoc
// @Summary      Delete a custom role
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Role ID"
// @Success      200 {object} response.APIResponse
// @Router       /api/roles/{id} [delete]
func (h *Handler) DeleteRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Error(apperr.Validation("invalid role id"))
		return
	}
	if err := h.svc.DeleteRole(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

// ── Permissions ──

// ListPermissions godoc
// @Summary      List all permissions
// @Tags         Permissions
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.APIResponse{data=[]Permission}
// @Router       /api/permissions [get]
func (h *Handler) ListPermissions(c *gin.Context) {
	perms, err := h.svc.ListPermissions(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, perms)
}

// Matrix godoc
// @Summary      Permission matrix (roles x modules x actions)
// @Tags         Permissions
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.APIResponse{data=[]MatrixEntry}
// @Router       /api/permissions/matrix [get]
func (h *Handler) Matrix(c *gin.Context) {
	m, err := h.svc.Matrix(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, m)
}

// UpdatePermission godoc
// @Summary      Set granted for one permission
// @Tags         Permissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        role_id path int    true "Role ID"
// @Param        module  path string true "Module name"
// @Param        action  path string true "Action (view, create, update, delete)"
// @Param        body    body UpdatePermissionRequest true "granted flag"
// @Success      200 {object} response.APIResponse{data=Permission}
// @Router       /api/permissions/{role_id}/{module}/{action} [put]
func (h *Handler) UpdatePermission(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.Error(apperr.Validation("invalid role_id"))
		return
	}
	module := c.Param("module")
	action := c.Param("action")
	if module == "" || action == "" {
		c.Error(apperr.Validation("module and action are required"))
		return
	}

	var req UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("invalid request body"))
		return
	}

	callerRoles := getRoles(c)
	p, err := h.svc.UpdatePermission(c.Request.Context(), roleID, module, action, req.Granted, callerRoles)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, p)
}

// BulkUpdate godoc
// @Summary      Bulk update permissions for a role
// @Tags         Permissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        role_id path int                   true "Role ID"
// @Param        body    body BulkPermissionRequest  true "Permissions array"
// @Success      200 {object} response.APIResponse{data=[]Permission}
// @Router       /api/permissions/{role_id}/bulk [put]
func (h *Handler) BulkUpdate(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.Error(apperr.Validation("invalid role_id"))
		return
	}

	var req BulkPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("invalid request body"))
		return
	}

	callerRoles := getRoles(c)
	perms, err := h.svc.BulkUpdatePermissions(c.Request.Context(), roleID, req, callerRoles)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, perms)
}

func getRoles(c *gin.Context) []string {
	rolesVal, _ := c.Get("roles")
	roles, _ := rolesVal.([]string)
	return roles
}

