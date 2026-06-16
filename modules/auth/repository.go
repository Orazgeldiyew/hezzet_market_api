// modules/auth/repository.go
package auth

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

// userScanDest returns the ordered scan destinations for a User row.
func userScanDest(u *User) []any {
	return []any{
		&u.ID, &u.Username, &u.PasswordHash, &u.FullName, &u.Phone, &u.Email,
		&u.IsActive, &u.BlockedAt, &u.BlockedReason, &u.PasswordChangedAt,
		&u.TokenVersion, &u.LastLoginAt, &u.CreatedBy, &u.UpdatedBy,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	}
}

const userCols = `id, username, password_hash, name, phone, email,
       is_active, blocked_at, blocked_reason, password_changed_at,
       token_version, last_login_at, created_by, updated_by,
       created_at, updated_at, deleted_at`

// ---------- User CRUD ----------

func (r *Repository) CreateUser(ctx context.Context, u *User) error {
	q := `
		INSERT INTO employees (username, password_hash, name, phone, email, is_active, created_by, has_account)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, q,
		u.Username, u.PasswordHash, u.FullName, u.Phone, u.Email, u.IsActive, u.CreatedBy,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

// CreateUserWithRoles inserts the user and assigns roles atomically.
// If role assignment fails, the user row is rolled back so username stays free
// and the caller can retry with the same payload.
func (r *Repository) CreateUserWithRoles(ctx context.Context, u *User, roleCodes []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := tx.QueryRow(ctx, `
		INSERT INTO employees (username, password_hash, name, phone, email, is_active, created_by, has_account)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true)
		RETURNING id, created_at, updated_at
	`,
		u.Username, u.PasswordHash, u.FullName, u.Phone, u.Email, u.IsActive, u.CreatedBy,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return err
	}

	for _, code := range roleCodes {
		ct, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id)
			SELECT $1, id FROM roles WHERE code = $2
		`, u.ID, code)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (User, error) {
	q := fmt.Sprintf(`SELECT %s FROM employees WHERE username = $1 AND deleted_at IS NULL`, userCols)
	var u User
	err := r.db.QueryRow(ctx, q, username).Scan(userScanDest(&u)...)
	return u, err
}

func (r *Repository) GetByID(ctx context.Context, id int64) (User, error) {
	q := fmt.Sprintf(`SELECT %s FROM employees WHERE id = $1 AND deleted_at IS NULL`, userCols)
	var u User
	err := r.db.QueryRow(ctx, q, id).Scan(userScanDest(&u)...)
	return u, err
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateUserRequest, updatedBy *int64) (User, error) {
	q := fmt.Sprintf(`
		UPDATE employees SET
			username   = COALESCE($1, username),
			name = COALESCE($2, name),
			phone      = COALESCE($3, phone),
			email      = COALESCE($4, email),
			is_active  = COALESCE($5, is_active),
			updated_by = $6,
			updated_at = now()
		WHERE id = $7 AND deleted_at IS NULL
		RETURNING %s
	`, userCols)

	var u User
	err := r.db.QueryRow(ctx, q,
		req.Username, req.FullName, req.Phone, req.Email, req.IsActive, updatedBy, id,
	).Scan(userScanDest(&u)...)
	return u, err
}

func (r *Repository) SoftDelete(ctx context.Context, id int64) error {
	ct, err := r.db.Exec(ctx,
		`UPDATE employees SET deleted_at = now(), is_active = false, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) UpdatePasswordAndBumpVersion(ctx context.Context, id int64, hash string, updatedBy int64) error {
	ct, err := r.db.Exec(ctx, `
		UPDATE employees SET
			password_hash       = $1,
			password_changed_at = now(),
			token_version       = token_version + 1,
			updated_by          = $2,
			updated_at          = now()
		WHERE id = $3 AND deleted_at IS NULL
	`, hash, updatedBy, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) UpdateLastLogin(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE employees SET last_login_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

// BumpTokenVersion atomically increments token_version and returns the new value.
// Called on every successful login to invalidate all prior sessions across all devices.
func (r *Repository) BumpTokenVersion(ctx context.Context, userID int64) (int, error) {
	var newVersion int
	err := r.db.QueryRow(ctx, `
		UPDATE employees
		SET token_version = token_version + 1, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING token_version
	`, userID).Scan(&newVersion)
	return newVersion, err
}

// ---------- Block / Unblock ----------

func (r *Repository) BlockUser(ctx context.Context, id int64, reason string, updatedBy int64) (User, error) {
	q := fmt.Sprintf(`
		UPDATE employees SET
			blocked_at     = now(),
			blocked_reason = $1,
			token_version  = token_version + 1,
			updated_by     = $2,
			updated_at     = now()
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING %s
	`, userCols)

	var u User
	err := r.db.QueryRow(ctx, q, reason, updatedBy, id).Scan(userScanDest(&u)...)
	return u, err
}

func (r *Repository) UnblockUser(ctx context.Context, id int64, updatedBy int64) (User, error) {
	// SECURITY: unblock changes security-state; bump token_version to invalidate tokens
	q := fmt.Sprintf(`
		UPDATE employees SET
			blocked_at     = NULL,
			blocked_reason = NULL,
			token_version  = token_version + 1,
			updated_by     = $1,
			updated_at     = now()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING %s
	`, userCols)

	var u User
	err := r.db.QueryRow(ctx, q, updatedBy, id).Scan(userScanDest(&u)...)
	return u, err
}

// ---------- Token version (used by middleware) ----------

// GetTokenVersion loads the user's current token_version.
// Returns an error if the user is deleted, disabled or blocked.
func (r *Repository) GetTokenVersion(ctx context.Context, userID int64) (int, error) {
	var version int
	var isActive bool
	var blockedAt *time.Time

	err := r.db.QueryRow(ctx, `
		SELECT token_version, is_active, blocked_at
		FROM employees
		WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(&version, &isActive, &blockedAt)
	if err != nil {
		return 0, err
	}
	if !isActive {
		return 0, fmt.Errorf("user disabled")
	}
	if blockedAt != nil {
		return 0, fmt.Errorf("user blocked")
	}
	return version, nil
}

// ---------- List ----------

func (r *Repository) List(ctx context.Context, limit, offset int, orderBy, orderDir, search string) ([]User, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	countQ := `
		SELECT COUNT(*) FROM employees
		WHERE deleted_at IS NULL AND has_account = true
		  AND ($1 = '' OR username ILIKE '%' || $1 || '%' OR name ILIKE '%' || $1 || '%')
	`
	var total int
	if err := r.db.QueryRow(ctx, countQ, search).Scan(&total); err != nil {
		return nil, 0, err
	}

	col := "created_at"
	switch orderBy {
	case "username":
		col = "username"
	case "created_at":
		col = "created_at"
	}

	dir := "DESC"
	if orderDir == "asc" {
		dir = "ASC"
	}

	q := fmt.Sprintf(`
		SELECT %s
		FROM employees
		WHERE deleted_at IS NULL AND has_account = true
		  AND ($1 = '' OR username ILIKE '%%' || $1 || '%%' OR name ILIKE '%%' || $1 || '%%')
		ORDER BY %s %s
		LIMIT $2 OFFSET $3
	`, userCols, col, dir)

	rows, err := r.db.Query(ctx, q, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(userScanDest(&u)...); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// ---------- Roles ----------

func (r *Repository) CountRolesByCodes(ctx context.Context, codes []string) (int, error) {
	q := `SELECT COUNT(*) FROM roles WHERE code = ANY($1)`
	var n int
	if err := r.db.QueryRow(ctx, q, codes).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (r *Repository) GetUserRoles(ctx context.Context, userID int64) ([]string, error) {
	q := `
		SELECT r.code
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY r.code
	`
	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		roles = append(roles, code)
	}
	return roles, rows.Err()
}

func (r *Repository) GetRolesByUserIDs(ctx context.Context, userIDs []int64) (map[int64][]string, error) {
	out := make(map[int64][]string, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}

	q := `
		SELECT ur.user_id, r.code
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = ANY($1)
		ORDER BY ur.user_id, r.code
	`
	rows, err := r.db.Query(ctx, q, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var uid int64
		var code string
		if err := rows.Scan(&uid, &code); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], code)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) SetUserRoles(ctx context.Context, userID int64, roleCodes []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID); err != nil {
		return err
	}

	for _, code := range roleCodes {
		ct, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id)
			SELECT $1, id FROM roles WHERE code = $2
		`, userID, code)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
	}

	// Bump token_version so the user must re-login to pick up new roles in JWT
	if _, err = tx.Exec(ctx, `
		UPDATE employees SET token_version = token_version + 1, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`, userID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}