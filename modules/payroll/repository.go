package payroll

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

const payrollCols = `id, worker_id, period, base_salary_cents, fines_cents, debts_cents,
	net_salary_cents, status, transaction_id, created_at, updated_at`

func scanPayroll(row pgx.Row) (PayrollRun, error) {
	var p PayrollRun
	err := row.Scan(
		&p.ID, &p.WorkerID, &p.Period, &p.BaseSalaryCents, &p.FinesCents, &p.DebtsCents,
		&p.NetSalaryCents, &p.Status, &p.TransactionID, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func (r *Repository) CreateTx(ctx context.Context, tx pgx.Tx, p *PayrollRun) error {
	return tx.QueryRow(ctx, `
		INSERT INTO payroll_runs (worker_id, period, base_salary_cents, fines_cents, debts_cents, net_salary_cents)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+payrollCols,
		p.WorkerID, p.Period, p.BaseSalaryCents, p.FinesCents, p.DebtsCents, p.NetSalaryCents,
	).Scan(
		&p.ID, &p.WorkerID, &p.Period, &p.BaseSalaryCents, &p.FinesCents, &p.DebtsCents,
		&p.NetSalaryCents, &p.Status, &p.TransactionID, &p.CreatedAt, &p.UpdatedAt,
	)
}

// InsertRunDebts records which debts (and how much of each) were included in
// the run. Called inside Calculate's tx; Pay later uses these rows to settle
// only the snapshotted debts.
func (r *Repository) InsertRunDebts(ctx context.Context, tx pgx.Tx, runID int64, debtAmounts map[int64]int64) error {
	if len(debtAmounts) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for debtID, amount := range debtAmounts {
		batch.Queue(
			`INSERT INTO payroll_run_debts (payroll_run_id, debt_id, amount_cents) VALUES ($1, $2, $3)`,
			runID, debtID, amount,
		)
	}
	br := tx.SendBatch(ctx, batch)
	defer br.Close()
	for range debtAmounts {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (PayrollRun, error) {
	return scanPayroll(r.db.QueryRow(ctx,
		`SELECT `+payrollCols+` FROM payroll_runs WHERE id = $1`, id,
	))
}

// LockPayrollRun acquires a row-level lock for atomic pay.
func (r *Repository) LockPayrollRun(ctx context.Context, tx pgx.Tx, id int64) (PayrollRun, error) {
	return scanPayroll(tx.QueryRow(ctx,
		`SELECT `+payrollCols+` FROM payroll_runs WHERE id = $1 FOR UPDATE`, id,
	))
}

// UpdateStatusPaid sets status='paid' and links the transaction.
func (r *Repository) UpdateStatusPaid(ctx context.Context, tx pgx.Tx, id int64, transactionID int64) (PayrollRun, error) {
	return scanPayroll(tx.QueryRow(ctx, `
		UPDATE payroll_runs SET status = 'paid', transaction_id = $2
		WHERE id = $1
		RETURNING `+payrollCols,
		id, transactionID,
	))
}

// ListForExport fetches all payroll runs for a period joined with worker name/position.
func (r *Repository) ListForExport(ctx context.Context, period string) ([]PayrollExportRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT w.name, w.position, pr.period,
		       pr.base_salary_cents, pr.fines_cents, pr.debts_cents,
		       pr.net_salary_cents, pr.status
		FROM payroll_runs pr
		JOIN workers w ON w.id = pr.worker_id
		WHERE pr.period = $1
		ORDER BY w.name
	`, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PayrollExportRow
	for rows.Next() {
		var row PayrollExportRow
		if err := rows.Scan(
			&row.WorkerName, &row.Position, &row.Period,
			&row.BaseSalaryCents, &row.FinesCents, &row.DebtsCents,
			&row.NetSalaryCents, &row.Status,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) List(ctx context.Context, workerID *int64, period *string, limit, offset int) ([]PayrollRun, int, error) {
	where := `WHERE ($1::bigint IS NULL OR worker_id = $1) AND ($2::text IS NULL OR period = $2)`

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM payroll_runs `+where, workerID, period,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, fmt.Sprintf(
		`SELECT %s FROM payroll_runs %s ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		payrollCols, where),
		workerID, period, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []PayrollRun
	for rows.Next() {
		p, err := scanPayroll(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}
