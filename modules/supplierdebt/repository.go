package supplierdebt

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateDebt creates a new supplier debt (manual or from purchase).
func (r *Repository) CreateDebt(ctx context.Context, supplierID int64, purchaseID *int64, amountCents int64, note string, createdBy *int64) (SupplierDebt, error) {
	var d SupplierDebt
	var n *string
	if note != "" {
		n = &note
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO supplier_debts (supplier_id, purchase_id, amount_cents, remaining_cents, note, created_by)
		VALUES ($1, $2, $3, $3, $4, $5)
		RETURNING id, supplier_id, purchase_id, amount_cents, remaining_cents, status, COALESCE(note,''), created_by, created_at, updated_at
	`, supplierID, purchaseID, amountCents, n, createdBy).Scan(
		&d.ID, &d.SupplierID, &d.PurchaseID, &d.AmountCents, &d.RemainingCents,
		&d.Status, &d.Note, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
	)
	return d, err
}

func (r *Repository) GetByID(ctx context.Context, id int64) (SupplierDebt, error) {
	var d SupplierDebt
	err := r.db.QueryRow(ctx, `
		SELECT sd.id, sd.supplier_id, s.name, sd.purchase_id, sd.amount_cents, sd.remaining_cents,
		       sd.status, COALESCE(sd.note,''), sd.created_by, COALESCE(u.name, ''), sd.created_at, sd.updated_at
		FROM supplier_debts sd
		JOIN suppliers s ON s.id = sd.supplier_id
		LEFT JOIN employees u ON u.id = sd.created_by
		WHERE sd.id = $1
	`, id).Scan(
		&d.ID, &d.SupplierID, &d.SupplierName, &d.PurchaseID, &d.AmountCents, &d.RemainingCents,
		&d.Status, &d.Note, &d.CreatedBy, &d.CreatedByName, &d.CreatedAt, &d.UpdatedAt,
	)
	return d, err
}

func (r *Repository) ListBySupplier(ctx context.Context, supplierID int64, onlyOpen bool, limit, offset int) ([]SupplierDebt, int, error) {
	statusFilter := ""
	if onlyOpen {
		statusFilter = "AND sd.status = 'open'"
	}

	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM supplier_debts sd WHERE sd.supplier_id = $1 `+statusFilter,
		supplierID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT sd.id, sd.supplier_id, s.name, sd.purchase_id, sd.amount_cents, sd.remaining_cents,
		       sd.status, COALESCE(sd.note,''), sd.created_by, COALESCE(u.name, ''), sd.created_at, sd.updated_at
		FROM supplier_debts sd
		JOIN suppliers s ON s.id = sd.supplier_id
		LEFT JOIN employees u ON u.id = sd.created_by
		WHERE sd.supplier_id = $1 `+statusFilter+`
		ORDER BY sd.created_at DESC
		LIMIT $2 OFFSET $3
	`, supplierID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return scanDebts(rows, total)
}

func (r *Repository) ListAll(ctx context.Context, limit, offset int) ([]SupplierDebt, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM supplier_debts WHERE status = 'open'`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT sd.id, sd.supplier_id, s.name, sd.purchase_id, sd.amount_cents, sd.remaining_cents,
		       sd.status, COALESCE(sd.note,''), sd.created_by, COALESCE(u.name, ''), sd.created_at, sd.updated_at
		FROM supplier_debts sd
		JOIN suppliers s ON s.id = sd.supplier_id
		LEFT JOIN employees u ON u.id = sd.created_by
		WHERE sd.status = 'open'
		ORDER BY sd.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return scanDebts(rows, total)
}

func (r *Repository) Pay(ctx context.Context, debtID int64, req PayRequest, userID *int64) (SupplierDebt, DebtPayment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SupplierDebt{}, DebtPayment{}, err
	}
	defer tx.Rollback(ctx)

	var remaining int64
	var status string
	err = tx.QueryRow(ctx,
		`SELECT remaining_cents, status FROM supplier_debts WHERE id = $1 FOR UPDATE`, debtID,
	).Scan(&remaining, &status)
	if err != nil {
		return SupplierDebt{}, DebtPayment{}, err
	}
	if status != "open" {
		return SupplierDebt{}, DebtPayment{}, pgx.ErrNoRows
	}
	if req.AmountCents > remaining {
		return SupplierDebt{}, DebtPayment{}, errOverpay
	}

	var note *string
	if req.Note != "" {
		note = &req.Note
	}
	var payment DebtPayment
	err = tx.QueryRow(ctx, `
		INSERT INTO supplier_debt_payments (debt_id, amount_cents, payment_type_id, note, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, debt_id, amount_cents, payment_type_id, COALESCE(note,''), created_by, created_at
	`, debtID, req.AmountCents, req.PaymentTypeID, note, userID).Scan(
		&payment.ID, &payment.DebtID, &payment.AmountCents, &payment.PaymentTypeID,
		&payment.Note, &payment.CreatedBy, &payment.CreatedAt,
	)
	if err != nil {
		return SupplierDebt{}, DebtPayment{}, err
	}

	newRemaining := remaining - req.AmountCents
	newStatus := "open"
	if newRemaining == 0 {
		newStatus = "settled"
	}

	var debt SupplierDebt
	err = tx.QueryRow(ctx, `
		UPDATE supplier_debts SET remaining_cents = $2, status = $3, updated_at = now()
		WHERE id = $1
		RETURNING id, supplier_id, purchase_id, amount_cents, remaining_cents, status, COALESCE(note,''), created_by, created_at, updated_at
	`, debtID, newRemaining, newStatus).Scan(
		&debt.ID, &debt.SupplierID, &debt.PurchaseID, &debt.AmountCents, &debt.RemainingCents,
		&debt.Status, &debt.Note, &debt.CreatedBy, &debt.CreatedAt, &debt.UpdatedAt,
	)
	if err != nil {
		return SupplierDebt{}, DebtPayment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SupplierDebt{}, DebtPayment{}, err
	}
	return debt, payment, nil
}

func (r *Repository) GetPayments(ctx context.Context, debtID int64) ([]DebtPayment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT dp.id, dp.debt_id, dp.amount_cents, dp.payment_type_id, COALESCE(dp.note,''),
		       dp.created_by, COALESCE(u.name, ''), dp.created_at
		FROM supplier_debt_payments dp
		LEFT JOIN employees u ON u.id = dp.created_by
		WHERE dp.debt_id = $1
		ORDER BY dp.created_at
	`, debtID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DebtPayment
	for rows.Next() {
		var p DebtPayment
		if err := rows.Scan(&p.ID, &p.DebtID, &p.AmountCents, &p.PaymentTypeID,
			&p.Note, &p.CreatedBy, &p.CreatedByName, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if out == nil {
		out = []DebtPayment{}
	}
	return out, rows.Err()
}

func (r *Repository) DebtorsSummary(ctx context.Context) ([]SupplierDebtSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT sd.supplier_id, s.name,
		       SUM(sd.remaining_cents) AS total_debt,
		       COUNT(*) AS open_debts
		FROM supplier_debts sd
		JOIN suppliers s ON s.id = sd.supplier_id
		WHERE sd.status = 'open'
		GROUP BY sd.supplier_id, s.name
		ORDER BY total_debt DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SupplierDebtSummary
	for rows.Next() {
		var s SupplierDebtSummary
		if err := rows.Scan(&s.SupplierID, &s.SupplierName, &s.TotalDebt, &s.OpenDebts); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if out == nil {
		out = []SupplierDebtSummary{}
	}
	return out, rows.Err()
}

// ─── helpers ────────────────────────────────────────────────────────────────

func scanDebts(rows interface{ Next() bool; Scan(...any) error; Err() error }, total int) ([]SupplierDebt, int, error) {
	var out []SupplierDebt
	for rows.Next() {
		var d SupplierDebt
		if err := rows.Scan(
			&d.ID, &d.SupplierID, &d.SupplierName, &d.PurchaseID, &d.AmountCents, &d.RemainingCents,
			&d.Status, &d.Note, &d.CreatedBy, &d.CreatedByName, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, d)
	}
	if out == nil {
		out = []SupplierDebt{}
	}
	return out, total, rows.Err()
}

type appError string

func (e appError) Error() string { return string(e) }

const errOverpay = appError("payment amount exceeds remaining debt")
