package warehouse

import (
	"context"
	"fmt"
	"strings"

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

// Update applies only the non-nil fields from UpdateRequest. Returns the row
// after the change, or pgx.ErrNoRows if the warehouse was missing/deleted.
func (r *Repository) Update(ctx context.Context, id int64, req UpdateRequest) (Warehouse, error) {
	sets := []string{}
	args := []any{id}
	idx := 2

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", idx))
		args = append(args, *req.Name)
		idx++
	}
	if req.Address != nil {
		sets = append(sets, fmt.Sprintf("address = $%d", idx))
		args = append(args, *req.Address)
		idx++
	}
	if req.IsActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *req.IsActive)
		idx++
	}

	if len(sets) == 0 {
		// Nothing to update — return the current row.
		return r.GetByID(ctx, id)
	}

	sets = append(sets, "updated_at = now()")
	q := fmt.Sprintf(`
		UPDATE warehouses SET %s
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, name, address, is_active, created_at, updated_at, deleted_at
	`, strings.Join(sets, ", "))

	var w Warehouse
	err := r.db.QueryRow(ctx, q, args...).Scan(
		&w.ID, &w.Name, &w.Address, &w.IsActive,
		&w.CreatedAt, &w.UpdatedAt, &w.DeletedAt,
	)
	return w, err
}

// HasInventory returns true if the warehouse has any product with a non-zero
// reserved or on-hand quantity. Used as a guard before soft-delete.
func (r *Repository) HasInventory(ctx context.Context, id int64) (bool, error) {
	var qty int64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(qty_milli), 0)
		FROM warehouse_items
		WHERE warehouse_id = $1
	`, id).Scan(&qty)
	if err != nil {
		return false, err
	}
	if qty != 0 {
		return true, nil
	}
	var reserved int64
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM stock_reservations
		WHERE warehouse_id = $1 AND status = 'active'
	`, id).Scan(&reserved)
	if err != nil {
		return false, err
	}
	return reserved > 0, nil
}

// SoftDelete marks the warehouse as deleted. Returns pgx.ErrNoRows if missing.
func (r *Repository) SoftDelete(ctx context.Context, id int64) error {
	ct, err := r.db.Exec(ctx, `
		UPDATE warehouses
		SET deleted_at = now(), is_active = FALSE, updated_at = now()
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
