// modules/auth/repository.go
package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func isNotFound(err error) bool { return err == pgx.ErrNoRows }

// ---------- User CRUD ----------

func (r *Repository) CreateUser(ctx context.Context, u *User) error {
	q := `
		INSERT INTO users (username, password_hash, full_name, phone, email, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, q,
		u.Username, u.PasswordHash, u.FullName, u.Phone, u.Email, u.IsActive,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (User, error) {
	var u User
	q := `
		SELECT id, username, password_hash, full_name, phone, email,
		       is_active, created_at, updated_at, deleted_at
		FROM users
		WHERE username = $1 AND deleted_at IS NULL
	`
	err := r.db.QueryRow(ctx, q, username).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.FullName, &u.Phone, &u.Email,
		&u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	return u, err
}

func (r *Repository) GetByID(ctx context.Context, id int64) (User, error) {
	var u User
	q := `
		SELECT id, username, password_hash, full_name, phone, email,
		       is_active, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := r.db.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.FullName, &u.Phone, &u.Email,
		&u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	return u, err
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateUserRequest) (User, error) {
	q := `
		UPDATE users SET
			username  = COALESCE($1, username),
			full_name = COALESCE($2, full_name),
			phone     = COALESCE($3, phone),
			email     = COALESCE($4, email),
			is_active = COALESCE($5, is_active),
			updated_at = now()
		WHERE id = $6 AND deleted_at IS NULL
		RETURNING id, username, password_hash, full_name, phone, email,
		          is_active, created_at, updated_at, deleted_at
	`
	var u User
	err := r.db.QueryRow(ctx, q,
		req.Username, req.FullName, req.Phone, req.Email, req.IsActive, id,
	).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.FullName, &u.Phone, &u.Email,
		&u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	return u, err
}

func (r *Repository) SoftDelete(ctx context.Context, id int64) error {
	ct, err := r.db.Exec(ctx,
		`UPDATE users SET deleted_at = now(), is_active=false, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) UpdatePassword(ctx context.Context, id int64, hash string) error {
	ct, err := r.db.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2 AND deleted_at IS NULL`, hash, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
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
		SELECT COUNT(*) FROM users
		WHERE deleted_at IS NULL
		  AND ($1 = '' OR username ILIKE '%' || $1 || '%' OR full_name ILIKE '%' || $1 || '%')
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
		SELECT id, username, password_hash, full_name, phone, email,
		       is_active, created_at, updated_at, deleted_at
		FROM users
		WHERE deleted_at IS NULL
		  AND ($1 = '' OR username ILIKE '%%' || $1 || '%%' OR full_name ILIKE '%%' || $1 || '%%')
		ORDER BY %s %s
		LIMIT $2 OFFSET $3
	`, col, dir)

	rows, err := r.db.Query(ctx, q, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.PasswordHash, &u.FullName, &u.Phone, &u.Email,
			&u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
		); err != nil {
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

	return tx.Commit(ctx)
}
