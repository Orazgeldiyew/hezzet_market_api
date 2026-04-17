package printer

import "time"

type Printer struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	IPAddress  string    `json:"ip_address"`
	Port       int       `json:"port"`
	RegisterID *int64    `json:"register_id,omitempty"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name       string `json:"name" binding:"required,min=1,max=100"`
	IPAddress  string `json:"ip_address" binding:"required,ip"`
	Port       int    `json:"port"`
	RegisterID *int64 `json:"register_id"`
	IsActive   *bool  `json:"is_active"`
}

type UpdateRequest struct {
	Name       *string `json:"name"`
	IPAddress  *string `json:"ip_address" binding:"omitempty,ip"`
	Port       *int    `json:"port"`
	RegisterID *int64  `json:"register_id"`
	IsActive   *bool   `json:"is_active"`
}
