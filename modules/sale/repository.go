package sale

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Repository struct {
	db      *pgxpool.Pool
	baseURL string // PUBLIC_BASE_URL for building product photo URLs
}

func NewRepository(db *pgxpool.Pool, baseURL string) *Repository {
	return &Repository{db: db, baseURL: baseURL}
}

// photoURL converts a nullable product photo_path to a full public URL.
func (r *Repository) photoURL(path *string) *string {
	if path == nil || *path == "" || r.baseURL == "" {
		return nil
	}
	u := r.baseURL + "/uploads/" + *path
	return &u
}

// ── column constants ─────────────────────────────────────────────────────────

const saleCols = `id, warehouse_id, customer_id, total_cents, cost_cents, items_count, note, created_by, created_at`

// ── scan helpers ─────────────────────────────────────────────────────────────

func scanSale(row pgx.Row) (Sale, error) {
	var s Sale
	err := row.Scan(
		&s.ID, &s.WarehouseID, &s.CustomerID, &s.TotalCents,
		&s.CostCents, &s.ItemsCount, &s.Note, &s.CreatedBy, &s.CreatedAt,
	)
	return s, err
}

func lineTotalCents(qtyMilli, unitPriceCents int64) int64 {
	return (qtyMilli * unitPriceCents) / 1000
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// ── CreateSale (atomic: sale + stock-out + finance) ──────────────────────────

func (r *Repository) CreateSale(
	ctx context.Context,
	req CreateSaleRequest,
	userID int64,
	finRepo *finance.Repository,
) (Sale, []SaleItem, *finance.Transaction, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Sale{}, nil, nil, err
	}
	defer tx.Rollback(ctx)

	uid := &userID

	// 1. Batch-fetch product sale prices
	productIDs := make([]int64, len(req.Items))
	for i, item := range req.Items {
		productIDs[i] = item.ProductID
	}

	priceMap := make(map[int64]int64, len(req.Items))
	rows, err := tx.Query(ctx,
		`SELECT id, sale_price FROM products WHERE id = ANY($1) AND is_active = true`,
		productIDs,
	)
	if err != nil {
		return Sale{}, nil, nil, err
	}
	for rows.Next() {
		var pid, sp int64
		if err := rows.Scan(&pid, &sp); err != nil {
			rows.Close()
			return Sale{}, nil, nil, err
		}
		priceMap[pid] = sp
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Sale{}, nil, nil, err
	}

	// Validate all products found
	for i, item := range req.Items {
		if _, ok := priceMap[item.ProductID]; !ok {
			return Sale{}, nil, nil, apperr.Validation(
				fmt.Sprintf("product %d not found or inactive (item index %d)", item.ProductID, i),
			)
		}
	}

	// 2. Calculate total revenue
	var totalCents int64
	for _, item := range req.Items {
		totalCents += lineTotalCents(item.QtyMilli, priceMap[item.ProductID])
	}

	// 3. Insert sale header
	sale, err := scanSale(tx.QueryRow(ctx, `
		INSERT INTO sales (warehouse_id, customer_id, total_cents, cost_cents, items_count, note, created_by)
		VALUES ($1, $2, $3, 0, $4, $5, $6)
		RETURNING `+saleCols,
		req.WarehouseID, req.CustomerID, totalCents, len(req.Items), req.Note, uid,
	))
	if err != nil {
		if isFKViolation(err) {
			return Sale{}, nil, nil, apperr.Validation("warehouse or customer does not exist")
		}
		return Sale{}, nil, nil, err
	}

	// 4. Process items sorted by product_id to avoid deadlocks
	type indexedItem struct {
		idx  int
		item CreateSaleItemRequest
	}
	sorted := make([]indexedItem, len(req.Items))
	for i, item := range req.Items {
		sorted[i] = indexedItem{idx: i, item: item}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].item.ProductID < sorted[j].item.ProductID
	})

	var totalCostCents int64

	for _, entry := range sorted {
		item := entry.item
		unitPrice := priceMap[item.ProductID]
		lineTotal := lineTotalCents(item.QtyMilli, unitPrice)

		// 4a. Lock warehouse_items row
		var currentQty, avgCost, wTotalCost int64
		err = tx.QueryRow(ctx, `
			SELECT qty_milli, avg_cost_cents, total_cost_cents
			FROM warehouse_items
			WHERE warehouse_id = $1 AND product_id = $2
			FOR UPDATE
		`, req.WarehouseID, item.ProductID).Scan(&currentQty, &avgCost, &wTotalCost)
		if err != nil {
			if err == pgx.ErrNoRows {
				return Sale{}, nil, nil, apperr.Validation(
					fmt.Sprintf("no stock for product %d in warehouse (item index %d)",
						item.ProductID, entry.idx),
				)
			}
			return Sale{}, nil, nil, err
		}
		if currentQty < item.QtyMilli {
			return Sale{}, nil, nil, apperr.Validation(
				fmt.Sprintf("insufficient stock for product %d: have %d, need %d (item index %d)",
					item.ProductID, currentQty, item.QtyMilli, entry.idx),
			)
		}

		// 4b. Calculate COGS by avg_cost
		itemCost := lineTotalCents(item.QtyMilli, avgCost)
		if wTotalCost < itemCost {
			return Sale{}, nil, nil, apperr.Internal(
				errors.New("total_cost_cents underflow for product " + strconv.FormatInt(item.ProductID, 10)),
			)
		}

		// 4c. Insert sale_item
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_items (sale_id, product_id, qty_milli, unit_price_cents, cost_cents, line_total_cents)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, sale.ID, item.ProductID, item.QtyMilli, unitPrice, itemCost, lineTotal)
		if err != nil {
			return Sale{}, nil, nil, err
		}

		// 4d. Insert stock ledger entry (type='sale', delta=-qty)
		_, err = tx.Exec(ctx, `
			INSERT INTO warehouse_item_details
				(idempotency_key, warehouse_id, product_id, delta_milli, type, price_cents, created_by)
			VALUES (gen_random_uuid(), $1, $2, $3, 'sale', $4, $5)
		`, req.WarehouseID, item.ProductID, -item.QtyMilli, unitPrice, uid)
		if err != nil {
			return Sale{}, nil, nil, err
		}

		// 4e. Update warehouse_items (decrement stock, subtract cost, recalc avg)
		_, err = tx.Exec(ctx, `
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
		`, req.WarehouseID, item.ProductID, item.QtyMilli, itemCost)
		if err != nil {
			return Sale{}, nil, nil, err
		}

		totalCostCents += itemCost
	}

	// 5. Update sale with accumulated cost
	_, err = tx.Exec(ctx, `UPDATE sales SET cost_cents = $2 WHERE id = $1`, sale.ID, totalCostCents)
	if err != nil {
		return Sale{}, nil, nil, err
	}
	sale.CostCents = totalCostCents

	// 6. Create finance transaction (income)
	reason := "Sale #" + strconv.FormatInt(sale.ID, 10)
	finTxn := finance.Transaction{
		Type:         "income",
		AmountCents:  totalCents,
		RelatedTable: "sale",
		RelatedID:    &sale.ID,
		Status:       "pending",
		Reason:       &reason,
		CreatedBy:    uid,
	}
	if err := finRepo.CreateTransaction(ctx, tx, &finTxn); err != nil {
		return Sale{}, nil, nil, err
	}

	// 7. Optional payment
	if req.PaymentAmount != nil && *req.PaymentAmount > 0 {
		if req.PaymentTypeID == nil {
			return Sale{}, nil, nil, apperr.Validation("payment_type_id is required when payment_amount is provided")
		}
		if *req.PaymentAmount > totalCents {
			return Sale{}, nil, nil, apperr.Validation("payment_amount cannot exceed sale total")
		}

		p := finance.Payment{
			TransactionID: finTxn.ID,
			PaymentTypeID: *req.PaymentTypeID,
			AmountCents:   *req.PaymentAmount,
			Note:          req.PaymentNote,
			CreatedBy:     uid,
		}
		if err := finRepo.CreatePayment(ctx, tx, &p); err != nil {
			return Sale{}, nil, nil, err
		}

		status := "partial"
		if *req.PaymentAmount >= totalCents {
			status = "paid"
		}
		if _, err := finRepo.UpdateTransactionStatus(ctx, tx, finTxn.ID, status); err != nil {
			return Sale{}, nil, nil, err
		}
		finTxn.Status = status
	}

	// 8. Commit
	if err := tx.Commit(ctx); err != nil {
		return Sale{}, nil, nil, err
	}

	// 9. Refetch items with product name + photo (after commit, outside TX)
	_, items, err := r.GetByID(ctx, sale.ID)
	if err != nil {
		return sale, nil, &finTxn, err
	}

	return sale, items, &finTxn, nil
}

// ── GetByID ──────────────────────────────────────────────────────────────────

func (r *Repository) GetByID(ctx context.Context, id int64) (Sale, []SaleItem, error) {
	sale, err := scanSale(r.db.QueryRow(ctx,
		`SELECT `+saleCols+` FROM sales WHERE id = $1`, id,
	))
	if err != nil {
		return Sale{}, nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT si.id, si.sale_id, si.product_id, si.qty_milli, si.unit_price_cents,
		       si.cost_cents, si.line_total_cents, si.created_at,
		       p.name, p.photo_path
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		WHERE si.sale_id = $1
		ORDER BY si.id
	`, id)
	if err != nil {
		return sale, nil, err
	}
	defer rows.Close()

	var items []SaleItem
	for rows.Next() {
		var si SaleItem
		var photoPath *string
		err := rows.Scan(
			&si.ID, &si.SaleID, &si.ProductID, &si.QtyMilli, &si.UnitPriceCents,
			&si.CostCents, &si.LineTotalCents, &si.CreatedAt,
			&si.ProductName, &photoPath,
		)
		if err != nil {
			return sale, nil, err
		}
		si.ProductPhotoURL = r.photoURL(photoPath)
		items = append(items, si)
	}
	if items == nil {
		items = []SaleItem{}
	}
	return sale, items, rows.Err()
}

// ── GetTransactionIDBySaleID ─────────────────────────────────────────────────

func (r *Repository) GetTransactionIDBySaleID(ctx context.Context, saleID int64) (int64, error) {
	var txnID int64
	err := r.db.QueryRow(ctx,
		`SELECT id FROM transactions WHERE related_table = 'sale' AND related_id = $1`, saleID,
	).Scan(&txnID)
	return txnID, err
}

// ── List ─────────────────────────────────────────────────────────────────────

func (r *Repository) List(
	ctx context.Context,
	warehouseID, customerID *int64,
	createdBy *int64,
	dateFrom, dateTo *time.Time,
	limit, offset int,
) ([]SaleListItem, int, error) {

	where := `
		WHERE ($1::bigint IS NULL OR s.warehouse_id = $1)
		  AND ($2::bigint IS NULL OR s.customer_id = $2)
		  AND ($3::timestamptz IS NULL OR s.created_at >= $3)
		  AND ($4::timestamptz IS NULL OR s.created_at <= $4)
		  AND ($5::bigint IS NULL OR s.created_by = $5)
	`

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM sales s `+where,
		warehouseID, customerID, dateFrom, dateTo, createdBy,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT s.`+saleCols+`, c.name
		FROM sales s
		LEFT JOIN customers c ON c.id = s.customer_id
		`+where+`
		ORDER BY s.created_at DESC, s.id DESC
		LIMIT $6 OFFSET $7
	`, warehouseID, customerID, dateFrom, dateTo, createdBy, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []SaleListItem
	for rows.Next() {
		var item SaleListItem
		err := rows.Scan(
			&item.ID, &item.WarehouseID, &item.CustomerID, &item.TotalCents,
			&item.CostCents, &item.ItemsCount, &item.Note, &item.CreatedBy, &item.CreatedAt,
			&item.CustomerName,
		)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}
