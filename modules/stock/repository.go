package stock

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// ── helpers ──────────────────────────────────────────────────────────────────

const detailCols = `id, idempotency_key::text, warehouse_id, product_id,
	delta_milli, type::text, price_cents, worker_id, created_by, created_at,
	supplier_id, note`

// ✅ добавили avg_cost_cents + total_cost_cents
const itemCols = `warehouse_id, product_id, qty_milli, avg_cost_cents, total_cost_cents, updated_at`

func scanDetail(row pgx.Row) (WarehouseItemDetail, error) {
	var d WarehouseItemDetail
	err := row.Scan(
		&d.ID, &d.IdempotencyKey, &d.WarehouseID, &d.ProductID,
		&d.DeltaMilli, &d.Type, &d.PriceCents, &d.WorkerID, &d.CreatedBy, &d.CreatedAt,
		&d.SupplierID, &d.Note,
	)
	return d, err
}

func scanItem(row pgx.Row) (WarehouseItem, error) {
	var it WarehouseItem
	err := row.Scan(
		&it.WarehouseID, &it.ProductID, &it.QtyMilli,
		&it.AvgCostCents, &it.TotalCostCents,
		&it.UpdatedAt,
	)
	return it, err
}

func isDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func normalizeType(t string) string {
	return strings.TrimSpace(strings.ToLower(t))
}

// qty_milli SCALE=1000:
// total_cents = qty_milli * unit_price_cents / 1000
func lineTotalCents(qtyMilli int64, unitPriceCents int64) int64 {
	return (qtyMilli * unitPriceCents) / 1000
}

// checkIdempotencyStrict: strict idempotency on (warehouse_id + idempotency_key), compares type+product_id+delta_milli.
func (r *Repository) checkIdempotencyStrict(
	ctx context.Context,
	key string,
	warehouseID int64,
	wantType string,
	wantProductID int64,
	wantDelta int64,
) (WarehouseItemDetail, bool, error) {

	existing, err := scanDetail(r.db.QueryRow(ctx, `
		SELECT `+detailCols+`
		FROM warehouse_item_details
		WHERE idempotency_key = $1 AND warehouse_id = $2
	`, key, warehouseID))

	if err != nil {
		if err == pgx.ErrNoRows {
			return WarehouseItemDetail{}, false, nil
		}
		return WarehouseItemDetail{}, false, err
	}

	if normalizeType(existing.Type) != normalizeType(wantType) ||
		existing.ProductID != wantProductID ||
		existing.DeltaMilli != wantDelta {
		return WarehouseItemDetail{}, true, apperr.Conflict(
			"IDEMPOTENCY_KEY_CONFLICT",
			"idempotency_key already used with different parameters",
		)
	}

	return existing, true, nil
}

// ── StockIn (cost-aware) ─────────────────────────────────────────────────────

func (r *Repository) StockIn(ctx context.Context, req InRequest, userID int64) (WarehouseItemDetail, WarehouseItem, error) {
	// strict idempotency
	if existing, exists, err := r.checkIdempotencyStrict(ctx, req.IdempotencyKey, req.WarehouseID, "in", req.ProductID, req.QtyMilli); err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	} else if exists {
		it, err := r.getItem(ctx, req.WarehouseID, req.ProductID)
		return existing, it, err
	}

	// ✅ для cost-aware IN нужна закупочная unit price
	if req.PriceCents == nil || *req.PriceCents < 0 {
		return WarehouseItemDetail{}, WarehouseItem{}, apperr.Validation("price_cents is required for stock in (unit price)")
	}
	unitPrice := *req.PriceCents
	inCost := lineTotalCents(req.QtyMilli, unitPrice)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}
	defer tx.Rollback(ctx)

	// Insert detail (price_cents = unit buy price)
	detail, err := scanDetail(tx.QueryRow(ctx, `
		INSERT INTO warehouse_item_details
			(idempotency_key, warehouse_id, product_id, delta_milli, type, price_cents, worker_id, created_by)
		VALUES ($1, $2, $3, $4, 'in', $5, $6, $7)
		RETURNING `+detailCols,
		req.IdempotencyKey, req.WarehouseID, req.ProductID,
		req.QtyMilli, req.PriceCents, req.WorkerID, userID,
	))
	if err != nil {
		if isDuplicateKey(err) {
			existing, _, e2 := r.checkIdempotencyStrict(ctx, req.IdempotencyKey, req.WarehouseID, "in", req.ProductID, req.QtyMilli)
			if e2 != nil {
				return WarehouseItemDetail{}, WarehouseItem{}, e2
			}
			it, e3 := r.getItem(ctx, req.WarehouseID, req.ProductID)
			return existing, it, e3
		}
		if isFKViolation(err) {
			return WarehouseItemDetail{}, WarehouseItem{}, apperr.Validation("warehouse or product does not exist")
		}
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}

	// ✅ Upsert item cache + costing
	// new_total_cost = old_total_cost + inCost
	// new_qty = old_qty + qty
	// new_avg_cost = (new_total_cost*1000)/new_qty
	item, err := scanItem(tx.QueryRow(ctx, `
		INSERT INTO warehouse_items (warehouse_id, product_id, qty_milli, avg_cost_cents, total_cost_cents, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (warehouse_id, product_id)
		DO UPDATE SET
			qty_milli = warehouse_items.qty_milli + EXCLUDED.qty_milli,
			total_cost_cents = warehouse_items.total_cost_cents + $5,
			avg_cost_cents = CASE
				WHEN (warehouse_items.qty_milli + EXCLUDED.qty_milli) > 0
					THEN ((warehouse_items.total_cost_cents + $5) * 1000) / (warehouse_items.qty_milli + EXCLUDED.qty_milli)
				ELSE 0
			END,
			updated_at = now()
		RETURNING `+itemCols,
		req.WarehouseID, req.ProductID, req.QtyMilli,
		unitPrice, inCost,
	))
	if err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}
	return detail, item, nil
}

// ── StockOut (cost-aware) ────────────────────────────────────────────────────

func (r *Repository) StockOut(ctx context.Context, req OutRequest, userID int64) (WarehouseItemDetail, WarehouseItem, error) {
	wantDelta := -req.QtyMilli

	if existing, exists, err := r.checkIdempotencyStrict(ctx, req.IdempotencyKey, req.WarehouseID, "out", req.ProductID, wantDelta); err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	} else if exists {
		it, err := r.getItem(ctx, req.WarehouseID, req.ProductID)
		return existing, it, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}
	defer tx.Rollback(ctx)

	// ✅ lock item row + get avg_cost and total_cost
	var currentQty, avgCost, totalCost int64
	err = tx.QueryRow(ctx, `
		SELECT qty_milli, avg_cost_cents, total_cost_cents
		FROM warehouse_items
		WHERE warehouse_id = $1 AND product_id = $2
		FOR UPDATE
	`, req.WarehouseID, req.ProductID).Scan(&currentQty, &avgCost, &totalCost)
	if err != nil {
		if err == pgx.ErrNoRows {
			return WarehouseItemDetail{}, WarehouseItem{}, apperr.Validation("no stock for this product in the given warehouse")
		}
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}
	if currentQty < req.QtyMilli {
		return WarehouseItemDetail{}, WarehouseItem{}, apperr.Validation("insufficient stock")
	}

	// cost to remove by average cost
	outCost := lineTotalCents(req.QtyMilli, avgCost)
	if totalCost < outCost {
		// safety (should not happen if logic is correct)
		return WarehouseItemDetail{}, WarehouseItem{}, apperr.Internal(errors.New("total_cost_cents underflow"))
	}

	// Insert detail (price_cents can be sale unit price, optional)
	detail, err := scanDetail(tx.QueryRow(ctx, `
		INSERT INTO warehouse_item_details
			(idempotency_key, warehouse_id, product_id, delta_milli, type, price_cents, worker_id, created_by)
		VALUES ($1, $2, $3, $4, 'out', $5, $6, $7)
		RETURNING `+detailCols,
		req.IdempotencyKey, req.WarehouseID, req.ProductID,
		wantDelta, req.PriceCents, req.WorkerID, userID,
	))
	if err != nil {
		if isDuplicateKey(err) {
			existing, _, e2 := r.checkIdempotencyStrict(ctx, req.IdempotencyKey, req.WarehouseID, "out", req.ProductID, wantDelta)
			if e2 != nil {
				return WarehouseItemDetail{}, WarehouseItem{}, e2
			}
			it, e3 := r.getItem(ctx, req.WarehouseID, req.ProductID)
			return existing, it, e3
		}
		if isFKViolation(err) {
			return WarehouseItemDetail{}, WarehouseItem{}, apperr.Validation("warehouse or product does not exist")
		}
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}

	// ✅ update qty + total_cost; recalc avg_cost from remaining
	item, err := scanItem(tx.QueryRow(ctx, `
		UPDATE warehouse_items
		SET
			qty_milli = qty_milli - $3,
			total_cost_cents = total_cost_cents - $4,
			avg_cost_cents = CASE
				WHEN (qty_milli - $3) > 0
					THEN ((total_cost_cents - $4) * 1000) / (qty_milli - $3)
				ELSE 0
			END,
			updated_at = now()
		WHERE warehouse_id = $1 AND product_id = $2
		RETURNING `+itemCols,
		req.WarehouseID, req.ProductID, req.QtyMilli, outCost,
	))
	if err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}
	return detail, item, nil
}

// ── Transfer (cost-aware) ────────────────────────────────────────────────────

func (r *Repository) Transfer(ctx context.Context, req TransferRequest, userID int64) (TransferResult, error) {
	if _, exists, err := r.checkIdempotencyStrict(ctx, req.IdempotencyKey, req.FromWarehouseID, "transfer_out", req.ProductID, -req.QtyMilli); err != nil {
		return TransferResult{}, err
	} else if exists {
		return r.existingTransferResult(ctx, req)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return TransferResult{}, err
	}
	defer tx.Rollback(ctx)

	// lock source and get avg cost for costing transfer
	var srcQty, srcAvgCost, srcTotalCost int64
	err = tx.QueryRow(ctx, `
		SELECT qty_milli, avg_cost_cents, total_cost_cents
		FROM warehouse_items
		WHERE warehouse_id = $1 AND product_id = $2
		FOR UPDATE
	`, req.FromWarehouseID, req.ProductID).Scan(&srcQty, &srcAvgCost, &srcTotalCost)
	if err != nil {
		if err == pgx.ErrNoRows {
			return TransferResult{}, apperr.Validation("no stock for this product in source warehouse")
		}
		return TransferResult{}, err
	}
	if srcQty < req.QtyMilli {
		return TransferResult{}, apperr.Validation("insufficient stock in source warehouse")
	}
	moveCost := lineTotalCents(req.QtyMilli, srcAvgCost)
	if srcTotalCost < moveCost {
		return TransferResult{}, apperr.Internal(errors.New("total_cost_cents underflow"))
	}

	// OUT detail
	outDetail, err := scanDetail(tx.QueryRow(ctx, `
		INSERT INTO warehouse_item_details
			(idempotency_key, warehouse_id, product_id, delta_milli, type, created_by)
		VALUES ($1, $2, $3, $4, 'transfer_out', $5)
		RETURNING `+detailCols,
		req.IdempotencyKey, req.FromWarehouseID, req.ProductID,
		-req.QtyMilli, userID,
	))
	if err != nil {
		if isDuplicateKey(err) {
			return r.existingTransferResult(ctx, req)
		}
		if isFKViolation(err) {
			return TransferResult{}, apperr.Validation("source warehouse or product does not exist")
		}
		return TransferResult{}, err
	}

	// IN detail
	inDetail, err := scanDetail(tx.QueryRow(ctx, `
		INSERT INTO warehouse_item_details
			(idempotency_key, warehouse_id, product_id, delta_milli, type, created_by)
		VALUES ($1, $2, $3, $4, 'transfer_in', $5)
		RETURNING `+detailCols,
		req.IdempotencyKey, req.ToWarehouseID, req.ProductID,
		req.QtyMilli, userID,
	))
	if err != nil {
		if isFKViolation(err) {
			return TransferResult{}, apperr.Validation("destination warehouse or product does not exist")
		}
		return TransferResult{}, err
	}

	// update source: qty - , total_cost -
	fromItem, err := scanItem(tx.QueryRow(ctx, `
		UPDATE warehouse_items
		SET
			qty_milli = qty_milli - $3,
			total_cost_cents = total_cost_cents - $4,
			avg_cost_cents = CASE
				WHEN (qty_milli - $3) > 0
					THEN ((total_cost_cents - $4) * 1000) / (qty_milli - $3)
				ELSE 0
			END,
			updated_at = now()
		WHERE warehouse_id = $1 AND product_id = $2
		RETURNING `+itemCols,
		req.FromWarehouseID, req.ProductID, req.QtyMilli, moveCost,
	))
	if err != nil {
		return TransferResult{}, err
	}

	// upsert dest: qty + , total_cost + (moveCost), avg recalculated
	toItem, err := scanItem(tx.QueryRow(ctx, `
		INSERT INTO warehouse_items (warehouse_id, product_id, qty_milli, avg_cost_cents, total_cost_cents, updated_at)
		VALUES ($1, $2, $3, 0, $4, now())
		ON CONFLICT (warehouse_id, product_id)
		DO UPDATE SET
			qty_milli = warehouse_items.qty_milli + EXCLUDED.qty_milli,
			total_cost_cents = warehouse_items.total_cost_cents + $4,
			avg_cost_cents = CASE
				WHEN (warehouse_items.qty_milli + EXCLUDED.qty_milli) > 0
					THEN ((warehouse_items.total_cost_cents + $4) * 1000) / (warehouse_items.qty_milli + EXCLUDED.qty_milli)
				ELSE 0
			END,
			updated_at = now()
		RETURNING `+itemCols,
		req.ToWarehouseID, req.ProductID, req.QtyMilli, moveCost,
	))
	if err != nil {
		return TransferResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return TransferResult{}, err
	}

	return TransferResult{
		OutDetail: outDetail,
		InDetail:  inDetail,
		FromItem:  fromItem,
		ToItem:    toItem,
	}, nil
}

// ── Move (cost-aware) ────────────────────────────────────────────────────────
// правила:
// - delta < 0 : списываем по avg_cost
// - delta > 0 : ТРЕБУЕМ price_cents (unit cost), иначе не сможем корректно поднять себестоимость

func (r *Repository) Move(ctx context.Context, req MoveRequest, userID int64) (WarehouseItemDetail, WarehouseItem, error) {
	t := normalizeType(req.Type)

	if existing, exists, err := r.checkIdempotencyStrict(ctx, req.IdempotencyKey, req.WarehouseID, t, req.ProductID, req.DeltaMilli); err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	} else if exists {
		it, err := r.getItem(ctx, req.WarehouseID, req.ProductID)
		return existing, it, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}
	defer tx.Rollback(ctx)

	// lock current item if exists (needed for negative delta and for correct avg calc)
	var currentQty, avgCost, totalCost int64
	err = tx.QueryRow(ctx, `
		SELECT qty_milli, avg_cost_cents, total_cost_cents
		FROM warehouse_items
		WHERE warehouse_id = $1 AND product_id = $2
		FOR UPDATE
	`, req.WarehouseID, req.ProductID).Scan(&currentQty, &avgCost, &totalCost)
	if err != nil && err != pgx.ErrNoRows {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}
	if err == pgx.ErrNoRows {
		currentQty, avgCost, totalCost = 0, 0, 0
	}

	var deltaCost int64 // how total_cost_cents changes (+/-)

	if req.DeltaMilli < 0 {
		if currentQty < -req.DeltaMilli {
			return WarehouseItemDetail{}, WarehouseItem{}, apperr.Validation("insufficient stock")
		}
		// remove by avg cost
		deltaCost = -lineTotalCents(-req.DeltaMilli, avgCost)
		if totalCost < -deltaCost {
			return WarehouseItemDetail{}, WarehouseItem{}, apperr.Internal(errors.New("total_cost_cents underflow"))
		}
	} else {
		// delta > 0 requires unit cost price_cents
		if req.PriceCents == nil || *req.PriceCents < 0 {
			return WarehouseItemDetail{}, WarehouseItem{}, apperr.Validation("price_cents is required for positive delta (unit cost)")
		}
		deltaCost = lineTotalCents(req.DeltaMilli, *req.PriceCents)
	}

	// insert detail
	detail, err := scanDetail(tx.QueryRow(ctx, `
		INSERT INTO warehouse_item_details
			(idempotency_key, warehouse_id, product_id, delta_milli, type, price_cents, worker_id, created_by, supplier_id, note)
		VALUES ($1, $2, $3, $4, $5::movement_type, $6, $7, $8, $9, $10)
		RETURNING `+detailCols,
		req.IdempotencyKey, req.WarehouseID, req.ProductID,
		req.DeltaMilli, t, req.PriceCents, req.WorkerID, userID,
		req.SupplierID, req.Note,
	))
	if err != nil {
		if isDuplicateKey(err) {
			existing, _, e2 := r.checkIdempotencyStrict(ctx, req.IdempotencyKey, req.WarehouseID, t, req.ProductID, req.DeltaMilli)
			if e2 != nil {
				return WarehouseItemDetail{}, WarehouseItem{}, e2
			}
			it, e3 := r.getItem(ctx, req.WarehouseID, req.ProductID)
			return existing, it, e3
		}
		if isFKViolation(err) {
			return WarehouseItemDetail{}, WarehouseItem{}, apperr.Validation("warehouse or product does not exist")
		}
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}

	// apply to cache using upsert with delta qty and delta cost
	item, err := scanItem(tx.QueryRow(ctx, `
		INSERT INTO warehouse_items (warehouse_id, product_id, qty_milli, avg_cost_cents, total_cost_cents, updated_at)
		VALUES ($1, $2, GREATEST($3,0), 0, GREATEST($4,0), now())
		ON CONFLICT (warehouse_id, product_id)
		DO UPDATE SET
			qty_milli = GREATEST(warehouse_items.qty_milli + $3, 0),
			total_cost_cents = GREATEST(warehouse_items.total_cost_cents + $4, 0),
			avg_cost_cents = CASE
				WHEN GREATEST(warehouse_items.qty_milli + $3, 0) > 0
					THEN (GREATEST(warehouse_items.total_cost_cents + $4, 0) * 1000) / GREATEST(warehouse_items.qty_milli + $3, 1)
				ELSE 0
			END,
			updated_at = now()
		RETURNING `+itemCols,
		req.WarehouseID, req.ProductID, req.DeltaMilli, deltaCost,
	))
	if err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}
	return detail, item, nil
}

// ── Items query ──────────────────────────────────────────────────────────────

func (r *Repository) GetItems(ctx context.Context, warehouseID, productID *int64) ([]WarehouseItem, error) {
	q := `
		SELECT ` + itemCols + `
		FROM warehouse_items
		WHERE ($1::bigint IS NULL OR warehouse_id = $1)
		  AND ($2::bigint IS NULL OR product_id = $2)
		ORDER BY warehouse_id, product_id
	`
	rows, err := r.db.Query(ctx, q, warehouseID, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []WarehouseItem
	for rows.Next() {
		var it WarehouseItem
		if err := rows.Scan(
			&it.WarehouseID, &it.ProductID, &it.QtyMilli,
			&it.AvgCostCents, &it.TotalCostCents,
			&it.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ── Details (ledger) ─────────────────────────────────────────────────────────

func (r *Repository) GetDetails(
	ctx context.Context,
	warehouseID, productID *int64,
	mType *string,
	supplierID *int64,
	dateFrom, dateTo *time.Time,
	limit, offset int,
) ([]WarehouseItemDetail, int, error) {

	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var t *string
	if mType != nil {
		tt := normalizeType(*mType)
		t = &tt
	}

	where := `
		WHERE ($1::bigint IS NULL OR warehouse_id = $1)
		  AND ($2::bigint IS NULL OR product_id = $2)
		  AND ($3::text   IS NULL OR type::text = $3)
		  AND ($4::timestamptz IS NULL OR created_at >= $4)
		  AND ($5::timestamptz IS NULL OR created_at <= $5)
		  AND ($6::bigint IS NULL OR supplier_id = $6)
	`

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM warehouse_item_details
		`+where,
		warehouseID, productID, t, dateFrom, dateTo, supplierID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT `+detailCols+`
		FROM warehouse_item_details
		`+where+`
		ORDER BY created_at DESC, id DESC
		LIMIT $7 OFFSET $8
	`, warehouseID, productID, t, dateFrom, dateTo, supplierID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []WarehouseItemDetail
	for rows.Next() {
		var d WarehouseItemDetail
		if err := rows.Scan(
			&d.ID, &d.IdempotencyKey, &d.WarehouseID, &d.ProductID,
			&d.DeltaMilli, &d.Type, &d.PriceCents, &d.WorkerID, &d.CreatedBy, &d.CreatedAt,
			&d.SupplierID, &d.Note,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}

// ── idempotent transfer return helpers ───────────────────────────────────────

func (r *Repository) existingTransferResult(ctx context.Context, req TransferRequest) (TransferResult, error) {
	outD, err := scanDetail(r.db.QueryRow(ctx, `
		SELECT `+detailCols+`
		FROM warehouse_item_details
		WHERE idempotency_key = $1 AND warehouse_id = $2
	`, req.IdempotencyKey, req.FromWarehouseID))
	if err != nil {
		return TransferResult{}, err
	}
	if normalizeType(outD.Type) != "transfer_out" || outD.ProductID != req.ProductID || outD.DeltaMilli != -req.QtyMilli {
		return TransferResult{}, apperr.Conflict("IDEMPOTENCY_KEY_CONFLICT", "idempotency_key already used with different parameters")
	}

	inD, err := scanDetail(r.db.QueryRow(ctx, `
		SELECT `+detailCols+`
		FROM warehouse_item_details
		WHERE idempotency_key = $1 AND warehouse_id = $2
	`, req.IdempotencyKey, req.ToWarehouseID))
	if err != nil {
		return TransferResult{}, err
	}
	if normalizeType(inD.Type) != "transfer_in" || inD.ProductID != req.ProductID || inD.DeltaMilli != req.QtyMilli {
		return TransferResult{}, apperr.Conflict("IDEMPOTENCY_KEY_CONFLICT", "idempotency_key already used with different parameters")
	}

	fromIt, err := r.getItem(ctx, req.FromWarehouseID, req.ProductID)
	if err != nil {
		return TransferResult{}, err
	}
	toIt, err := r.getItem(ctx, req.ToWarehouseID, req.ProductID)
	if err != nil {
		return TransferResult{}, err
	}

	return TransferResult{
		OutDetail: outD,
		InDetail:  inD,
		FromItem:  fromIt,
		ToItem:    toIt,
	}, nil
}

func (r *Repository) getItem(ctx context.Context, warehouseID, productID int64) (WarehouseItem, error) {
	it, err := scanItem(r.db.QueryRow(ctx, `
		SELECT `+itemCols+`
		FROM warehouse_items
		WHERE warehouse_id = $1 AND product_id = $2
	`, warehouseID, productID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return WarehouseItem{WarehouseID: warehouseID, ProductID: productID}, nil
		}
		return WarehouseItem{}, err
	}
	return it, nil
}
func (r *Repository) OpeningBalance(ctx context.Context, req OpeningBalanceRequest, userID int64) (WarehouseItemDetail, WarehouseItem, error) {
	// strict idempotency: same key must match type+product+delta
	if existing, exists, err := r.checkIdempotencyStrict(ctx, req.IdempotencyKey, req.WarehouseID, "opening_balance", req.ProductID, req.QtyMilli); err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	} else if exists {
		it, err := r.getItem(ctx, req.WarehouseID, req.ProductID)
		return existing, it, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}
	defer tx.Rollback(ctx)

	// detail (ledger)
	detail, err := scanDetail(tx.QueryRow(ctx, `
		INSERT INTO warehouse_item_details
			(idempotency_key, warehouse_id, product_id, delta_milli, type, price_cents, created_by)
		VALUES ($1, $2, $3, $4, 'opening_balance', $5, $6)
		RETURNING `+detailCols,
		req.IdempotencyKey, req.WarehouseID, req.ProductID, req.QtyMilli, req.PriceCents, userID,
	))
	if err != nil {
		if isDuplicateKey(err) {
			existing, _, e2 := r.checkIdempotencyStrict(ctx, req.IdempotencyKey, req.WarehouseID, "opening_balance", req.ProductID, req.QtyMilli)
			if e2 != nil {
				return WarehouseItemDetail{}, WarehouseItem{}, e2
			}
			it, e3 := r.getItem(ctx, req.WarehouseID, req.ProductID)
			return existing, it, e3
		}
		if isFKViolation(err) {
			return WarehouseItemDetail{}, WarehouseItem{}, apperr.Validation("warehouse or product does not exist")
		}
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}

	// total_cost = qty * unit_price / 1000
	totalCost := (req.QtyMilli * req.PriceCents) / 1000

	// overwrite cache to exact state
	item, err := scanItem(tx.QueryRow(ctx, `
		INSERT INTO warehouse_items (warehouse_id, product_id, qty_milli, avg_cost_cents, total_cost_cents, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (warehouse_id, product_id)
		DO UPDATE SET
			qty_milli = EXCLUDED.qty_milli,
			avg_cost_cents = EXCLUDED.avg_cost_cents,
			total_cost_cents = EXCLUDED.total_cost_cents,
			updated_at = now()
		RETURNING `+itemCols+`, avg_cost_cents, total_cost_cents
	`, req.WarehouseID, req.ProductID, req.QtyMilli, req.PriceCents, totalCost))
	if err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return WarehouseItemDetail{}, WarehouseItem{}, err
	}
	return detail, item, nil
}
