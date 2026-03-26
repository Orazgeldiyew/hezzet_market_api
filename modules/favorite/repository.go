package favorite

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, userID int64) ([]FavoriteProduct, error) {
	rows, err := r.db.Query(ctx, `
		SELECT fp.id, fp.user_id, fp.product_id, fp.position, fp.created_at,
		       p.name, p.sale_price, p.photo_path
		FROM favorite_products fp
		JOIN products p ON p.id = fp.product_id
		WHERE fp.user_id = $1 AND p.is_active = true AND p.deleted_at IS NULL
		ORDER BY fp.position, fp.created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []FavoriteProduct{}
	for rows.Next() {
		var f FavoriteProduct
		if err := rows.Scan(&f.ID, &f.UserID, &f.ProductID, &f.Position, &f.CreatedAt,
			&f.Name, &f.PriceCents, &f.PhotoURL); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repository) Add(ctx context.Context, userID, productID int64) (FavoriteProduct, error) {
	// Get next position
	var maxPos int
	_ = r.db.QueryRow(ctx,
		`SELECT COALESCE(MAX(position), -1) FROM favorite_products WHERE user_id = $1`,
		userID,
	).Scan(&maxPos)

	var f FavoriteProduct
	err := r.db.QueryRow(ctx, `
		INSERT INTO favorite_products (user_id, product_id, position)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, product_id) DO NOTHING
		RETURNING id, user_id, product_id, position, created_at
	`, userID, productID, maxPos+1).Scan(&f.ID, &f.UserID, &f.ProductID, &f.Position, &f.CreatedAt)
	if err != nil {
		return FavoriteProduct{}, apperr.Conflict("ALREADY_FAVORITE", "product is already in favorites")
	}

	// Fill product info
	_ = r.db.QueryRow(ctx,
		`SELECT name, sale_price, photo_path FROM products WHERE id = $1`,
		productID,
	).Scan(&f.Name, &f.PriceCents, &f.PhotoURL)

	return f, nil
}

func (r *Repository) Remove(ctx context.Context, userID, productID int64) error {
	ct, err := r.db.Exec(ctx,
		`DELETE FROM favorite_products WHERE user_id = $1 AND product_id = $2`,
		userID, productID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return apperr.NotFound("NOT_FOUND", "favorite not found")
	}
	return nil
}

func (r *Repository) Reorder(ctx context.Context, userID int64, items []ReorderItem) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, item := range items {
		_, err := tx.Exec(ctx,
			`UPDATE favorite_products SET position = $3 WHERE user_id = $1 AND product_id = $2`,
			userID, item.ProductID, item.Position,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
