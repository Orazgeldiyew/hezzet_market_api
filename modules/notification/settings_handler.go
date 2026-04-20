package notification

import (
	"github.com/gin-gonic/gin"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type SettingsHandler struct {
	repo *SettingsRepository
}

func NewSettingsHandler(repo *SettingsRepository) *SettingsHandler {
	return &SettingsHandler{repo: repo}
}

// Get godoc
// @Summary      Get notification settings
// @Description  Returns the runtime-configurable notification settings (digest hour, etc.).
// @Tags         Notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.APIResponse{data=Settings}
// @Failure      500  {object}  response.APIResponse
// @Router       /api/notifications/settings [get]
func (h *SettingsHandler) Get(c *gin.Context) {
	s, err := h.repo.Get(c.Request.Context())
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, s)
}

// Update godoc
// @Summary      Update notification settings
// @Description  Partial update of notification settings. admin/manager only.
// @Tags         Notifications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  UpdateSettingsRequest  true  "fields to change"
// @Success      200  {object}  response.APIResponse{data=Settings}
// @Failure      400  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Router       /api/notifications/settings [put]
func (h *SettingsHandler) Update(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation(err.Error()))
		return
	}
	s, err := h.repo.Update(c.Request.Context(), req)
	if err != nil {
		c.Error(apperr.Internal(err))
		return
	}
	response.OK(c, s)
}
