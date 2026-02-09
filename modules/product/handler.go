package product

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
// @Summary      Create product
// @Description  Create a new product
// @Tags         Products
// @Accept       json
// @Produce      json
// @Param        body  body      CreateRequest  true  "Product data"
// @Success      201   {object}  response.APIResponse{data=Product}
// @Failure      400   {object}  response.APIResponse
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
// @Description  Get paginated list of products with optional search
// @Tags         Products
// @Produce      json
// @Param        limit   query     int     false  "Limit (default 50, max 200)"
// @Param        offset  query     int     false  "Offset (default 0)"
// @Param        q       query     string  false  "Search by name, SKU, or barcode"
// @Success      200     {object}  response.APIResponse{data=[]Product}
// @Failure      500     {object}  response.APIResponse
// @Router       /products [get]
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
// @Summary      Get product by ID
// @Description  Get a single product by its ID
// @Tags         Products
// @Produce      json
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
// @Param        id    path      int            true  "Product ID"
// @Param        body  body      UpdateRequest  true  "Update data"
// @Success      200   {object}  response.APIResponse{data=Product}
// @Failure      400   {object}  response.APIResponse
// @Failure      404   {object}  response.APIResponse
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

// GetCard godoc
// @Summary      Get product card
// @Description  Get product with stock, categories, and additional info
// @Tags         Products
// @Produce      json
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
