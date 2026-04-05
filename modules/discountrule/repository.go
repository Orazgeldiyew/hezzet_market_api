package discountrule

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

func (r *Repository) Create(ctx context.Context, req CreateRequest, userID *int64) (DiscountRule, error) {
	var d DiscountRule
	err := r.db.QueryRow(ctx, `
		INSERT INTO discount_rules (name, min_amount_cents, discount_percent, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, min_amount_cents, discount_percent, is_active, created_by, created_at, updated_at
	`, req.Name, req.MinAmountCents, req.DiscountPercent, userID).Scan(
		&d.ID, &d.Name, &d.MinAmountCents, &d.DiscountPercent, &d.IsActive, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
	)
	return d, err
}

func (r *Repository) GetByID(ctx context.Context, id int64) (DiscountRule, error) {
	var d DiscountRule
	err := r.db.QueryRow(ctx, `
		SELECT id, name, min_amount_cents, discount_percent, is_active, created_by, created_at, updated_at
		FROM discount_rules WHERE id = $1
	`, id).Scan(&d.ID, &d.Name, &d.MinAmountCents, &d.DiscountPercent, &d.IsActive, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (r *Repository) List(ctx context.Context) ([]DiscountRule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, min_amount_cents, discount_percent, is_active, created_by, created_at, updated_at
		FROM discount_rules ORDER BY min_amount_cents ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DiscountRule
	for rows.Next() {
		var d DiscountRule
		if err := rows.Scan(&d.ID, &d.Name, &d.MinAmountCents, &d.DiscountPercent, &d.IsActive, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if out == nil {
		out = []DiscountRule{}
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateRequest) (DiscountRule, error) {
	var d DiscountRule
	err := r.db.QueryRow(ctx, `
		UPDATE discount_rules SET
			name = COALESCE($2, name),
			min_amount_cents = COALESCE($3, min_amount_cents),
			discount_percent = COALESCE($4, discount_percent),
			is_active = COALESCE($5, is_active),
			updated_at = now()
		WHERE id = $1
		RETURNING id, name, min_amount_cents, discount_percent, is_active, created_by, created_at, updated_at
	`, id, req.Name, req.MinAmountCents, req.DiscountPercent, req.IsActive).Scan(
		&d.ID, &d.Name, &d.MinAmountCents, &d.DiscountPercent, &d.IsActive, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
	)
	return d, err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	ct, err := r.db.Exec(ctx, `DELETE FROM discount_rules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// FindMatchingRule returns the best matching active rule for a given amount (highest min_amount that the amount exceeds).
func (r *Repository) FindMatchingRule(ctx context.Context, amountCents int64) (*DiscountRule, error) {
	var d DiscountRule
	err := r.db.QueryRow(ctx, `
		SELECT id, name, min_amount_cents, discount_percent, is_active, created_by, created_at, updated_at
		FROM discount_rules
		WHERE is_active = true AND min_amount_cents <= $1
		ORDER BY min_amount_cents DESC
		LIMIT 1
	`, amountCents).Scan(&d.ID, &d.Name, &d.MinAmountCents, &d.DiscountPercent, &d.IsActive, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}
