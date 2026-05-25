package supplier

import (
	"context"
	"fmt"

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

// supplierCols is the shared column list — keep RETURNING/SELECT/Scan in sync.
const supplierCols = `id, user_id, name, COALESCE(legal_name,''), COALESCE(tax_id,''), phone, email, address, is_active, created_at`

func scanSupplier(row pgx.Row, s *Supplier) error {
	return row.Scan(&s.ID, &s.UserID, &s.Name, &s.LegalName, &s.TaxID, &s.Phone, &s.Email, &s.Address, &s.IsActive, &s.CreatedAt)
}

func (r *Repository) Create(ctx context.Context, s *Supplier) error {
	q := `
		INSERT INTO suppliers (name, legal_name, tax_id, phone, email, address, is_active, user_id)
		VALUES ($1, NULLIF($2,''), NULLIF($3,''), $4, $5, $6, $7, $8)
		RETURNING ` + supplierCols
	return scanSupplier(r.db.QueryRow(ctx, q,
		s.Name, s.LegalName, s.TaxID, s.Phone, s.Email, s.Address, s.IsActive, s.UserID,
	), s)
}

func (r *Repository) GetByID(ctx context.Context, id int) (Supplier, error) {
	var s Supplier
	q := `SELECT ` + supplierCols + ` FROM suppliers WHERE id=$1`
	err := scanSupplier(r.db.QueryRow(ctx, q, id), &s)
	return s, err
}

func (r *Repository) List(ctx context.Context, limit, offset int, orderBy, orderDir, q string) ([]Supplier, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// Count total matching rows
	countSQL := `
		SELECT COUNT(*) FROM suppliers
		WHERE is_active = true
		  AND ($1 = '' OR
		       name ILIKE '%' || $1 || '%' OR
		       phone ILIKE '%' || $1 || '%' OR
		       email ILIKE '%' || $1 || '%')
	`
	var total int
	if err := r.db.QueryRow(ctx, countSQL, q).Scan(&total); err != nil {
		return nil, 0, err
	}

	// whitelist order column
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

	sql := fmt.Sprintf(`
		SELECT %s FROM suppliers
		WHERE is_active = true
		  AND ($1 = '' OR
		       name ILIKE '%%' || $1 || '%%' OR
		       phone ILIKE '%%' || $1 || '%%' OR
		       email ILIKE '%%' || $1 || '%%')
		ORDER BY %s %s
		LIMIT $2 OFFSET $3
	`, supplierCols, col, dir)

	rows, err := r.db.Query(ctx, sql, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Supplier
	for rows.Next() {
		var s Supplier
		if err := scanSupplier(rows, &s); err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *Repository) Update(ctx context.Context, id int, req UpdateRequest) (Supplier, error) {
	q := `
		UPDATE suppliers SET
			name       = COALESCE($1, name),
			legal_name = COALESCE($2, legal_name),
			tax_id     = COALESCE($3, tax_id),
			phone      = COALESCE($4, phone),
			email      = COALESCE($5, email),
			address    = COALESCE($6, address),
			is_active  = COALESCE($7, is_active)
		WHERE id=$8
		RETURNING ` + supplierCols
	var s Supplier
	err := scanSupplier(r.db.QueryRow(ctx, q,
		req.Name, req.LegalName, req.TaxID, req.Phone, req.Email, req.Address, req.IsActive, id,
	), &s)
	return s, err
}

func (r *Repository) SoftDelete(ctx context.Context, id int) error {
	ct, err := r.db.Exec(ctx,
		`UPDATE suppliers SET is_active=false WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
