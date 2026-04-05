package discountrule

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	var userID *int64
	if v, ok := c.Get("user_id"); ok {
		if uid, ok := v.(int64); ok {
			userID = &uid
		}
	}
	d, err := h.repo.Create(c.Request.Context(), req, userID)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.Created(c, d)
}

func (h *Handler) List(c *gin.Context) {
	rules, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, rules)
}

func (h *Handler) GetByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	d, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.Error(apperr.NotFound("NOT_FOUND", "discount rule not found"))
			return
		}
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, d)
}

func (h *Handler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	d, err := h.repo.Update(c.Request.Context(), id, req)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.Error(apperr.NotFound("NOT_FOUND", "discount rule not found"))
			return
		}
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, d)
}

func (h *Handler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		if err == pgx.ErrNoRows {
			c.Error(apperr.NotFound("NOT_FOUND", "discount rule not found"))
			return
		}
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
