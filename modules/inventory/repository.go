package inventory

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Repository struct {
	db      *pgxpool.Pool
	finRepo *finance.Repository
}

func NewRepository(db *pgxpool.Pool, finRepo *finance.Repository) *Repository {
	return &Repository{db: db, finRepo: finRepo}
}

func (r *Repository) Create(ctx context.Context, req CreateRequest, userID int64) (InventoryDetail, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return InventoryDetail{}, apperr.Internal(err)
	}
	defer tx.Rollback(ctx)

	// Create header
	var ic InventoryCount
	err = tx.QueryRow(ctx, `
		INSERT INTO inventory_counts (warehouse_id, note, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, warehouse_id, status, note, created_by, created_at
	`, req.WarehouseID, req.Note, userID).Scan(
		&ic.ID, &ic.WarehouseID, &ic.Status, &ic.Note, &ic.CreatedBy, &ic.CreatedAt,
	)
	if err != nil {
		return InventoryDetail{}, apperr.Internal(err)
	}

	// Load all products from warehouse with current qty
	rows, err := tx.Query(ctx, `
		INSERT INTO inventory_count_items (count_id, product_id, system_qty_milli, actual_qty_milli, diff_milli)
		SELECT $1, wi.product_id, wi.qty_milli, 0, (0 - wi.qty_milli)
		FROM warehouse_items wi
		JOIN products p ON p.id = wi.product_id AND p.deleted_at IS NULL AND p.is_active = true
		WHERE wi.warehouse_id = $2
		ORDER BY wi.product_id
		RETURNING id, count_id, product_id, system_qty_milli, actual_qty_milli, diff_milli
	`, ic.ID, req.WarehouseID)
	if err != nil {
		return InventoryDetail{}, apperr.Internal(err)
	}

	var items []InventoryCountItem
	for rows.Next() {
		var it InventoryCountItem
		if err := rows.Scan(&it.ID, &it.CountID, &it.ProductID, &it.SystemQtyMilli, &it.ActualQtyMilli, &it.DiffMilli); err != nil {
			rows.Close()
			return InventoryDetail{}, apperr.Internal(err)
		}
		items = append(items, it)
	}
	rows.Close()

	if items == nil {
		items = []InventoryCountItem{}
	}

	if err := tx.Commit(ctx); err != nil {
		return InventoryDetail{}, apperr.Internal(err)
	}

	return InventoryDetail{InventoryCount: ic, Items: items}, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (InventoryDetail, error) {
	var ic InventoryCount
	err := r.db.QueryRow(ctx, `
		SELECT ic.id, ic.warehouse_id, w.name, ic.status, ic.note,
		       ic.created_by, ic.confirmed_by, ic.created_at, ic.confirmed_at
		FROM inventory_counts ic
		JOIN warehouses w ON w.id = ic.warehouse_id
		WHERE ic.id = $1
	`, id).Scan(
		&ic.ID, &ic.WarehouseID, &ic.WarehouseName, &ic.Status, &ic.Note,
		&ic.CreatedBy, &ic.ConfirmedBy, &ic.CreatedAt, &ic.ConfirmedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return InventoryDetail{}, apperr.NotFound("INVENTORY_NOT_FOUND", "inventory count not found")
		}
		return InventoryDetail{}, apperr.Internal(err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT ici.id, ici.count_id, ici.product_id, p.name,
		       ici.system_qty_milli, ici.actual_qty_milli, ici.diff_milli
		FROM inventory_count_items ici
		JOIN products p ON p.id = ici.product_id
		WHERE ici.count_id = $1
		ORDER BY p.name
	`, id)
	if err != nil {
		return InventoryDetail{}, apperr.Internal(err)
	}
	defer rows.Close()

	var items []InventoryCountItem
	for rows.Next() {
		var it InventoryCountItem
		if err := rows.Scan(&it.ID, &it.CountID, &it.ProductID, &it.ProductName,
			&it.SystemQtyMilli, &it.ActualQtyMilli, &it.DiffMilli); err != nil {
			return InventoryDetail{}, apperr.Internal(err)
		}
		items = append(items, it)
	}
	if items == nil {
		items = []InventoryCountItem{}
	}

	return InventoryDetail{InventoryCount: ic, Items: items}, nil
}

func (r *Repository) List(ctx context.Context, warehouseID *int64, limit, offset int) ([]InventoryCount, int, error) {
	// Build args dynamically so we don't pass a nil warehouseID into a query
	// that has no $1 placeholder — pgx rejects that as "extra arguments".
	where := "WHERE 1=1"
	countArgs := []any{}
	listArgs := []any{}
	if warehouseID != nil {
		where += " AND ic.warehouse_id = $1"
		countArgs = append(countArgs, *warehouseID)
		listArgs = append(listArgs, *warehouseID)
	}
	listArgs = append(listArgs, limit, offset)

	// LIMIT/OFFSET placeholders shift by one if a warehouse filter is present.
	limitIdx := len(listArgs) - 1
	offsetIdx := len(listArgs)
	listSQL := `
		SELECT ic.id, ic.warehouse_id, w.name, ic.status, ic.note,
		       ic.created_by, ic.confirmed_by, ic.created_at, ic.confirmed_at
		FROM inventory_counts ic
		JOIN warehouses w ON w.id = ic.warehouse_id
		` + where + `
		ORDER BY ic.created_at DESC
		LIMIT $` + strconv.Itoa(limitIdx) + ` OFFSET $` + strconv.Itoa(offsetIdx)

	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM inventory_counts ic `+where, countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []InventoryCount
	for rows.Next() {
		var ic InventoryCount
		if err := rows.Scan(&ic.ID, &ic.WarehouseID, &ic.WarehouseName, &ic.Status, &ic.Note,
			&ic.CreatedBy, &ic.ConfirmedBy, &ic.CreatedAt, &ic.ConfirmedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, ic)
	}
	if out == nil {
		out = []InventoryCount{}
	}
	return out, total, rows.Err()
}

func (r *Repository) UpdateItems(ctx context.Context, countID int64, req UpdateItemsRequest) error {
	// Check status
	var status string
	err := r.db.QueryRow(ctx, `SELECT status FROM inventory_counts WHERE id = $1`, countID).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return apperr.NotFound("INVENTORY_NOT_FOUND", "inventory count not found")
		}
		return apperr.Internal(err)
	}
	if status != "draft" {
		return apperr.Conflict("INVENTORY_NOT_DRAFT", "can only update items in draft status")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return apperr.Internal(err)
	}
	defer tx.Rollback(ctx)

	for _, item := range req.Items {
		_, err := tx.Exec(ctx, `
			UPDATE inventory_count_items
			SET actual_qty_milli = $3,
			    diff_milli = $3 - system_qty_milli
			WHERE count_id = $1 AND product_id = $2
		`, countID, item.ProductID, item.ActualQtyMilli)
		if err != nil {
			return apperr.Internal(err)
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) Confirm(ctx context.Context, countID, userID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return apperr.Internal(err)
	}
	defer tx.Rollback(ctx)

	// Lock and check
	var status string
	var warehouseID int64
	err = tx.QueryRow(ctx,
		`SELECT status, warehouse_id FROM inventory_counts WHERE id = $1 FOR UPDATE`,
		countID,
	).Scan(&status, &warehouseID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return apperr.NotFound("INVENTORY_NOT_FOUND", "inventory count not found")
		}
		return apperr.Internal(err)
	}
	if status != "draft" {
		return apperr.Conflict("INVENTORY_NOT_DRAFT", "can only confirm draft inventory")
	}

	// Get items with diff != 0
	rows, err := tx.Query(ctx, `
		SELECT product_id, diff_milli
		FROM inventory_count_items
		WHERE count_id = $1 AND diff_milli != 0
	`, countID)
	if err != nil {
		return apperr.Internal(err)
	}

	type adjItem struct {
		productID int64
		diffMilli int64
	}
	var adjustments []adjItem
	for rows.Next() {
		var a adjItem
		if err := rows.Scan(&a.productID, &a.diffMilli); err != nil {
			rows.Close()
			return apperr.Internal(err)
		}
		adjustments = append(adjustments, a)
	}
	rows.Close()

	// Apply adjustments, valuing shortages at current avg_cost for P&L tracking.
	uid := &userID
	var shortageCents int64
	for _, a := range adjustments {
		// Capture current avg_cost BEFORE the stock change so shortage valuation
		// reflects the actual cost of the missing goods. When the warehouse
		// has no row yet for this product (first time it appears here),
		// fall back to products.purchase_price so we don't seed the row at
		// avg=0 — that "free goods" state would make every future sale show
		// a 100% margin until the next stock-in.
		var avgCostCents int64
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(avg_cost_cents, 0) FROM warehouse_items
			WHERE warehouse_id = $1 AND product_id = $2
		`, warehouseID, a.productID).Scan(&avgCostCents)
		if err == pgx.ErrNoRows {
			_ = tx.QueryRow(ctx, `
				SELECT COALESCE(purchase_price, 0) FROM products WHERE id = $1
			`, a.productID).Scan(&avgCostCents)
		} else if err != nil {
			return apperr.Internal(err)
		}

		if a.diffMilli < 0 {
			// Shortage: |diff_milli| × avg_cost_cents / 1000  (milli → whole units)
			shortageCents += (-a.diffMilli) * avgCostCents / 1000
		}

		// Ledger entry — store price_cents so reports can value shortage/surplus per product.
		_, err = tx.Exec(ctx, `
			INSERT INTO warehouse_item_details
				(idempotency_key, warehouse_id, product_id, delta_milli, type, price_cents, created_by, note)
			VALUES (gen_random_uuid(), $1, $2, $3, 'adjustment', $4, $5, 'Inventory count')
		`, warehouseID, a.productID, a.diffMilli, avgCostCents, uid)
		if err != nil {
			return apperr.Internal(err)
		}

		// Upsert warehouse_items: keep avg_cost stable (don't repaint history),
		// but ALWAYS recompute total_cost from the new qty × avg so the row's
		// three-number invariant (qty × avg / 1000 ≈ total) holds after the
		// count. Pre-fix this UPSERT touched only qty_milli, leaving avg=0,
		// total=0 on freshly-inserted rows and an ever-growing drift on
		// updated rows. GREATEST clamps total to 0 if qty went negative
		// (under-count of a previously empty row).
		_, err = tx.Exec(ctx, `
			INSERT INTO warehouse_items (warehouse_id, product_id, qty_milli, avg_cost_cents, total_cost_cents, updated_at)
			VALUES ($1, $2, $3, $4, GREATEST($3::bigint, 0) * $4 / 1000, now())
			ON CONFLICT (warehouse_id, product_id) DO UPDATE SET
				qty_milli        = warehouse_items.qty_milli + $3::bigint,
				total_cost_cents = GREATEST(warehouse_items.qty_milli + $3::bigint, 0)
				                   * warehouse_items.avg_cost_cents / 1000,
				updated_at = now()
		`, warehouseID, a.productID, a.diffMilli, avgCostCents)
		if err != nil {
			return apperr.Internal(err)
		}
	}

	// Record shrinkage as an expense transaction so it hits the P&L.
	if shortageCents > 0 && r.finRepo != nil {
		reason := fmt.Sprintf("Inventory shrinkage (count #%d)", countID)
		txn := &finance.Transaction{
			Status:       "paid",
			AmountCents:  shortageCents,
			Type:         "expense",
			RelatedTable: "adjustment",
			RelatedID:    &countID,
			Reason:       &reason,
			CreatedBy:    uid,
		}
		if err := r.finRepo.CreateTransaction(ctx, tx, txn); err != nil {
			return apperr.Internal(err)
		}
	}

	// Mark confirmed
	_, err = tx.Exec(ctx, `
		UPDATE inventory_counts
		SET status = 'confirmed', confirmed_by = $2, confirmed_at = now()
		WHERE id = $1
	`, countID, userID)
	if err != nil {
		return apperr.Internal(err)
	}

	return tx.Commit(ctx)
}

func (r *Repository) Cancel(ctx context.Context, countID int64) error {
	ct, err := r.db.Exec(ctx, `
		UPDATE inventory_counts SET status = 'cancelled'
		WHERE id = $1 AND status = 'draft'
	`, countID)
	if err != nil {
		return apperr.Internal(err)
	}
	if ct.RowsAffected() == 0 {
		return apperr.NotFound("INVENTORY_NOT_FOUND", "inventory not found or not in draft status")
	}
	return nil
}
