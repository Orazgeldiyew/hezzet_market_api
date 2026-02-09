package client

import "time"

// Client represents a customer
type Client struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateRequest for creating new client
type CreateRequest struct {
	Name  string `json:"name" binding:"required,min=1,max=255"`
	Phone string `json:"phone" binding:"omitempty,max=50"`
	Email string `json:"email" binding:"omitempty,email,max=255"`
}

// UpdateRequest for updating client
type UpdateRequest struct {
	Name     *string `json:"name" binding:"omitempty,min=1,max=255"`
	Phone    *string `json:"phone" binding:"omitempty,max=50"`
	Email    *string `json:"email" binding:"omitempty,email,max=255"`
	IsActive *bool   `json:"is_active"`
}

// ListResponse for paginated client list
type ListResponse struct {
	Items  []Client `json:"items"`
	Total  int      `json:"total"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
}
