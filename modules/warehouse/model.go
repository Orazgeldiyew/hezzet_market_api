package warehouse

import "time"

type Warehouse struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Address   string     `json:"address"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type CreateRequest struct {
	Name    string `json:"name" binding:"required,min=1,max=255"`
	Address string `json:"address"`
}

type UpdateRequest struct {
	Name     *string `json:"name" binding:"omitempty,min=1,max=255"`
	Address  *string `json:"address"`
	IsActive *bool   `json:"is_active"`
}

type ListResponse struct {
	Items  []Warehouse `json:"items"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}
