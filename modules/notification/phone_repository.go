package notification

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type PhoneRepository struct {
	db *pgxpool.Pool
}

func NewPhoneRepository(db *pgxpool.Pool) *PhoneRepository {
	return &PhoneRepository{db: db}
}

func (r *PhoneRepository) GetActivePhones(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT phone FROM notification_phones WHERE enabled = true ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var phones []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		phones = append(phones, p)
	}
	return phones, rows.Err()
}

func (r *PhoneRepository) List(ctx context.Context) ([]NotificationPhone, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, phone, label, enabled, created_at FROM notification_phones ORDER BY id`,
	)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()

	out := []NotificationPhone{}
	for rows.Next() {
		var p NotificationPhone
		if err := rows.Scan(&p.ID, &p.Phone, &p.Label, &p.Enabled, &p.CreatedAt); err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PhoneRepository) Add(ctx context.Context, phone, label string) (NotificationPhone, error) {
	var p NotificationPhone
	err := r.db.QueryRow(ctx, `
		INSERT INTO notification_phones (phone, label)
		VALUES ($1, $2)
		RETURNING id, phone, label, enabled, created_at
	`, phone, label).Scan(&p.ID, &p.Phone, &p.Label, &p.Enabled, &p.CreatedAt)
	if err != nil {
		return NotificationPhone{}, apperr.Internal(err)
	}
	return p, nil
}

func (r *PhoneRepository) Update(ctx context.Context, id int64, label *string, enabled *bool) (NotificationPhone, error) {
	var p NotificationPhone
	err := r.db.QueryRow(ctx, `
		UPDATE notification_phones
		SET label   = COALESCE($2, label),
		    enabled = COALESCE($3, enabled)
		WHERE id = $1
		RETURNING id, phone, label, enabled, created_at
	`, id, label, enabled).Scan(&p.ID, &p.Phone, &p.Label, &p.Enabled, &p.CreatedAt)
	if err == pgx.ErrNoRows {
		return NotificationPhone{}, apperr.NotFound("PHONE_NOT_FOUND", "phone not found")
	}
	if err != nil {
		return NotificationPhone{}, apperr.Internal(err)
	}
	return p, nil
}

func (r *PhoneRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM notification_phones WHERE id = $1`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("PHONE_NOT_FOUND", "phone not found")
	}
	return nil
}
