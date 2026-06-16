package supplierreturn

import (
	"context"
	"errors"
	"fmt"
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

const retCols = `id, supplier_id, warehouse_id, status, total_cents, items_count,
	note, created_by, created_at, confirmed_at, confirmed_by`

func scanReturn(row pgx.Row) (SupplierReturn, error) {
	var r SupplierReturn
	err := row.Scan(
		&r.ID, &r.SupplierID, &r.WarehouseID, &r.Status, &r.TotalCents, &r.ItemsCount,
		&r.Note, &r.CreatedBy, &r.CreatedAt, &r.ConfirmedAt, &r.ConfirmedBy,
	)
	return r, err
}

func lineTotalCents(qtyMilli, unitCostCents int64) int64 {
	return (qtyMilli*unitCostCents + 500) / 1000
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// ── Create ──────────────────────────────────────────────────────────────────

func (r *Repository) Create(
	ctx context.Context,
	req CreateRequest,
	userID int64,
) (SupplierReturn, []ReturnItem, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SupplierReturn{}, nil, err
	}
	defer tx.Rollback(ctx)

	uid := &userID

	// 1. Validate product IDs
	productIDs := make([]int64, len(req.Items))
	for i, item := range req.Items {
		productIDs[i] = item.ProductID
	}

	rows, err := tx.Query(ctx,
		`SELECT id FROM products WHERE id = ANY($1) AND is_active = true`,
		productIDs,
	)
	if err != nil {
		return SupplierReturn{}, nil, err
	}
	validIDs := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return SupplierReturn{}, nil, err
		}
		validIDs[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return SupplierReturn{}, nil, err
	}

	for i, item := range req.Items {
		if !validIDs[item.ProductID] {
			return SupplierReturn{}, nil, apperr.Validation(
				fmt.Sprintf("product %d not found or inactive (item index %d)", item.ProductID, i),
			)
		}
	}

	// 2. Calculate total
	var totalCents int64
	for _, item := range req.Items {
		totalCents += lineTotalCents(item.QtyMilli, item.UnitCostCents)
	}

	// 3. Insert return header
	ret, err := scanReturn(tx.QueryRow(ctx, `
		INSERT INTO supplier_returns
			(supplier_id, warehouse_id, total_cents, items_count, note, created_by, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'draft')
		RETURNING `+retCols,
		req.SupplierID, req.WarehouseID, totalCents, len(req.Items), req.Note, uid,
	))
	if err != nil {
		if isFKViolation(err) {
			return SupplierReturn{}, nil, apperr.Validation("supplier or warehouse does not exist")
		}
		return SupplierReturn{}, nil, err
	}

	// 4. Insert line items
	for _, item := range req.Items {
		lineTotal := lineTotalCents(item.QtyMilli, item.UnitCostCents)
		_, err = tx.Exec(ctx, `
			INSERT INTO supplier_return_items (return_id, product_id, qty_milli, unit_cost_cents, line_total_cents)
			VALUES ($1, $2, $3, $4, $5)
		`, ret.ID, item.ProductID, item.QtyMilli, item.UnitCostCents, lineTotal)
		if err != nil {
			return SupplierReturn{}, nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return SupplierReturn{}, nil, err
	}

	// Refetch with product names
	_, items, err := r.GetByID(ctx, ret.ID)
	if err != nil {
		return ret, nil, err
	}
	return ret, items, nil
}

// ── Confirm (draft → confirmed: stock out + reduce supplier debt) ───────────

func (r *Repository) Confirm(
	ctx context.Context,
	returnID int64,
	userID int64,
) (SupplierReturn, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SupplierReturn{}, err
	}
	defer tx.Rollback(ctx)

	uid := &userID

	// 1. Lock and check status
	var status string
	var supplierID, warehouseID, totalCents int64
	err = tx.QueryRow(ctx, `
		SELECT status, supplier_id, warehouse_id, total_cents
		FROM supplier_returns WHERE id = $1 FOR UPDATE
	`, returnID).Scan(&status, &supplierID, &warehouseID, &totalCents)
	if err != nil {
		if err == pgx.ErrNoRows {
			return SupplierReturn{}, apperr.NotFound("RETURN_NOT_FOUND", "supplier return not found")
		}
		return SupplierReturn{}, err
	}
	if status != "draft" {
		return SupplierReturn{}, apperr.Conflict("RETURN_NOT_DRAFT", "supplier return is not in draft status")
	}

	// 2. Fetch items
	type retItem struct {
		productID     int64
		qtyMilli      int64
		unitCostCents int64
	}
	itemRows, err := tx.Query(ctx, `
		SELECT product_id, qty_milli, unit_cost_cents
		FROM supplier_return_items
		WHERE return_id = $1
		ORDER BY product_id
	`, returnID)
	if err != nil {
		return SupplierReturn{}, err
	}
	var items []retItem
	for itemRows.Next() {
		var it retItem
		if err := itemRows.Scan(&it.productID, &it.qtyMilli, &it.unitCostCents); err != nil {
			itemRows.Close()
			return SupplierReturn{}, err
		}
		items = append(items, it)
	}
	itemRows.Close()
	if err := itemRows.Err(); err != nil {
		return SupplierReturn{}, err
	}

	// 3. For each item: stock ledger (delta = -qty) + update warehouse_items
	for _, it := range items {
		outCost := lineTotalCents(it.qtyMilli, it.unitCostCents)

		// Stock ledger entry (type='supplier_return', delta = -qty)
		_, err = tx.Exec(ctx, `
			INSERT INTO warehouse_item_details
				(idempotency_key, warehouse_id, product_id, delta_milli, type, price_cents, created_by)
			VALUES (gen_random_uuid(), $1, $2, $3, 'supplier_return', $4, $5)
		`, warehouseID, it.productID, -it.qtyMilli, it.unitCostCents, uid)
		if err != nil {
			return SupplierReturn{}, err
		}

		// Update warehouse_items: subtract qty and cost
		_, err = tx.Exec(ctx, `
			UPDATE warehouse_items
			SET qty_milli        = qty_milli - $3,
			    total_cost_cents = total_cost_cents - $4,
			    avg_cost_cents   = CASE
			        WHEN (qty_milli - $3) > 0
			            THEN ((total_cost_cents - $4) * 1000) / (qty_milli - $3)
			        ELSE 0
			    END,
			    updated_at = now()
			WHERE warehouse_id = $1 AND product_id = $2
		`, warehouseID, it.productID, it.qtyMilli, outCost)
		if err != nil {
			return SupplierReturn{}, err
		}
	}

	// 4. Update return status
	ret, err := scanReturn(tx.QueryRow(ctx, `
		UPDATE supplier_returns
		SET status = 'confirmed', confirmed_at = now(), confirmed_by = $2
		WHERE id = $1
		RETURNING `+retCols,
		returnID, uid,
	))
	if err != nil {
		return SupplierReturn{}, err
	}

	// 5. Reduce supplier debt (find the oldest open debt and reduce remaining_cents)
	var remaining int64
	remaining = totalCents
	debtRows, err := tx.Query(ctx, `
		SELECT id, remaining_cents FROM supplier_debts
		WHERE supplier_id = $1 AND status = 'open'
		ORDER BY created_at ASC
		FOR UPDATE
	`, supplierID)
	if err != nil {
		return SupplierReturn{}, err
	}
	type debtRow struct {
		id        int64
		remaining int64
	}
	var debts []debtRow
	for debtRows.Next() {
		var d debtRow
		if err := debtRows.Scan(&d.id, &d.remaining); err != nil {
			debtRows.Close()
			return SupplierReturn{}, err
		}
		debts = append(debts, d)
	}
	debtRows.Close()
	if err := debtRows.Err(); err != nil {
		return SupplierReturn{}, err
	}

	for _, d := range debts {
		if remaining <= 0 {
			break
		}
		reduce := d.remaining
		if reduce > remaining {
			reduce = remaining
		}
		newRemaining := d.remaining - reduce
		newStatus := "open"
		if newRemaining == 0 {
			newStatus = "settled"
		}
		_, err = tx.Exec(ctx, `
			UPDATE supplier_debts
			SET remaining_cents = $2, status = $3, updated_at = now()
			WHERE id = $1
		`, d.id, newRemaining, newStatus)
		if err != nil {
			return SupplierReturn{}, err
		}
		remaining -= reduce
	}

	if err := tx.Commit(ctx); err != nil {
		return SupplierReturn{}, err
	}
	return ret, nil
}

// ── Cancel ──────────────────────────────────────────────────────────────────

func (r *Repository) Cancel(ctx context.Context, returnID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status string
	err = tx.QueryRow(ctx, `
		SELECT status FROM supplier_returns WHERE id = $1 FOR UPDATE
	`, returnID).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return apperr.NotFound("RETURN_NOT_FOUND", "supplier return not found")
		}
		return err
	}

	switch status {
	case "confirmed":
		return apperr.Conflict("RETURN_ALREADY_CONFIRMED", "cannot cancel a confirmed return")
	case "cancelled":
		return apperr.Conflict("RETURN_ALREADY_CANCELLED", "return is already cancelled")
	}

	_, err = tx.Exec(ctx, `UPDATE supplier_returns SET status = 'cancelled' WHERE id = $1`, returnID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ── GetByID ─────────────────────────────────────────────────────────────────

func (r *Repository) GetByID(ctx context.Context, id int64) (SupplierReturn, []ReturnItem, error) {
	ret, err := scanReturn(r.db.QueryRow(ctx,
		`SELECT `+retCols+` FROM supplier_returns WHERE id = $1`, id,
	))
	if err != nil {
		return SupplierReturn{}, nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT ri.id, ri.return_id, ri.product_id,
		       ri.qty_milli, ri.unit_cost_cents, ri.line_total_cents, ri.created_at,
		       p.name
		FROM supplier_return_items ri
		JOIN products p ON p.id = ri.product_id
		WHERE ri.return_id = $1
		ORDER BY ri.id
	`, id)
	if err != nil {
		return ret, nil, err
	}
	defer rows.Close()

	var items []ReturnItem
	for rows.Next() {
		var it ReturnItem
		if err := rows.Scan(
			&it.ID, &it.ReturnID, &it.ProductID,
			&it.QtyMilli, &it.UnitCostCents, &it.LineTotalCents, &it.CreatedAt,
			&it.ProductName,
		); err != nil {
			return ret, nil, err
		}
		items = append(items, it)
	}
	if items == nil {
		items = []ReturnItem{}
	}
	return ret, items, rows.Err()
}

// ── List ────────────────────────────────────────────────────────────────────

func (r *Repository) List(
	ctx context.Context,
	supplierID, warehouseID *int64,
	status *string,
	dateFrom, dateTo *time.Time,
	limit, offset int,
) ([]ReturnListItem, int, error) {

	where := `
		WHERE ($1::bigint IS NULL OR sr.supplier_id = $1)
		  AND ($2::bigint IS NULL OR sr.warehouse_id = $2)
		  AND ($3::text   IS NULL OR sr.status = $3)
		  AND ($4::timestamptz IS NULL OR sr.created_at >= $4)
		  AND ($5::timestamptz IS NULL OR sr.created_at <= $5)
	`

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM supplier_returns sr `+where,
		supplierID, warehouseID, status, dateFrom, dateTo,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT sr.id, sr.supplier_id, sr.warehouse_id, sr.status, sr.total_cents, sr.items_count,
		       sr.note, sr.created_by, sr.created_at, sr.confirmed_at, sr.confirmed_by,
		       s.name, w.name,
		       COALESCE(uc.name, ''), COALESCE(ucf.name, '')
		FROM supplier_returns sr
		LEFT JOIN suppliers s  ON s.id  = sr.supplier_id
		LEFT JOIN warehouses w ON w.id  = sr.warehouse_id
		LEFT JOIN employees uc     ON uc.id = sr.created_by
		LEFT JOIN employees ucf    ON ucf.id = sr.confirmed_by
		`+where+`
		ORDER BY sr.created_at DESC, sr.id DESC
		LIMIT $6 OFFSET $7
	`, supplierID, warehouseID, status, dateFrom, dateTo, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []ReturnListItem
	for rows.Next() {
		var item ReturnListItem
		var supplierName, warehouseName *string
		err := rows.Scan(
			&item.ID, &item.SupplierID, &item.WarehouseID, &item.Status, &item.TotalCents, &item.ItemsCount,
			&item.Note, &item.CreatedBy, &item.CreatedAt, &item.ConfirmedAt, &item.ConfirmedBy,
			&supplierName, &warehouseName,
			&item.CreatedByName, &item.ConfirmedByName,
		)
		if err != nil {
			return nil, 0, err
		}
		if supplierName != nil {
			item.SupplierName = *supplierName
		}
		if warehouseName != nil {
			item.WarehouseName = *warehouseName
		}
		out = append(out, item)
	}
	if out == nil {
		out = []ReturnListItem{}
	}
	return out, total, rows.Err()
}
