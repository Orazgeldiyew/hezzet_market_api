package product

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StockRepository struct {
	db *pgxpool.Pool
}

func NewStockRepository(db *pgxpool.Pool) *StockRepository {
	return &StockRepository{db: db}
}

// GetStock returns total qty_milli across all warehouses for a product.
func (r *StockRepository) GetStock(ctx context.Context, productID int64) (int64, error) {
	var stock int64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(qty_milli), 0)
		FROM warehouse_items
		WHERE product_id = $1
	`, productID).Scan(&stock)
	return stock, err
}
