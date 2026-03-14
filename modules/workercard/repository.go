package workercard

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, workerID *int64) ([]WorkerCard, error) {
	q := `
		SELECT id, worker_id, card_code, label, enabled, created_at
		FROM worker_cards
		WHERE ($1::bigint IS NULL OR worker_id = $1)
		ORDER BY id
	`
	rows, err := r.db.Query(ctx, q, workerID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()

	out := []WorkerCard{}
	for rows.Next() {
		var c WorkerCard
		if err := rows.Scan(&c.ID, &c.WorkerID, &c.CardCode, &c.Label, &c.Enabled, &c.CreatedAt); err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetByCardCode finds an enabled worker card by its QR/barcode code.
func (r *Repository) GetByCardCode(ctx context.Context, code string) (WorkerCard, error) {
	var c WorkerCard
	err := r.db.QueryRow(ctx, `
		SELECT id, worker_id, card_code, label, enabled, created_at
		FROM worker_cards
		WHERE card_code = $1 AND enabled = true
	`, code).Scan(&c.ID, &c.WorkerID, &c.CardCode, &c.Label, &c.Enabled, &c.CreatedAt)
	if err == pgx.ErrNoRows {
		return WorkerCard{}, apperr.NotFound("CARD_NOT_FOUND", "worker card not found")
	}
	if err != nil {
		return WorkerCard{}, apperr.Internal(err)
	}
	return c, nil
}

func (r *Repository) Add(ctx context.Context, workerID int64, label string) (WorkerCard, error) {
	var c WorkerCard
	err := r.db.QueryRow(ctx, `
		INSERT INTO worker_cards (worker_id, label, card_code)
		VALUES ($1, $2, upper(substring(md5(gen_random_uuid()::text), 1, 8)))
		RETURNING id, worker_id, card_code, label, enabled, created_at
	`, workerID, label).Scan(&c.ID, &c.WorkerID, &c.CardCode, &c.Label, &c.Enabled, &c.CreatedAt)
	if err != nil {
		return WorkerCard{}, apperr.Internal(err)
	}
	return c, nil
}

func (r *Repository) Update(ctx context.Context, id int64, label *string, enabled *bool) (WorkerCard, error) {
	var c WorkerCard
	err := r.db.QueryRow(ctx, `
		UPDATE worker_cards
		SET label   = COALESCE($2, label),
		    enabled = COALESCE($3, enabled)
		WHERE id = $1
		RETURNING id, worker_id, card_code, label, enabled, created_at
	`, id, label, enabled).Scan(&c.ID, &c.WorkerID, &c.CardCode, &c.Label, &c.Enabled, &c.CreatedAt)
	if err == pgx.ErrNoRows {
		return WorkerCard{}, apperr.NotFound("CARD_NOT_FOUND", "worker card not found")
	}
	if err != nil {
		return WorkerCard{}, apperr.Internal(err)
	}
	return c, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM worker_cards WHERE id = $1`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("CARD_NOT_FOUND", "worker card not found")
	}
	return nil
}
