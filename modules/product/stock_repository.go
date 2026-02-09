package product

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StockRepository struct {
	db *pgxpool.Pool
}

func NewStockRepository(db *pgxpool.Pool) *StockRepository { return &StockRepository{db: db} }

func (r *StockRepository) GetStock(ctx context.Context, productID int64) (float64, error) {
	q := `SELECT COALESCE(SUM(qty), 0) FROM stock_movements WHERE product_id=$1`
	var stock float64
	if err := r.db.QueryRow(ctx, q, productID).Scan(&stock); err != nil {
		return 0, err
	}
	return stock, nil
}
