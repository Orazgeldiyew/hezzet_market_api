package auditlog

import (
	"encoding/json"
	"time"
)

// Snapshot is the JSONB before/after state. nil = entity did not exist
// (create has Old=nil, delete has New=nil).
type Snapshot map[string]any

type AuditLog struct {
	ID         int64           `json:"id"`
	UserID     int64           `json:"user_id"`
	Username   string          `json:"username"`
	Action     string          `json:"action"`
	EntityType string          `json:"entity_type"`
	EntityID   *string         `json:"entity_id,omitempty"`
	Method     string          `json:"method"`
	Path       string          `json:"path"`
	IPAddress  *string         `json:"ip_address,omitempty"`
	RequestID  *string         `json:"request_id,omitempty"`
	OldValue   json.RawMessage `json:"old_value,omitempty"`
	NewValue   json.RawMessage `json:"new_value,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

type AuditFilter struct {
	UserID     *int64
	EntityType string
	Action     string
	From       *time.Time
	To         *time.Time
}

type AuditListResult struct {
	Items  []AuditLog `json:"items"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}
