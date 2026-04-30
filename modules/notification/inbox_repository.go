package notification

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

var errRepoNotConfigured = errors.New("inbox repository not configured")

func sortByCreatedAtDesc(items []InboxItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}

func paginate(items []InboxItem, limit, offset int) []InboxItem {
	if offset >= len(items) {
		return []InboxItem{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

// InboxItem is an in-app notification shown in the bell dropdown. It projects
// an sms_logs row plus the current user's read state.
type InboxItem struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
	IsRead    bool      `json:"is_read"`
}

// InboxResult is the paginated list returned to the UI.
type InboxResult struct {
	Items       []InboxItem `json:"items"`
	Total       int         `json:"total"`
	UnreadCount int         `json:"unread_count"`
	Limit       int         `json:"limit"`
	Offset      int         `json:"offset"`
}

// InboxRepository reads/writes the per-user inbox state on top of sms_logs.
type InboxRepository struct {
	db *pgxpool.Pool
}

func NewInboxRepository(db *pgxpool.Pool) *InboxRepository {
	if db == nil {
		return nil
	}
	return &InboxRepository{db: db}
}

// inboxTypes is the whitelist of sms_logs.type values that surface in the UI.
// Transactional customer-facing SMS (order confirmations, etc.) stay out — the
// bell is for internal staff alerts only.
var inboxTypes = []string{
	"admin_low_stock",
	"admin_reorder_digest",
}

// List returns the inbox page for userID, newest first, with read flags.
// Dedupes by dedup_key so the same low-stock alert sent to N admin phones
// shows up once per incident rather than N times.
func (r *InboxRepository) List(ctx context.Context, userID int64, limit, offset int) (*InboxResult, error) {
	if r == nil {
		return nil, apperr.Internal(errRepoNotConfigured)
	}

	total, unread, err := r.counts(ctx, userID)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	// DISTINCT ON (dedup_key, type) collapses fan-out to multiple phones.
	// Logs without dedup_key are treated as unique via COALESCE(dedup_key, job_id).
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT ON (COALESCE(NULLIF(l.dedup_key, ''), l.job_id))
		       l.id, l.type, l.message, l.created_at,
		       (nr.user_id IS NOT NULL) AS is_read
		FROM sms_logs l
		LEFT JOIN notification_reads nr
		       ON nr.sms_log_id = l.id AND nr.user_id = $1
		WHERE l.type = ANY($2)
		ORDER BY COALESCE(NULLIF(l.dedup_key, ''), l.job_id), l.created_at DESC
	`, userID, inboxTypes)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()

	var all []InboxItem
	for rows.Next() {
		var it InboxItem
		if err := rows.Scan(&it.ID, &it.Type, &it.Message, &it.CreatedAt, &it.IsRead); err != nil {
			return nil, apperr.Internal(err)
		}
		all = append(all, it)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err)
	}

	// Sort deduped results newest-first and slice the requested page in Go —
	// simpler than a second window query and the deduped set is small.
	sortByCreatedAtDesc(all)
	items := paginate(all, limit, offset)

	return &InboxResult{
		Items:       items,
		Total:       total,
		UnreadCount: unread,
		Limit:       limit,
		Offset:      offset,
	}, nil
}

// counts returns (total, unread) after dedup by dedup_key/job_id.
func (r *InboxRepository) counts(ctx context.Context, userID int64) (int, int, error) {
	var total, unread int
	err := r.db.QueryRow(ctx, `
		WITH deduped AS (
		    SELECT DISTINCT ON (COALESCE(NULLIF(l.dedup_key, ''), l.job_id))
		           l.id,
		           (nr.user_id IS NOT NULL) AS is_read
		    FROM sms_logs l
		    LEFT JOIN notification_reads nr
		           ON nr.sms_log_id = l.id AND nr.user_id = $1
		    WHERE l.type = ANY($2)
		    ORDER BY COALESCE(NULLIF(l.dedup_key, ''), l.job_id), l.created_at DESC
		)
		SELECT COUNT(*), COUNT(*) FILTER (WHERE NOT is_read) FROM deduped
	`, userID, inboxTypes).Scan(&total, &unread)
	return total, unread, err
}

// UnreadCount returns only the unread count — cheap endpoint for badge polling.
func (r *InboxRepository) UnreadCount(ctx context.Context, userID int64) (int, error) {
	if r == nil {
		return 0, apperr.Internal(errRepoNotConfigured)
	}
	_, unread, err := r.counts(ctx, userID)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return unread, nil
}

// MarkRead records that userID has seen sms_log.id. Idempotent via PK conflict.
func (r *InboxRepository) MarkRead(ctx context.Context, userID, logID int64) error {
	if r == nil {
		return apperr.Internal(errRepoNotConfigured)
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO notification_reads (user_id, sms_log_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, sms_log_id) DO NOTHING
	`, userID, logID)
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// MarkAllRead inserts reads for every currently-unread inbox item. Returns the
// number of rows inserted.
func (r *InboxRepository) MarkAllRead(ctx context.Context, userID int64) (int, error) {
	if r == nil {
		return 0, apperr.Internal(errRepoNotConfigured)
	}
	ct, err := r.db.Exec(ctx, `
		INSERT INTO notification_reads (user_id, sms_log_id)
		SELECT $1, l.id FROM sms_logs l
		WHERE l.type = ANY($2)
		ON CONFLICT (user_id, sms_log_id) DO NOTHING
	`, userID, inboxTypes)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return int(ct.RowsAffected()), nil
}
