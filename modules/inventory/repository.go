package inventory

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
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
	where := "WHERE 1=1"
	if warehouseID != nil {
		where += " AND ic.warehouse_id = $1"
	}

	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM inventory_counts ic `+where,
		warehouseID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT ic.id, ic.warehouse_id, w.name, ic.status, ic.note,
		       ic.created_by, ic.confirmed_by, ic.created_at, ic.confirmed_at
		FROM inventory_counts ic
		JOIN warehouses w ON w.id = ic.warehouse_id
		`+where+`
		ORDER BY ic.created_at DESC
		LIMIT $2 OFFSET $3
	`, warehouseID, limit, offset)
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

	// Apply adjustments
	uid := &userID
	for _, a := range adjustments {
		// Ledger entry
		_, err = tx.Exec(ctx, `
			INSERT INTO warehouse_item_details
				(idempotency_key, warehouse_id, product_id, delta_milli, type, created_by, note)
			VALUES (gen_random_uuid(), $1, $2, $3, 'adjustment', $4, 'Inventory count')
		`, warehouseID, a.productID, a.diffMilli, uid)
		if err != nil {
			return apperr.Internal(err)
		}

		// Update warehouse_items
		_, err = tx.Exec(ctx, `
			INSERT INTO warehouse_items (warehouse_id, product_id, qty_milli, avg_cost_cents, total_cost_cents, updated_at)
			VALUES ($1, $2, $3, 0, 0, now())
			ON CONFLICT (warehouse_id, product_id) DO UPDATE SET
				qty_milli = warehouse_items.qty_milli + $3::bigint,
				updated_at = now()
		`, warehouseID, a.productID, a.diffMilli)
		if err != nil {
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
