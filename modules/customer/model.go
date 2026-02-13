// modules/customer/model.go
package customer

import "time"

type Customer struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Phone       string     `json:"phone"`
	Email       string     `json:"email"`
	Type        string     `json:"type"`
	TotalSpent  float64    `json:"total_spent"`
	BonusPoints float64    `json:"bonus_points"`
	IsActive    bool       `json:"is_active"`
	Notes       string     `json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type CreateRequest struct {
	Name  string  `json:"name" binding:"required,min=1,max=255"`
	Phone string  `json:"phone" binding:"omitempty,max=50"`
	Email string  `json:"email" binding:"omitempty,email,max=255"`
	Type  *string `json:"type" binding:"omitempty,oneof=regular wholesale"`
	Notes string  `json:"notes"`
}

type UpdateRequest struct {
	Name     *string `json:"name" binding:"omitempty,min=1,max=255"`
	Phone    *string `json:"phone" binding:"omitempty,max=50"`
	Email    *string `json:"email" binding:"omitempty,email,max=255"`
	Type     *string `json:"type" binding:"omitempty,oneof=regular wholesale"`
	IsActive *bool   `json:"is_active"`
	Notes    *string `json:"notes"`
}

type ListResponse struct {
	Items  []Customer `json:"items"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}
type UpdateContactRequest struct {
	Name  *string `json:"name" binding:"omitempty,min=1,max=255"`
	Phone *string `json:"phone" binding:"omitempty,max=50"`
	Email *string `json:"email" binding:"omitempty,email,max=255"`
	Notes *string `json:"notes"`
}
