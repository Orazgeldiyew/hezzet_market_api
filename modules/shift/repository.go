package shift

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgconn"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) OpenShift(ctx context.Context, userID int64, req OpenRequest) (Shift, error) {
	// The unique partial index idx_shifts_user_open_unique enforces "at most one
	// open shift per user" at the database level — catching 23505 here closes
	// the race window between SELECT-then-INSERT.
	var s Shift
	err := r.db.QueryRow(ctx, `
		INSERT INTO shifts (register_id, user_id, opening_cash)
		VALUES ($1, $2, $3)
		RETURNING id, register_id, user_id, opened_at, opening_cash, sales_count, sales_total, returns_total, status
	`, req.RegisterID, userID, req.OpeningCash).Scan(
		&s.ID, &s.RegisterID, &s.UserID, &s.OpenedAt, &s.OpeningCash,
		&s.SalesCount, &s.SalesTotal, &s.ReturnsTotal, &s.Status,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Shift{}, apperr.Conflict("SHIFT_ALREADY_OPEN", "you already have an open shift")
		}
		return Shift{}, apperr.Internal(err)
	}
	return s, nil
}

func (r *Repository) CloseShift(ctx context.Context, shiftID, callerID int64, callerRoles []string, req CloseRequest) (Shift, error) {
	// Calculate sales stats for the shift period
	var salesCount int
	var salesTotal, returnsTotal int64

	var openedAt interface{}
	var shiftUserID int64
	var openingCash int64
	var status string

	err := r.db.QueryRow(ctx,
		`SELECT user_id, opened_at, opening_cash, status FROM shifts WHERE id = $1 FOR UPDATE`,
		shiftID,
	).Scan(&shiftUserID, &openedAt, &openingCash, &status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Shift{}, apperr.NotFound("SHIFT_NOT_FOUND", "shift not found")
		}
		return Shift{}, apperr.Internal(err)
	}
	if status != "open" {
		return Shift{}, apperr.Conflict("SHIFT_ALREADY_CLOSED", "shift is already closed")
	}

	// Only the shift owner or a manager/admin may close. Without this, any
	// authenticated cashier could close another user's shift and skew their
	// expected-cash totals.
	isPrivileged := false
	for _, role := range callerRoles {
		if role == "manager" || role == "admin" {
			isPrivileged = true
			break
		}
	}
	if callerID != shiftUserID && !isPrivileged {
		return Shift{}, apperr.Forbidden("only the shift owner or a manager can close this shift")
	}

	// Count confirmed sales by this user during shift
	_ = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total_cents), 0)
		FROM sales
		WHERE created_by = $1 AND status = 'confirmed' AND created_at >= (SELECT opened_at FROM shifts WHERE id = $2)
	`, shiftUserID, shiftID).Scan(&salesCount, &salesTotal)

	// Count returns
	_ = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(sr.total_cents), 0)
		FROM sale_returns sr
		WHERE sr.created_by = $1 AND sr.created_at >= (SELECT opened_at FROM shifts WHERE id = $2)
	`, shiftUserID, shiftID).Scan(&returnsTotal)

	expectedCash := openingCash + salesTotal - returnsTotal

	var s Shift
	err = r.db.QueryRow(ctx, `
		UPDATE shifts SET
			closed_at     = now(),
			closing_cash  = $2,
			expected_cash = $3,
			sales_count   = $4,
			sales_total   = $5,
			returns_total = $6,
			status        = 'closed',
			note          = $7,
			closed_by     = $8
		WHERE id = $1
		RETURNING id, register_id, user_id, opened_at, closed_at,
		          opening_cash, closing_cash, expected_cash,
		          sales_count, sales_total, returns_total, status, note, closed_by
	`, shiftID, req.ClosingCash, expectedCash, salesCount, salesTotal, returnsTotal, req.Note, callerID).Scan(
		&s.ID, &s.RegisterID, &s.UserID, &s.OpenedAt, &s.ClosedAt,
		&s.OpeningCash, &s.ClosingCash, &s.ExpectedCash,
		&s.SalesCount, &s.SalesTotal, &s.ReturnsTotal, &s.Status, &s.Note, &s.ClosedBy,
	)
	if err != nil {
		return Shift{}, apperr.Internal(err)
	}

	// Calculate difference
	if s.ClosingCash != nil && s.ExpectedCash != nil {
		diff := *s.ClosingCash - *s.ExpectedCash
		s.Difference = &diff
	}

	return s, nil
}

func (r *Repository) GetCurrent(ctx context.Context, userID int64) (Shift, error) {
	var s Shift
	err := r.db.QueryRow(ctx, `
		SELECT sh.id, sh.register_id, cr.name, sh.user_id, u.username,
		       sh.opened_at, sh.status
		FROM shifts sh
		JOIN cash_registers cr ON cr.id = sh.register_id
		JOIN employees u ON u.id = sh.user_id
		WHERE sh.user_id = $1 AND sh.status = 'open'
		ORDER BY sh.opened_at DESC LIMIT 1
	`, userID).Scan(
		&s.ID, &s.RegisterID, &s.RegisterName, &s.UserID, &s.CashierName,
		&s.OpenedAt, &s.Status,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Shift{}, apperr.NotFound("NO_OPEN_SHIFT", "no open shift found")
		}
		return Shift{}, apperr.Internal(err)
	}
	return s, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Shift, error) {
	var s Shift
	err := r.db.QueryRow(ctx, `
		SELECT sh.id, sh.register_id, cr.name, sh.user_id, u.username,
		       sh.opened_at, sh.closed_at,
		       sh.opening_cash, sh.closing_cash, sh.expected_cash,
		       sh.sales_count, sh.sales_total, sh.returns_total,
		       sh.status, sh.note, sh.closed_by
		FROM shifts sh
		JOIN cash_registers cr ON cr.id = sh.register_id
		JOIN employees u ON u.id = sh.user_id
		WHERE sh.id = $1
	`, id).Scan(
		&s.ID, &s.RegisterID, &s.RegisterName, &s.UserID, &s.CashierName,
		&s.OpenedAt, &s.ClosedAt,
		&s.OpeningCash, &s.ClosingCash, &s.ExpectedCash,
		&s.SalesCount, &s.SalesTotal, &s.ReturnsTotal,
		&s.Status, &s.Note, &s.ClosedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Shift{}, apperr.NotFound("SHIFT_NOT_FOUND", "shift not found")
		}
		return Shift{}, apperr.Internal(err)
	}

	if s.ClosingCash != nil && s.ExpectedCash != nil {
		diff := *s.ClosingCash - *s.ExpectedCash
		s.Difference = &diff
	}

	return s, nil
}

func (r *Repository) List(ctx context.Context, userID *int64, registerID *int64, status *string, limit, offset int) ([]Shift, int, error) {
	// Build placeholders dynamically so we never pass unused args (pgx rejects
	// "extra args"). The original code hard-coded $1/$2/$3 even when filters
	// were nil — that's the source of the 500 on unfiltered queries.
	where := `WHERE 1=1`
	args := []any{}
	add := func(clause string, v any) {
		args = append(args, v)
		where += ` AND ` + clause + ` $` + strconv.Itoa(len(args))
	}
	if userID != nil {
		add("sh.user_id =", *userID)
	}
	if registerID != nil {
		add("sh.register_id =", *registerID)
	}
	if status != nil {
		add("sh.status =", *status)
	}

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM shifts sh `+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)
	limitIdx := strconv.Itoa(len(args) + 1)
	offsetIdx := strconv.Itoa(len(args) + 2)

	rows, err := r.db.Query(ctx, `
		SELECT sh.id, sh.register_id, cr.name, sh.user_id, u.username,
		       sh.opened_at, sh.closed_at,
		       sh.opening_cash, sh.closing_cash, sh.expected_cash,
		       sh.sales_count, sh.sales_total, sh.returns_total,
		       sh.status, sh.note, sh.closed_by
		FROM shifts sh
		JOIN cash_registers cr ON cr.id = sh.register_id
		JOIN employees u ON u.id = sh.user_id
		`+where+`
		ORDER BY sh.opened_at DESC
		LIMIT $`+limitIdx+` OFFSET $`+offsetIdx,
		listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Shift
	for rows.Next() {
		var s Shift
		err := rows.Scan(
			&s.ID, &s.RegisterID, &s.RegisterName, &s.UserID, &s.CashierName,
			&s.OpenedAt, &s.ClosedAt,
			&s.OpeningCash, &s.ClosingCash, &s.ExpectedCash,
			&s.SalesCount, &s.SalesTotal, &s.ReturnsTotal,
			&s.Status, &s.Note, &s.ClosedBy,
		)
		if err != nil {
			return nil, 0, err
		}
		if s.ClosingCash != nil && s.ExpectedCash != nil {
			diff := *s.ClosingCash - *s.ExpectedCash
			s.Difference = &diff
		}
		out = append(out, s)
	}
	if out == nil {
		out = []Shift{}
	}
	return out, total, rows.Err()
}

func (r *Repository) ListRegisters(ctx context.Context) ([]CashRegister, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, is_active, created_at FROM cash_registers WHERE is_active = true ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CashRegister
	for rows.Next() {
		var cr CashRegister
		if err := rows.Scan(&cr.ID, &cr.Name, &cr.IsActive, &cr.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, cr)
	}
	if out == nil {
		out = []CashRegister{}
	}
	return out, rows.Err()
}
