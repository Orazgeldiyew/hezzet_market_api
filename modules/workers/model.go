package workers

import "time"

type Worker struct {
	ID         int64      `json:"id"`
	UserID     *int64     `json:"user_id,omitempty"`
	Name       string     `json:"name"`
	Position   string     `json:"position"`
	Department string     `json:"department"`
	Phone      string     `json:"phone"`
	Email      string     `json:"email"`
	Address    string     `json:"address"`
	Salary     float64    `json:"salary"`
	HireDate   *time.Time `json:"hire_date,omitempty"`
	IsActive   bool       `json:"is_active"`
	Notes      string     `json:"notes"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

type CreateRequest struct {
	UserID     *int64   `json:"user_id"`
	Name       string   `json:"name" binding:"required,min=1,max=255"`
	Position   string   `json:"position" binding:"omitempty,max=100"`
	Department string   `json:"department" binding:"omitempty,max=100"`
	Phone      string   `json:"phone" binding:"omitempty,max=50"`
	Email      string   `json:"email" binding:"omitempty,email,max=255"`
	Address    string   `json:"address"`
	Salary     *float64 `json:"salary" binding:"omitempty,gte=0"`
	HireDate   *string  `json:"hire_date"`
	Notes      string   `json:"notes"`
}

type UpdateRequest struct {
	Name       *string  `json:"name" binding:"omitempty,min=1,max=255"`
	Position   *string  `json:"position" binding:"omitempty,max=100"`
	Department *string  `json:"department" binding:"omitempty,max=100"`
	Phone      *string  `json:"phone" binding:"omitempty,max=50"`
	Email      *string  `json:"email" binding:"omitempty,email,max=255"`
	Address    *string  `json:"address"`
	Salary     *float64 `json:"salary" binding:"omitempty,gte=0"`
	HireDate   *string  `json:"hire_date"`
	IsActive   *bool    `json:"is_active"`
	Notes      *string  `json:"notes"`
}

type ListResponse struct {
	Items  []Worker `json:"items"`
	Total  int      `json:"total"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
}
