package warehouse

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)


type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Create godoc
//
//	@Summary		Create warehouse
//	@Description	Creates a new warehouse (soft-delete supported). Requires manager role.
//	@Tags			Warehouses
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateRequest	true	"Create warehouse request"
//	@Success		201		{object}	response.APIResponse{data=Warehouse}
//	@Failure		400		{object}	response.APIResponse{error=response.APIError}
//	@Failure		401		{object}	response.APIResponse{error=response.APIError}
//	@Failure		403		{object}	response.APIResponse{error=response.APIError}
//	@Failure		409		{object}	response.APIResponse{error=response.APIError}
//	@Failure		500		{object}	response.APIResponse{error=response.APIError}
//	@Router			/api/warehouses [post]
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
//
//	@Summary		List warehouses
//	@Description	Returns active (not deleted) warehouses with pagination.
//	@Tags			Warehouses
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page	query		int	false	"Page number"		minimum(1)
//	@Param			limit	query		int	false	"Page size"			minimum(1)	maximum(200)
//	@Success		200		{object}	response.APIResponse{data=[]Warehouse}
//	@Failure		401		{object}	response.APIResponse{error=response.APIError}
//	@Failure		403		{object}	response.APIResponse{error=response.APIError}
//	@Failure		500		{object}	response.APIResponse{error=response.APIError}
//	@Router			/api/warehouses [get]
func (h *Handler) List(c *gin.Context) {
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

	out, err := h.svc.List(c.Request.Context(), limit, offset)
	if err != nil {
		c.Error(err)
		return
	}
	response.List(c, out.Items, page, out.Limit, out.Offset, out.Total)
}

// Get godoc
//
//	@Summary		Get warehouse by ID
//	@Description	Returns a single warehouse by ID (not deleted).
//	@Tags			Warehouses
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int64	true	"Warehouse ID"
//	@Success		200	{object}	response.APIResponse{data=Warehouse}
//	@Failure		400	{object}	response.APIResponse{error=response.APIError}
//	@Failure		401	{object}	response.APIResponse{error=response.APIError}
//	@Failure		403	{object}	response.APIResponse{error=response.APIError}
//	@Failure		404	{object}	response.APIResponse{error=response.APIError}
//	@Failure		500	{object}	response.APIResponse{error=response.APIError}
//	@Router			/api/warehouses/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid warehouse id"))
		return
	}

	out, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// Update godoc
//
//	@Summary		Update warehouse
//	@Description	Updates the editable fields of a warehouse (name, address, is_active). Manager+.
//	@Tags			Warehouses
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int64			true	"Warehouse ID"
//	@Param			body	body		UpdateRequest	true	"Fields to update"
//	@Success		200		{object}	response.APIResponse{data=Warehouse}
//	@Failure		400		{object}	response.APIResponse{error=response.APIError}
//	@Failure		404		{object}	response.APIResponse{error=response.APIError}
//	@Failure		409		{object}	response.APIResponse{error=response.APIError}
//	@Router			/api/warehouses/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid warehouse id"))
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
//
//	@Summary		Delete warehouse (soft)
//	@Description	Soft-deletes a warehouse. Blocked when it still holds stock or has active reservations.
//	@Tags			Warehouses
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int64	true	"Warehouse ID"
//	@Success		200	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse{error=response.APIError}
//	@Failure		409	{object}	response.APIResponse{error=response.APIError}
//	@Router			/api/warehouses/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid warehouse id"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
