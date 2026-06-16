package customerdebt

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

// CreateDebt creates a new customer debt record (called from sale confirm).
func (r *Repository) CreateDebt(ctx context.Context, tx pgx.Tx, customerID int64, saleID *int64, amountCents int64, note string, createdBy *int64) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO customer_debts (customer_id, sale_id, amount_cents, remaining_cents, note, created_by)
		VALUES ($1, $2, $3, $3, $4, $5)
		RETURNING id
	`, customerID, saleID, amountCents, note, createdBy).Scan(&id)
	return id, err
}

// GetByID returns a single debt with customer and user names.
func (r *Repository) GetByID(ctx context.Context, id int64) (CustomerDebt, error) {
	var d CustomerDebt
	err := r.db.QueryRow(ctx, `
		SELECT cd.id, cd.customer_id, c.name, cd.sale_id, cd.amount_cents, cd.remaining_cents,
		       cd.status, cd.note, cd.created_by, COALESCE(u.name, ''), cd.created_at, cd.updated_at
		FROM customer_debts cd
		JOIN customers c ON c.id = cd.customer_id
		LEFT JOIN employees u ON u.id = cd.created_by
		WHERE cd.id = $1
	`, id).Scan(
		&d.ID, &d.CustomerID, &d.CustomerName, &d.SaleID, &d.AmountCents, &d.RemainingCents,
		&d.Status, &d.Note, &d.CreatedBy, &d.CreatedByName, &d.CreatedAt, &d.UpdatedAt,
	)
	return d, err
}

// ListByCustomer returns all debts for a customer, newest first.
func (r *Repository) ListByCustomer(ctx context.Context, customerID int64, onlyOpen bool, limit, offset int) ([]CustomerDebt, int, error) {
	statusFilter := ""
	if onlyOpen {
		statusFilter = "AND cd.status = 'open'"
	}

	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM customer_debts cd WHERE cd.customer_id = $1 `+statusFilter,
		customerID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT cd.id, cd.customer_id, c.name, cd.sale_id, cd.amount_cents, cd.remaining_cents,
		       cd.status, cd.note, cd.created_by, COALESCE(u.name, ''), cd.created_at, cd.updated_at
		FROM customer_debts cd
		JOIN customers c ON c.id = cd.customer_id
		LEFT JOIN employees u ON u.id = cd.created_by
		WHERE cd.customer_id = $1 `+statusFilter+`
		ORDER BY cd.created_at DESC
		LIMIT $2 OFFSET $3
	`, customerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []CustomerDebt
	for rows.Next() {
		var d CustomerDebt
		if err := rows.Scan(
			&d.ID, &d.CustomerID, &d.CustomerName, &d.SaleID, &d.AmountCents, &d.RemainingCents,
			&d.Status, &d.Note, &d.CreatedBy, &d.CreatedByName, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, d)
	}
	if out == nil {
		out = []CustomerDebt{}
	}
	return out, total, rows.Err()
}

// ListAll returns all open debts across customers (for manager view).
func (r *Repository) ListAll(ctx context.Context, limit, offset int) ([]CustomerDebt, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM customer_debts WHERE status = 'open'`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT cd.id, cd.customer_id, c.name, cd.sale_id, cd.amount_cents, cd.remaining_cents,
		       cd.status, cd.note, cd.created_by, COALESCE(u.name, ''), cd.created_at, cd.updated_at
		FROM customer_debts cd
		JOIN customers c ON c.id = cd.customer_id
		LEFT JOIN employees u ON u.id = cd.created_by
		WHERE cd.status = 'open'
		ORDER BY cd.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []CustomerDebt
	for rows.Next() {
		var d CustomerDebt
		if err := rows.Scan(
			&d.ID, &d.CustomerID, &d.CustomerName, &d.SaleID, &d.AmountCents, &d.RemainingCents,
			&d.Status, &d.Note, &d.CreatedBy, &d.CreatedByName, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, d)
	}
	if out == nil {
		out = []CustomerDebt{}
	}
	return out, total, rows.Err()
}

// Pay records a payment against a debt. Auto-settles if remaining becomes 0.
func (r *Repository) Pay(ctx context.Context, debtID int64, req PayRequest, userID *int64) (CustomerDebt, DebtPayment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return CustomerDebt{}, DebtPayment{}, err
	}
	defer tx.Rollback(ctx)

	// Lock debt row
	var remaining int64
	var status string
	err = tx.QueryRow(ctx, `
		SELECT remaining_cents, status FROM customer_debts WHERE id = $1 FOR UPDATE
	`, debtID).Scan(&remaining, &status)
	if err != nil {
		return CustomerDebt{}, DebtPayment{}, err
	}
	if status != "open" {
		return CustomerDebt{}, DebtPayment{}, pgx.ErrNoRows // will be mapped to 404
	}
	if req.AmountCents > remaining {
		return CustomerDebt{}, DebtPayment{}, errOverpay
	}

	// Insert payment
	var note *string
	if req.Note != "" {
		note = &req.Note
	}
	var payment DebtPayment
	err = tx.QueryRow(ctx, `
		INSERT INTO customer_debt_payments (debt_id, amount_cents, payment_type_id, note, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, debt_id, amount_cents, payment_type_id, note, created_by, created_at
	`, debtID, req.AmountCents, req.PaymentTypeID, note, userID).Scan(
		&payment.ID, &payment.DebtID, &payment.AmountCents, &payment.PaymentTypeID,
		&payment.Note, &payment.CreatedBy, &payment.CreatedAt,
	)
	if err != nil {
		return CustomerDebt{}, DebtPayment{}, err
	}

	// Update remaining
	newRemaining := remaining - req.AmountCents
	newStatus := "open"
	if newRemaining == 0 {
		newStatus = "settled"
	}

	var debt CustomerDebt
	err = tx.QueryRow(ctx, `
		UPDATE customer_debts SET remaining_cents = $2, status = $3, updated_at = now()
		WHERE id = $1
		RETURNING id, customer_id, sale_id, amount_cents, remaining_cents, status, note, created_by, created_at, updated_at
	`, debtID, newRemaining, newStatus).Scan(
		&debt.ID, &debt.CustomerID, &debt.SaleID, &debt.AmountCents, &debt.RemainingCents,
		&debt.Status, &debt.Note, &debt.CreatedBy, &debt.CreatedAt, &debt.UpdatedAt,
	)
	if err != nil {
		return CustomerDebt{}, DebtPayment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return CustomerDebt{}, DebtPayment{}, err
	}
	return debt, payment, nil
}

// GetPayments returns all payments for a debt.
func (r *Repository) GetPayments(ctx context.Context, debtID int64) ([]DebtPayment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT dp.id, dp.debt_id, dp.amount_cents, dp.payment_type_id, dp.note,
		       dp.created_by, COALESCE(u.name, ''), dp.created_at
		FROM customer_debt_payments dp
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

// GetCustomerTotalDebt returns total remaining debt for a customer.
func (r *Repository) GetCustomerTotalDebt(ctx context.Context, customerID int64) (int64, error) {
	var total int64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(remaining_cents), 0) FROM customer_debts
		WHERE customer_id = $1 AND status = 'open'
	`, customerID).Scan(&total)
	return total, err
}

// DebtorsSummary returns all customers with open debts.
func (r *Repository) DebtorsSummary(ctx context.Context) ([]CustomerDebtSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT cd.customer_id, c.name,
		       SUM(cd.remaining_cents) AS total_debt,
		       COUNT(*) AS open_debts
		FROM customer_debts cd
		JOIN customers c ON c.id = cd.customer_id
		WHERE cd.status = 'open'
		GROUP BY cd.customer_id, c.name
		ORDER BY total_debt DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CustomerDebtSummary
	for rows.Next() {
		var s CustomerDebtSummary
		if err := rows.Scan(&s.CustomerID, &s.CustomerName, &s.TotalDebt, &s.OpenDebts); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if out == nil {
		out = []CustomerDebtSummary{}
	}
	return out, rows.Err()
}

// sentinel error for overpayment
type appError string

func (e appError) Error() string { return string(e) }

const errOverpay = appError("payment amount exceeds remaining debt")
