package finance

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// ── column constants ────────────────────────────────────────────────────────

const transactionCols = `id, payment_type_id, reason, status, amount_cents, type,
	table_payment_id, warehouse_item_detail_id, related_table, related_id,
	created_by, created_at, updated_at`

const paymentCols = `id, transaction_id, payment_type_id, amount_cents, note, created_by, created_at`

const paymentTypeCols = `id, code, name, is_active, created_at`

// ── scan helpers ────────────────────────────────────────────────────────────

func scanTransaction(row pgx.Row) (Transaction, error) {
	var t Transaction
	err := row.Scan(
		&t.ID, &t.PaymentTypeID, &t.Reason, &t.Status, &t.AmountCents, &t.Type,
		&t.TablePaymentID, &t.WarehouseItemDetailID, &t.RelatedTable, &t.RelatedID,
		&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	return t, err
}

func scanPayment(row pgx.Row) (Payment, error) {
	var p Payment
	err := row.Scan(
		&p.ID, &p.TransactionID, &p.PaymentTypeID, &p.AmountCents,
		&p.Note, &p.CreatedBy, &p.CreatedAt,
	)
	return p, err
}

func scanPaymentType(row pgx.Row) (PaymentType, error) {
	var pt PaymentType
	err := row.Scan(&pt.ID, &pt.Code, &pt.Name, &pt.IsActive, &pt.CreatedAt)
	return pt, err
}

func isNotFound(err error) bool { return err == pgx.ErrNoRows }

// ── payment_types ───────────────────────────────────────────────────────────

func (r *Repository) ListPaymentTypes(ctx context.Context) ([]PaymentType, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+paymentTypeCols+`
		FROM payment_types
		WHERE is_active = true
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PaymentType
	for rows.Next() {
		pt, err := scanPaymentType(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, pt)
	}
	return out, rows.Err()
}

// GetPaymentTypeIDByCode looks up an active payment type by code.
func (r *Repository) GetPaymentTypeIDByCode(ctx context.Context, code string) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx,
		`SELECT id FROM payment_types WHERE code = $1 AND is_active = true`, code,
	).Scan(&id)
	return id, err
}

// ── transactions CRUD ───────────────────────────────────────────────────────

func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func (r *Repository) CreateTransaction(ctx context.Context, tx pgx.Tx, t *Transaction) error {
	return tx.QueryRow(ctx, `
		INSERT INTO transactions
			(payment_type_id, reason, status, amount_cents, type,
			 table_payment_id, warehouse_item_detail_id, related_table, related_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING `+transactionCols,
		t.PaymentTypeID, t.Reason, t.Status, t.AmountCents, t.Type,
		t.TablePaymentID, t.WarehouseItemDetailID, t.RelatedTable, t.RelatedID, t.CreatedBy,
	).Scan(
		&t.ID, &t.PaymentTypeID, &t.Reason, &t.Status, &t.AmountCents, &t.Type,
		&t.TablePaymentID, &t.WarehouseItemDetailID, &t.RelatedTable, &t.RelatedID,
		&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
}

// FindByIdempotencyKey returns the existing transaction created with the given
// key, or (nil, nil) if no match. Used by manual-transaction flow to deduplicate
// double-clicks and network retries.
func (r *Repository) FindByIdempotencyKey(ctx context.Context, key string) (*Transaction, error) {
	if key == "" {
		return nil, nil
	}
	var t Transaction
	err := r.db.QueryRow(ctx,
		`SELECT `+transactionCols+` FROM transactions WHERE idempotency_key = $1`, key,
	).Scan(
		&t.ID, &t.PaymentTypeID, &t.Reason, &t.Status, &t.AmountCents, &t.Type,
		&t.TablePaymentID, &t.WarehouseItemDetailID, &t.RelatedTable, &t.RelatedID,
		&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// CreateManualTransactionWithKey is the idempotent variant of CreateTransaction
// used for manual income/expense entries. If a row with the same idempotency_key
// already exists, the existing row is returned and no new row is inserted.
func (r *Repository) CreateManualTransactionWithKey(ctx context.Context, tx pgx.Tx, t *Transaction, key string) (alreadyExisted bool, err error) {
	if key == "" {
		// Backward compatibility: fall through to a plain insert.
		return false, r.CreateTransaction(ctx, tx, t)
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions
			(payment_type_id, reason, status, amount_cents, type,
			 table_payment_id, warehouse_item_detail_id, related_table, related_id, created_by, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING
		RETURNING `+transactionCols,
		t.PaymentTypeID, t.Reason, t.Status, t.AmountCents, t.Type,
		t.TablePaymentID, t.WarehouseItemDetailID, t.RelatedTable, t.RelatedID, t.CreatedBy, key,
	).Scan(
		&t.ID, &t.PaymentTypeID, &t.Reason, &t.Status, &t.AmountCents, &t.Type,
		&t.TablePaymentID, &t.WarehouseItemDetailID, &t.RelatedTable, &t.RelatedID,
		&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == nil {
		return false, nil
	}
	if err != pgx.ErrNoRows {
		return false, err
	}
	// ON CONFLICT skipped the insert — load the original row outside the tx
	// (we can't run a second SELECT in the same tx after a failed RETURNING).
	existing, ferr := r.FindByIdempotencyKey(ctx, key)
	if ferr != nil {
		return false, ferr
	}
	if existing == nil {
		return false, fmt.Errorf("idempotency_key not found after conflict")
	}
	*t = *existing
	return true, nil
}

func (r *Repository) CreatePayment(ctx context.Context, tx pgx.Tx, p *Payment) error {
	return tx.QueryRow(ctx, `
		INSERT INTO payments (transaction_id, payment_type_id, amount_cents, note, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+paymentCols,
		p.TransactionID, p.PaymentTypeID, p.AmountCents, p.Note, p.CreatedBy,
	).Scan(
		&p.ID, &p.TransactionID, &p.PaymentTypeID, &p.AmountCents,
		&p.Note, &p.CreatedBy, &p.CreatedAt,
	)
}

func (r *Repository) UpdateTransactionStatus(ctx context.Context, tx pgx.Tx, id int64, status string) (Transaction, error) {
	return scanTransaction(tx.QueryRow(ctx, `
		UPDATE transactions SET status = $2 WHERE id = $1
		RETURNING `+transactionCols,
		id, status,
	))
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Transaction, error) {
	return scanTransaction(r.db.QueryRow(ctx, `
		SELECT `+transactionCols+` FROM transactions WHERE id = $1`,
		id,
	))
}

func (r *Repository) GetPayments(ctx context.Context, transactionID int64) ([]Payment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+paymentCols+`
		FROM payments
		WHERE transaction_id = $1
		ORDER BY created_at ASC`,
		transactionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Payment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) GetPaidSum(ctx context.Context, tx pgx.Tx, transactionID int64) (int64, error) {
	var sum int64
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_cents), 0)
		FROM payments
		WHERE transaction_id = $1`,
		transactionID,
	).Scan(&sum)
	return sum, err
}

// LockTransaction acquires a row-level lock and returns the transaction.
func (r *Repository) LockTransaction(ctx context.Context, tx pgx.Tx, id int64) (Transaction, error) {
	return scanTransaction(tx.QueryRow(ctx, `
		SELECT `+transactionCols+`
		FROM transactions
		WHERE id = $1
		FOR UPDATE`,
		id,
	))
}

// SetCanceled sets status to 'canceled'. Returns the updated row.
// Returns CONFLICT error if the transaction is already fully paid.
func (r *Repository) SetCanceled(ctx context.Context, id int64) (Transaction, error) {
	t, err := scanTransaction(r.db.QueryRow(ctx, `
		UPDATE transactions
		SET status = 'canceled'
		WHERE id = $1 AND status NOT IN ('canceled', 'paid')
		RETURNING `+transactionCols,
		id,
	))
	if err != nil {
		if isNotFound(err) {
			// Could be already canceled/paid or truly not found — check
			existing, err2 := r.GetByID(ctx, id)
			if err2 != nil {
				if isNotFound(err2) {
					return Transaction{}, apperr.NotFound("TRANSACTION_NOT_FOUND", "transaction not found")
				}
				return Transaction{}, err2
			}
			if existing.Status == "paid" {
				return Transaction{}, apperr.Conflict("TRANSACTION_PAID", "cannot cancel a fully paid transaction")
			}
			// Already canceled — return idempotently
			return existing, nil
		}
		return Transaction{}, err
	}
	return t, nil
}

// ── list with filters ───────────────────────────────────────────────────────

func (r *Repository) ListTransactions(
	ctx context.Context,
	f TransactionFilter,
	limit, offset int,
) ([]Transaction, int, error) {

	where := `
		WHERE ($1::text IS NULL OR type = $1)
		  AND ($2::text IS NULL OR status = $2)
		  AND ($3::text IS NULL OR related_table = $3)
		  AND ($4::bigint IS NULL OR related_id = $4)
		  AND ($5::timestamptz IS NULL OR created_at >= $5)
		  AND ($6::timestamptz IS NULL OR created_at <= $6)
		  AND ($7::bigint IS NULL OR payment_type_id = $7)
	`

	args := []any{f.Type, f.Status, f.RelatedTable, f.RelatedID, f.DateFrom, f.DateTo, f.PaymentTypeID}

	// Count
	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM transactions `+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Select
	query := fmt.Sprintf(`
		SELECT %s FROM transactions %s
		ORDER BY created_at DESC, id DESC
		LIMIT $8 OFFSET $9`,
		transactionCols, where,
	)

	rows, err := r.db.Query(ctx, query, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}
