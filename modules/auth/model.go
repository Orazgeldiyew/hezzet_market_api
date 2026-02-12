// modules/auth/model.go
package auth

import "time"

type User struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	FullName     string     `json:"full_name"`
	Phone        string     `json:"phone"`
	Email        string     `json:"email"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// ---------- Requests ----------

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CreateUserRequest struct {
	Username string   `json:"username" binding:"required,min=3,max=50"`
	Password string   `json:"password" binding:"required,min=6"`
	FullName string   `json:"full_name"`
	Phone    string   `json:"phone"`
	Email    string   `json:"email"`
	IsActive *bool    `json:"is_active"`
	Roles    []string `json:"roles" binding:"required,min=1"`
}

type UpdateUserRequest struct {
	Username *string  `json:"username" binding:"omitempty,min=3,max=50"`
	FullName *string  `json:"full_name"`
	Phone    *string  `json:"phone"`
	Email    *string  `json:"email"`
	IsActive *bool    `json:"is_active"`
	Roles    []string `json:"roles"`
}

type ChangePasswordRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}

// ---------- Responses ----------

type LoginResponse struct {
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresIn   int64    `json:"expires_in"`
	User        UserDTO  `json:"user"`
	Roles       []string `json:"roles"`
}

type UserDTO struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	FullName  string    `json:"full_name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
		ID:        u.ID,
		Username:  u.Username,
		FullName:  u.FullName,
		Phone:     u.Phone,
		Email:     u.Email,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
