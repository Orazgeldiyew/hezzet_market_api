package printer

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const cols = `id, name, ip_address, port, register_id, is_active, created_at, updated_at`

func scanPrinter(row pgx.Row) (Printer, error) {
	var p Printer
	err := row.Scan(
		&p.ID, &p.Name, &p.IPAddress, &p.Port, &p.RegisterID, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func (r *Repository) Create(ctx context.Context, req CreateRequest) (Printer, error) {
	port := req.Port
	if port == 0 {
		port = 9100
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}

	return scanPrinter(r.db.QueryRow(ctx, `
		INSERT INTO printers (name, ip_address, port, register_id, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+cols,
		req.Name, req.IPAddress, port, req.RegisterID, active,
	))
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Printer, error) {
	return scanPrinter(r.db.QueryRow(ctx,
		`SELECT `+cols+` FROM printers WHERE id = $1`, id,
	))
}

func (r *Repository) GetByRegisterID(ctx context.Context, registerID int64) (Printer, error) {
	return scanPrinter(r.db.QueryRow(ctx,
		`SELECT `+cols+` FROM printers WHERE register_id = $1 AND is_active = true LIMIT 1`,
		registerID,
	))
}

func (r *Repository) List(ctx context.Context) ([]Printer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+cols+` FROM printers ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Printer
	for rows.Next() {
		p, err := scanPrinter(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if out == nil {
		out = []Printer{}
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateRequest) (Printer, error) {
	return scanPrinter(r.db.QueryRow(ctx, `
		UPDATE printers SET
			name        = COALESCE($2, name),
			ip_address  = COALESCE($3, ip_address),
			port        = COALESCE($4, port),
			register_id = COALESCE($5, register_id),
			is_active   = COALESCE($6, is_active),
			updated_at  = now()
		WHERE id = $1
		RETURNING `+cols,
		id, req.Name, req.IPAddress, req.Port, req.RegisterID, req.IsActive,
	))
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM printers WHERE id = $1`, id)
	return err
}
