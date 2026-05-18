// modules/auth/handler.go
package auth

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// helper: extract current user_id from context (set by JWT middleware)
func currentUserID(c *gin.Context) (int64, bool) {
	raw, ok := c.Get("user_id")
	if !ok {
		return 0, false
	}
	id, ok := raw.(int64)
	return id, ok
}

func currentRoles(c *gin.Context) []string {
	v, _ := c.Get("roles")
	roles, _ := v.([]string)
	return roles
}

func isAdmin(roles []string) bool {
	for _, r := range roles {
		if r == "admin" {
			return true
		}
	}
	return false
}

// Register godoc
// @Summary      Self-register
// @Description  Create a new account (public, no roles assigned)
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      RegisterRequest  true  "Registration data"
// @Success      201   {object}  response.APIResponse{data=UserWithRoles}
// @Failure      400   {object}  response.APIResponse
// @Failure      409   {object}  response.APIResponse
// @Router       /api/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	c.Set("audit_username", req.Username)

	out, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("user_id", out.ID)
	c.Set("username", out.Username)
	response.Created(c, out)
}

// Login godoc
// @Summary      User login
// @Description  Authenticate with username and password
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      LoginRequest  true  "Login credentials"
// @Success      200   {object}  response.APIResponse{data=LoginResponse}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Router       /api/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	// Capture the attempted username so audit middleware can log it
	// even when authentication fails.
	c.Set("audit_username", req.Username)

	out, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}
	// Surface the resolved user to the audit middleware on success.
	c.Set("user_id", out.User.ID)
	c.Set("username", out.User.Username)
	response.OK(c, out)
}

// RefreshToken godoc
// @Summary      Refresh tokens
// @Description  Get new access and refresh tokens using a valid refresh token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      RefreshRequest  true  "Refresh token"
// @Success      200   {object}  response.APIResponse{data=TokenResponse}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Router       /api/auth/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	out, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.Error(err)
		return
	}
	// The service doesn't return the resolved user — for audit purposes the
	// IP, request_id, and TOKEN_REFRESH action are enough to correlate
	// suspicious refresh patterns even with user_id=0.
	response.OK(c, out)
}

// CreateUser godoc
// @Summary      Create user
// @Description  Create a new user with roles (admin only)
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CreateUserRequest  true  "User data"
// @Success      201   {object}  response.APIResponse{data=UserWithRoles}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      403   {object}  response.APIResponse
// @Failure      409   {object}  response.APIResponse
// @Router       /api/auth/users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		c.Error(apperr.Unauthorized("unauthorized"))
		return
	}

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	out, err := h.svc.CreateUser(c.Request.Context(), req, callerID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// ListUsers godoc
// @Summary      List users
// @Description  Get paginated list of users (admin only)
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Param        page             query  int     false  "Page (default 1)"
// @Param        limit            query  int     false  "Limit (default 10, max 100)"
// @Param        skip             query  int     false  "Skip (legacy, default 0)"
// @Param        search           query  string  false  "Search by username or full_name"
// @Param        order_by         query  string  false  "Order by field (username, created_at)" Enums(username,created_at)
// @Param        order_direction  query  string  false  "Order direction (asc/desc)" Enums(asc,desc)
// @Success      200              {object}  response.APIResponse{data=ListResponse}
// @Failure      401              {object}  response.APIResponse
// @Failure      403              {object}  response.APIResponse
// @Router       /api/auth/users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	page := 1
	limit := 10
	offset := 0

	if pRaw, ok := c.Get("pagination"); ok {
		if p, ok := pRaw.(middleware.Pagination); ok {
			if p.Page > 0 {
				page = p.Page
			}
			if p.Limit > 0 {
				limit = p.Limit
			}
			if p.Offset >= 0 {
				offset = p.Offset
			}
		}
	}

	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if v := c.Query("skip"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
			if limit > 0 {
				page = (offset / limit) + 1
			}
		}
	}

	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	search := c.Query("search")

	orderBy := c.Query("order_by")
	if orderBy == "" {
		orderBy = "created_at"
	}
	switch orderBy {
	case "username", "created_at":
	default:
		c.Error(apperr.Validation("order_by must be 'username' or 'created_at'"))
		return
	}

	orderDir := strings.ToLower(c.Query("order_direction"))
	if orderDir == "" {
		orderDir = "desc"
	}
	if orderDir != "asc" && orderDir != "desc" {
		c.Error(apperr.Validation("order_direction must be 'asc' or 'desc'"))
		return
	}

	out, err := h.svc.ListUsers(c.Request.Context(), limit, offset, orderBy, orderDir, search)
	if err != nil {
		c.Error(err)
		return
	}

	response.List(c, out.Items, page, out.Limit, out.Offset, out.Total)
}

// GetUser godoc
// @Summary      Get user by ID
// @Description  Get a single user by ID (admin only)
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  response.APIResponse{data=UserWithRoles}
// @Failure      401  {object}  response.APIResponse
// @Failure      403  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /api/auth/users/{id} [get]
func (h *Handler) GetUser(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	out, err := h.svc.GetUser(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// UpdateUser godoc
// @Summary      Update user
// @Description  Update user fields (admin only)
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int               true  "User ID"
// @Param        body  body      UpdateUserRequest  true  "Update data"
// @Success      200   {object}  response.APIResponse{data=UserWithRoles}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      403   {object}  response.APIResponse
// @Failure      404   {object}  response.APIResponse
// @Router       /api/auth/users/{id} [patch]
func (h *Handler) UpdateUser(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		c.Error(apperr.Unauthorized("unauthorized"))
		return
	}

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	out, err := h.svc.UpdateUser(c.Request.Context(), id, req, callerID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// DeleteUser godoc
// @Summary      Delete user (soft)
// @Description  Soft delete user (admin only)
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  response.APIResponse{data=object}
// @Failure      401  {object}  response.APIResponse
// @Failure      403  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /api/auth/users/{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	actorID, ok := currentUserID(c)
	if !ok {
		c.Error(apperr.Unauthorized("unauthorized"))
		return
	}

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.DeleteUser(c.Request.Context(), id, actorID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

// ChangePassword godoc
// @Summary      Change user password
// @Description  Change password (admin or self). Invalidates all existing tokens.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                    true  "User ID"
// @Param        body  body      ChangePasswordRequest  true  "New password"
// @Success      200   {object}  response.APIResponse{data=object}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      403   {object}  response.APIResponse
// @Failure      404   {object}  response.APIResponse
// @Router       /api/auth/users/{id}/password [post]
func (h *Handler) ChangePassword(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	currentUID, ok := currentUserID(c)
	if !ok {
		c.Error(apperr.Unauthorized("unauthorized"))
		return
	}

	if !isAdmin(currentRoles(c)) {
		c.Error(apperr.Forbidden("only admin can change passwords"))
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), id, req.Password, currentUID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"changed": true})
}

// BlockUser godoc
// @Summary      Block user
// @Description  Block a user account (admin only). Invalidates all existing tokens.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int               true  "User ID"
// @Param        body  body      BlockUserRequest  true  "Block reason"
// @Success      200   {object}  response.APIResponse{data=UserWithRoles}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      403   {object}  response.APIResponse
// @Failure      404   {object}  response.APIResponse
// @Router       /api/auth/users/{id}/block [post]
func (h *Handler) BlockUser(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		c.Error(apperr.Unauthorized("unauthorized"))
		return
	}

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req BlockUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	out, err := h.svc.BlockUser(c.Request.Context(), id, req.Reason, callerID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// UnblockUser godoc
// @Summary      Unblock user
// @Description  Unblock a user account (admin only)
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  response.APIResponse{data=UserWithRoles}
// @Failure      401  {object}  response.APIResponse
// @Failure      403  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /api/auth/users/{id}/unblock [post]
func (h *Handler) UnblockUser(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		c.Error(apperr.Unauthorized("unauthorized"))
		return
	}

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	out, err := h.svc.UnblockUser(c.Request.Context(), id, callerID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}