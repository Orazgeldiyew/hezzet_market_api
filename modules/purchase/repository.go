package purchase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/modules/auditlog"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Repository struct {
	db        *pgxpool.Pool
	auditRepo *auditlog.Repository
}

func NewRepository(db *pgxpool.Pool, auditRepo *auditlog.Repository) *Repository {
	return &Repository{db: db, auditRepo: auditRepo}
}



const poCols = `id, supplier_id, warehouse_id, status, total_cents, items_count,
	note, created_by, created_at, received_at, received_by`



func scanPO(row pgx.Row) (PurchaseOrder, error) {
	var p PurchaseOrder
	err := row.Scan(
		&p.ID, &p.SupplierID, &p.WarehouseID, &p.Status, &p.TotalCents, &p.ItemsCount,
		&p.Note, &p.CreatedBy, &p.CreatedAt, &p.ReceivedAt, &p.ReceivedBy,
	)
	return p, err
}

func lineTotalCents(qtyMilli, unitCostCents int64) int64 {
	return (qtyMilli*unitCostCents + 500) / 1000
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}



func (r *Repository) CreatePO(
	ctx context.Context,
	req CreatePORequest,
	userID int64,
) (PurchaseOrder, []PurchaseItem, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return PurchaseOrder{}, nil, err
	}
	defer tx.Rollback(ctx)

	uid := &userID

	// 1. Validate all product IDs are active
	productIDs := make([]int64, len(req.Items))
	for i, item := range req.Items {
		productIDs[i] = item.ProductID
	}

	rows, err := tx.Query(ctx,
		`SELECT id FROM products WHERE id = ANY($1) AND is_active = true`,
		productIDs,
	)
	if err != nil {
		return PurchaseOrder{}, nil, err
	}
	validIDs := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return PurchaseOrder{}, nil, err
		}
		validIDs[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return PurchaseOrder{}, nil, err
	}

	for i, item := range req.Items {
		if !validIDs[item.ProductID] {
			return PurchaseOrder{}, nil, apperr.Validation(
				fmt.Sprintf("product %d not found or inactive (item index %d)", item.ProductID, i),
			)
		}
	}

	// 2. Calculate total
	var totalCents int64
	for _, item := range req.Items {
		totalCents += lineTotalCents(item.QtyMilli, item.UnitCostCents)
	}

	// 3. Insert PO header (status='draft')
	po, err := scanPO(tx.QueryRow(ctx, `
		INSERT INTO purchase_orders
			(supplier_id, warehouse_id, total_cents, items_count, note, created_by, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'draft')
		RETURNING `+poCols,
		req.SupplierID, req.WarehouseID, totalCents, len(req.Items), req.Note, uid,
	))
	if err != nil {
		if isFKViolation(err) {
			return PurchaseOrder{}, nil, apperr.Validation("supplier or warehouse does not exist")
		}
		return PurchaseOrder{}, nil, err
	}

	// 4. Insert line items
	for _, item := range req.Items {
		lineTotal := lineTotalCents(item.QtyMilli, item.UnitCostCents)
		_, err = tx.Exec(ctx, `
			INSERT INTO purchase_order_items (po_id, product_id, qty_milli, unit_cost_cents, sale_price_cents, line_total_cents)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, po.ID, item.ProductID, item.QtyMilli, item.UnitCostCents, item.SalePriceCents, lineTotal)
		if err != nil {
			return PurchaseOrder{}, nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return PurchaseOrder{}, nil, err
	}

	// Refetch with product names (after commit)
	_, items, err := r.GetByID(ctx, po.ID)
	if err != nil {
		return po, nil, err
	}
	return po, items, nil
}

// ── ReceivePO (draft → received: stock in + expense transaction) ─────────────

func (r *Repository) ReceivePO(
	ctx context.Context,
	poID int64,
	userID int64,
	finRepo *finance.Repository,
) (PurchaseOrder, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return PurchaseOrder{}, err
	}
	defer tx.Rollback(ctx)

	uid := &userID

	// 1. Lock PO and check status
	var status string
	var supplierID, warehouseID, totalCents int64
	err = tx.QueryRow(ctx, `
		SELECT status, supplier_id, warehouse_id, total_cents FROM purchase_orders WHERE id = $1 FOR UPDATE
	`, poID).Scan(&status, &supplierID, &warehouseID, &totalCents)
	if err != nil {
		if err == pgx.ErrNoRows {
			return PurchaseOrder{}, apperr.NotFound("PO_NOT_FOUND", "purchase order not found")
		}
		return PurchaseOrder{}, err
	}
	if status != "draft" {
		return PurchaseOrder{}, apperr.Conflict("PO_NOT_DRAFT", "purchase order is not in draft status")
	}

	// 2. Fetch items sorted by product_id to avoid deadlocks
	type poItem struct {
		productID      int64
		qtyMilli       int64
		unitCostCents  int64
		salePriceCents int64
	}
	itemRows, err := tx.Query(ctx, `
		SELECT product_id, qty_milli, unit_cost_cents, sale_price_cents
		FROM purchase_order_items
		WHERE po_id = $1
		ORDER BY product_id
	`, poID)
	if err != nil {
		return PurchaseOrder{}, err
	}
	var items []poItem
	for itemRows.Next() {
		var it poItem
		if err := itemRows.Scan(&it.productID, &it.qtyMilli, &it.unitCostCents, &it.salePriceCents); err != nil {
			itemRows.Close()
			return PurchaseOrder{}, err
		}
		items = append(items, it)
	}
	itemRows.Close()
	if err := itemRows.Err(); err != nil {
		return PurchaseOrder{}, err
	}

	// 3. For each item: insert stock ledger + upsert warehouse_items
	for _, it := range items {
		inCost := lineTotalCents(it.qtyMilli, it.unitCostCents)

		// Insert stock ledger entry (type='purchase', delta=+qty)
		_, err = tx.Exec(ctx, `
			INSERT INTO warehouse_item_details
				(idempotency_key, warehouse_id, product_id, delta_milli, type, price_cents, created_by)
			VALUES (gen_random_uuid(), $1, $2, $3, 'purchase', $4, $5)
		`, warehouseID, it.productID, it.qtyMilli, it.unitCostCents, uid)
		if err != nil {
			return PurchaseOrder{}, err
		}

		// Upsert warehouse_items: add qty + cost, recalc avg_cost
		_, err = tx.Exec(ctx, `
			INSERT INTO warehouse_items
				(warehouse_id, product_id, qty_milli, avg_cost_cents, total_cost_cents, updated_at)
			VALUES ($1, $2, $3, $4, $5, now())
			ON CONFLICT (warehouse_id, product_id)
			DO UPDATE SET
				qty_milli        = warehouse_items.qty_milli + EXCLUDED.qty_milli,
				total_cost_cents = warehouse_items.total_cost_cents + $5,
				avg_cost_cents   = CASE
					WHEN (warehouse_items.qty_milli + EXCLUDED.qty_milli) > 0
						THEN ((warehouse_items.total_cost_cents + $5) * 1000)
						     / (warehouse_items.qty_milli + EXCLUDED.qty_milli)
					ELSE 0
				END,
				updated_at = now()
		`, warehouseID, it.productID, it.qtyMilli, it.unitCostCents, inCost)
		if err != nil {
			return PurchaseOrder{}, err
		}

		// Propagate prices to the product master record:
		//   - purchase_price always reflects the most recent buy cost
		//   - sale_price updates only when the PO line set it explicitly (>0),
		//     so PO lines without a sale price leave the catalog price alone.
		//
		// Read old prices first so we can write an audit_logs entry showing
		// the before/after diff — the audit middleware doesn't see these
		// internal UPDATEs, only the receive HTTP call itself.
		var oldPurchase, oldSale int64
		var productName string
		if err = tx.QueryRow(ctx, `
			SELECT purchase_price, sale_price, name FROM products WHERE id = $1
		`, it.productID).Scan(&oldPurchase, &oldSale, &productName); err != nil {
			return PurchaseOrder{}, err
		}

		newSale := oldSale
		if it.salePriceCents > 0 {
			newSale = it.salePriceCents
			_, err = tx.Exec(ctx, `
				UPDATE products
				SET purchase_price = $2,
				    sale_price     = $3,
				    updated_at     = now()
				WHERE id = $1
			`, it.productID, it.unitCostCents, it.salePriceCents)
		} else {
			_, err = tx.Exec(ctx, `
				UPDATE products
				SET purchase_price = $2,
				    updated_at     = now()
				WHERE id = $1
			`, it.productID, it.unitCostCents)
		}
		if err != nil {
			return PurchaseOrder{}, err
		}

		// Emit a per-product audit entry when anything actually changed.
		// Skipped when prices match — avoids audit-log noise on re-receives
		// of an unchanged PO.
		r.emitPriceChangeAudit(ctx, it.productID, productName, poID, userID,
			oldPurchase, it.unitCostCents, oldSale, newSale)
	}

	// 4. Update PO status to received
	po, err := scanPO(tx.QueryRow(ctx, `
		UPDATE purchase_orders
		SET status = 'received', received_at = now(), received_by = $2
		WHERE id = $1
		RETURNING `+poCols,
		poID, uid,
	))
	if err != nil {
		return PurchaseOrder{}, err
	}

	// 5. Create expense finance transaction (debt to supplier)
	reason := "Purchase Order #" + strconv.FormatInt(poID, 10)
	finTxn := finance.Transaction{
		Type:         "expense",
		AmountCents:  totalCents,
		RelatedTable: "purchase",
		RelatedID:    &poID,
		Status:       "pending",
		Reason:       &reason,
		CreatedBy:    uid,
	}
	if err := finRepo.CreateTransaction(ctx, tx, &finTxn); err != nil {
		return PurchaseOrder{}, err
	}

	// 6. Create supplier debt record
	_, err = tx.Exec(ctx, `
		INSERT INTO supplier_debts
			(supplier_id, purchase_id, amount_cents, remaining_cents, note, created_by)
		VALUES ($1, $2, $3, $3, $4, $5)
	`, supplierID, poID, totalCents, &reason, uid)
	if err != nil {
		return PurchaseOrder{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return PurchaseOrder{}, err
	}
	return po, nil
}

// ── CancelPO ──────────────────────────────────────────────────────────────────

func (r *Repository) CancelPO(ctx context.Context, poID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status string
	err = tx.QueryRow(ctx, `
		SELECT status FROM purchase_orders WHERE id = $1 FOR UPDATE
	`, poID).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return apperr.NotFound("PO_NOT_FOUND", "purchase order not found")
		}
		return err
	}

	switch status {
	case "received":
		return apperr.Conflict("PO_ALREADY_RECEIVED", "cannot cancel a received purchase order")
	case "cancelled":
		return apperr.Conflict("PO_ALREADY_CANCELLED", "purchase order is already cancelled")
	}

	_, err = tx.Exec(ctx, `UPDATE purchase_orders SET status = 'cancelled' WHERE id = $1`, poID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ── AddPayment ────────────────────────────────────────────────────────────────

func (r *Repository) AddPayment(
	ctx context.Context,
	poID int64,
	req AddPaymentRequest,
	userID int64,
	finRepo *finance.Repository,
) (finance.Payment, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return finance.Payment{}, err
	}
	defer tx.Rollback(ctx)

	uid := &userID

	// 1. Lock transaction for this PO
	var txnID int64
	var txnStatus string
	var txnAmount int64
	err = tx.QueryRow(ctx, `
		SELECT id, status, amount_cents
		FROM transactions
		WHERE related_table = 'purchase' AND related_id = $1
		FOR UPDATE
	`, poID).Scan(&txnID, &txnStatus, &txnAmount)
	if err != nil {
		if err == pgx.ErrNoRows {
			return finance.Payment{}, apperr.Conflict("PO_NOT_RECEIVED", "purchase order has not been received yet")
		}
		return finance.Payment{}, err
	}
	if txnStatus == "canceled" {
		return finance.Payment{}, apperr.Conflict("TRANSACTION_CANCELLED", "transaction has been cancelled")
	}
	if txnStatus == "paid" {
		return finance.Payment{}, apperr.Conflict("ALREADY_PAID", "purchase order is already fully paid")
	}

	// 2. Check existing payments
	var paidSum int64
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount_cents), 0) FROM payments WHERE transaction_id = $1`,
		txnID,
	).Scan(&paidSum); err != nil {
		return finance.Payment{}, err
	}
	if paidSum+req.AmountCents > txnAmount {
		return finance.Payment{}, apperr.Validation(
			fmt.Sprintf("payment amount exceeds outstanding balance (%d cents remaining)",
				txnAmount-paidSum),
		)
	}

	// 3. Create payment
	p := finance.Payment{
		TransactionID: txnID,
		PaymentTypeID: req.PaymentTypeID,
		AmountCents:   req.AmountCents,
		Note:          req.Note,
		CreatedBy:     uid,
	}
	if err := finRepo.CreatePayment(ctx, tx, &p); err != nil {
		return finance.Payment{}, err
	}

	// 4. Update transaction status
	newPaid := paidSum + req.AmountCents
	newStatus := "partial"
	if newPaid >= txnAmount {
		newStatus = "paid"
	}
	if _, err := finRepo.UpdateTransactionStatus(ctx, tx, txnID, newStatus); err != nil {
		return finance.Payment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return finance.Payment{}, err
	}
	return p, nil
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func (r *Repository) GetByID(ctx context.Context, id int64) (PurchaseOrder, []PurchaseItem, error) {
	po, err := scanPO(r.db.QueryRow(ctx,
		`SELECT `+poCols+` FROM purchase_orders WHERE id = $1`, id,
	))
	if err != nil {
		return PurchaseOrder{}, nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT poi.id, poi.po_id, poi.product_id,
		       poi.qty_milli, poi.unit_cost_cents, poi.sale_price_cents, poi.line_total_cents, poi.created_at,
		       p.name
		FROM purchase_order_items poi
		JOIN products p ON p.id = poi.product_id
		WHERE poi.po_id = $1
		ORDER BY poi.id
	`, id)
	if err != nil {
		return po, nil, err
	}
	defer rows.Close()

	var items []PurchaseItem
	for rows.Next() {
		var it PurchaseItem
		if err := rows.Scan(
			&it.ID, &it.POID, &it.ProductID,
			&it.QtyMilli, &it.UnitCostCents, &it.SalePriceCents, &it.LineTotalCents, &it.CreatedAt,
			&it.ProductName,
		); err != nil {
			return po, nil, err
		}
		items = append(items, it)
	}
	if items == nil {
		items = []PurchaseItem{}
	}
	return po, items, rows.Err()
}

// ── GetTransactionID ──────────────────────────────────────────────────────────

func (r *Repository) GetTransactionID(ctx context.Context, poID int64) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx,
		`SELECT id FROM transactions WHERE related_table = 'purchase' AND related_id = $1`, poID,
	).Scan(&id)
	return id, err
}

// ── List ──────────────────────────────────────────────────────────────────────

func (r *Repository) List(
	ctx context.Context,
	supplierID, warehouseID *int64,
	status *string,
	dateFrom, dateTo *time.Time,
	limit, offset int,
) ([]POListItem, int, error) {

	where := `
		WHERE ($1::bigint IS NULL OR po.supplier_id = $1)
		  AND ($2::bigint IS NULL OR po.warehouse_id = $2)
		  AND ($3::text   IS NULL OR po.status = $3)
		  AND ($4::timestamptz IS NULL OR po.created_at >= $4)
		  AND ($5::timestamptz IS NULL OR po.created_at <= $5)
	`

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM purchase_orders po `+where,
		supplierID, warehouseID, status, dateFrom, dateTo,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT po.id, po.supplier_id, po.warehouse_id, po.status, po.total_cents, po.items_count,
		       po.note, po.created_by, po.created_at, po.received_at, po.received_by,
		       s.name, w.name,
		       COALESCE(uc.full_name, ''), COALESCE(ur.full_name, '')
		FROM purchase_orders po
		LEFT JOIN suppliers s  ON s.id  = po.supplier_id
		LEFT JOIN warehouses w ON w.id  = po.warehouse_id
		LEFT JOIN users uc     ON uc.id = po.created_by
		LEFT JOIN users ur     ON ur.id = po.received_by
		`+where+`
		ORDER BY po.created_at DESC, po.id DESC
		LIMIT $6 OFFSET $7
	`, supplierID, warehouseID, status, dateFrom, dateTo, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []POListItem
	for rows.Next() {
		var item POListItem
		var supplierName, warehouseName *string
		err := rows.Scan(
			&item.ID, &item.SupplierID, &item.WarehouseID, &item.Status, &item.TotalCents, &item.ItemsCount,
			&item.Note, &item.CreatedBy, &item.CreatedAt, &item.ReceivedAt, &item.ReceivedBy,
			&supplierName, &warehouseName,
			&item.CreatedByName, &item.ReceivedByName,
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
	return out, total, rows.Err()
}

// ── DebtSummary ───────────────────────────────────────────────────────────────

func (r *Repository) DebtSummary(ctx context.Context) ([]SupplierDebtRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
		    s.id,
		    s.name,
		    SUM(t.amount_cents)                            AS total_cents,
		    COALESCE(SUM(ps.paid_cents), 0)                AS paid_cents,
		    SUM(t.amount_cents) - COALESCE(SUM(ps.paid_cents), 0) AS debt_cents
		FROM purchase_orders po
		JOIN suppliers s ON s.id = po.supplier_id
		JOIN transactions t
		    ON t.related_table = 'purchase'
		    AND t.related_id = po.id
		    AND t.status != 'canceled'
		LEFT JOIN (
		    SELECT transaction_id, SUM(amount_cents) AS paid_cents
		    FROM payments
		    GROUP BY transaction_id
		) ps ON ps.transaction_id = t.id
		GROUP BY s.id, s.name
		HAVING SUM(t.amount_cents) - COALESCE(SUM(ps.paid_cents), 0) > 0
		ORDER BY debt_cents DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SupplierDebtRow
	for rows.Next() {
		var row SupplierDebtRow
		if err := rows.Scan(
			&row.SupplierID, &row.SupplierName,
			&row.TotalCents, &row.PaidCents, &row.DebtCents,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if out == nil {
		out = []SupplierDebtRow{}
	}
	return out, rows.Err()
}

// emitPriceChangeAudit writes a per-product audit_logs entry when ReceivePO
// changes purchase_price or sale_price on the products table. The middleware
// only sees the outer HTTP call (PO_RECEIVE) and would miss these internal
// UPDATEs, so we synthesize the entry here.
//
// Fire-and-forget: errors are swallowed by the auditlog repository itself
// (it logs to stderr and bumps the failed-writes counter).
func (r *Repository) emitPriceChangeAudit(
	ctx context.Context,
	productID int64, productName string,
	poID, userID int64,
	oldPurchase, newPurchase, oldSale, newSale int64,
) {
	if r.auditRepo == nil {
		return
	}
	if oldPurchase == newPurchase && oldSale == newSale {
		return // nothing actually changed
	}

	oldVal := map[string]any{
		"purchase_price": oldPurchase,
		"sale_price":     oldSale,
	}
	newVal := map[string]any{
		"purchase_price": newPurchase,
		"sale_price":     newSale,
	}
	// Stash product name in the "before" snapshot so the frontend diff view
	// shows what was changed without an extra lookup.
	oldVal["product_name"] = productName
	newVal["product_name"] = productName

	oldRaw, _ := json.Marshal(oldVal)
	newRaw, _ := json.Marshal(newVal)

	entityID := strconv.FormatInt(productID, 10)
	path := "/api/purchases/" + strconv.FormatInt(poID, 10) + "/receive"

	r.auditRepo.Create(ctx, &auditlog.AuditLog{
		UserID:     userID,
		Action:     "PRODUCT_PRICE_CHANGE_VIA_PO",
		EntityType: "product",
		EntityID:   &entityID,
		Method:     "POST",
		Path:       path,
		OldValue:   oldRaw,
		NewValue:   newRaw,
	})
}
