package client

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
// @Summary      Create client
// @Description  Create a new client (customer)
// @Tags         Clients
// @Accept       json
// @Produce      json
// @Param        body  body      CreateRequest  true  "Client data"
// @Success      201   {object}  response.APIResponse{data=Client}
// @Failure      400   {object}  response.APIResponse
// @Failure      409   {object}  response.APIResponse
// @Router       /clients [post]
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
// @Summary      List clients
// @Description  Get paginated list of active clients with optional search
// @Tags         Clients
// @Produce      json
// @Param        limit   query     int     false  "Limit (default 50, max 200)"
// @Param        offset  query     int     false  "Offset (default 0)"
// @Param        q       query     string  false  "Search by name or phone"
// @Success      200     {object}  response.APIResponse{data=ListResponse}
// @Failure      500     {object}  response.APIResponse
// @Router       /clients [get]
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
// @Summary      Get client by ID
// @Description  Get a single active client by its ID
// @Tags         Clients
// @Produce      json
// @Param        id   path      int  true  "Client ID"
// @Success      200  {object}  response.APIResponse{data=Client}
// @Failure      404  {object}  response.APIResponse
// @Router       /clients/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	out, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// Update godoc
// @Summary      Update client
// @Description  Update client fields (name, phone, email, is_active)
// @Tags         Clients
// @Accept       json
// @Produce      json
// @Param        id    path      int            true  "Client ID"
// @Param        body  body      UpdateRequest  true  "Update data"
// @Success      200   {object}  response.APIResponse{data=Client}
// @Failure      400   {object}  response.APIResponse
// @Failure      404   {object}  response.APIResponse
// @Failure      409   {object}  response.APIResponse
// @Router       /clients/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
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
// @Summary      Delete client (soft)
// @Description  Soft delete client by setting is_active=false
// @Tags         Clients
// @Produce      json
// @Param        id   path      int  true  "Client ID"
// @Success      200  {object}  response.APIResponse{data=object}
// @Failure      404  {object}  response.APIResponse
// @Router       /clients/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
