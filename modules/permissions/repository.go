package permissions

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	cacheKeyFmt = "perm:v1:%s:%s"
	cacheTTL    = 5 * time.Minute
)

type Repository struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

func NewRepository(db *pgxpool.Pool, rdb *redis.Client) *Repository {
	return &Repository{db: db, rdb: rdb}
}

func cacheKey(role, module string) string {
	return fmt.Sprintf(cacheKeyFmt, role, module)
}

// IsEnabled checks if a role has access to a module.
// Checks Redis first (cache), falls back to DB on miss.
func (r *Repository) IsEnabled(ctx context.Context, role, module string) (bool, error) {
	// 1. Try Redis cache
	if r.rdb != nil {
		val, err := r.rdb.Get(ctx, cacheKey(role, module)).Result()
		if err == nil {
			return val == "true", nil
		}
	}

	// 2. Query DB
	var enabled bool
	err := r.db.QueryRow(ctx,
		`SELECT enabled FROM role_permissions WHERE role = $1 AND module = $2`,
		role, module,
	).Scan(&enabled)
	if err != nil {
		// Row not found means no explicit permission → default allow
		return true, nil
	}

	// 3. Cache result
	if r.rdb != nil {
		v := "false"
		if enabled {
			v = "true"
		}
		_ = r.rdb.Set(ctx, cacheKey(role, module), v, cacheTTL).Err()
	}

	return enabled, nil
}

// List returns all role_permissions rows.
func (r *Repository) List(ctx context.Context) ([]Permission, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, role, module, enabled, updated_at FROM role_permissions ORDER BY role, module`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Role, &p.Module, &p.Enabled, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if out == nil {
		out = []Permission{}
	}
	return out, rows.Err()
}

// Update sets enabled for a role+module and invalidates Redis cache.
func (r *Repository) Update(ctx context.Context, role, module string, enabled bool) (Permission, error) {
	var p Permission
	err := r.db.QueryRow(ctx, `
		INSERT INTO role_permissions (role, module, enabled, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (role, module) DO UPDATE
		SET enabled = $3, updated_at = now()
		RETURNING id, role, module, enabled, updated_at
	`, role, module, enabled).Scan(&p.ID, &p.Role, &p.Module, &p.Enabled, &p.UpdatedAt)
	if err != nil {
		return Permission{}, err
	}

	// Invalidate cache
	if r.rdb != nil {
		_ = r.rdb.Del(ctx, cacheKey(role, module)).Err()
	}

	return p, nil
}
