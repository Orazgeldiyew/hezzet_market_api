package supplier

import "time"

// Supplier represents a product supplier
type Supplier struct {
	ID        int       `json:"id"`
	UserID    *int64    `json:"user_id,omitempty"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Address   string    `json:"address"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateRequest for creating new supplier
type CreateRequest struct {
	UserID  *int64 `json:"user_id"`
	Name    string `json:"name" binding:"required,min=1,max=255"`
	Phone   string `json:"phone" binding:"max=50"`
	Email   string `json:"email" binding:"omitempty,email,max=255"`
	Address string `json:"address"`
}

// UpdateRequest for updating supplier
type UpdateRequest struct {
	Name     *string `json:"name" binding:"omitempty,min=1,max=255"`
	Phone    *string `json:"phone" binding:"omitempty,max=50"`
	Email    *string `json:"email" binding:"omitempty,email,max=255"`
	Address  *string `json:"address"`
	IsActive *bool   `json:"is_active"`
}

// ListResponse for paginated supplier list
type ListResponse struct {
	Items  []Supplier `json:"items"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}
