package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func isNotFound(err error) bool { return err == pgx.ErrNoRows }

func (r *Repository) Create(ctx context.Context, w *Worker) error {
	q := `
		INSERT INTO workers (name, position, department, phone, email, address, salary, hire_date, notes, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, is_active, created_at, updated_at
	`
	return r.db.QueryRow(ctx, q,
		w.Name, w.Position, w.Department, w.Phone, w.Email,
		w.Address, w.Salary, w.HireDate, w.Notes, w.IsActive,
	).Scan(&w.ID, &w.IsActive, &w.CreatedAt, &w.UpdatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Worker, error) {
	var w Worker
	q := `
		SELECT id, name, position, department, phone, email, address,
		       salary, hire_date, is_active, notes, created_at, updated_at, deleted_at
		FROM workers
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := r.db.QueryRow(ctx, q, id).Scan(
		&w.ID, &w.Name, &w.Position, &w.Department, &w.Phone, &w.Email, &w.Address,
		&w.Salary, &w.HireDate, &w.IsActive, &w.Notes, &w.CreatedAt, &w.UpdatedAt, &w.DeletedAt,
	)
	return w, err
}

func (r *Repository) List(ctx context.Context, limit, offset int, orderBy, orderDir, search string, activeOnly bool) ([]Worker, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	countSQL := `
		SELECT COUNT(*) FROM workers
		WHERE deleted_at IS NULL
		  AND ($2 = false OR is_active = true)
		  AND ($1 = '' OR
		       name ILIKE '%' || $1 || '%' OR
		       phone ILIKE '%' || $1 || '%' OR
		       email ILIKE '%' || $1 || '%')
	`
	var total int
	if err := r.db.QueryRow(ctx, countSQL, search, activeOnly).Scan(&total); err != nil {
		return nil, 0, err
	}

	col := "created_at"
	switch orderBy {
	case "name":
		col = "name"
	case "created_at":
		col = "created_at"
	}

	dir := "DESC"
	if orderDir == "asc" {
		dir = "ASC"
	}

	q := fmt.Sprintf(`
		SELECT id, name, position, department, phone, email, address,
		       salary, hire_date, is_active, notes, created_at, updated_at, deleted_at
		FROM workers
		WHERE deleted_at IS NULL
		  AND ($2 = false OR is_active = true)
		  AND ($1 = '' OR
		       name ILIKE '%%' || $1 || '%%' OR
		       phone ILIKE '%%' || $1 || '%%' OR
		       email ILIKE '%%' || $1 || '%%')
		ORDER BY %s %s
		LIMIT $3 OFFSET $4
	`, col, dir)

	rows, err := r.db.Query(ctx, q, search, activeOnly, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Worker
	for rows.Next() {
		var w Worker
		if err := rows.Scan(
			&w.ID, &w.Name, &w.Position, &w.Department, &w.Phone, &w.Email, &w.Address,
			&w.Salary, &w.HireDate, &w.IsActive, &w.Notes, &w.CreatedAt, &w.UpdatedAt, &w.DeletedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, w)
	}
	return out, total, rows.Err()
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateRequest) (Worker, error) {
	var hireDate *time.Time
	if req.HireDate != nil {
		if *req.HireDate == "" {
			hireDate = nil
		} else {
			t, err := time.Parse("2006-01-02", *req.HireDate)
			if err != nil {
				return Worker{}, err
			}
			hireDate = &t
		}
	}

	q := `
		UPDATE workers SET
			name       = COALESCE($1, name),
			position   = COALESCE($2, position),
			department = COALESCE($3, department),
			phone      = COALESCE($4, phone),
			email      = COALESCE($5, email),
			address    = COALESCE($6, address),
			salary     = COALESCE($7, salary),
			hire_date  = COALESCE($8, hire_date),
			is_active  = COALESCE($9, is_active),
			notes      = COALESCE($10, notes),
			updated_at = now()
		WHERE id = $11 AND deleted_at IS NULL
		RETURNING id, name, position, department, phone, email, address,
		          salary, hire_date, is_active, notes, created_at, updated_at, deleted_at
	`
	var w Worker
	err := r.db.QueryRow(ctx, q,
		req.Name, req.Position, req.Department, req.Phone, req.Email,
		req.Address, req.Salary, hireDate, req.IsActive, req.Notes, id,
	).Scan(
		&w.ID, &w.Name, &w.Position, &w.Department, &w.Phone, &w.Email, &w.Address,
		&w.Salary, &w.HireDate, &w.IsActive, &w.Notes, &w.CreatedAt, &w.UpdatedAt, &w.DeletedAt,
	)
	return w, err
}

// SoftDelete sets deleted_at=now() and is_active=false
func (r *Repository) SoftDelete(ctx context.Context, id int64) error {
	ct, err := r.db.Exec(ctx, `
		UPDATE workers
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
