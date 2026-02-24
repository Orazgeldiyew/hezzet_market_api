package notification

import "time"

// SMS log status constants.
const (
	StatusQueued      = "queued"
	StatusSending     = "sending"
	StatusSent        = "sent"
	StatusRetrying    = "retrying"
	StatusRateLimited = "rate_limited"
	StatusDLQ         = "dlq"
	StatusFailed      = "failed"
)

// SMSLog represents a row in the sms_logs table.
type SMSLog struct {
	ID          int64      `json:"id"`
	JobID       string     `json:"job_id"`
	Type        string     `json:"type"`
	ToPhone     string     `json:"to_phone"`
	Message     string     `json:"message"`
	SMSFrom     string     `json:"sms_from"`
	Status      string     `json:"status"`
	Attempt     int        `json:"attempt"`
	MaxAttempts int        `json:"max_attempts"`
	DedupKey    string     `json:"dedup_key,omitempty"`
	Provider    string     `json:"provider"`
	LastError   *string    `json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
}

// SMSLogFilter is used for listing/searching sms_logs.
type SMSLogFilter struct {
	Status   string
	Type     string
	Phone    string
	JobID    string
	FromDate *time.Time
	ToDate   *time.Time
}

// SMSLogListResult is returned by LogRepository.List.
type SMSLogListResult struct {
	Items  []SMSLog `json:"items"`
	Total  int      `json:"total"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
}
