package notification

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type PhoneHandler struct {
	repo *PhoneRepository
}

func NewPhoneHandler(repo *PhoneRepository) *PhoneHandler {
	return &PhoneHandler{repo: repo}
}

// List godoc
// @Summary      List notification phones
// @Description  Returns all admin phone numbers registered for SMS notifications
// @Tags         Notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.APIResponse{data=[]NotificationPhone}
// @Failure      401  {object}  response.APIResponse
// @Router       /api/notifications/phones [get]
func (h *PhoneHandler) List(c *gin.Context) {
	phones, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, phones)
}

// Add godoc
// @Summary      Add notification phone
// @Description  Registers a new phone number to receive SMS notifications
// @Tags         Notifications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  AddPhoneRequest  true  "Phone number and optional label"
// @Success      201  {object}  response.APIResponse{data=NotificationPhone}
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Router       /api/notifications/phones [post]
func (h *PhoneHandler) Add(c *gin.Context) {
	var req AddPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("phone is required"))
		return
	}
	p, err := h.repo.Add(c.Request.Context(), req.Phone, req.Label)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, p)
}

// Update godoc
// @Summary      Update notification phone
// @Description  Updates the label and/or enabled status of a notification phone
// @Tags         Notifications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int                true  "Phone ID"
// @Param        body  body  UpdatePhoneRequest  true  "Fields to update"
// @Success      200  {object}  response.APIResponse{data=NotificationPhone}
// @Failure      400  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /api/notifications/phones/{id} [put]
func (h *PhoneHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(apperr.Validation("invalid id"))
		return
	}
	var req UpdatePhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("invalid request body"))
		return
	}
	p, err := h.repo.Update(c.Request.Context(), id, req.Label, req.Enabled)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, p)
}

// Delete godoc
// @Summary      Delete notification phone
// @Description  Permanently removes a notification phone number by ID
// @Tags         Notifications
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  int  true  "Phone ID"
// @Success      200  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /api/notifications/phones/{id} [delete]
func (h *PhoneHandler) Delete(c *gin.Context) {
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
