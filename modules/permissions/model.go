package permissions

import "time"

// ── Role ──

type Role struct {
	ID          int          `json:"id"`
	Code        string       `json:"code"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsSystem    bool         `json:"is_system"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type CreateRoleRequest struct {
	Code        string `json:"code"        binding:"required,min=2,max=50"`
	Name        string `json:"name"        binding:"required,min=1,max=100"`
	Description string `json:"description"`
}

type UpdateRoleRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// ── Permission ──

type Permission struct {
	ID        int64     `json:"id"`
	RoleID    int       `json:"role_id"`
	RoleCode  string    `json:"role_code"`
	Module    string    `json:"module"`
	Action    string    `json:"action"`
	Granted   bool      `json:"granted"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdatePermissionRequest struct {
	Granted bool `json:"granted"`
}

type BulkPermissionItem struct {
	Module  string `json:"module"  binding:"required"`
	Action  string `json:"action"  binding:"required"`
	Granted bool   `json:"granted"`
}

type BulkPermissionRequest struct {
	Permissions []BulkPermissionItem `json:"permissions" binding:"required,min=1,dive"`
}

// ── Matrix (for GET /permissions/matrix) ──

type MatrixEntry struct {
	RoleID   int    `json:"role_id"`
	RoleCode string `json:"role_code"`
	RoleName string `json:"role_name"`
	Module   string `json:"module"`
	View     bool   `json:"view"`
	Create   bool   `json:"create"`
	Update   bool   `json:"update"`
	Delete   bool   `json:"delete"`
}
