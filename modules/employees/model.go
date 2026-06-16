package employees

import "time"

// Employee is the unified actor record. Two flags split the formerly-separate
// worker/user concepts:
//
//	has_account = true  → can log into the system (was a "user")
//	is_worker   = true  → market staff with payroll / fines / debts (was a "worker")
//
// An employee can be one, the other, both, or — for a system account that
// happens to be no longer needed — neither.
type Employee struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Phone      string     `json:"phone"`
	Email      string     `json:"email"`
	Address    string     `json:"address"`
	Position   string     `json:"position"`
	Department string     `json:"department"`
	Salary     float64    `json:"salary"`
	HireDate   *time.Time `json:"hire_date,omitempty"`
	Notes      string     `json:"notes"`
	IsActive   bool       `json:"is_active"`

	HasAccount bool    `json:"has_account"`
	IsWorker   bool    `json:"is_worker"`
	Username   *string `json:"username,omitempty"`
	Role       *string `json:"role,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type CreateRequest struct {
	Name       string   `json:"name" binding:"required,min=1,max=255"`
	Phone      string   `json:"phone" binding:"omitempty,max=50"`
	Email      string   `json:"email" binding:"omitempty,email,max=255"`
	Address    string   `json:"address"`
	Position   string   `json:"position" binding:"omitempty,max=100"`
	Department string   `json:"department" binding:"omitempty,max=100"`
	Salary     *float64 `json:"salary" binding:"omitempty,gte=0"`
	HireDate   *string  `json:"hire_date"`
	Notes      string   `json:"notes"`

	HasAccount bool    `json:"has_account"`
	IsWorker   bool    `json:"is_worker"`
	Username   *string `json:"username" binding:"omitempty,min=3,max=50"`
	Password   *string `json:"password" binding:"omitempty,min=8"`
	Role       *string `json:"role"`
}

type UpdateRequest struct {
	Name       *string  `json:"name" binding:"omitempty,min=1,max=255"`
	Phone      *string  `json:"phone" binding:"omitempty,max=50"`
	Email      *string  `json:"email" binding:"omitempty,email,max=255"`
	Address    *string  `json:"address"`
	Position   *string  `json:"position" binding:"omitempty,max=100"`
	Department *string  `json:"department" binding:"omitempty,max=100"`
	Salary     *float64 `json:"salary" binding:"omitempty,gte=0"`
	HireDate   *string  `json:"hire_date"`
	Notes      *string  `json:"notes"`
	IsActive   *bool    `json:"is_active"`

	IsWorker *bool   `json:"is_worker"`
	Username *string `json:"username" binding:"omitempty,min=3,max=50"`
	Role     *string `json:"role"`
}

type ListFilter struct {
	Search     string
	IsWorker   *bool
	HasAccount *bool
	ActiveOnly bool
}

type ListResponse struct {
	Items  []Employee `json:"items"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}
