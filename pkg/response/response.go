package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Success    bool        `json:"success"`
	StatusCode int         `json:"status_code"`
	Data       interface{} `json:"data,omitempty"`
	Error      *APIError   `json:"error,omitempty"`
	Meta       *Meta       `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Meta struct {
	// Useful for frontend debugging / UX
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

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Success:    true,
		StatusCode: http.StatusOK,
		Data:       data,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Success:    true,
		StatusCode: http.StatusCreated,
		Data:       data,
	})
}

func Error(c *gin.Context, status int, code, msg string) {
	c.JSON(status, APIResponse{
		Success:    false,
		StatusCode: status,
		Error:      &APIError{Code: code, Message: msg},
	})
}

// OKMeta returns 200 with data + meta
func OKMeta(c *gin.Context, data interface{}, meta *Meta) {
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

// List is a standard helper for list endpoints: 200 with data + pagination meta
func List(c *gin.Context, data interface{}, page, limit, offset, total int) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	if page <= 0 {
		// fallback derived from offset/limit
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
