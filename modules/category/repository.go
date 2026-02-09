package category

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

func IsNotFound(err error) bool {
	return err == pgx.ErrNoRows
}

func (r *Repository) Create(ctx context.Context, c *Category) error {
	q := `
		INSERT INTO categories (name, parent_id, is_active)
		VALUES ($1, $2, $3)
		RETURNING id, is_active, created_at
	`
	return r.db.QueryRow(ctx, q, c.Name, c.ParentID, c.IsActive).
		Scan(&c.ID, &c.IsActive, &c.CreatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id int) (Category, error) {
	var c Category
	q := `
		SELECT id, name, parent_id, is_active, created_at
		FROM categories
		WHERE id=$1
	`
	err := r.db.QueryRow(ctx, q, id).
		Scan(&c.ID, &c.Name, &c.ParentID, &c.IsActive, &c.CreatedAt)
	return c, err
}

func (r *Repository) List(ctx context.Context, limit, offset int, orderBy, orderDir, q string) ([]Category, int, error){
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// Count total matching rows
	countSQL := `
		SELECT COUNT(*) FROM categories
		WHERE is_active = true
		  AND ($1 = '' OR name ILIKE '%' || $1 || '%')
	`
	var total int
	if err := r.db.QueryRow(ctx, countSQL, q).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get data
	sql := `
		SELECT id, name, parent_id, is_active, created_at
		FROM categories
		WHERE is_active = true
		  AND ($1 = '' OR name ILIKE '%' || $1 || '%')
		ORDER BY name
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, sql, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

func (r *Repository) ListAll(ctx context.Context) ([]Category, error) {
	sql := `
		SELECT id, name, parent_id, is_active, created_at
		FROM categories
		WHERE is_active = true
		ORDER BY name
	`
	rows, err := r.db.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, id int, req UpdateRequest) (Category, error) {
	q := `
		UPDATE categories SET
			name = COALESCE($1, name),
			parent_id = COALESCE($2, parent_id),
			is_active = COALESCE($3, is_active)
		WHERE id=$4
		RETURNING id, name, parent_id, is_active, created_at
	`
	var c Category
	err := r.db.QueryRow(ctx, q,
		req.Name,
		req.ParentID,
		req.IsActive,
		id,
	).Scan(&c.ID, &c.Name, &c.ParentID, &c.IsActive, &c.CreatedAt)

	return c, err
}

func (r *Repository) SoftDelete(ctx context.Context, id int) error {
	ct, err := r.db.Exec(ctx,
		`UPDATE categories SET is_active=false WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
