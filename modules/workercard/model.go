package workercard

import "time"

type WorkerCard struct {
	ID        int64     `json:"id"`
	WorkerID  int64     `json:"worker_id"`
	CardCode  string    `json:"card_code"`
	Label     string    `json:"label"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateCardRequest struct {
	WorkerID int64   `json:"worker_id" binding:"required,gt=0"`
	Label    string  `json:"label"`
	CardCode *string `json:"card_code"`
}

type UpdateCardRequest struct {
	Label   *string `json:"label"`
	Enabled *bool   `json:"enabled"`
}
