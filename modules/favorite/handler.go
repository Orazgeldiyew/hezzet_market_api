package favorite

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func getUserID(c *gin.Context) int64 {
	raw, _ := c.Get("user_id")
	id, _ := raw.(int64)
	return id
}

// List godoc
// @Summary      List favorite products
// @Description  Returns the current user's favorite products sorted by position
// @Tags         Favorites
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.APIResponse{data=[]FavoriteProduct}
// @Router       /api/favorites [get]
func (h *Handler) List(c *gin.Context) {
	userID := getUserID(c)
	favorites, err := h.repo.List(c.Request.Context(), userID)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, favorites)
}

// Add godoc
// @Summary      Add product to favorites
// @Description  Adds a product to the current user's favorites
// @Tags         Favorites
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  AddRequest  true  "Product ID"
// @Success      201  {object}  response.APIResponse{data=FavoriteProduct}
// @Failure      409  {object}  response.APIResponse
// @Router       /api/favorites [post]
func (h *Handler) Add(c *gin.Context) {
	var req AddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("product_id is required"))
		return
	}
	userID := getUserID(c)
	fav, err := h.repo.Add(c.Request.Context(), userID, req.ProductID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, fav)
}

// Remove godoc
// @Summary      Remove product from favorites
// @Description  Removes a product from the current user's favorites
// @Tags         Favorites
// @Produce      json
// @Security     BearerAuth
// @Param        product_id  path  int  true  "Product ID"
// @Success      200  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /api/favorites/{product_id} [delete]
func (h *Handler) Remove(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("product_id"), 10, 64)
	if err != nil {
		c.Error(apperr.Validation("invalid product_id"))
		return
	}
	userID := getUserID(c)
	if err := h.repo.Remove(c.Request.Context(), userID, productID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

// Reorder godoc
// @Summary      Reorder favorite products
// @Description  Updates the position of favorite products for the current user
// @Tags         Favorites
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  ReorderRequest  true  "New positions"
// @Success      200  {object}  response.APIResponse
// @Router       /api/favorites/reorder [put]
func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("invalid request body"))
		return
	}
	userID := getUserID(c)
	if err := h.repo.Reorder(c.Request.Context(), userID, req.Items); err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, gin.H{"reordered": true})
}
