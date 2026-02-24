package category

import (
	"time"
)

// Category represents a product category with hierarchy support
type Category struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	ParentID  *int      `json:"parent_id"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// CategoryBrief is a lightweight category representation
type CategoryBrief struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// CategoryResponse is the enriched response DTO with parent_name resolved via JOIN
type CategoryResponse struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	ParentID   *int      `json:"parent_id"`
	ParentName *string   `json:"parent_name"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// CategoryTree represents category with children
type CategoryTree struct {
	ID       int             `json:"id"`
	Name     string          `json:"name"`
	ParentID *int            `json:"parent_id,omitempty"`
	Children []*CategoryTree `json:"children,omitempty"`
}

// CreateRequest for creating new category
type CreateRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=255"`
	ParentID *int   `json:"parent_id"`
	IsActive *bool  `json:"is_active"`
}

// UpdateRequest for updating category
type UpdateRequest struct {
	Name     *string `json:"name" binding:"omitempty,min=1,max=255"`
	ParentID *int    `json:"parent_id"`
	IsActive *bool   `json:"is_active"`
}

// ListResponse for paginated category list
type ListResponse struct {
	Items  []CategoryResponse `json:"items"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

// TreeResponse for category tree
type TreeResponse struct {
	Items []*CategoryTree `json:"items"`
}
