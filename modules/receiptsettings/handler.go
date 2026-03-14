package receiptsettings

import (
	"html/template"

	"github.com/gin-gonic/gin"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// GetSettings godoc
// @Summary      Get receipt settings
// @Description  Returns the current receipt print settings (store name, footer, template, etc.)
// @Tags         ReceiptSettings
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.APIResponse{data=receiptsettings.ReceiptSettings}
// @Failure      401  {object}  response.APIResponse
// @Router       /api/receipt-settings [get]
func (h *Handler) GetSettings(c *gin.Context) {
	s, err := h.repo.Get(c.Request.Context())
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, s)
}

// UpdateSettings godoc
// @Summary      Update receipt settings
// @Description  Updates receipt print settings; validates custom Go template syntax if provided
// @Tags         ReceiptSettings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  UpdateRequest  true  "Receipt settings to update"
// @Success      200  {object}  response.APIResponse{data=receiptsettings.ReceiptSettings}
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Router       /api/receipt-settings [put]
func (h *Handler) UpdateSettings(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	// Validate custom template syntax if provided
	if req.Template != nil && *req.Template != "" {
		if _, err := template.New("receipt").Parse(*req.Template); err != nil {
			c.Error(apperr.Validation("invalid template syntax: " + err.Error()))
			return
		}
	}

	if err := h.repo.Update(c.Request.Context(), req); err != nil {
		c.Error(apperr.Internal(err))
		return
	}

	// Return merged settings
	s, err := h.repo.Get(c.Request.Context())
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, s)
}
