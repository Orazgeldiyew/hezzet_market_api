// modules/customer/repository.go
package customer

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

func IsNotFound(err error) bool { return err == pgx.ErrNoRows }

func (r *Repository) Create(ctx context.Context, c *Customer) error {
	q := `
		INSERT INTO customers (name, phone, email, type, notes, is_active, card_code)
		VALUES ($1, $2, $3, $4, $5, $6, upper(substring(md5(gen_random_uuid()::text), 1, 8)))
		RETURNING id, total_spent, bonus_points, card_code, is_active, created_at, updated_at
	`
	return r.db.QueryRow(ctx, q,
		c.Name, c.Phone, c.Email, c.Type, c.Notes, c.IsActive,
	).Scan(&c.ID, &c.TotalSpent, &c.BonusPoints, &c.CardCode, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
}

// IMPORTANT: Get returns row even if is_active=false. 404 only when deleted_at IS NOT NULL.
func (r *Repository) GetByID(ctx context.Context, id int64) (Customer, error) {
	var c Customer
	q := `
		SELECT id, name, phone, email, type, total_spent, bonus_points, card_code,
		       is_active, notes, created_at, updated_at, deleted_at
		FROM customers
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := r.db.QueryRow(ctx, q, id).Scan(
		&c.ID, &c.Name, &c.Phone, &c.Email, &c.Type, &c.TotalSpent, &c.BonusPoints, &c.CardCode,
		&c.IsActive, &c.Notes, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	return c, err
}

// GetByCardCode finds a customer by their loyalty card QR/barcode.
func (r *Repository) GetByCardCode(ctx context.Context, code string) (Customer, error) {
	var c Customer
	q := `
		SELECT id, name, phone, email, type, total_spent, bonus_points, card_code,
		       is_active, notes, created_at, updated_at, deleted_at
		FROM customers
		WHERE card_code = $1 AND deleted_at IS NULL
	`
	err := r.db.QueryRow(ctx, q, code).Scan(
		&c.ID, &c.Name, &c.Phone, &c.Email, &c.Type, &c.TotalSpent, &c.BonusPoints, &c.CardCode,
		&c.IsActive, &c.Notes, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	return c, err
}

// List returns NOT deleted; by default activeOnly=true.
func (r *Repository) List(ctx context.Context, limit, offset int, orderBy, orderDir, search string, activeOnly bool) ([]Customer, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	countSQL := `
		SELECT COUNT(*) FROM customers
		WHERE deleted_at IS NULL
		  AND ($2 = false OR is_active = true)
		  AND ($1 = '' OR
		       name ILIKE '%' || $1 || '%' OR
		       phone ILIKE '%' || $1 || '%' OR
		       email ILIKE '%' || $1 || '%' OR
		       card_code ILIKE '%' || $1 || '%')
	`
	var total int
	if err := r.db.QueryRow(ctx, countSQL, search, activeOnly).Scan(&total); err != nil {
		return nil, 0, err
	}

	col := "created_at"
	switch orderBy {
	case "name":
		col = "name"
	case "total_spent":
		col = "total_spent"
	case "created_at":
		col = "created_at"
	}

	dir := "DESC"
	if orderDir == "asc" {
		dir = "ASC"
	}

	q := fmt.Sprintf(`
		SELECT id, name, phone, email, type, total_spent, bonus_points, card_code,
		       is_active, notes, created_at, updated_at, deleted_at
		FROM customers
		WHERE deleted_at IS NULL
		  AND ($2 = false OR is_active = true)
		  AND ($1 = '' OR
		       name ILIKE '%%' || $1 || '%%' OR
		       phone ILIKE '%%' || $1 || '%%' OR
		       email ILIKE '%%' || $1 || '%%' OR
		       card_code ILIKE '%%' || $1 || '%%')
		ORDER BY %s %s
		LIMIT $3 OFFSET $4
	`, col, dir)

	rows, err := r.db.Query(ctx, q, search, activeOnly, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Customer
	for rows.Next() {
		var c Customer
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Phone, &c.Email, &c.Type, &c.TotalSpent, &c.BonusPoints, &c.CardCode,
			&c.IsActive, &c.Notes, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

// Delete policy: deleted_at=now(), is_active=false
func (r *Repository) SoftDelete(ctx context.Context, id int64) error {
	ct, err := r.db.Exec(ctx, `
		UPDATE customers
		SET deleted_at = now(), is_active = false, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) AddSpent(ctx context.Context, id int64, amountCents int64, bonusCents int64) (Customer, error) {
	q := `
		UPDATE customers SET
			total_spent  = total_spent + $1,
			bonus_points = bonus_points + $2,
			updated_at   = now()
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING id, name, phone, email, type, total_spent, bonus_points, card_code,
		          is_active, notes, created_at, updated_at, deleted_at
	`
	var c Customer
	err := r.db.QueryRow(ctx, q, amountCents, bonusCents, id).Scan(
		&c.ID, &c.Name, &c.Phone, &c.Email, &c.Type, &c.TotalSpent, &c.BonusPoints, &c.CardCode,
		&c.IsActive, &c.Notes, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	return c, err
}

// AddSpentTx adds total_spent and bonus_points within an existing transaction.
func (r *Repository) AddSpentTx(ctx context.Context, tx pgx.Tx, id, amountCents, bonusCents int64) error {
	_, err := tx.Exec(ctx, `
		UPDATE customers SET
			total_spent  = total_spent + $1,
			bonus_points = bonus_points + $2,
			updated_at   = now()
		WHERE id = $3 AND deleted_at IS NULL
	`, amountCents, bonusCents, id)
	return err
}

// DeductBonus atomically removes bonusAmount from bonus_points within an existing transaction.
// Returns VALIDATION error if bonus_points < bonusAmount.
func (r *Repository) DeductBonus(ctx context.Context, tx pgx.Tx, customerID, bonusAmount int64) error {
	ct, err := tx.Exec(ctx, `
		UPDATE customers SET
			bonus_points = bonus_points - $2,
			updated_at   = now()
		WHERE id = $1 AND deleted_at IS NULL AND bonus_points >= $2
	`, customerID, bonusAmount)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return apperr.Validation("insufficient bonus points")
	}
	return nil
}

// UpdateContact updates only contact fields (name/phone/email/notes).
func (r *Repository) UpdateContact(ctx context.Context, id int64, req UpdateContactRequest) (Customer, error) {
	q := `
		UPDATE customers SET
			name      = COALESCE($1, name),
			phone     = COALESCE($2, phone),
			email     = COALESCE($3, email),
			notes     = COALESCE($4, notes),
			updated_at = now()
		WHERE id = $5 AND deleted_at IS NULL
		RETURNING id, name, phone, email, type, total_spent, bonus_points, card_code,
		          is_active, notes, created_at, updated_at, deleted_at
	`

	var c Customer
	err := r.db.QueryRow(ctx, q,
		req.Name, req.Phone, req.Email, req.Notes, id,
	).Scan(
		&c.ID, &c.Name, &c.Phone, &c.Email, &c.Type, &c.TotalSpent, &c.BonusPoints, &c.CardCode,
		&c.IsActive, &c.Notes, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	return c, err
}

// UpdateAdmin updates only business fields (type, is_active).
func (r *Repository) UpdateAdmin(ctx context.Context, id int64, req UpdateAdminRequest) (Customer, error) {
	q := `
		UPDATE customers SET
			type      = COALESCE($1, type),
			is_active = COALESCE($2, is_active),
			updated_at = now()
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING id, name, phone, email, type, total_spent, bonus_points, card_code,
		          is_active, notes, created_at, updated_at, deleted_at
	`
	var c Customer
	err := r.db.QueryRow(ctx, q, req.Type, req.IsActive, id).Scan(
		&c.ID, &c.Name, &c.Phone, &c.Email, &c.Type, &c.TotalSpent, &c.BonusPoints, &c.CardCode,
		&c.IsActive, &c.Notes, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	return c, err
}
