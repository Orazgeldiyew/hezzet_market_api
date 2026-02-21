package warehouse

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
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


// responseErrorBadRequest creates a plain error that your global error middleware should map.
// If you already have apperr.Validation(...) available here, replace this helper with it.
// func responseErrorBadRequest(msg string) error {
// 	// Using gin.Error would work too, but you already use c.Error(err) pattern.
// 	// If your project has apperr.Validation, prefer that:
// 	// return apperr.Validation(msg)
// 	return &gin.Error{Err: strconv.ErrSyntax, Type: gin.ErrorTypeBind, Meta: msg}
// }
