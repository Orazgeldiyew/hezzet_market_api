package permissions

import "time"

type Permission struct {
	ID        int64     `json:"id"`
	Role      string    `json:"role"`
	Module    string    `json:"module"`
	Enabled   bool      `json:"enabled"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdatePermissionRequest struct {
	Enabled bool `json:"enabled"`
}
