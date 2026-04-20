package notification

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Settings is the singleton runtime configuration for notifications.
// Admins update it from the UI; workers read it on each tick.
type Settings struct {
	ReorderDigestHour    int  `json:"reorder_digest_hour"`
	ReorderDigestEnabled bool `json:"reorder_digest_enabled"`
}

// UpdateSettingsRequest is the payload for PUT /api/notifications/settings.
type UpdateSettingsRequest struct {
	ReorderDigestHour    *int  `json:"reorder_digest_hour"    binding:"omitempty,min=0,max=23"`
	ReorderDigestEnabled *bool `json:"reorder_digest_enabled"`
}

// SettingsRepository persists notification_settings.
type SettingsRepository struct {
	db *pgxpool.Pool
}

func NewSettingsRepository(db *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get returns the singleton row, creating it on-the-fly if missing.
func (r *SettingsRepository) Get(ctx context.Context) (Settings, error) {
	var s Settings
	err := r.db.QueryRow(ctx, `
		SELECT reorder_digest_hour, reorder_digest_enabled
		FROM notification_settings WHERE id = 1
	`).Scan(&s.ReorderDigestHour, &s.ReorderDigestEnabled)
	if errors.Is(err, pgx.ErrNoRows) {
		// Bootstrap row on first read.
		_, insErr := r.db.Exec(ctx, `INSERT INTO notification_settings (id) VALUES (1)
			ON CONFLICT (id) DO NOTHING`)
		if insErr != nil {
			return s, insErr
		}
		s.ReorderDigestHour = 9
		s.ReorderDigestEnabled = true
		return s, nil
	}
	return s, err
}

// Update applies only non-nil fields. Returns the updated row.
func (r *SettingsRepository) Update(ctx context.Context, req UpdateSettingsRequest) (Settings, error) {
	var s Settings
	err := r.db.QueryRow(ctx, `
		UPDATE notification_settings SET
		    reorder_digest_hour    = COALESCE($1, reorder_digest_hour),
		    reorder_digest_enabled = COALESCE($2, reorder_digest_enabled),
		    updated_at             = now()
		WHERE id = 1
		RETURNING reorder_digest_hour, reorder_digest_enabled
	`, req.ReorderDigestHour, req.ReorderDigestEnabled,
	).Scan(&s.ReorderDigestHour, &s.ReorderDigestEnabled)
	return s, err
}
