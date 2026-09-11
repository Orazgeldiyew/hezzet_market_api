package printer

import "time"

const (
	ConnectionNetwork = "network"
	ConnectionUSB     = "usb"
)

type Printer struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	ConnectionType string    `json:"connection_type"` // network | usb
	IPAddress      string    `json:"ip_address,omitempty"`
	Port           int       `json:"port"`
	RegisterID     *int64    `json:"register_id,omitempty"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name           string `json:"name" binding:"required,min=1,max=100"`
	ConnectionType string `json:"connection_type"` // network (default) | usb
	IPAddress      string `json:"ip_address" binding:"omitempty,ip"`
	Port           int    `json:"port"`
	RegisterID     *int64 `json:"register_id"`
	IsActive       *bool  `json:"is_active"`
}

type UpdateRequest struct {
	Name           *string `json:"name"`
	ConnectionType *string `json:"connection_type"`
	IPAddress      *string `json:"ip_address" binding:"omitempty,ip"`
	Port           *int    `json:"port"`
	RegisterID     *int64  `json:"register_id"`
	IsActive       *bool   `json:"is_active"`
}
