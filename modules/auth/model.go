// modules/auth/model.go
package auth

import "time"

// ---------- Domain ----------

type User struct {
	ID                int64      `json:"id"`
	Username          string     `json:"username"`
	PasswordHash      string     `json:"-"`
	FullName          string     `json:"full_name"`
	Phone             string     `json:"phone"`
	Email             string     `json:"email"`
	IsActive          bool       `json:"is_active"`
	BlockedAt         *time.Time `json:"blocked_at,omitempty"`
	BlockedReason     *string    `json:"blocked_reason,omitempty"`
	PasswordChangedAt *time.Time `json:"password_changed_at,omitempty"`
	TokenVersion      int        `json:"-"`
	LastLoginAt       *time.Time `json:"last_login_at,omitempty"`
	CreatedBy         *int64     `json:"created_by,omitempty"`
	UpdatedBy         *int64     `json:"updated_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
}

// ---------- Requests ----------

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
}

type CreateUserRequest struct {
	Username string   `json:"username" binding:"required,min=3,max=50"`
	Password string   `json:"password" binding:"required,min=6"`
	FullName string   `json:"full_name"`
	Phone    string   `json:"phone"`
	Email    string   `json:"email"`
	IsActive *bool    `json:"is_active"`
	Roles    []string `json:"roles" binding:"required,min=1,dive,required" enums:"admin,cashier,operator,manager"`
}

type UpdateUserRequest struct {
	Username *string   `json:"username" binding:"omitempty,min=3,max=50"`
	FullName *string   `json:"full_name"`
	Phone    *string   `json:"phone"`
	Email    *string   `json:"email"`
	IsActive *bool     `json:"is_active"`
	Roles    *[]string `json:"roles"`
}

type ChangePasswordRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}

type BlockUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ---------- JWT ----------

type JWTPayload struct {
	UserID       int64    `json:"user_id"`
	Username     string   `json:"username"`
	Role         []string `json:"role"`
	TokenVersion int      `json:"token_version"`
}

// ---------- Responses ----------

type LoginResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	TokenType    string   `json:"token_type"`
	ExpiresIn    int64    `json:"expires_in"`
	User         UserDTO  `json:"user"`
	Roles        []string `json:"roles"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type UserDTO struct {
	ID                int64      `json:"id"`
	Username          string     `json:"username"`
	FullName          string     `json:"full_name"`
	Phone             string     `json:"phone"`
	Email             string     `json:"email"`
	IsActive          bool       `json:"is_active"`
	BlockedAt         *time.Time `json:"blocked_at,omitempty"`
	BlockedReason     *string    `json:"blocked_reason,omitempty"`
	PasswordChangedAt *time.Time `json:"password_changed_at,omitempty"`
	LastLoginAt       *time.Time `json:"last_login_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type UserWithRoles struct {
	UserDTO
	Roles []string `json:"roles"`
}

type ListResponse struct {
	Items  []UserWithRoles `json:"items"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

func toDTO(u User) UserDTO {
	return UserDTO{
		ID:                u.ID,
		Username:          u.Username,
		FullName:          u.FullName,
		Phone:             u.Phone,
		Email:             u.Email,
		IsActive:          u.IsActive,
		BlockedAt:         u.BlockedAt,
		BlockedReason:     u.BlockedReason,
		PasswordChangedAt: u.PasswordChangedAt,
		LastLoginAt:       u.LastLoginAt,
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
	}
}
