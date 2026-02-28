package auditlog

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const auditCols = `id, user_id, username, action, entity_type, entity_id,
	method, path, ip_address, request_id, created_at`

// Create inserts one audit log row. Errors are silently discarded — audit must
// never break the main request flow. Call this inside a goroutine.
func (r *Repository) Create(ctx context.Context, a *AuditLog) {
	r.db.Exec(ctx, `
		INSERT INTO audit_logs
			(user_id, username, action, entity_type, entity_id, method, path, ip_address, request_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, a.UserID, a.Username, a.Action, a.EntityType, a.EntityID,
		a.Method, a.Path, a.IPAddress, a.RequestID)
}

// List returns paginated audit logs filtered by the given criteria.
func (r *Repository) List(ctx context.Context, f AuditFilter, limit, offset int) ([]AuditLog, int, error) {
	var (
		where []string
		args  []any
		idx   = 1
	)

	if f.UserID != nil {
		where = append(where, fmt.Sprintf("user_id = $%d", idx))
		args = append(args, *f.UserID)
		idx++
	}
	if f.EntityType != "" {
		where = append(where, fmt.Sprintf("entity_type = $%d", idx))
		args = append(args, f.EntityType)
		idx++
	}
	if f.Action != "" {
		where = append(where, fmt.Sprintf("action = $%d", idx))
		args = append(args, f.Action)
		idx++
	}
	if f.From != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", idx))
		args = append(args, *f.From)
		idx++
	}
	if f.To != nil {
		where = append(where, fmt.Sprintf("created_at <= $%d", idx))
		args = append(args, *f.To)
		idx++
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE "
		for i, w := range where {
			if i > 0 {
				whereSQL += " AND "
			}
			whereSQL += w
		}
	}

	var total int
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM audit_logs "+whereSQL, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataQ := fmt.Sprintf(
		"SELECT %s FROM audit_logs %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		auditCols, whereSQL, idx, idx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []AuditLog
	for rows.Next() {
		var a AuditLog
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.Username, &a.Action, &a.EntityType, &a.EntityID,
			&a.Method, &a.Path, &a.IPAddress, &a.RequestID, &a.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	if out == nil {
		out = []AuditLog{}
	}
	return out, total, rows.Err()
}
