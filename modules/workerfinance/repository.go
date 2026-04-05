package workerfinance

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func isNotFound(err error) bool { return err == pgx.ErrNoRows }

// ── column constants ────────────────────────────────────────────────────────

const compensationCols = `id, worker_id, base_salary_cents, pay_day, is_active, created_at, updated_at`

const fineCols = `id, worker_id, amount_cents, reason, status, occurred_at, created_by, created_at, updated_at`

const debtCols = `id, worker_id, amount_cents, remaining_cents, type, status, note, created_by, created_at, updated_at`

// ── scan helpers ────────────────────────────────────────────────────────────

func scanCompensation(row pgx.Row) (WorkerCompensation, error) {
	var c WorkerCompensation
	err := row.Scan(&c.ID, &c.WorkerID, &c.BaseSalaryCents, &c.PayDay, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func scanFine(row pgx.Row) (WorkerFine, error) {
	var f WorkerFine
	err := row.Scan(&f.ID, &f.WorkerID, &f.AmountCents, &f.Reason, &f.Status, &f.OccurredAt, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt)
	return f, err
}

func scanDebt(row pgx.Row) (WorkerDebt, error) {
	var d WorkerDebt
	err := row.Scan(&d.ID, &d.WorkerID, &d.AmountCents, &d.RemainingCents, &d.Type, &d.Status, &d.Note, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

// ── compensation ────────────────────────────────────────────────────────────

func (r *Repository) UpsertCompensation(ctx context.Context, c *WorkerCompensation) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO worker_compensation (worker_id, base_salary_cents, pay_day)
		VALUES ($1, $2, $3)
		ON CONFLICT (worker_id)
		DO UPDATE SET base_salary_cents = EXCLUDED.base_salary_cents,
		              pay_day = EXCLUDED.pay_day,
		              is_active = true
		RETURNING `+compensationCols,
		c.WorkerID, c.BaseSalaryCents, c.PayDay,
	).Scan(&c.ID, &c.WorkerID, &c.BaseSalaryCents, &c.PayDay, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
}

func (r *Repository) GetCompensation(ctx context.Context, workerID int64) (WorkerCompensation, error) {
	return scanCompensation(r.db.QueryRow(ctx,
		`SELECT `+compensationCols+` FROM worker_compensation WHERE worker_id = $1 AND is_active = true`,
		workerID,
	))
}

// ── fines ───────────────────────────────────────────────────────────────────

func (r *Repository) CreateFine(ctx context.Context, f *WorkerFine) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO worker_fines (worker_id, amount_cents, reason, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING `+fineCols,
		f.WorkerID, f.AmountCents, f.Reason, f.CreatedBy,
	).Scan(&f.ID, &f.WorkerID, &f.AmountCents, &f.Reason, &f.Status, &f.OccurredAt, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt)
}

func (r *Repository) ListFines(ctx context.Context, workerID int64, limit, offset int) ([]WorkerFine, int, error) {
	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM worker_fines WHERE worker_id = $1`, workerID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, fmt.Sprintf(
		`SELECT %s FROM worker_fines WHERE worker_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		fineCols),
		workerID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []WorkerFine
	for rows.Next() {
		f, err := scanFine(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, f)
	}
	return out, total, rows.Err()
}

// SumOpenFines returns the total cents of open fines for a worker.
func (r *Repository) SumOpenFines(ctx context.Context, workerID int64) (int64, error) {
	var sum int64
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount_cents), 0) FROM worker_fines WHERE worker_id = $1 AND status = 'open'`,
		workerID,
	).Scan(&sum)
	return sum, err
}

// MarkFinesDeducted marks all open fines as deducted for the worker within a tx.
func (r *Repository) MarkFinesDeducted(ctx context.Context, tx pgx.Tx, workerID int64) error {
	_, err := tx.Exec(ctx,
		`UPDATE worker_fines SET status = 'deducted' WHERE worker_id = $1 AND status = 'open'`,
		workerID,
	)
	return err
}

// ── debts ───────────────────────────────────────────────────────────────────

func (r *Repository) CreateDebt(ctx context.Context, tx pgx.Tx, d *WorkerDebt) error {
	return tx.QueryRow(ctx, `
		INSERT INTO worker_debts (worker_id, amount_cents, remaining_cents, type, note, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+debtCols,
		d.WorkerID, d.AmountCents, d.RemainingCents, d.Type, d.Note, d.CreatedBy,
	).Scan(&d.ID, &d.WorkerID, &d.AmountCents, &d.RemainingCents, &d.Type, &d.Status, &d.Note, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
}

func (r *Repository) ListDebts(ctx context.Context, workerID int64, limit, offset int) ([]WorkerDebt, int, error) {
	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM worker_debts WHERE worker_id = $1`, workerID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, fmt.Sprintf(
		`SELECT %s FROM worker_debts WHERE worker_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		debtCols),
		workerID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []WorkerDebt
	for rows.Next() {
		d, err := scanDebt(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}

// SumOpenDebts returns the total remaining_cents of open debts for a worker.
func (r *Repository) SumOpenDebts(ctx context.Context, workerID int64) (int64, error) {
	var sum int64
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(remaining_cents), 0) FROM worker_debts WHERE worker_id = $1 AND status = 'open'`,
		workerID,
	).Scan(&sum)
	return sum, err
}

// SettleDebts marks all open debts as settled for the worker within a tx.
func (r *Repository) SettleDebts(ctx context.Context, tx pgx.Tx, workerID int64) error {
	_, err := tx.Exec(ctx,
		`UPDATE worker_debts SET remaining_cents = 0, status = 'settled' WHERE worker_id = $1 AND status = 'open'`,
		workerID,
	)
	return err
}

// GetDebtByID returns a single debt.
func (r *Repository) GetDebtByID(ctx context.Context, id int64) (WorkerDebt, error) {
	return scanDebt(r.db.QueryRow(ctx, fmt.Sprintf(`SELECT %s FROM worker_debts WHERE id = $1`, debtCols), id))
}

// AllDebtors returns workers with open debts.
func (r *Repository) AllDebtors(ctx context.Context) ([]WorkerDebtSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT wd.worker_id, w.name,
		       SUM(wd.remaining_cents) AS total_debt,
		       COUNT(*) AS open_debts
		FROM worker_debts wd
		JOIN workers w ON w.id = wd.worker_id
		WHERE wd.status = 'open'
		GROUP BY wd.worker_id, w.name
		ORDER BY total_debt DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []WorkerDebtSummary
	for rows.Next() {
		var s WorkerDebtSummary
		if err := rows.Scan(&s.WorkerID, &s.WorkerName, &s.TotalDebt, &s.OpenDebts); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if out == nil {
		out = []WorkerDebtSummary{}
	}
	return out, rows.Err()
}

// PayDebt records a partial/full payment on a debt.
func (r *Repository) PayDebt(ctx context.Context, debtID int64, amountCents int64) (WorkerDebt, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return WorkerDebt{}, err
	}
	defer tx.Rollback(ctx)

	var remaining int64
	var status string
	err = tx.QueryRow(ctx,
		`SELECT remaining_cents, status FROM worker_debts WHERE id = $1 FOR UPDATE`, debtID,
	).Scan(&remaining, &status)
	if err != nil {
		return WorkerDebt{}, err
	}
	if status != "open" {
		return WorkerDebt{}, pgx.ErrNoRows
	}
	if amountCents > remaining {
		return WorkerDebt{}, fmt.Errorf("payment amount exceeds remaining debt")
	}

	newRemaining := remaining - amountCents
	newStatus := "open"
	if newRemaining == 0 {
		newStatus = "settled"
	}

	d, err := scanDebt(tx.QueryRow(ctx, fmt.Sprintf(`
		UPDATE worker_debts SET remaining_cents = $2, status = $3, updated_at = now()
		WHERE id = $1 RETURNING %s`, debtCols), debtID, newRemaining, newStatus))
	if err != nil {
		return WorkerDebt{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return WorkerDebt{}, err
	}
	return d, nil
}

// BeginTx starts a database transaction.
func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}
