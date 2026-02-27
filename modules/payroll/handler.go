package payroll

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func extractUserID(c *gin.Context) int64 {
	v, _ := c.Get("user_id")
	uid, _ := v.(int64)
	return uid
}

// List godoc
// @Summary List payroll runs
// @Description List payroll runs with optional worker_id and period filters.
// @Tags Payroll
// @Produce json
// @Security BearerAuth
// @Param worker_id query int false "Worker ID"
// @Param period query string false "Period (YYYY-MM)"
// @Param page query int false "page"
// @Param limit query int false "limit"
// @Success 200 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/payroll [get]
func (h *Handler) List(c *gin.Context) {
	var workerID *int64
	var period *string

	if v := c.Query("worker_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			workerID = &n
		}
	}
	if v := c.Query("period"); v != "" {
		period = &v
	}

	page := 1
	limit := 10
	offset := 0
	if pRaw, ok := c.Get("pagination"); ok {
		if p, ok := pRaw.(middleware.Pagination); ok {
			page = p.Page
			limit = p.Limit
			offset = p.Offset
		}
	}

	out, err := h.svc.List(c.Request.Context(), workerID, period, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}
	response.List(c, out.Items, page, out.Limit, out.Offset, out.Total)
}

// Calculate godoc
// @Summary Calculate payroll
// @Description Calculate monthly payroll for a worker (does NOT pay).
// @Tags Payroll
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CalculateRequest true "Calculate request"
// @Success 201 {object} response.APIResponse{data=PayrollRun}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/payroll/calculate [post]
func (h *Handler) Calculate(c *gin.Context) {
	var req CalculateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.Calculate(c.Request.Context(), req, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.Created(c, out)
}

// Pay godoc
// @Summary Pay payroll
// @Description Pay a calculated payroll run. Creates transaction + payment, marks fines/debts.
// @Tags Payroll
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Payroll Run ID"
// @Param body body PayRequest true "Payment type"
// @Success 200 {object} response.APIResponse{data=PayrollRun}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/payroll/{id}/pay [post]
func (h *Handler) Pay(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Error(apperr.Validation("invalid payroll run id"))
		return
	}
	var req PayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	userID := extractUserID(c)

	out, err := h.svc.Pay(c.Request.Context(), id, req.PaymentTypeCode, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, out)
}

// ExportExcel godoc
// @Summary      Export payroll as Excel
// @Description  Download all payroll runs for a given period as .xlsx file
// @Tags         Payroll
// @Produce      application/octet-stream
// @Security     BearerAuth
// @Param        period  query  string  true  "Period in YYYY-MM format"
// @Success      200  {file}   binary
// @Failure      400  {object} response.APIResponse
// @Failure      401  {object} response.APIResponse
// @Failure      403  {object} response.APIResponse
// @Router       /api/payroll/export [get]
func (h *Handler) ExportExcel(c *gin.Context) {
	period := c.Query("period")
	if period == "" {
		c.Error(apperr.Validation("period query param is required"))
		return
	}

	data, err := h.svc.ExportPeriod(c.Request.Context(), period)
	if err != nil {
		c.Error(err)
		return
	}

	filename := "payroll_" + period + ".xlsx"
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write(data)
}
