package warehouse

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func isNotFound(err error) bool { return err == pgx.ErrNoRows }

func (r *Repository) Create(ctx context.Context, w *Warehouse) error {
	q := `
		INSERT INTO warehouses (name, address)
		VALUES ($1, $2)
		RETURNING id, is_active, created_at, updated_at
	`
	return r.db.QueryRow(ctx, q, w.Name, w.Address).
		Scan(&w.ID, &w.IsActive, &w.CreatedAt, &w.UpdatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Warehouse, error) {
	var w Warehouse
	q := `
		SELECT id, name, address, is_active, created_at, updated_at, deleted_at
		FROM warehouses
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := r.db.QueryRow(ctx, q, id).Scan(
		&w.ID, &w.Name, &w.Address, &w.IsActive,
		&w.CreatedAt, &w.UpdatedAt, &w.DeletedAt,
	)
	return w, err
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]Warehouse, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM warehouses WHERE deleted_at IS NULL AND is_active = TRUE
	`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, name, address, is_active, created_at, updated_at, deleted_at
		FROM warehouses
		WHERE deleted_at IS NULL AND is_active = TRUE
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Warehouse
	for rows.Next() {
		var w Warehouse
		if err := rows.Scan(
			&w.ID, &w.Name, &w.Address, &w.IsActive,
			&w.CreatedAt, &w.UpdatedAt, &w.DeletedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, w)
	}
	return out, total, rows.Err()
}
