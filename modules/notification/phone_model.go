package notification

import "time"

type NotificationPhone struct {
	ID        int64     `json:"id"`
	Phone     string    `json:"phone"`
	Label     string    `json:"label"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type AddPhoneRequest struct {
	Phone string `json:"phone" binding:"required"`
	Label string `json:"label"`
}

type UpdatePhoneRequest struct {
	Label   *string `json:"label"`
	Enabled *bool   `json:"enabled"`
}
