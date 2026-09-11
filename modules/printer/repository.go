package printer

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

const cols = `id, name, connection_type, COALESCE(ip_address, ''), port, register_id, is_active, created_at, updated_at`

func scanPrinter(row pgx.Row) (Printer, error) {
	var p Printer
	err := row.Scan(
		&p.ID, &p.Name, &p.ConnectionType, &p.IPAddress, &p.Port, &p.RegisterID, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func normalizeConnectionType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case ConnectionUSB:
		return ConnectionUSB
	default:
		return ConnectionNetwork
	}
}

func (r *Repository) Create(ctx context.Context, req CreateRequest) (Printer, error) {
	connType := normalizeConnectionType(req.ConnectionType)
	port := req.Port
	if port == 0 {
		port = 9100
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}

	ip := strings.TrimSpace(req.IPAddress)
	if connType == ConnectionNetwork {
		if ip == "" {
			return Printer{}, fmt.Errorf("ip_address is required for network printers")
		}
	} else {
		// USB: IP/port unused — store empty / default port.
		ip = ""
		port = 9100
	}

	var ipArg any
	if ip == "" {
		ipArg = nil
	} else {
		ipArg = ip
	}

	return scanPrinter(r.db.QueryRow(ctx, `
		INSERT INTO printers (name, connection_type, ip_address, port, register_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+cols,
		req.Name, connType, ipArg, port, req.RegisterID, active,
	))
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Printer, error) {
	return scanPrinter(r.db.QueryRow(ctx,
		`SELECT `+cols+` FROM printers WHERE id = $1`, id,
	))
}

func (r *Repository) GetByRegisterID(ctx context.Context, registerID int64) (Printer, error) {
	return scanPrinter(r.db.QueryRow(ctx,
		`SELECT `+cols+` FROM printers WHERE register_id = $1 AND is_active = true ORDER BY id LIMIT 1`,
		registerID,
	))
}

// GetFirstActive prefers a register-bound printer, then any active one.
func (r *Repository) GetFirstActive(ctx context.Context) (Printer, error) {
	return scanPrinter(r.db.QueryRow(ctx,
		`SELECT `+cols+` FROM printers WHERE is_active = true ORDER BY register_id NULLS LAST, id LIMIT 1`,
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
	cur, err := r.GetByID(ctx, id)
	if err != nil {
		return Printer{}, err
	}

	name := cur.Name
	if req.Name != nil {
		name = *req.Name
	}
	connType := cur.ConnectionType
	if req.ConnectionType != nil {
		connType = normalizeConnectionType(*req.ConnectionType)
	}
	ip := cur.IPAddress
	if req.IPAddress != nil {
		ip = strings.TrimSpace(*req.IPAddress)
	}
	port := cur.Port
	if req.Port != nil {
		port = *req.Port
	}
	if port == 0 {
		port = 9100
	}
	registerID := cur.RegisterID
	if req.RegisterID != nil {
		registerID = req.RegisterID
	}
	active := cur.IsActive
	if req.IsActive != nil {
		active = *req.IsActive
	}

	if connType == ConnectionNetwork {
		if ip == "" {
			return Printer{}, fmt.Errorf("ip_address is required for network printers")
		}
	} else {
		ip = ""
		port = 9100
	}

	var ipArg any
	if ip == "" {
		ipArg = nil
	} else {
		ipArg = ip
	}

	return scanPrinter(r.db.QueryRow(ctx, `
		UPDATE printers SET
			name            = $2,
			connection_type = $3,
			ip_address      = $4,
			port            = $5,
			register_id     = $6,
			is_active       = $7,
			updated_at      = now()
		WHERE id = $1
		RETURNING `+cols,
		id, name, connType, ipArg, port, registerID, active,
	))
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM printers WHERE id = $1`, id)
	return err
}
