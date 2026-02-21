package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// APIResponse is the standard JSON envelope for every response.
//
// Success responses:
//
//	{ "success": true,  "status_code": 200, "data": <any>, "meta": <optional> }
//
// Error responses (written by ErrorMiddleware, not by handlers directly):
//
//	{ "success": false, "status_code": 4xx/5xx, "error": { ... } }
type APIResponse struct {
	Success    bool        `json:"success"`
	StatusCode int         `json:"status_code"`
	Data       any         `json:"data,omitempty"`
	Error      *APIError   `json:"error,omitempty"`
	Meta       *Meta       `json:"meta,omitempty"`
}

// APIError is the structured error payload inside an error envelope.
// Details and RequestID are omitted from JSON when empty.
type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

type Meta struct {
	Timestamp  string          `json:"timestamp,omitempty"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

type PaginationMeta struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Offset     int  `json:"offset"`
	Total      int  `json:"total"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
	TotalPages int  `json:"total_pages"`
}

// OK writes a 200 success response.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, APIResponse{
		Success:    true,
		StatusCode: http.StatusOK,
		Data:       data,
	})
}

// Created writes a 201 success response.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, APIResponse{
		Success:    true,
		StatusCode: http.StatusCreated,
		Data:       data,
	})
}

// Error writes a structured error response directly.
// Prefer using c.Error(err) in handlers and letting ErrorMiddleware handle it.
// Use this only from middleware or places where the middleware cannot intercept.
func Error(c *gin.Context, status int, code, msg string) {
	c.JSON(status, APIResponse{
		Success:    false,
		StatusCode: status,
		Error:      &APIError{Code: code, Message: msg},
	})
}

// OKMeta writes 200 with data + arbitrary meta.
func OKMeta(c *gin.Context, data any, meta *Meta) {
	if meta != nil && meta.Timestamp == "" {
		meta.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	c.JSON(http.StatusOK, APIResponse{
		Success:    true,
		StatusCode: http.StatusOK,
		Data:       data,
		Meta:       meta,
	})
}

// List writes 200 with data + pagination meta.
func List(c *gin.Context, data any, page, limit, offset, total int) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	if page <= 0 {
		page = (offset / limit) + 1
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}
	hasPrev := offset > 0
	hasNext := offset+limit < total

	meta := &Meta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Pagination: &PaginationMeta{
			Page:       page,
			Limit:      limit,
			Offset:     offset,
			Total:      total,
			HasPrev:    hasPrev,
			HasNext:    hasNext,
			TotalPages: totalPages,
		},
	}

	c.JSON(http.StatusOK, APIResponse{
		Success:    true,
		StatusCode: http.StatusOK,
		Data:       data,
		Meta:       meta,
	})
}
