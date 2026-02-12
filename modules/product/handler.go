package product

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

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Create godoc
// @Summary      Create product
// @Description  Create a new product
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CreateRequest  true  "Product data"
// @Success      201   {object}  response.APIResponse{data=Product}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      403   {object}  response.APIResponse
// @Failure      409   {object}  response.APIResponse
// @Router       /products [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	p, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, p)
}

// List godoc
// @Summary      List products
// @Description  Get paginated list of active products with optional search
// @Tags         Products
// @Produce      json
// @Security     BearerAuth
// @Param        page             query  int     false  "Page (default 1)"
// @Param        limit            query  int     false  "Limit (default 10, max 100)"
// @Param        skip             query  int     false  "Skip (legacy, default 0). If provided, overrides page/offset."
// @Param        search           query  string  false  "Search by name, SKU, or barcode"
// @Param        order_by         query  string  false  "Order by field (name, created_at)" Enums(name,created_at)
// @Param        order_direction  query  string  false  "Order direction (asc/desc)" Enums(asc,desc)
// @Success      200              {object}  response.APIResponse{data=ListResponse}
// @Failure      400              {object}  response.APIResponse
// @Failure      500              {object}  response.APIResponse
// @Router       /products [get]
func (h *Handler) List(c *gin.Context) {
	// Defaults from pagination middleware (page+limit -> offset)
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

	// legacy overrides
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

	// Validate limit/offset hard
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	q := c.Query("search")

	orderBy := c.Query("order_by")
	if orderBy == "" {
		orderBy = "created_at"
	}
	switch orderBy {
	case "name", "created_at":
	default:
		c.Error(apperr.Validation("order_by must be 'name' or 'created_at'"))
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

	out, err := h.svc.List(c.Request.Context(), limit, offset, orderBy, orderDir, q)
	if err != nil {
		c.Error(err)
		return
	}

	response.List(c, out, page, out.Limit, out.Offset, out.Total)
}

// Get godoc
// @Summary      Get product by ID
// @Description  Get a single product by its ID
// @Tags         Products
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Product ID"
// @Success      200  {object}  response.APIResponse{data=Product}
// @Failure      404  {object}  response.APIResponse
// @Router       /products/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	p, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, p)
}

// Update godoc
// @Summary      Update product
// @Description  Update product fields
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int            true  "Product ID"
// @Param        body  body      UpdateRequest  true  "Update data"
// @Success      200   {object}  response.APIResponse{data=Product}
// @Failure      400   {object}  response.APIResponse
// @Failure      404   {object}  response.APIResponse
// @Failure      409   {object}  response.APIResponse
// @Router       /products/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	p, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, p)
}

// Delete godoc
// @Summary      Delete product (soft)
// @Description  Soft delete product by setting is_active=false
// @Tags         Products
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Product ID"
// @Success      200  {object}  response.APIResponse{data=object}
// @Failure      401  {object}  response.APIResponse
// @Failure      403  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /products/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

// GetCard godoc
// @Summary      Get product card
// @Description  Get product with stock, categories, and additional info
// @Tags         Products
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Product ID"
// @Success      200  {object}  response.APIResponse{data=Card}
// @Failure      404  {object}  response.APIResponse
// @Router       /products/{id}/card [get]
func (h *Handler) GetCard(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	card, err := h.svc.GetCard(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, card)
}

// GetCategories godoc
// @Summary      Get product categories
// @Description  Get all categories associated with a product
// @Tags         Products
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Product ID"
// @Success      200  {object}  response.APIResponse{data=[]CategoryBrief}
// @Failure      404  {object}  response.APIResponse
// @Router       /products/{id}/categories [get]
func (h *Handler) GetCategories(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	cats, err := h.svc.GetCategories(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, cats)
}

// SetCategories godoc
// @Summary      Set product categories
// @Description  Replace all categories for a product
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                   true  "Product ID"
// @Param        body  body      SetCategoriesRequest  true  "Category IDs"
// @Success      200   {object}  response.APIResponse{data=[]CategoryBrief}
// @Failure      400   {object}  response.APIResponse
// @Failure      404   {object}  response.APIResponse
// @Router       /products/{id}/categories [put]
func (h *Handler) SetCategories(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req SetCategoriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	cats, err := h.svc.SetCategories(c.Request.Context(), id, req.CategoryIDs)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, cats)
}

// RemoveCategory godoc
// @Summary      Remove product category
// @Description  Remove a single category from a product
// @Tags         Products
// @Produce      json
// @Security     BearerAuth
// @Param        id          path      int  true  "Product ID"
// @Param        categoryId  path      int  true  "Category ID"
// @Success      200         {object}  response.APIResponse{data=object}
// @Failure      404         {object}  response.APIResponse
// @Router       /products/{id}/categories/{categoryId} [delete]
func (h *Handler) RemoveCategory(c *gin.Context) {
	productID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	categoryID, _ := strconv.ParseInt(c.Param("categoryId"), 10, 64)

	if err := h.svc.RemoveCategory(c.Request.Context(), productID, categoryID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
