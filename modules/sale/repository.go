package sale

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/modules/customer"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/customerdebt"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/discountrule"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/receiptsettings"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Repository struct {
	db               *pgxpool.Pool
	baseURL          string // PUBLIC_BASE_URL for building product photo URLs
	customerRepo     *customer.Repository
	debtRepo         *customerdebt.Repository
	discountRuleRepo *discountrule.Repository
	receiptRepo      *receiptsettings.Repository

	// printerSv is read inside the auto-print goroutine spawned per sale
	// (see sale/service.go ConfirmSale). The setter is called once at
	// startup wiring, but Go's memory model still treats a concurrent
	// write/read on an interface field as a data race — guard it.
	muPrinter sync.RWMutex
	printerSv PrinterService
}

func NewRepository(db *pgxpool.Pool, baseURL string, customerRepo *customer.Repository) *Repository {
	return &Repository{db: db, baseURL: baseURL, customerRepo: customerRepo}
}

func (r *Repository) SetDebtRepo(repo *customerdebt.Repository)         { r.debtRepo = repo }
func (r *Repository) SetDiscountRuleRepo(repo *discountrule.Repository) { r.discountRuleRepo = repo }
func (r *Repository) SetPrinterService(p PrinterService) {
	r.muPrinter.Lock()
	r.printerSv = p
	r.muPrinter.Unlock()
}
func (r *Repository) PrinterService() PrinterService {
	r.muPrinter.RLock()
	defer r.muPrinter.RUnlock()
	return r.printerSv
}
func (r *Repository) DB() *pgxpool.Pool { return r.db }

// photoURL converts a nullable product photo_path to a full public URL.
func (r *Repository) photoURL(path *string) *string {
	if path == nil || *path == "" || r.baseURL == "" {
		return nil
	}
	u := r.baseURL + "/uploads/" + *path
	return &u
}

// ── column constants ─────────────────────────────────────────────────────────

const saleCols = `id, warehouse_id, customer_id, worker_id, total_cents, cost_cents, bonus_used_cents, discount_percent, discount_cents, items_count, note, created_by, created_at, status`

// ── scan helpers ─────────────────────────────────────────────────────────────

func scanSale(row pgx.Row) (Sale, error) {
	var s Sale
	err := row.Scan(
		&s.ID, &s.WarehouseID, &s.CustomerID, &s.WorkerID, &s.TotalCents,
		&s.CostCents, &s.BonusUsedCents, &s.DiscountPercent, &s.DiscountCents,
		&s.ItemsCount, &s.Note, &s.CreatedBy, &s.CreatedAt,
		&s.Status,
	)
	return s, err
}

func lineTotalCents(qtyMilli, unitPriceCents int64) int64 {
	return (qtyMilli*unitPriceCents + 500) / 1000
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// ── CreateSale (draft: reserves stock, no immediate deduction) ───────────────

func (r *Repository) CreateSale(
	ctx context.Context,
	req CreateSaleRequest,
	userID int64,
) (Sale, []SaleItem, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Sale{}, nil, err
	}
	defer tx.Rollback(ctx)

	uid := &userID

	// 1. Batch-fetch product sale prices
	productIDs := make([]int64, len(req.Items))
	for i, item := range req.Items {
		productIDs[i] = item.ProductID
	}

	priceMap := make(map[int64]int64, len(req.Items))
	productDiscountMap := make(map[int64]int, len(req.Items))
	rows, err := tx.Query(ctx,
		`SELECT id, sale_price, discount_percent FROM products WHERE id = ANY($1) AND is_active = true`,
		productIDs,
	)
	if err != nil {
		return Sale{}, nil, err
	}
	for rows.Next() {
		var pid, sp int64
		var dp int
		if err := rows.Scan(&pid, &sp, &dp); err != nil {
			rows.Close()
			return Sale{}, nil, err
		}
		priceMap[pid] = sp
		productDiscountMap[pid] = dp
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Sale{}, nil, err
	}

	for i, item := range req.Items {
		sp, ok := priceMap[item.ProductID]
		if !ok {
			return Sale{}, nil, apperr.Validation(
				fmt.Sprintf("product %d not found or inactive (item index %d)", item.ProductID, i),
			)
		}
		// Block selling at zero price — bulk-imported rows start with
		// purchase_price=0, sale_price=0 and need a human to set them in
		// /products/edit before they can hit the till. Catches the "kassir
		// rings up a free item" footgun.
		if sp == 0 {
			return Sale{}, nil, apperr.Validation(
				fmt.Sprintf("product %d has no sale price set — please set it in /products before selling (item index %d)", item.ProductID, i),
			)
		}
	}

	// 2. Auto-apply product discount if item has no explicit discount
	for i := range req.Items {
		if req.Items[i].DiscountPercent == 0 {
			if pd := productDiscountMap[req.Items[i].ProductID]; pd > 0 {
				req.Items[i].DiscountPercent = pd
			}
		}
	}

	// 3. Calculate total revenue (with per-item discounts)
	var totalCents int64
	for _, item := range req.Items {
		lt := lineTotalCents(item.QtyMilli, priceMap[item.ProductID])
		if item.DiscountPercent > 0 && item.DiscountPercent <= 100 {
			lt = lt - (lt*int64(item.DiscountPercent))/100
		}
		totalCents += lt
	}

	// 3. Insert sale header (status='draft', cost_cents=0)
	sale, err := scanSale(tx.QueryRow(ctx, `
		INSERT INTO sales (warehouse_id, customer_id, worker_id, total_cents, cost_cents, items_count, note, created_by, status)
		VALUES ($1, $2, $3, $4, 0, $5, $6, $7, 'draft')
		RETURNING `+saleCols,
		req.WarehouseID, req.CustomerID, req.WorkerID, totalCents, len(req.Items), req.Note, uid,
	))
	if err != nil {
		if isFKViolation(err) {
			return Sale{}, nil, apperr.Validation("warehouse or customer does not exist")
		}
		return Sale{}, nil, err
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

	for _, entry := range sorted {
		item := entry.item
		unitPrice := priceMap[item.ProductID]
		lineTotal := lineTotalCents(item.QtyMilli, unitPrice)
		if item.DiscountPercent > 0 && item.DiscountPercent <= 100 {
			lineTotal = lineTotal - (lineTotal*int64(item.DiscountPercent))/100
		}

		// 4a. Lock warehouse_items row
		var currentQty int64
		err = tx.QueryRow(ctx, `
			SELECT qty_milli
			FROM warehouse_items
			WHERE warehouse_id = $1 AND product_id = $2
			FOR UPDATE
		`, req.WarehouseID, item.ProductID).Scan(&currentQty)
		if err != nil {
			if err == pgx.ErrNoRows {
				if !req.Force {
					return Sale{}, nil, apperr.Validation(
						fmt.Sprintf("no stock for product %d in warehouse (item index %d)",
							item.ProductID, entry.idx),
					)
				}
				currentQty = 0 // force=true: строки нет, считаем 0
			} else {
				return Sale{}, nil, err
			}
		}

		// 4b. Count active reservations (while holding the lock)
		var activeReserved int64
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(qty_milli), 0)
			FROM stock_reservations
			WHERE warehouse_id = $1 AND product_id = $2 AND status = 'active'
		`, req.WarehouseID, item.ProductID).Scan(&activeReserved)
		if err != nil {
			return Sale{}, nil, err
		}

		available := currentQty - activeReserved
		if available < item.QtyMilli && !req.Force {
			return Sale{}, nil, apperr.Validation(
				fmt.Sprintf("insufficient available stock for product %d: have %d milli, reserved %d milli, need %d milli (use force=true to sell in deficit)",
					item.ProductID, currentQty, activeReserved, item.QtyMilli),
			)
		}

		// 4c. Insert sale_item (cost_cents=0, will be set at confirm time)
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_items (sale_id, product_id, qty_milli, unit_price_cents, cost_cents, line_total_cents, discount_percent)
			VALUES ($1, $2, $3, $4, 0, $5, $6)
		`, sale.ID, item.ProductID, item.QtyMilli, unitPrice, lineTotal, item.DiscountPercent)
		if err != nil {
			return Sale{}, nil, err
		}
	}

	// 5. Insert stock reservations (one per item)
	for _, entry := range sorted {
		item := entry.item
		_, err = tx.Exec(ctx, `
			INSERT INTO stock_reservations (sale_id, warehouse_id, product_id, qty_milli)
			VALUES ($1, $2, $3, $4)
		`, sale.ID, req.WarehouseID, item.ProductID, item.QtyMilli)
		if err != nil {
			return Sale{}, nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Sale{}, nil, err
	}

	// Refetch items with product name + photo (after commit, outside TX)
	_, items, err := r.GetByID(ctx, sale.ID)
	if err != nil {
		return sale, nil, err
	}
	return sale, items, nil
}

// reservationRow holds data fetched from stock_reservations for ConfirmSale.
type reservationRow struct {
	id          int64
	productID   int64
	warehouseID int64
	qtyMilli    int64
}

// ── ConfirmSale (draft → confirmed: deducts stock, creates finance tx) ───────

func (r *Repository) ConfirmSale(
	ctx context.Context,
	saleID int64,
	req ConfirmSaleRequest,
	userID int64,
	finRepo *finance.Repository,
) (Sale, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Sale{}, err
	}
	defer tx.Rollback(ctx)

	uid := &userID

	// 1. Lock sale row and check status
	var status string
	var totalCents int64
	var saleCustomerID *int64
	var saleWorkerID *int64
	err = tx.QueryRow(ctx, `
		SELECT status, total_cents, customer_id, worker_id FROM sales WHERE id = $1 FOR UPDATE
	`, saleID).Scan(&status, &totalCents, &saleCustomerID, &saleWorkerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Sale{}, apperr.NotFound("SALE_NOT_FOUND", "sale not found")
		}
		return Sale{}, err
	}
	if status != "draft" {
		return Sale{}, apperr.Conflict("SALE_NOT_DRAFT", "sale is not in draft status")
	}

	// 1b. Validate and deduct bonus points if requested
	bonusUsed := int64(0)
	if req.BonusUsedCents != nil && *req.BonusUsedCents > 0 {
		bonusUsed = *req.BonusUsedCents
		if saleCustomerID == nil {
			return Sale{}, apperr.Validation("bonus redemption requires a customer on the sale")
		}
		if bonusUsed > totalCents {
			return Sale{}, apperr.Validation("bonus_used_cents exceeds sale total")
		}
		if err := r.customerRepo.DeductBonus(ctx, tx, *saleCustomerID, bonusUsed); err != nil {
			return Sale{}, err
		}
	}

	// 2. Get active reservations (sorted by product_id to avoid deadlocks)
	resRows, err := tx.Query(ctx, `
		SELECT sr.id, sr.product_id, sr.warehouse_id, sr.qty_milli
		FROM stock_reservations sr
		WHERE sr.sale_id = $1 AND sr.status = 'active'
		ORDER BY sr.product_id
	`, saleID)
	if err != nil {
		return Sale{}, err
	}
	var reservations []reservationRow
	for resRows.Next() {
		var res reservationRow
		if err := resRows.Scan(&res.id, &res.productID, &res.warehouseID, &res.qtyMilli); err != nil {
			resRows.Close()
			return Sale{}, err
		}
		reservations = append(reservations, res)
	}
	resRows.Close()
	if err := resRows.Err(); err != nil {
		return Sale{}, err
	}

	var totalCostCents int64

	// 3. For each reservation: deduct stock and record COGS
	for _, res := range reservations {
		var currentQty, avgCost int64
		err = tx.QueryRow(ctx, `
			SELECT qty_milli, avg_cost_cents
			FROM warehouse_items
			WHERE warehouse_id = $1 AND product_id = $2
			FOR UPDATE
		`, res.warehouseID, res.productID).Scan(&currentQty, &avgCost)
		if err != nil {
			if err == pgx.ErrNoRows {
				if !req.Force {
					return Sale{}, apperr.Validation(
						fmt.Sprintf("no stock for product %d in warehouse", res.productID),
					)
				}
				currentQty, avgCost = 0, 0 // force: строки нет, себестоимость 0
			} else {
				return Sale{}, err
			}
		}
		if currentQty < res.qtyMilli && !req.Force {
			return Sale{}, apperr.Validation(
				fmt.Sprintf("insufficient stock for product %d: have %d milli, need %d milli (use force=true to override)",
					res.productID, currentQty, res.qtyMilli),
			)
		}

		itemCost := lineTotalCents(res.qtyMilli, avgCost)

		// Insert stock ledger entry (type='sale', delta=-qty)
		_, err = tx.Exec(ctx, `
			INSERT INTO warehouse_item_details
				(idempotency_key, warehouse_id, product_id, delta_milli, type, created_by)
			VALUES (gen_random_uuid(), $1, $2, $3, 'sale', $4)
		`, res.warehouseID, res.productID, -res.qtyMilli, uid)
		if err != nil {
			return Sale{}, err
		}

		// Upsert warehouse_items (decrement stock; INSERT if row didn't exist)
		_, err = tx.Exec(ctx, `
			INSERT INTO warehouse_items
				(warehouse_id, product_id, qty_milli, avg_cost_cents, total_cost_cents, updated_at)
			VALUES ($1, $2, (0 - $3::bigint), 0, 0, now())
			ON CONFLICT (warehouse_id, product_id) DO UPDATE SET
				qty_milli        = warehouse_items.qty_milli - $3::bigint,
				total_cost_cents = GREATEST(warehouse_items.total_cost_cents - $4::bigint, 0),
				-- NUMERIC intermediate so the *1000 multiply can't overflow
				-- int64 on large warehouse totals before the divide.
				avg_cost_cents   = CASE
					WHEN (warehouse_items.qty_milli - $3::bigint) > 0
						THEN (((warehouse_items.total_cost_cents - $4::bigint)::numeric * 1000)
						     / (warehouse_items.qty_milli - $3::bigint))::bigint
					ELSE 0
				END,
				updated_at = now()
		`, res.warehouseID, res.productID, res.qtyMilli, itemCost)
		if err != nil {
			return Sale{}, err
		}

		// Update sale_item.cost_cents
		_, err = tx.Exec(ctx, `
			UPDATE sale_items SET cost_cents = $3 WHERE sale_id = $1 AND product_id = $2
		`, saleID, res.productID, itemCost)
		if err != nil {
			return Sale{}, err
		}

		// Mark reservation as fulfilled
		_, err = tx.Exec(ctx, `
			UPDATE stock_reservations SET status = 'fulfilled', released_at = now() WHERE id = $1
		`, res.id)
		if err != nil {
			return Sale{}, err
		}

		totalCostCents += itemCost
	}

	// 4. Apply per-sale discount (manual or auto from threshold rules)
	saleDiscountPercent := 0
	var saleDiscountCents int64
	if req.DiscountPercent != nil && *req.DiscountPercent > 0 && *req.DiscountPercent <= 100 {
		// Manual discount from manager
		saleDiscountPercent = *req.DiscountPercent
	} else if r.discountRuleRepo != nil {
		// Auto-apply threshold discount rule
		rule, err := r.discountRuleRepo.FindMatchingRule(ctx, totalCents)
		if err == nil && rule != nil {
			saleDiscountPercent = rule.DiscountPercent
		}
	}
	if saleDiscountPercent > 0 {
		saleDiscountCents = (totalCents * int64(saleDiscountPercent)) / 100
		totalCents = totalCents - saleDiscountCents
	}

	// 5. Update sale to confirmed
	sale, err := scanSale(tx.QueryRow(ctx, `
		UPDATE sales SET status = 'confirmed', cost_cents = $2, bonus_used_cents = $3,
			total_cents = $4, discount_percent = $5, discount_cents = $6
		WHERE id = $1
		RETURNING `+saleCols,
		saleID, totalCostCents, bonusUsed, totalCents, saleDiscountPercent, saleDiscountCents,
	))
	if err != nil {
		return Sale{}, err
	}

	// 5a. Create bonus_payment transaction if bonus was used.
	if bonusUsed > 0 {
		bonusReason := "Sale #" + strconv.FormatInt(saleID, 10) + " (bonus)"
		bonusTxn := finance.Transaction{
			Type:         "bonus_payment",
			AmountCents:  bonusUsed,
			RelatedTable: "sale",
			RelatedID:    &saleID,
			Status:       "paid",
			Reason:       &bonusReason,
			CreatedBy:    uid,
		}
		if err := finRepo.CreateTransaction(ctx, tx, &bonusTxn); err != nil {
			return Sale{}, err
		}
	}

	// 5b. Create income transaction for the cash portion (if any).
	effectiveAmount := totalCents - bonusUsed
	var finTxn finance.Transaction
	if effectiveAmount > 0 {
		reason := "Sale #" + strconv.FormatInt(saleID, 10)
		finTxn = finance.Transaction{
			Type:         "income",
			AmountCents:  effectiveAmount,
			RelatedTable: "sale",
			RelatedID:    &saleID,
			Status:       "pending",
			Reason:       &reason,
			CreatedBy:    uid,
		}
		if err := finRepo.CreateTransaction(ctx, tx, &finTxn); err != nil {
			return Sale{}, err
		}
	}

	// 6. Worker credit purchase — create a worker_debt of type='purchase'
	if saleWorkerID != nil {
		debtNote := "Sale #" + strconv.FormatInt(saleID, 10)
		_, err = tx.Exec(ctx, `
			INSERT INTO worker_debts (worker_id, amount_cents, remaining_cents, type, note, created_by)
			VALUES ($1, $2, $2, 'purchase', $3, $4)
		`, *saleWorkerID, totalCents, debtNote, uid)
		if err != nil {
			return Sale{}, err
		}
	}

	// 7. Optional payment (only when there is a cash portion to pay)
	if req.PaymentAmount != nil && *req.PaymentAmount > 0 {
		if effectiveAmount <= 0 {
			return Sale{}, apperr.Validation("payment not accepted: entire sale was paid by bonus")
		}
		if req.PaymentTypeID == nil {
			return Sale{}, apperr.Validation("payment_type_id is required when payment_amount is provided")
		}
		// Allow payment_amount > effectiveAmount: the excess is customer change (sdaça),
		// and the stored amount is what the cashier physically received.

		p := finance.Payment{
			TransactionID: finTxn.ID,
			PaymentTypeID: *req.PaymentTypeID,
			AmountCents:   *req.PaymentAmount,
			Note:          req.PaymentNote,
			CreatedBy:     uid,
		}
		if err := finRepo.CreatePayment(ctx, tx, &p); err != nil {
			return Sale{}, err
		}

		txnStatus := "partial"
		if *req.PaymentAmount >= effectiveAmount {
			txnStatus = "paid"
		}
		if _, err := finRepo.UpdateTransactionStatus(ctx, tx, finTxn.ID, txnStatus); err != nil {
			return Sale{}, err
		}
	}

	// 8. Customer debt — if payment_type is "debt" (id=4) and customer is set
	if req.PaymentTypeID != nil && *req.PaymentTypeID == 4 && saleCustomerID != nil && r.debtRepo != nil {
		paidAmount := int64(0)
		if req.PaymentAmount != nil {
			paidAmount = *req.PaymentAmount
		}
		debtAmount := effectiveAmount - paidAmount
		if debtAmount > 0 {
			debtNote := "Sale #" + strconv.FormatInt(saleID, 10)
			if _, err := r.debtRepo.CreateDebt(ctx, tx, *saleCustomerID, &saleID, debtAmount, debtNote, uid); err != nil {
				return Sale{}, err
			}
		}
	}

	// 9. Auto-accrue bonus points for customer (based on bonus_percent setting)
	if saleCustomerID != nil && r.customerRepo != nil && totalCents > 0 {
		bonusPercent := 1 // default 1%
		if r.receiptRepo != nil {
			if bp, err := r.receiptRepo.GetBonusPercent(ctx); err == nil && bp > 0 {
				bonusPercent = bp
			}
		}
		bonusCents := (totalCents * int64(bonusPercent)) / 100
		if bonusCents > 0 {
			if err := r.customerRepo.AddSpentTx(ctx, tx, *saleCustomerID, totalCents, bonusCents); err != nil {
				return Sale{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Sale{}, err
	}
	return sale, nil
}

// ── CancelSale (draft → cancelled or confirmed → cancelled) ─────────────────

func (r *Repository) TransferDraft(ctx context.Context, saleID, currentUserID, newCashierID int64, callerRoles []string) error {
	var status string
	var createdBy *int64
	err := r.db.QueryRow(ctx,
		`SELECT status, created_by FROM sales WHERE id = $1`, saleID,
	).Scan(&status, &createdBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return apperr.NotFound("SALE_NOT_FOUND", "sale not found")
		}
		return err
	}

	if status != "draft" {
		return apperr.Validation("only draft sales can be transferred")
	}

	// Only owner or manager/admin can transfer
	isManagerOrAdmin := false
	for _, r := range callerRoles {
		if r == "admin" || r == "manager" {
			isManagerOrAdmin = true
			break
		}
	}
	if createdBy != nil && *createdBy != currentUserID && !isManagerOrAdmin {
		return apperr.Forbidden("only the owner or manager can transfer a draft")
	}

	ct, err := r.db.Exec(ctx,
		`UPDATE sales SET created_by = $2 WHERE id = $1 AND status = 'draft'`,
		saleID, newCashierID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return apperr.NotFound("SALE_NOT_FOUND", "sale not found or not draft")
	}
	return nil
}

func (r *Repository) CancelSale(ctx context.Context, saleID int64, userID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	uid := &userID

	// 1. Lock sale row and check status
	var status string
	var warehouseID int64
	var saleCustomerID *int64
	var saleWorkerID *int64
	var bonusUsedCents int64
	err = tx.QueryRow(ctx, `
		SELECT status, warehouse_id, customer_id, worker_id, bonus_used_cents FROM sales WHERE id = $1 FOR UPDATE
	`, saleID).Scan(&status, &warehouseID, &saleCustomerID, &saleWorkerID, &bonusUsedCents)
	if err != nil {
		if err == pgx.ErrNoRows {
			return apperr.NotFound("SALE_NOT_FOUND", "sale not found")
		}
		return err
	}

	switch status {
	case "draft":
		// Release all active reservations
		_, err = tx.Exec(ctx, `
			UPDATE stock_reservations
			SET status = 'released', released_at = now()
			WHERE sale_id = $1 AND status = 'active'
		`, saleID)
		if err != nil {
			return err
		}

	case "confirmed":
		// Fetch sale items for stock restoration (sorted by product_id)
		type saleItemRow struct {
			productID int64
			qtyMilli  int64
			costCents int64
		}
		siRows, err := tx.Query(ctx, `
			SELECT product_id, qty_milli, cost_cents
			FROM sale_items
			WHERE sale_id = $1
			ORDER BY product_id
		`, saleID)
		if err != nil {
			return err
		}
		var saleItems []saleItemRow
		for siRows.Next() {
			var si saleItemRow
			if err := siRows.Scan(&si.productID, &si.qtyMilli, &si.costCents); err != nil {
				siRows.Close()
				return err
			}
			saleItems = append(saleItems, si)
		}
		siRows.Close()
		if err := siRows.Err(); err != nil {
			return err
		}

		// Restore stock for each item
		for _, si := range saleItems {
			var currentQty, avgCost, wTotalCost int64
			rowExists := true
			err = tx.QueryRow(ctx, `
				SELECT qty_milli, avg_cost_cents, total_cost_cents
				FROM warehouse_items
				WHERE warehouse_id = $1 AND product_id = $2
				FOR UPDATE
			`, warehouseID, si.productID).Scan(&currentQty, &avgCost, &wTotalCost)
			if err != nil {
				if err != pgx.ErrNoRows {
					return err
				}
				rowExists = false
			}

			// Use the stored COGS as restoration cost (exact reversal)
			inCost := si.costCents
			newQty := currentQty + si.qtyMilli
			newTotalCost := wTotalCost + inCost
			newAvgCost := int64(0)
			if newQty > 0 {
				newAvgCost = (newTotalCost * 1000) / newQty
			}

			// Insert stock ledger entry (type='sale_return', delta=+qty)
			_, err = tx.Exec(ctx, `
				INSERT INTO warehouse_item_details
					(idempotency_key, warehouse_id, product_id, delta_milli, type, created_by)
				VALUES (gen_random_uuid(), $1, $2, $3, 'sale_return', $4)
			`, warehouseID, si.productID, si.qtyMilli, uid)
			if err != nil {
				return err
			}

			if rowExists {
				_, err = tx.Exec(ctx, `
					UPDATE warehouse_items
					SET qty_milli = $3, total_cost_cents = $4, avg_cost_cents = $5, updated_at = now()
					WHERE warehouse_id = $1 AND product_id = $2
				`, warehouseID, si.productID, newQty, newTotalCost, newAvgCost)
			} else {
				_, err = tx.Exec(ctx, `
					INSERT INTO warehouse_items
						(warehouse_id, product_id, qty_milli, avg_cost_cents, total_cost_cents, updated_at)
					VALUES ($1, $2, $3, $4, $5, now())
				`, warehouseID, si.productID, newQty, newAvgCost, newTotalCost)
			}
			if err != nil {
				return err
			}
		}

		// Cancel the finance transaction for this sale
		_, err = tx.Exec(ctx, `
			UPDATE transactions SET status = 'canceled'
			WHERE related_table = 'sale' AND related_id = $1
		`, saleID)
		if err != nil {
			return err
		}

		// Restore bonus points to customer if any were used
		if bonusUsedCents > 0 && saleCustomerID != nil {
			_, err = tx.Exec(ctx, `
				UPDATE customers SET bonus_points = bonus_points + $2, updated_at = now()
				WHERE id = $1 AND deleted_at IS NULL
			`, *saleCustomerID, bonusUsedCents)
			if err != nil {
				return err
			}
		}

		// Cancel open worker credit debt created during ConfirmSale. Use
		// soft-cancel (status='canceled') instead of DELETE — payroll_run_debts
		// may hold a FK to this row, and a hard DELETE would either fail with
		// a constraint violation or, with future ON DELETE CASCADE, silently
		// erase a snapshotted debt from a calculated payroll run.
		if saleWorkerID != nil {
			debtNote := "Sale #" + strconv.FormatInt(saleID, 10)
			_, err = tx.Exec(ctx, `
				UPDATE worker_debts
				SET status = 'canceled', remaining_cents = 0, updated_at = now()
				WHERE worker_id = $1 AND type = 'purchase' AND note = $2 AND status = 'open'
			`, *saleWorkerID, debtNote)
			if err != nil {
				return err
			}
		}

	case "cancelled":
		return apperr.Conflict("SALE_ALREADY_CANCELLED", "sale is already cancelled")

	default:
		return apperr.Internal(errors.New("unknown sale status: " + status))
	}

	// Update sale status to cancelled
	_, err = tx.Exec(ctx, `UPDATE sales SET status = 'cancelled' WHERE id = $1`, saleID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
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
		       si.cost_cents, si.line_total_cents, si.discount_percent, si.created_at,
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
			&si.CostCents, &si.LineTotalCents, &si.DiscountPercent, &si.CreatedAt,
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

// ── GetReceiptData fetches sale + items with unit_type + cashier/warehouse/customer names ──

type ReceiptSaleRow struct {
	Sale
	CashierName   string
	WarehouseName string
	CustomerName  string
	WorkerName    string
	PaymentMethod string // cash, card, bank_transfer, debt, etc.
	PaidCents     int64  // amount actually paid
}

type ReceiptItemRow struct {
	ProductName     string
	QtyMilli        int64
	UnitType        string
	UnitPriceCents  int64
	LineTotalCents  int64
	DiscountPercent int
}

func (r *Repository) GetReceiptData(ctx context.Context, id int64) (ReceiptSaleRow, []ReceiptItemRow, error) {
	var row ReceiptSaleRow
	var cashierName, warehouseName, customerName, workerName, paymentMethod *string

	err := r.db.QueryRow(ctx, `
		SELECT s.id, s.warehouse_id, s.customer_id, s.worker_id, s.total_cents, s.cost_cents,
		       s.bonus_used_cents, s.discount_percent, s.discount_cents,
		       s.items_count, s.note, s.created_by, s.created_at, s.status,
		       u.username,
		       w.name,
		       c.name,
		       wk.name,
		       (SELECT STRING_AGG(DISTINCT pt.name, ', ' ORDER BY pt.name)
		        FROM payments p
		        JOIN transactions t ON t.id = p.transaction_id
		        JOIN payment_types pt ON pt.id = p.payment_type_id
		        WHERE t.related_table = 'sale' AND t.related_id = s.id),
		       COALESCE((SELECT SUM(p.amount_cents) FROM payments p
		        JOIN transactions t ON t.id = p.transaction_id
		        WHERE t.related_table = 'sale' AND t.related_id = s.id), 0)
		FROM sales s
		LEFT JOIN employees u ON u.id = s.created_by
		LEFT JOIN warehouses w ON w.id = s.warehouse_id
		LEFT JOIN customers c ON c.id = s.customer_id
		LEFT JOIN employees wk ON wk.id = s.worker_id AND wk.deleted_at IS NULL
		WHERE s.id = $1
	`, id).Scan(
		&row.ID, &row.WarehouseID, &row.CustomerID, &row.WorkerID, &row.TotalCents,
		&row.CostCents, &row.BonusUsedCents, &row.DiscountPercent, &row.DiscountCents,
		&row.ItemsCount, &row.Note, &row.CreatedBy, &row.CreatedAt,
		&row.Status,
		&cashierName, &warehouseName, &customerName, &workerName, &paymentMethod, &row.PaidCents,
	)
	if err != nil {
		return ReceiptSaleRow{}, nil, err
	}
	if cashierName != nil {
		row.CashierName = *cashierName
	}
	if warehouseName != nil {
		row.WarehouseName = *warehouseName
	}
	if customerName != nil {
		row.CustomerName = *customerName
	}
	if workerName != nil {
		row.WorkerName = *workerName
	}
	if paymentMethod != nil {
		row.PaymentMethod = *paymentMethod
	}

	rows, err := r.db.Query(ctx, `
		SELECT p.name, si.qty_milli, p.unit_type, si.unit_price_cents, si.line_total_cents, si.discount_percent
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		WHERE si.sale_id = $1
		ORDER BY si.id
	`, id)
	if err != nil {
		return row, nil, err
	}
	defer rows.Close()

	var items []ReceiptItemRow
	for rows.Next() {
		var it ReceiptItemRow
		if err := rows.Scan(&it.ProductName, &it.QtyMilli, &it.UnitType, &it.UnitPriceCents, &it.LineTotalCents, &it.DiscountPercent); err != nil {
			return row, nil, err
		}
		items = append(items, it)
	}
	if items == nil {
		items = []ReceiptItemRow{}
	}
	return row, items, rows.Err()
}

// ── GetTransactionIDBySaleID ─────────────────────────────────────────────────

func (r *Repository) GetTransactionIDBySaleID(ctx context.Context, saleID int64) (int64, error) {
	var txnID int64
	err := r.db.QueryRow(ctx,
		`SELECT id FROM transactions WHERE related_table = 'sale' AND related_id = $1`, saleID,
	).Scan(&txnID)
	return txnID, err
}

// ── DeleteSaleItem (draft only: removes item, releases reservation, updates totals) ──

// ── DeleteSaleItem (draft only: removes item, releases reservation, updates totals) ──

func (r *Repository) DeleteSaleItem(
	ctx context.Context,
	saleID int64,
	itemID int64,
) (Sale, []SaleItem, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Sale{}, nil, err
	}
	defer tx.Rollback(ctx)

	// 1. Lock sale row and check status
	var status string
	var itemsCount int
	err = tx.QueryRow(ctx, `
		SELECT status, items_count FROM sales WHERE id = $1 FOR UPDATE
	`, saleID).Scan(&status, &itemsCount)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Sale{}, nil, apperr.NotFound("SALE_NOT_FOUND", "sale not found")
		}
		return Sale{}, nil, err
	}
	if status != "draft" {
		return Sale{}, nil, apperr.Conflict("SALE_NOT_DRAFT", "sale must be in draft status")
	}

	// 2. Fetch sale item
	var productID, lineTotalCents int64
	err = tx.QueryRow(ctx, `
		SELECT product_id, line_total_cents
		FROM sale_items
		WHERE id = $1 AND sale_id = $2
	`, itemID, saleID).Scan(&productID, &lineTotalCents)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Sale{}, nil, apperr.NotFound("SALE_ITEM_NOT_FOUND", "sale item not found")
		}
		return Sale{}, nil, err
	}

	// 3. If this is the last item, cancel the whole draft sale
	if itemsCount == 1 {
		// release all active reservations of this sale
		_, err = tx.Exec(ctx, `
			UPDATE stock_reservations
			SET status = 'released', released_at = now()
			WHERE sale_id = $1 AND status = 'active'
		`, saleID)
		if err != nil {
			return Sale{}, nil, err
		}

		// delete all sale items of this sale
		_, err = tx.Exec(ctx, `
			DELETE FROM sale_items
			WHERE sale_id = $1
		`, saleID)
		if err != nil {
			return Sale{}, nil, err
		}

		// cancel sale and zero totals
		sale, err := scanSale(tx.QueryRow(ctx, `
			UPDATE sales
			SET status = 'cancelled',
			    total_cents = 0,
			    items_count = 0
			WHERE id = $1
			RETURNING `+saleCols,
			saleID,
		))
		if err != nil {
			return Sale{}, nil, err
		}

		if err := tx.Commit(ctx); err != nil {
			return Sale{}, nil, err
		}

		return sale, []SaleItem{}, nil
	}

	// 4. Release stock reservation for this item's product
	_, err = tx.Exec(ctx, `
		UPDATE stock_reservations
		SET status = 'released', released_at = now()
		WHERE sale_id = $1 AND product_id = $2 AND status = 'active'
	`, saleID, productID)
	if err != nil {
		return Sale{}, nil, err
	}

	// 5. Delete sale item
	_, err = tx.Exec(ctx, `
		DELETE FROM sale_items
		WHERE id = $1
	`, itemID)
	if err != nil {
		return Sale{}, nil, err
	}

	// 6. Update sale header
	_, err = tx.Exec(ctx, `
		UPDATE sales
		SET total_cents = total_cents - $2,
		    items_count = items_count - 1
		WHERE id = $1
	`, saleID, lineTotalCents)
	if err != nil {
		return Sale{}, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Sale{}, nil, err
	}

	// 7. Refetch updated sale + items
	sale, items, err := r.GetByID(ctx, saleID)
	if err != nil {
		return Sale{}, nil, err
	}
	return sale, items, nil
}
// ── List ─────────────────────────────────────────────────────────────────────

func (r *Repository) List(
	ctx context.Context,
	warehouseID, customerID *int64,
	createdBy *int64,
	status *string,
	dateFrom, dateTo *time.Time,
	limit, offset int,
) ([]SaleListItem, int, error) {

	where := `
		WHERE ($1::bigint IS NULL OR s.warehouse_id = $1)
		  AND ($2::bigint IS NULL OR s.customer_id = $2)
		  AND ($3::timestamptz IS NULL OR s.created_at >= $3)
		  AND ($4::timestamptz IS NULL OR s.created_at <= $4)
		  AND ($5::bigint IS NULL OR s.created_by = $5)
		  AND ($6::text IS NULL OR s.status = $6)
	`

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM sales s `+where,
		warehouseID, customerID, dateFrom, dateTo, createdBy, status,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT s.id, s.warehouse_id, s.customer_id, s.worker_id, s.total_cents, s.cost_cents,
		       s.bonus_used_cents, s.discount_percent, s.discount_cents, s.items_count, s.note,
		       s.created_by, s.created_at, s.status, c.name
		FROM sales s
		LEFT JOIN customers c ON c.id = s.customer_id
		`+where+`
		ORDER BY s.created_at DESC, s.id DESC
		LIMIT $7 OFFSET $8
	`, warehouseID, customerID, dateFrom, dateTo, createdBy, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []SaleListItem
	for rows.Next() {
		var item SaleListItem
		err := rows.Scan(
			&item.ID, &item.WarehouseID, &item.CustomerID, &item.WorkerID, &item.TotalCents,
			&item.CostCents, &item.BonusUsedCents, &item.DiscountPercent, &item.DiscountCents,
			&item.ItemsCount, &item.Note, &item.CreatedBy, &item.CreatedAt,
			&item.Status,
			&item.CustomerName,
		)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

// ── ReturnSale (partial or full return of confirmed sale) ────────────────────

func (r *Repository) ReturnSale(ctx context.Context, saleID int64, req ReturnSaleRequest, userID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	uid := &userID

	// 1. Lock sale and check status
	var status string
	var warehouseID int64
	var saleCustomerID *int64
	var totalCents, bonusUsedCents int64
	err = tx.QueryRow(ctx, `
		SELECT status, warehouse_id, customer_id, total_cents, bonus_used_cents
		FROM sales WHERE id = $1 FOR UPDATE
	`, saleID).Scan(&status, &warehouseID, &saleCustomerID, &totalCents, &bonusUsedCents)
	if err != nil {
		if err == pgx.ErrNoRows {
			return apperr.NotFound("SALE_NOT_FOUND", "sale not found")
		}
		return err
	}
	if status != "confirmed" && status != "partially_returned" {
		return apperr.Conflict("SALE_NOT_RETURNABLE", "only confirmed or partially_returned sales can be returned")
	}

	// 2. If items is empty → full return (all items)
	type saleItemInfo struct {
		id             int64
		productID      int64
		qtyMilli       int64
		costCents      int64
		unitPriceCents int64
		lineTotalCents int64
	}

	returnItems := req.Items
	if len(returnItems) == 0 {
		// Full return: fetch all sale items
		rows, err := tx.Query(ctx, `
			SELECT id, product_id, qty_milli, cost_cents, unit_price_cents, line_total_cents
			FROM sale_items WHERE sale_id = $1 ORDER BY product_id
		`, saleID)
		if err != nil {
			return err
		}
		var allItems []saleItemInfo
		for rows.Next() {
			var si saleItemInfo
			if err := rows.Scan(&si.id, &si.productID, &si.qtyMilli, &si.costCents, &si.unitPriceCents, &si.lineTotalCents); err != nil {
				rows.Close()
				return err
			}
			allItems = append(allItems, si)
		}
		rows.Close()

		// Check already returned quantities
		for _, si := range allItems {
			var alreadyReturned int64
			_ = tx.QueryRow(ctx, `
				SELECT COALESCE(SUM(qty_milli), 0) FROM sale_return_items WHERE sale_item_id = $1
			`, si.id).Scan(&alreadyReturned)
			remaining := si.qtyMilli - alreadyReturned
			if remaining > 0 {
				returnItems = append(returnItems, ReturnItemRequest{
					SaleItemID: si.id,
					QtyMilli:   remaining,
				})
			}
		}
	}

	if len(returnItems) == 0 {
		return apperr.Conflict("NOTHING_TO_RETURN", "all items already returned")
	}

	// 3. Create sale_returns header
	var returnID int64
	var returnTotalCents int64
	err = tx.QueryRow(ctx, `
		INSERT INTO sale_returns (sale_id, reason, total_cents, created_by)
		VALUES ($1, $2, 0, $3)
		RETURNING id
	`, saleID, req.Reason, uid).Scan(&returnID)
	if err != nil {
		return err
	}

	// 4. Process each return item
	for _, ri := range returnItems {
		// Fetch original sale item
		var origProductID, origQtyMilli, origCostCents, origUnitPrice, origLineTotal int64
		err = tx.QueryRow(ctx, `
			SELECT product_id, qty_milli, cost_cents, unit_price_cents, line_total_cents
			FROM sale_items WHERE id = $1 AND sale_id = $2
		`, ri.SaleItemID, saleID).Scan(&origProductID, &origQtyMilli, &origCostCents, &origUnitPrice, &origLineTotal)
		if err != nil {
			if err == pgx.ErrNoRows {
				return apperr.NotFound("SALE_ITEM_NOT_FOUND", "sale item not found in this sale")
			}
			return err
		}

		// Check already returned quantity
		var alreadyReturned int64
		_ = tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(qty_milli), 0) FROM sale_return_items WHERE sale_item_id = $1
		`, ri.SaleItemID).Scan(&alreadyReturned)

		if ri.QtyMilli > origQtyMilli-alreadyReturned {
			return apperr.Validation(
				fmt.Sprintf("return qty exceeds remaining for sale_item %d: remaining=%d, requested=%d",
					ri.SaleItemID, origQtyMilli-alreadyReturned, ri.QtyMilli),
			)
		}

		// Calculate refund (proportional to qty)
		refundCents := lineTotalCents(ri.QtyMilli, origUnitPrice)
		returnTotalCents += refundCents

		// Calculate cost to restore (proportional). Rounded half-up so the
		// inventory cost adjustment matches the refund (which is also rounded
		// in lineTotalCents) — otherwise repeated partial returns drift the
		// warehouse total_cost_cents away from the per-item average.
		costToRestore := (origCostCents*ri.QtyMilli + origQtyMilli/2) / origQtyMilli

		// Insert sale_return_items
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_return_items (return_id, sale_item_id, product_id, qty_milli, refund_cents)
			VALUES ($1, $2, $3, $4, $5)
		`, returnID, ri.SaleItemID, origProductID, ri.QtyMilli, refundCents)
		if err != nil {
			return err
		}

		// Restore stock
		var currentQty, avgCost, wTotalCost int64
		rowExists := true
		err = tx.QueryRow(ctx, `
			SELECT qty_milli, avg_cost_cents, total_cost_cents
			FROM warehouse_items
			WHERE warehouse_id = $1 AND product_id = $2
			FOR UPDATE
		`, warehouseID, origProductID).Scan(&currentQty, &avgCost, &wTotalCost)
		if err != nil {
			if err != pgx.ErrNoRows {
				return err
			}
			rowExists = false
		}

		newQty := currentQty + ri.QtyMilli
		newTotalCost := wTotalCost + costToRestore
		newAvgCost := int64(0)
		if newQty > 0 {
			// Round half-up to keep newQty*newAvgCost/1000 close to newTotalCost.
			newAvgCost = (newTotalCost*1000 + newQty/2) / newQty
		}

		// Ledger entry
		_, err = tx.Exec(ctx, `
			INSERT INTO warehouse_item_details
				(idempotency_key, warehouse_id, product_id, delta_milli, type, created_by)
			VALUES (gen_random_uuid(), $1, $2, $3, 'sale_return', $4)
		`, warehouseID, origProductID, ri.QtyMilli, uid)
		if err != nil {
			return err
		}

		if rowExists {
			_, err = tx.Exec(ctx, `
				UPDATE warehouse_items
				SET qty_milli = $3, total_cost_cents = $4, avg_cost_cents = $5, updated_at = now()
				WHERE warehouse_id = $1 AND product_id = $2
			`, warehouseID, origProductID, newQty, newTotalCost, newAvgCost)
		} else {
			_, err = tx.Exec(ctx, `
				INSERT INTO warehouse_items
					(warehouse_id, product_id, qty_milli, avg_cost_cents, total_cost_cents, updated_at)
				VALUES ($1, $2, $3, $4, $5, now())
			`, warehouseID, origProductID, newQty, newAvgCost, newTotalCost)
		}
		if err != nil {
			return err
		}
	}

	// 5. Update return total
	_, err = tx.Exec(ctx, `UPDATE sale_returns SET total_cents = $2 WHERE id = $1`, returnID, returnTotalCents)
	if err != nil {
		return err
	}

	// 6. Create refund finance transaction
	if returnTotalCents > 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO transactions
				(payment_type_id, reason, status, amount_cents, type, related_table, related_id, created_by, created_at, updated_at)
			VALUES (NULL, $1, 'paid', $2, 'expense', 'sale', $3, $4, now(), now())
		`, fmt.Sprintf("Return for Sale #%d", saleID), returnTotalCents, saleID, uid)
		if err != nil {
			return err
		}
	}

	// 7. Proportional bonus restoration
	if bonusUsedCents > 0 && saleCustomerID != nil && totalCents > 0 {
		// Round half-up so customers don't lose fractional bonus points on
		// repeated partial returns.
		bonusToRestore := (bonusUsedCents*returnTotalCents + totalCents/2) / totalCents
		if bonusToRestore > 0 {
			_, err = tx.Exec(ctx, `
				UPDATE customers SET bonus_points = bonus_points + $2, updated_at = now()
				WHERE id = $1 AND deleted_at IS NULL
			`, *saleCustomerID, bonusToRestore)
			if err != nil {
				return err
			}
		}
	}

	// 8. Determine new sale status
	var totalSoldMilli, totalReturnedMilli int64
	_ = tx.QueryRow(ctx, `SELECT COALESCE(SUM(qty_milli), 0) FROM sale_items WHERE sale_id = $1`, saleID).Scan(&totalSoldMilli)
	_ = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(sri.qty_milli), 0)
		FROM sale_return_items sri
		JOIN sale_returns sr ON sr.id = sri.return_id
		WHERE sr.sale_id = $1
	`, saleID).Scan(&totalReturnedMilli)

	newStatus := "partially_returned"
	if totalReturnedMilli >= totalSoldMilli {
		newStatus = "returned"
	}
	_, err = tx.Exec(ctx, `UPDATE sales SET status = $2 WHERE id = $1`, saleID, newStatus)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
func (r *Repository) DecreaseDraftSaleItemQty(
	ctx context.Context,
	saleID int64,
	itemID int64,
) (Sale, []SaleItem, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Sale{}, nil, err
	}
	defer tx.Rollback(ctx)

	// 1. Проверка sale = draft
	var status string
	err = tx.QueryRow(ctx, `
		SELECT status FROM sales WHERE id = $1 FOR UPDATE
	`, saleID).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Sale{}, nil, apperr.NotFound("SALE_NOT_FOUND", "sale not found")
		}
		return Sale{}, nil, err
	}
	if status != "draft" {
		return Sale{}, nil, apperr.Conflict("SALE_NOT_DRAFT", "sale must be draft")
	}

	// 2. Берём item
	var productID, qtyMilli, unitPrice int64
	var discount int

	err = tx.QueryRow(ctx, `
		SELECT product_id, qty_milli, unit_price_cents, discount_percent
		FROM sale_items
		WHERE id = $1 AND sale_id = $2
		FOR UPDATE
	`, itemID, saleID).Scan(&productID, &qtyMilli, &unitPrice, &discount)

	if err != nil {
		if err == pgx.ErrNoRows {
			return Sale{}, nil, apperr.NotFound("SALE_ITEM_NOT_FOUND", "item not found")
		}
		return Sale{}, nil, err
	}

	// ❗ ГЛАВНАЯ СТРОКА (фикс)
	if qtyMilli <= 1000 {
		return Sale{}, nil, apperr.Conflict(
			"MIN_QTY",
			"cannot decrease below 1, use delete",
		)
	}

	newQty := qtyMilli - 1000

	oldTotal := lineTotalCents(qtyMilli, unitPrice)
	newTotal := lineTotalCents(newQty, unitPrice)

	if discount > 0 {
		oldTotal -= (oldTotal * int64(discount)) / 100
		newTotal -= (newTotal * int64(discount)) / 100
	}

	diff := oldTotal - newTotal

	// 3. update item
	_, err = tx.Exec(ctx, `
		UPDATE sale_items
		SET qty_milli = $3,
		    line_total_cents = $4
		WHERE id = $1 AND sale_id = $2
	`, itemID, saleID, newQty, newTotal)
	if err != nil {
		return Sale{}, nil, err
	}

	// 4. update reservation
	_, err = tx.Exec(ctx, `
		UPDATE stock_reservations
		SET qty_milli = $3
		WHERE sale_id = $1 AND product_id = $2 AND status = 'active'
	`, saleID, productID, newQty)
	if err != nil {
		return Sale{}, nil, err
	}

	// 5. update total
	_, err = tx.Exec(ctx, `
		UPDATE sales
		SET total_cents = total_cents - $2
		WHERE id = $1
	`, saleID, diff)
	if err != nil {
		return Sale{}, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Sale{}, nil, err
	}

	return r.GetByID(ctx, saleID)
}
