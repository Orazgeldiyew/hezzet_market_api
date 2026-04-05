package discountrule

import "time"

type DiscountRule struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	MinAmountCents  int64     `json:"min_amount_cents"`
	DiscountPercent int       `json:"discount_percent"`
	IsActive        bool      `json:"is_active"`
	CreatedBy       *int64    `json:"created_by,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name            string `json:"name" binding:"required"`
	MinAmountCents  int64  `json:"min_amount_cents" binding:"required,gt=0"`
	DiscountPercent int    `json:"discount_percent" binding:"required,gt=0,lte=100"`
}

type UpdateRequest struct {
	Name            *string `json:"name"`
	MinAmountCents  *int64  `json:"min_amount_cents"`
	DiscountPercent *int    `json:"discount_percent"`
	IsActive        *bool   `json:"is_active"`
}
