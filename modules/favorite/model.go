package favorite

import "time"

type FavoriteProduct struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	ProductID  int64     `json:"product_id"`
	Position   int       `json:"position"`
	CreatedAt  time.Time `json:"created_at"`
	Name       string    `json:"name"`
	PriceCents int64     `json:"price_cents"`
	PhotoURL   *string   `json:"photo_url"`
}

type AddRequest struct {
	ProductID int64 `json:"product_id" binding:"required,gt=0"`
}

type ReorderItem struct {
	ProductID int64 `json:"product_id" binding:"required,gt=0"`
	Position  int   `json:"position"   binding:"min=0"`
}

type ReorderRequest struct {
	Items []ReorderItem `json:"items" binding:"required,min=1,dive"`
}
