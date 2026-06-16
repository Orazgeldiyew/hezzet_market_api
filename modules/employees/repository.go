package employees

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func isNotFound(err error) bool { return err == pgx.ErrNoRows }

const empCols = `id, name, COALESCE(phone,''), COALESCE(email,''), COALESCE(address,''),
	COALESCE(position,''), COALESCE(department,''), salary, hire_date, COALESCE(notes,''),
	is_active, has_account, is_worker, username, created_at, updated_at, deleted_at`

func scanEmployee(row pgx.Row) (Employee, error) {
	var e Employee
	err := row.Scan(
		&e.ID, &e.Name, &e.Phone, &e.Email, &e.Address,
		&e.Position, &e.Department, &e.Salary, &e.HireDate, &e.Notes,
		&e.IsActive, &e.HasAccount, &e.IsWorker, &e.Username,
		&e.CreatedAt, &e.UpdatedAt, &e.DeletedAt,
	)
	return e, err
}

// GetByID returns a single employee with roles attached.
func (r *Repository) GetByID(ctx context.Context, id int64) (Employee, error) {
	q := fmt.Sprintf(`SELECT %s FROM employees WHERE id = $1 AND deleted_at IS NULL`, empCols)
	e, err := scanEmployee(r.db.QueryRow(ctx, q, id))
	if err != nil {
		return e, err
	}
	role, err := r.getRole(ctx, id)
	if err != nil {
		return e, err
	}
	e.Role = role
	return e, nil
}

func (r *Repository) getRole(ctx context.Context, id int64) (*string, error) {
	var code string
	err := r.db.QueryRow(ctx, `
		SELECT r.code FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1 LIMIT 1
	`, id).Scan(&code)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &code, nil
}

// List filters employees by is_worker/has_account/search/active.
func (r *Repository) List(ctx context.Context, f ListFilter, limit, offset int) ([]Employee, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	where := []string{"deleted_at IS NULL"}
	args := []any{}
	idx := 1

	if f.Search != "" {
		where = append(where, fmt.Sprintf(
			"(name ILIKE '%%' || $%d || '%%' OR COALESCE(username,'') ILIKE '%%' || $%d || '%%' OR COALESCE(phone,'') ILIKE '%%' || $%d || '%%' OR COALESCE(email,'') ILIKE '%%' || $%d || '%%')",
			idx, idx, idx, idx))
		args = append(args, f.Search)
		idx++
	}
	if f.IsWorker != nil {
		where = append(where, fmt.Sprintf("is_worker = $%d", idx))
		args = append(args, *f.IsWorker)
		idx++
	}
	if f.HasAccount != nil {
		where = append(where, fmt.Sprintf("has_account = $%d", idx))
		args = append(args, *f.HasAccount)
		idx++
	}
	if f.ActiveOnly {
		where = append(where, "is_active = true")
	}

	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM employees WHERE `+whereSQL, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	q := fmt.Sprintf(
		`SELECT %s FROM employees WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		empCols, whereSQL, idx, idx+1,
	)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Employee
	ids := []int64{}
	for rows.Next() {
		e, err := scanEmployee(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, e)
		ids = append(ids, e.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Bulk-load the single role for each returned employee. Schema enforces
	// at most one row per user_id (see migration 076), so we just take the
	// first hit per id.
	if len(ids) > 0 {
		roleRows, err := r.db.Query(ctx, `
			SELECT ur.user_id, r.code FROM user_roles ur
			JOIN roles r ON r.id = ur.role_id
			WHERE ur.user_id = ANY($1)
		`, ids)
		if err != nil {
			return nil, 0, err
		}
		defer roleRows.Close()
		roleByID := map[int64]string{}
		for roleRows.Next() {
			var uid int64
			var code string
			if err := roleRows.Scan(&uid, &code); err != nil {
				return nil, 0, err
			}
			roleByID[uid] = code
		}
		for i := range out {
			if c, ok := roleByID[out[i].ID]; ok {
				out[i].Role = &c
			}
		}
	}

	return out, total, nil
}

// Create inserts a new employee. If has_account=true, username + password
// must be provided; the caller passes a bcrypt hash via passwordHash.
func (r *Repository) Create(ctx context.Context, req CreateRequest, passwordHash *string, createdBy *int64) (Employee, error) {
	var hireDate *time.Time
	if req.HireDate != nil && *req.HireDate != "" {
		t, err := time.Parse("2006-01-02", *req.HireDate)
		if err != nil {
			return Employee{}, fmt.Errorf("invalid hire_date: %w", err)
		}
		hireDate = &t
	}

	salary := 0.0
	if req.Salary != nil {
		salary = *req.Salary
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Employee{}, err
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO employees (
			name, phone, email, address, position, department, salary, hire_date, notes,
			is_active, has_account, is_worker, username, password_hash, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9, true, $10, $11, $12, $13, $14)
		RETURNING id
	`,
		req.Name, nullStr(req.Phone), nullStr(req.Email), nullStr(req.Address),
		nullStr(req.Position), nullStr(req.Department), salary, hireDate, nullStr(req.Notes),
		req.HasAccount, req.IsWorker, req.Username, passwordHash, createdBy,
	).Scan(&id)
	if err != nil {
		return Employee{}, err
	}

	if req.HasAccount && req.Role != nil && *req.Role != "" {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id)
			SELECT $1, id FROM roles WHERE code = $2
		`, id, *req.Role); err != nil {
			return Employee{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Employee{}, err
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateRequest, updatedBy *int64) (Employee, error) {
	sets := []string{}
	args := []any{id}
	idx := 2

	add := func(col string, val any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, idx))
		args = append(args, val)
		idx++
	}

	if req.Name != nil {
		add("name", *req.Name)
	}
	if req.Phone != nil {
		add("phone", nullStr(*req.Phone))
	}
	if req.Email != nil {
		add("email", nullStr(*req.Email))
	}
	if req.Address != nil {
		add("address", nullStr(*req.Address))
	}
	if req.Position != nil {
		add("position", nullStr(*req.Position))
	}
	if req.Department != nil {
		add("department", nullStr(*req.Department))
	}
	if req.Salary != nil {
		add("salary", *req.Salary)
	}
	if req.HireDate != nil {
		if *req.HireDate == "" {
			add("hire_date", nil)
		} else {
			t, err := time.Parse("2006-01-02", *req.HireDate)
			if err != nil {
				return Employee{}, fmt.Errorf("invalid hire_date: %w", err)
			}
			add("hire_date", t)
		}
	}
	if req.Notes != nil {
		add("notes", nullStr(*req.Notes))
	}
	if req.IsActive != nil {
		add("is_active", *req.IsActive)
	}
	if req.IsWorker != nil {
		add("is_worker", *req.IsWorker)
	}
	if req.Username != nil {
		add("username", nullStr(*req.Username))
	}
	if updatedBy != nil {
		add("updated_by", *updatedBy)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Employee{}, err
	}
	defer tx.Rollback(ctx)

	if len(sets) > 0 {
		sets = append(sets, "updated_at = now()")
		q := fmt.Sprintf(`UPDATE employees SET %s WHERE id = $1 AND deleted_at IS NULL`,
			strings.Join(sets, ", "))
		ct, err := tx.Exec(ctx, q, args...)
		if err != nil {
			return Employee{}, err
		}
		if ct.RowsAffected() == 0 {
			return Employee{}, pgx.ErrNoRows
		}
	}

	if req.Role != nil {
		// Always wipe the old row first; a single-role schema means there's
		// at most one to drop. If req.Role is "" the caller is explicitly
		// removing the role; otherwise we insert the new one.
		if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, id); err != nil {
			return Employee{}, err
		}
		if *req.Role != "" {
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_roles (user_id, role_id)
				SELECT $1, id FROM roles WHERE code = $2
			`, id, *req.Role); err != nil {
				return Employee{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Employee{}, err
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) SoftDelete(ctx context.Context, id int64) error {
	ct, err := r.db.Exec(ctx, `
		UPDATE employees SET deleted_at = now(), is_active = false, updated_at = now()
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

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
