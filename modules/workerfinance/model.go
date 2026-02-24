package workerfinance

import "time"

// ---------- Domain ----------

type WorkerCompensation struct {
	ID              int64     `json:"id"`
	WorkerID        int64     `json:"worker_id"`
	BaseSalaryCents int64     `json:"base_salary_cents"`
	PayDay          int       `json:"pay_day"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type WorkerFine struct {
	ID          int64     `json:"id"`
	WorkerID    int64     `json:"worker_id"`
	AmountCents int64     `json:"amount_cents"`
	Reason      string    `json:"reason"`
	Status      string    `json:"status"`
	OccurredAt  time.Time `json:"occurred_at"`
	CreatedBy   *int64    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type WorkerDebt struct {
	ID             int64     `json:"id"`
	WorkerID       int64     `json:"worker_id"`
	AmountCents    int64     `json:"amount_cents"`
	RemainingCents int64     `json:"remaining_cents"`
	Type           string    `json:"type"`
	Status         string    `json:"status"`
	Note           *string   `json:"note,omitempty"`
	CreatedBy      *int64    `json:"created_by,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ---------- Requests ----------

type SetCompensationRequest struct {
	WorkerID        int64 `json:"worker_id" binding:"required,gt=0"`
	BaseSalaryCents int64 `json:"base_salary_cents" binding:"required,gt=0"`
	PayDay          int   `json:"pay_day" binding:"required,min=1,max=31"`
}

type CreateFineRequest struct {
	AmountCents int64  `json:"amount_cents" binding:"required,gt=0"`
	Reason      string `json:"reason" binding:"required,min=1"`
}

type CreateDebtRequest struct {
	AmountCents int64   `json:"amount_cents" binding:"required,gt=0"`
	Type        string  `json:"type" binding:"required,oneof=advance loan"`
	Note        *string `json:"note"`
}

// ---------- Responses ----------

type FineListResult struct {
	Items  []WorkerFine `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type DebtListResult struct {
	Items  []WorkerDebt `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}
