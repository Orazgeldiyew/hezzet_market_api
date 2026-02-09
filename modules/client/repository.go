package client

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

func (r *Repository) Create(ctx context.Context, c *Client) error {
	q := `
		INSERT INTO clients (name, phone, email, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, is_active, created_at, updated_at
	`
	return r.db.QueryRow(ctx, q, c.Name, c.Phone, c.Email, c.IsActive).
		Scan(&c.ID, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Client, error) {
	var c Client
	q := `
		SELECT id, name, phone, email, is_active, created_at, updated_at
		FROM clients
		WHERE id=$1
	`
	err := r.db.QueryRow(ctx, q, id).
		Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *Repository) List(ctx context.Context, limit, offset int, q string) ([]Client, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// Count total matching rows
	countSQL := `
		SELECT COUNT(*) FROM clients
		WHERE is_active = true
		  AND ($1 = '' OR
		       name ILIKE '%' || $1 || '%' OR
		       phone ILIKE '%' || $1 || '%')
	`
	var total int
	if err := r.db.QueryRow(ctx, countSQL, q).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get data
	sql := `
		SELECT id, name, phone, email, is_active, created_at, updated_at
		FROM clients
		WHERE is_active = true
		  AND ($1 = '' OR
		       name ILIKE '%' || $1 || '%' OR
		       phone ILIKE '%' || $1 || '%')
		ORDER BY id DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, sql, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Client
	for rows.Next() {
		var c Client
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateRequest) (Client, error) {
	q := `
		UPDATE clients SET
			name = COALESCE($1, name),
			phone = COALESCE($2, phone),
			email = COALESCE($3, email),
			is_active = COALESCE($4, is_active),
			updated_at = now()
		WHERE id=$5
		RETURNING id, name, phone, email, is_active, created_at, updated_at
	`
	var c Client
	err := r.db.QueryRow(ctx, q,
		req.Name,
		req.Phone,
		req.Email,
		req.IsActive,
		id,
	).Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)

	return c, err
}

func (r *Repository) SoftDelete(ctx context.Context, id int64) error {
	ct, err := r.db.Exec(ctx,
		`UPDATE clients SET is_active=false, updated_at=now() WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
