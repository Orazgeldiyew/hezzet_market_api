package workercard

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

// List godoc
// @Summary      List worker cards
// @Description  Returns all worker cards, optionally filtered by worker_id
// @Tags         WorkerCards
// @Produce      json
// @Security     BearerAuth
// @Param        worker_id  query   int  false  "Filter by worker ID"
// @Success      200  {object}  response.APIResponse{data=[]WorkerCard}
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Router       /api/worker-cards [get]
func (h *Handler) List(c *gin.Context) {
	var workerID *int64
	if v := c.Query("worker_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			c.Error(apperr.Validation("invalid worker_id"))
			return
		}
		workerID = &id
	}
	cards, err := h.repo.List(c.Request.Context(), workerID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, cards)
}

// GetByCard godoc
// @Summary      Find worker card by code
// @Description  Returns an enabled worker card matching the given QR/barcode code
// @Tags         WorkerCards
// @Produce      json
// @Security     BearerAuth
// @Param        code  path  string  true  "Card QR or barcode code"
// @Success      200  {object}  response.APIResponse{data=WorkerCard}
// @Failure      404  {object}  response.APIResponse
// @Router       /api/worker-cards/by-card/{code} [get]
func (h *Handler) GetByCard(c *gin.Context) {
	code := c.Param("code")
	card, err := h.repo.GetByCardCode(c.Request.Context(), code)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, card)
}

// Add godoc
// @Summary      Create a worker card
// @Description  Generates a new card with a unique code for the specified worker
// @Tags         WorkerCards
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  CreateCardRequest  true  "Worker ID and optional label"
// @Success      201  {object}  response.APIResponse{data=WorkerCard}
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Router       /api/worker-cards [post]
func (h *Handler) Add(c *gin.Context) {
	var req CreateCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("worker_id is required"))
		return
	}
	card, err := h.repo.Add(c.Request.Context(), req.WorkerID, req.Label, req.CardCode)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, card)
}

// Update godoc
// @Summary      Update a worker card
// @Description  Updates the label and/or enabled status of a worker card
// @Tags         WorkerCards
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int               true  "Card ID"
// @Param        body  body  UpdateCardRequest  true  "Fields to update"
// @Success      200  {object}  response.APIResponse{data=WorkerCard}
// @Failure      400  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /api/worker-cards/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(apperr.Validation("invalid id"))
		return
	}
	var req UpdateCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("invalid request body"))
		return
	}
	card, err := h.repo.Update(c.Request.Context(), id, req.Label, req.Enabled)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, card)
}

// Delete godoc
// @Summary      Delete a worker card
// @Description  Permanently removes a worker card by ID
// @Tags         WorkerCards
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  int  true  "Card ID"
// @Success      200  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /api/worker-cards/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(apperr.Validation("invalid id"))
		return
	}
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
