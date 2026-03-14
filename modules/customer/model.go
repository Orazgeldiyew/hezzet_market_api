// modules/customer/model.go
package customer

import "time"

type Customer struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Phone       string     `json:"phone"`
	Email       string     `json:"email"`
	Type        string     `json:"type"`
	TotalSpent  int64      `json:"total_spent"`
	BonusPoints int64      `json:"bonus_points"`
	CardCode    string     `json:"card_code"`
	IsActive    bool       `json:"is_active"`
	Notes       string     `json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// ---------- Requests ----------

type CreateRequest struct {
	Name  string  `json:"name" binding:"required,min=1,max=255"`
	Phone string  `json:"phone" binding:"omitempty,max=50"`
	Email string  `json:"email" binding:"omitempty,email,max=255"`
	Type  *string `json:"type" binding:"omitempty,oneof=regular vip wholesale"`
	Notes string  `json:"notes"`
}

// UpdateRequest используется в handler Update (если ты его оставляешь).
// Если ты хочешь полностью убрать общий Update — тогда нужно удалить h.Update и repo.Update/service.Update.
// Пока оставляю совместимость с твоим текущим кодом.
type UpdateRequest struct {
	Name  *string `json:"name" binding:"omitempty,min=1,max=255"`
	Phone *string `json:"phone" binding:"omitempty,max=50"`
	Email *string `json:"email" binding:"omitempty,email,max=255"`
	Type  *string `json:"type" binding:"omitempty,oneof=regular vip wholesale"`

	IsActive *bool   `json:"is_active"`
	Notes    *string `json:"notes"`
}

// Только контактные поля (cashier/operator/manager)
type UpdateContactRequest struct {
	Name  *string `json:"name" binding:"omitempty,min=1,max=255"`
	Phone *string `json:"phone" binding:"omitempty,max=50"`
	Email *string `json:"email" binding:"omitempty,email,max=255"`
	Notes *string `json:"notes"`
}

// Только бизнес-поля (manager/admin)
type UpdateAdminRequest struct {
	Type     *string `json:"type" binding:"omitempty"`
	IsActive *bool   `json:"is_active"`
}

// ---------- Responses ----------

type ListResponse struct {
	Items  []Customer `json:"items"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}
