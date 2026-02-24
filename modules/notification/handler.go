package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

// Handler exposes admin endpoints for inspecting and managing SMS logs.
type Handler struct {
	logRepo *LogRepository
	queue   *Queue
}

// NewHandler creates a new notification Handler.
func NewHandler(logRepo *LogRepository, queue *Queue) *Handler {
	return &Handler{logRepo: logRepo, queue: queue}
}

// ListSMSLogs godoc
// @Summary List SMS logs
// @Description List SMS audit logs with filters and pagination (admin only).
// @Tags Notifications
// @Produce json
// @Security BearerAuth
// @Param status   query string false "Filter by status (queued|sending|sent|retrying|rate_limited|dlq|failed)"
// @Param type     query string false "Filter by notification type"
// @Param phone    query string false "Filter by phone number"
// @Param job_id   query string false "Filter by job ID"
// @Param from_date query string false "Created at from (RFC3339)"
// @Param to_date   query string false "Created at to (RFC3339)"
// @Param page     query int    false "Page number"
// @Param limit    query int    false "Items per page"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/notifications/sms [get]
func (h *Handler) ListSMSLogs(c *gin.Context) {
	if err := h.checkReady(); err != nil {
		c.Error(err)
		return
	}

	var f SMSLogFilter
	f.Status = c.Query("status")
	f.Type = c.Query("type")
	f.Phone = c.Query("phone")
	f.JobID = c.Query("job_id")

	if v := c.Query("from_date"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			c.Error(apperr.Validation("from_date must be RFC3339"))
			return
		}
		f.FromDate = &t
	}
	if v := c.Query("to_date"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			c.Error(apperr.Validation("to_date must be RFC3339"))
			return
		}
		f.ToDate = &t
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

	result, err := h.logRepo.List(c.Request.Context(), f, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}

	response.List(c, result.Items, page, result.Limit, result.Offset, result.Total)
}

// GetSMSLog godoc
// @Summary Get SMS log by job ID
// @Description Get a single SMS audit log entry by job ID (admin only).
// @Tags Notifications
// @Produce json
// @Security BearerAuth
// @Param job_id path string true "Job ID"
// @Success 200 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/notifications/sms/{job_id} [get]
func (h *Handler) GetSMSLog(c *gin.Context) {
	if err := h.checkReady(); err != nil {
		c.Error(err)
		return
	}

	jobID := c.Param("job_id")
	if jobID == "" {
		c.Error(apperr.Validation("job_id is required"))
		return
	}

	log, err := h.logRepo.GetByJobID(c.Request.Context(), jobID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, log)
}

// RequeueSMSLog godoc
// @Summary Requeue a failed/DLQ SMS job
// @Description Re-enqueue a job from DLQ/failed back into the SMS queue (admin only).
// @Tags Notifications
// @Produce json
// @Security BearerAuth
// @Param job_id path string true "Job ID"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/notifications/sms/{job_id}/requeue [post]
func (h *Handler) RequeueSMSLog(c *gin.Context) {
	if err := h.checkReady(); err != nil {
		c.Error(err)
		return
	}
	if h.queue == nil {
		c.Error(apperr.Internal(fmt.Errorf("notifications disabled (redis not configured)")))
		return
	}

	jobID := c.Param("job_id")
	if jobID == "" {
		c.Error(apperr.Validation("job_id is required"))
		return
	}

	ctx := c.Request.Context()

	// Load the log row to get the job details.
	logEntry, err := h.logRepo.GetByJobID(ctx, jobID)
	if err != nil {
		c.Error(err)
		return
	}

	// Only allow requeue from terminal statuses.
	if logEntry.Status != StatusDLQ && logEntry.Status != StatusFailed && logEntry.Status != StatusRateLimited {
		c.Error(apperr.Validation("job is not in a requeueable status (must be dlq, failed, or rate_limited)"))
		return
	}

	// Re-enqueue into Redis.
	job := SMSJob{
		JobID:       logEntry.JobID,
		Type:        logEntry.Type,
		ToPhone:     logEntry.ToPhone,
		Message:     logEntry.Message,
		Attempt:     0,
		MaxAttempts: logEntry.MaxAttempts,
		DedupKey:    "", // no dedup on manual requeue
		CreatedAt:   time.Now().Unix(),
	}
	if err := h.pushToQueue(ctx, job); err != nil {
		c.Error(apperr.Internal(err))
		return
	}

	// Update DB status.
	if err := h.logRepo.ResetForRequeue(ctx, jobID); err != nil {
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"job_id": jobID, "status": "queued"})
}

// pushToQueue directly pushes a job to the Redis main queue (bypassing dedup).
func (h *Handler) pushToQueue(ctx context.Context, job SMSJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	return h.queue.rdb.RPush(ctx, keyQueue, data).Err()
}

// checkReady returns an error if the log repository is not configured.
func (h *Handler) checkReady() error {
	if h.logRepo == nil {
		return apperr.Internal(fmt.Errorf("notification logging not configured"))
	}
	return nil
}
