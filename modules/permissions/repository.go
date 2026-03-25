package permissions

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	cacheKeyFmt = "perm:v2:%s:%s:%s"
	cacheTTL    = 5 * time.Minute
)

type Repository struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

func NewRepository(db *pgxpool.Pool, rdb *redis.Client) *Repository {
	return &Repository{db: db, rdb: rdb}
}

func cacheKey(roleCode, module, action string) string {
	return fmt.Sprintf(cacheKeyFmt, roleCode, module, action)
}

// ──────────────────────────────────────────────
// Permission checking (used by middleware)
// ──────────────────────────────────────────────

// IsAllowed checks if ANY of the given role codes has granted=true for module+action.
// Returns true if no explicit row exists (default allow).
func (r *Repository) IsAllowed(ctx context.Context, roleCodes []string, module, action string) (bool, error) {
	for _, code := range roleCodes {
		// 1. Try Redis cache
		if r.rdb != nil {
			val, err := r.rdb.Get(ctx, cacheKey(code, module, action)).Result()
			if err == nil {
				if val == "1" {
					return true, nil
				}
				continue // cached as denied, check next role
			}
		}

		// 2. Query DB
		var granted bool
		err := r.db.QueryRow(ctx, `
			SELECT rp.granted
			FROM role_permissions rp
			JOIN roles ro ON ro.id = rp.role_id
			WHERE ro.code = $1 AND rp.module = $2 AND rp.action = $3
		`, code, module, action).Scan(&granted)

		if err != nil {
			if err == pgx.ErrNoRows {
				// No explicit permission → default allow
				r.cacheSet(ctx, code, module, action, true)
				return true, nil
			}
			return false, err
		}

		// 3. Cache result
		r.cacheSet(ctx, code, module, action, granted)

		if granted {
			return true, nil
		}
	}

	return false, nil
}

// IsEnabled provides backward compatibility with ModuleChecker interface.
func (r *Repository) IsEnabled(ctx context.Context, role, module string) (bool, error) {
	return r.IsAllowed(ctx, []string{role}, module, "view")
}

func (r *Repository) cacheSet(ctx context.Context, roleCode, module, action string, granted bool) {
	if r.rdb == nil {
		return
	}
	v := "0"
	if granted {
		v = "1"
	}
	_ = r.rdb.Set(ctx, cacheKey(roleCode, module, action), v, cacheTTL).Err()
}

func (r *Repository) invalidateRoleCache(ctx context.Context, roleCode string) {
	if r.rdb == nil {
		return
	}
	pattern := fmt.Sprintf("perm:v2:%s:*", roleCode)
	iter := r.rdb.Scan(ctx, 0, pattern, 200).Iterator()
	for iter.Next(ctx) {
		_ = r.rdb.Del(ctx, iter.Val()).Err()
	}
}

// ──────────────────────────────────────────────
// Role CRUD
// ──────────────────────────────────────────────

func (r *Repository) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, code, name, description, is_system, created_at, updated_at
		FROM roles ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Code, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, role)
	}
	if out == nil {
		out = []Role{}
	}
	return out, rows.Err()
}

func (r *Repository) GetRole(ctx context.Context, id int) (Role, error) {
	var role Role
	err := r.db.QueryRow(ctx, `
		SELECT id, code, name, description, is_system, created_at, updated_at
		FROM roles WHERE id = $1
	`, id).Scan(&role.ID, &role.Code, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt)
	return role, err
}

func (r *Repository) CreateRole(ctx context.Context, req CreateRoleRequest) (Role, error) {
	var role Role
	err := r.db.QueryRow(ctx, `
		INSERT INTO roles (code, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, code, name, description, is_system, created_at, updated_at
	`, req.Code, req.Name, req.Description).Scan(
		&role.ID, &role.Code, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
	)
	return role, err
}

func (r *Repository) UpdateRole(ctx context.Context, id int, req UpdateRoleRequest) (Role, error) {
	var role Role
	err := r.db.QueryRow(ctx, `
		UPDATE roles SET
			name        = COALESCE($2, name),
			description = COALESCE($3, description),
			updated_at  = now()
		WHERE id = $1
		RETURNING id, code, name, description, is_system, created_at, updated_at
	`, id, req.Name, req.Description).Scan(
		&role.ID, &role.Code, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
	)
	return role, err
}

func (r *Repository) DeleteRole(ctx context.Context, id int) error {
	ct, err := r.db.Exec(ctx, `DELETE FROM roles WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) IsRoleAssigned(ctx context.Context, roleID int) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM user_roles WHERE role_id = $1
	`, roleID).Scan(&count)
	return count > 0, err
}

// ──────────────────────────────────────────────
// Permission CRUD
// ──────────────────────────────────────────────

func (r *Repository) ListPermissions(ctx context.Context) ([]Permission, error) {
	rows, err := r.db.Query(ctx, `
		SELECT rp.id, rp.role_id, ro.code, rp.module, rp.action, rp.granted, rp.updated_at
		FROM role_permissions rp
		JOIN roles ro ON ro.id = rp.role_id
		ORDER BY ro.code, rp.module, rp.action
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.RoleID, &p.RoleCode, &p.Module, &p.Action, &p.Granted, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if out == nil {
		out = []Permission{}
	}
	return out, rows.Err()
}

func (r *Repository) GetPermissionsByRole(ctx context.Context, roleID int) ([]Permission, error) {
	rows, err := r.db.Query(ctx, `
		SELECT rp.id, rp.role_id, ro.code, rp.module, rp.action, rp.granted, rp.updated_at
		FROM role_permissions rp
		JOIN roles ro ON ro.id = rp.role_id
		WHERE rp.role_id = $1
		ORDER BY rp.module, rp.action
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.RoleID, &p.RoleCode, &p.Module, &p.Action, &p.Granted, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if out == nil {
		out = []Permission{}
	}
	return out, rows.Err()
}

func (r *Repository) UpdatePermission(ctx context.Context, roleID int, module, action string, granted bool) (Permission, error) {
	var p Permission
	err := r.db.QueryRow(ctx, `
		INSERT INTO role_permissions (role_id, module, action, granted, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (role_id, module, action) DO UPDATE
		SET granted = $4, updated_at = now()
		RETURNING id, role_id, module, action, granted, updated_at
	`, roleID, module, action, granted).Scan(&p.ID, &p.RoleID, &p.Module, &p.Action, &p.Granted, &p.UpdatedAt)
	if err != nil {
		return Permission{}, err
	}

	// Get role code for cache invalidation
	var roleCode string
	_ = r.db.QueryRow(ctx, `SELECT code FROM roles WHERE id = $1`, roleID).Scan(&roleCode)
	if roleCode != "" {
		r.invalidateRoleCache(ctx, roleCode)
	}

	p.RoleCode = roleCode
	return p, nil
}

func (r *Repository) BulkUpdatePermissions(ctx context.Context, roleID int, items []BulkPermissionItem) ([]Permission, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var results []Permission
	for _, item := range items {
		var p Permission
		err := tx.QueryRow(ctx, `
			INSERT INTO role_permissions (role_id, module, action, granted, updated_at)
			VALUES ($1, $2, $3, $4, now())
			ON CONFLICT (role_id, module, action) DO UPDATE
			SET granted = $4, updated_at = now()
			RETURNING id, role_id, module, action, granted, updated_at
		`, roleID, item.Module, item.Action, item.Granted).Scan(
			&p.ID, &p.RoleID, &p.Module, &p.Action, &p.Granted, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, p)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Invalidate cache
	var roleCode string
	_ = r.db.QueryRow(ctx, `SELECT code FROM roles WHERE id = $1`, roleID).Scan(&roleCode)
	if roleCode != "" {
		r.invalidateRoleCache(ctx, roleCode)
	}

	return results, nil
}

// MatrixForRoles returns the matrix filtered to the given role codes, merging with OR logic.
func (r *Repository) MatrixForRoles(ctx context.Context, roleCodes []string) ([]MatrixEntry, error) {
	if len(roleCodes) == 0 {
		return []MatrixEntry{}, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT rp.module,
			   COALESCE(bool_or(rp.action = 'view'   AND rp.granted), false) AS can_view,
			   COALESCE(bool_or(rp.action = 'create'  AND rp.granted), false) AS can_create,
			   COALESCE(bool_or(rp.action = 'update'  AND rp.granted), false) AS can_update,
			   COALESCE(bool_or(rp.action = 'delete'  AND rp.granted), false) AS can_delete
		FROM role_permissions rp
		JOIN roles ro ON ro.id = rp.role_id
		WHERE ro.code = ANY($1)
		GROUP BY rp.module
		ORDER BY rp.module
	`, roleCodes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MatrixEntry
	for rows.Next() {
		var m MatrixEntry
		if err := rows.Scan(&m.Module, &m.View, &m.Create, &m.Update, &m.Delete); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if out == nil {
		out = []MatrixEntry{}
	}
	return out, rows.Err()
}

// Matrix returns a compact role x module x actions view.
func (r *Repository) Matrix(ctx context.Context) ([]MatrixEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ro.id, ro.code, ro.name, rp.module,
			   COALESCE(bool_or(rp.action = 'view'   AND rp.granted), false) AS can_view,
			   COALESCE(bool_or(rp.action = 'create'  AND rp.granted), false) AS can_create,
			   COALESCE(bool_or(rp.action = 'update'  AND rp.granted), false) AS can_update,
			   COALESCE(bool_or(rp.action = 'delete'  AND rp.granted), false) AS can_delete
		FROM role_permissions rp
		JOIN roles ro ON ro.id = rp.role_id
		GROUP BY ro.id, ro.code, ro.name, rp.module
		ORDER BY ro.id, rp.module
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MatrixEntry
	for rows.Next() {
		var m MatrixEntry
		if err := rows.Scan(&m.RoleID, &m.RoleCode, &m.RoleName, &m.Module, &m.View, &m.Create, &m.Update, &m.Delete); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if out == nil {
		out = []MatrixEntry{}
	}
	return out, rows.Err()
}
