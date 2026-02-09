package supplier

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Create godoc
// @Summary      Create supplier
// @Description  Create a new supplier
// @Tags         Suppliers
// @Accept       json
// @Produce      json
// @Param        body  body      CreateRequest  true  "Supplier data"
// @Success      201   {object}  response.APIResponse{data=Supplier}
// @Failure      400   {object}  response.APIResponse
// @Router       /suppliers [post]
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
// @Summary      List suppliers
// @Description  Get paginated list of active suppliers with optional search by name/phone/email
// @Tags         Suppliers
// @Produce      json
// @Param        limit   query     int     false  "Limit (default 50, max 200)"
// @Param        offset  query     int     false  "Offset (default 0)"
// @Param        q       query     string  false  "Search by name, phone, or email"
// @Success      200     {object}  response.APIResponse{data=ListResponse}
// @Failure      500     {object}  response.APIResponse
// @Router       /suppliers [get]
func (h *Handler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	q := c.Query("q")

	out, err := h.svc.List(c.Request.Context(), limit, offset, q)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// Get godoc
// @Summary      Get supplier by ID
// @Description  Get a single supplier by its ID
// @Tags         Suppliers
// @Produce      json
// @Param        id   path      int  true  "Supplier ID"
// @Success      200  {object}  response.APIResponse{data=Supplier}
// @Failure      404  {object}  response.APIResponse
// @Router       /suppliers/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	out, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// Update godoc
// @Summary      Update supplier
// @Description  Update supplier fields (name, phone, email, address, is_active)
// @Tags         Suppliers
// @Accept       json
// @Produce      json
// @Param        id    path      int            true  "Supplier ID"
// @Param        body  body      UpdateRequest  true  "Update data"
// @Success      200   {object}  response.APIResponse{data=Supplier}
// @Failure      400   {object}  response.APIResponse
// @Failure      404   {object}  response.APIResponse
// @Router       /suppliers/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
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
// @Summary      Delete supplier (soft)
// @Description  Soft delete supplier by setting is_active=false
// @Tags         Suppliers
// @Produce      json
// @Param        id   path      int  true  "Supplier ID"
// @Success      200  {object}  response.APIResponse{data=object}
// @Failure      404  {object}  response.APIResponse
// @Router       /suppliers/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
