package notification

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

// LogRepository persists SMS job lifecycle events into PostgreSQL.
// All methods are nil-safe: calling any method on a nil receiver is a no-op.
type LogRepository struct {
	db *pgxpool.Pool
}

// NewLogRepository creates a new LogRepository.
// Returns nil if db is nil (allows graceful degradation).
func NewLogRepository(db *pgxpool.Pool) *LogRepository {
	if db == nil {
		return nil
	}
	return &LogRepository{db: db}
}

// columns shared by all scan operations.
const smsLogColumns = `id, job_id, type, to_phone, message, sms_from, status,
	attempt, max_attempts, dedup_key, provider, last_error,
	created_at, updated_at, sent_at`

func scanSMSLog(row pgx.Row) (SMSLog, error) {
	var l SMSLog
	err := row.Scan(
		&l.ID, &l.JobID, &l.Type, &l.ToPhone, &l.Message, &l.SMSFrom,
		&l.Status, &l.Attempt, &l.MaxAttempts, &l.DedupKey, &l.Provider,
		&l.LastError, &l.CreatedAt, &l.UpdatedAt, &l.SentAt,
	)
	return l, err
}

// UpsertQueued inserts a new sms_logs row with status=queued or updates
// an existing row (idempotent on job_id conflict).
func (r *LogRepository) UpsertQueued(ctx context.Context, job SMSJob, provider, smsFrom string) {
	if r == nil {
		return
	}
	const q = `
		INSERT INTO sms_logs (job_id, type, to_phone, message, sms_from, status, attempt, max_attempts, dedup_key, provider)
		VALUES ($1, $2, $3, $4, $5, 'queued', $6, $7, $8, $9)
		ON CONFLICT (job_id) DO UPDATE SET
			status     = 'queued',
			attempt    = EXCLUDED.attempt,
			updated_at = now()`
	_, err := r.db.Exec(ctx, q,
		job.JobID, job.Type, job.ToPhone, job.Message, smsFrom,
		job.Attempt, job.MaxAttempts, job.DedupKey, provider,
	)
	if err != nil {
		logRepoError("UpsertQueued", job.JobID, err)
	}
}

// MarkSending sets status=sending.
func (r *LogRepository) MarkSending(ctx context.Context, jobID string) {
	if r == nil {
		return
	}
	const q = `UPDATE sms_logs SET status = 'sending' WHERE job_id = $1`
	_, err := r.db.Exec(ctx, q, jobID)
	if err != nil {
		logRepoError("MarkSending", jobID, err)
	}
}

// MarkSent sets status=sent, records sent_at, clears last_error.
func (r *LogRepository) MarkSent(ctx context.Context, jobID string) {
	if r == nil {
		return
	}
	const q = `UPDATE sms_logs SET status = 'sent', sent_at = now(), last_error = NULL WHERE job_id = $1`
	_, err := r.db.Exec(ctx, q, jobID)
	if err != nil {
		logRepoError("MarkSent", jobID, err)
	}
}

// MarkRetrying sets status=retrying with the current attempt count and error.
func (r *LogRepository) MarkRetrying(ctx context.Context, jobID string, attempt int, lastError string) {
	if r == nil {
		return
	}
	const q = `UPDATE sms_logs SET status = 'retrying', attempt = $2, last_error = $3 WHERE job_id = $1`
	_, err := r.db.Exec(ctx, q, jobID, attempt, lastError)
	if err != nil {
		logRepoError("MarkRetrying", jobID, err)
	}
}

// MarkRateLimited sets status=rate_limited.
func (r *LogRepository) MarkRateLimited(ctx context.Context, jobID string) {
	if r == nil {
		return
	}
	const q = `UPDATE sms_logs SET status = 'rate_limited', last_error = 'rate_limited' WHERE job_id = $1`
	_, err := r.db.Exec(ctx, q, jobID)
	if err != nil {
		logRepoError("MarkRateLimited", jobID, err)
	}
}

// MarkDLQ sets status=dlq.
func (r *LogRepository) MarkDLQ(ctx context.Context, jobID string, lastError string) {
	if r == nil {
		return
	}
	const q = `UPDATE sms_logs SET status = 'dlq', last_error = $2 WHERE job_id = $1`
	_, err := r.db.Exec(ctx, q, jobID, lastError)
	if err != nil {
		logRepoError("MarkDLQ", jobID, err)
	}
}

// MarkFailed sets status=failed (terminal, non-retryable).
func (r *LogRepository) MarkFailed(ctx context.Context, jobID string, lastError string) {
	if r == nil {
		return
	}
	const q = `UPDATE sms_logs SET status = 'failed', last_error = $2 WHERE job_id = $1`
	_, err := r.db.Exec(ctx, q, jobID, lastError)
	if err != nil {
		logRepoError("MarkFailed", jobID, err)
	}
}

// InsertFailed inserts a new row with status=failed (when enqueue itself fails
// and no row exists yet). Idempotent on job_id conflict.
func (r *LogRepository) InsertFailed(ctx context.Context, job SMSJob, provider, smsFrom, lastError string) {
	if r == nil {
		return
	}
	const q = `
		INSERT INTO sms_logs (job_id, type, to_phone, message, sms_from, status, attempt, max_attempts, dedup_key, provider, last_error)
		VALUES ($1, $2, $3, $4, $5, 'failed', $6, $7, $8, $9, $10)
		ON CONFLICT (job_id) DO UPDATE SET
			status     = 'failed',
			last_error = EXCLUDED.last_error,
			updated_at = now()`
	_, err := r.db.Exec(ctx, q,
		job.JobID, job.Type, job.ToPhone, job.Message, smsFrom,
		job.Attempt, job.MaxAttempts, job.DedupKey, provider, lastError,
	)
	if err != nil {
		logRepoError("InsertFailed", job.JobID, err)
	}
}

// GetByJobID returns a single SMS log by job_id.
func (r *LogRepository) GetByJobID(ctx context.Context, jobID string) (*SMSLog, error) {
	if r == nil {
		return nil, apperr.Internal(fmt.Errorf("log repository not configured"))
	}
	q := `SELECT ` + smsLogColumns + ` FROM sms_logs WHERE job_id = $1`
	l, err := scanSMSLog(r.db.QueryRow(ctx, q, jobID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperr.NotFound("SMS_LOG_NOT_FOUND", "sms log not found")
		}
		return nil, apperr.Internal(err)
	}
	return &l, nil
}

// List returns paginated sms_logs with optional filters.
func (r *LogRepository) List(ctx context.Context, f SMSLogFilter, limit, offset int) (*SMSLogListResult, error) {
	if r == nil {
		return nil, apperr.Internal(fmt.Errorf("log repository not configured"))
	}

	var (
		where []string
		args  []any
		idx   = 1
	)

	addFilter := func(clause string, val any) {
		where = append(where, fmt.Sprintf(clause, idx))
		args = append(args, val)
		idx++
	}

	if f.Status != "" {
		addFilter("status = $%d", f.Status)
	}
	if f.Type != "" {
		addFilter("type = $%d", f.Type)
	}
	if f.Phone != "" {
		addFilter("to_phone = $%d", f.Phone)
	}
	if f.JobID != "" {
		addFilter("job_id = $%d", f.JobID)
	}
	if f.FromDate != nil {
		addFilter("created_at >= $%d", *f.FromDate)
	}
	if f.ToDate != nil {
		addFilter("created_at <= $%d", *f.ToDate)
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	// Count total matching rows.
	countQ := "SELECT count(*) FROM sms_logs " + whereSQL
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, apperr.Internal(err)
	}

	// Fetch page.
	dataQ := fmt.Sprintf(
		"SELECT %s FROM sms_logs %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		smsLogColumns, whereSQL, idx, idx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()

	items := make([]SMSLog, 0)
	for rows.Next() {
		var l SMSLog
		if err := rows.Scan(
			&l.ID, &l.JobID, &l.Type, &l.ToPhone, &l.Message, &l.SMSFrom,
			&l.Status, &l.Attempt, &l.MaxAttempts, &l.DedupKey, &l.Provider,
			&l.LastError, &l.CreatedAt, &l.UpdatedAt, &l.SentAt,
		); err != nil {
			return nil, apperr.Internal(err)
		}
		items = append(items, l)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err)
	}

	return &SMSLogListResult{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// ResetForRequeue updates status to queued and resets attempt for re-enqueue.
// Only allowed from dlq or failed status.
func (r *LogRepository) ResetForRequeue(ctx context.Context, jobID string) error {
	if r == nil {
		return apperr.Internal(fmt.Errorf("log repository not configured"))
	}
	const q = `
		UPDATE sms_logs
		SET status = 'queued', attempt = 0, last_error = NULL
		WHERE job_id = $1 AND status IN ('dlq', 'failed', 'rate_limited')`
	ct, err := r.db.Exec(ctx, q, jobID)
	if err != nil {
		return apperr.Internal(err)
	}
	if ct.RowsAffected() == 0 {
		return apperr.Validation("job is not in a requeueable status (must be dlq, failed, or rate_limited)")
	}
	return nil
}

// logRepoError logs errors from fire-and-forget repository methods.
// These must not block or panic — the SMS flow is the primary concern.
func logRepoError(method, jobID string, err error) {
	// Use standard log to match the project style (worker.go, service.go use log.Printf).
	fmt.Printf("[sms-log-repo] %s job=%s error: %v\n", method, jobID, err)
}
