package notification

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const qtyMilliScale int64 = 1000

// PhoneQuerier is implemented by PhoneRepository.
// Service uses it to look up active admin phones at runtime.
type PhoneQuerier interface {
	GetActivePhones(ctx context.Context) ([]string, error)
}

// Service provides high-level, fire-and-forget notification helpers.
// All methods are safe to call with a nil receiver (no-op) so callers
// don't need nil-guards.
type Service struct {
	queue           *Queue
	logRepo         *LogRepository
	phoneRepo       PhoneQuerier
	db              *pgxpool.Pool // used for dynamic reorder-point calculation
	adminPhones     []string      // fallback when phoneRepo returns nothing
	smsFrom         string
	providerName    string
	lowStockDefault int64 // fallback threshold in regular units (not milli)
	dedupTTL        time.Duration
}

func NewService(
	queue *Queue,
	logRepo *LogRepository,
	adminPhones []string,
	smsFrom string,
	providerName string,
	lowStockDefault int64,
	dedupTTL time.Duration,
	phoneRepo PhoneQuerier,
	db *pgxpool.Pool,
) *Service {
	return &Service{
		queue:           queue,
		logRepo:         logRepo,
		phoneRepo:       phoneRepo,
		db:              db,
		adminPhones:     adminPhones,
		smsFrom:         smsFrom,
		providerName:    providerName,
		lowStockDefault: lowStockDefault,
		dedupTTL:        dedupTTL,
	}
}

// ---------------------------------------------------------------------------
// Customer notifications
// ---------------------------------------------------------------------------

// NotifyCustomerOrderCreated enqueues an SMS about order creation.
func (s *Service) NotifyCustomerOrderCreated(ctx context.Context, phone string, orderID, totalCents int64) {
	if s == nil {
		return
	}
	msg := fmt.Sprintf(
		"Your order #%d has been created. Total: %.2f. Thank you!",
		orderID, float64(totalCents)/100,
	)

	job := SMSJob{
		JobID:       generateUUID(),
		Type:        "customer_order_created",
		ToPhone:     phone,
		Message:     msg,
		MaxAttempts: 3,
		DedupKey:    fmt.Sprintf("order_created:%d", orderID),
	}

	s.enqueueAndLog(ctx, job)
}

// NotifyCustomerPaymentReceived enqueues an SMS about payment confirmation.
func (s *Service) NotifyCustomerPaymentReceived(ctx context.Context, phone string, orderID, amountCents int64) {
	if s == nil {
		return
	}
	msg := fmt.Sprintf(
		"Payment of %.2f received for order #%d. Thank you!",
		float64(amountCents)/100, orderID,
	)

	job := SMSJob{
		JobID:       generateUUID(),
		Type:        "customer_payment_received",
		ToPhone:     phone,
		Message:     msg,
		MaxAttempts: 3,
		DedupKey:    fmt.Sprintf("payment_received:%d", orderID),
	}

	s.enqueueAndLog(ctx, job)
}

// ---------------------------------------------------------------------------
// Admin / operator notifications
// ---------------------------------------------------------------------------

// NotifyAdminLowStock enqueues SMS to all admin phones when stock crosses
// below the product's dynamic reorder point (avg_daily × lead_time + safety).
// Falls back to the static LOW_STOCK_DEFAULT for products without sales history.
// Dedup key prevents spam within the TTL window.
func (s *Service) NotifyAdminLowStock(ctx context.Context, productID, warehouseID, currentQtyMilli int64) {
	if s == nil {
		return
	}

	info, err := s.loadReorderInfo(ctx, productID, warehouseID)
	if err != nil {
		log.Printf("[notification] reorder info lookup failed product=%d wh=%d: %v", productID, warehouseID, err)
		// Fall back to static threshold so we still alert on obvious shortages.
		info = reorderInfo{
			thresholdMilli: s.lowStockDefault * qtyMilliScale,
		}
	}

	if currentQtyMilli > info.thresholdMilli {
		return
	}

	msg := buildLowStockMessage(info, currentQtyMilli)

	// Prefer DB phones; fall back to config phones if DB returns nothing.
	phones := s.adminPhones
	if s.phoneRepo != nil {
		if dbPhones, err := s.phoneRepo.GetActivePhones(ctx); err == nil && len(dbPhones) > 0 {
			phones = dbPhones
		}
	}

	for _, phone := range phones {
		job := SMSJob{
			JobID:       generateUUID(),
			Type:        "admin_low_stock",
			ToPhone:     phone,
			Message:     msg,
			MaxAttempts: 5,
			DedupKey:    fmt.Sprintf("low_stock:%d:%d", productID, warehouseID),
		}

		s.enqueueAndLog(ctx, job)
	}
}

// reorderInfo bundles the data needed to build a low-stock SMS.
type reorderInfo struct {
	productName     string
	warehouseName   string
	avgDailyMilli   float64 // 0 when no sales history
	leadTimeDays    int
	safetyStockMilli int64
	thresholdMilli  int64 // reorder point, or static fallback
}

// loadReorderInfo fetches product/warehouse names plus the inputs needed to
// compute the dynamic reorder point. Returns a best-effort struct even when
// some optional data (e.g. sales history) is missing.
func (s *Service) loadReorderInfo(ctx context.Context, productID, warehouseID int64) (reorderInfo, error) {
	var info reorderInfo
	if s.db == nil {
		info.thresholdMilli = s.lowStockDefault * qtyMilliScale
		return info, nil
	}

	err := s.db.QueryRow(ctx, `
		SELECT p.name,
		       w.name,
		       COALESCE((
		           SELECT SUM(si.qty_milli)::float / 30.0
		           FROM sale_items si
		           JOIN sales s ON s.id = si.sale_id
		           WHERE si.product_id = $1
		             AND s.status = 'confirmed'
		             AND s.created_at >= now() - interval '30 days'
		       ), 0) AS avg_daily_milli,
		       p.lead_time_days,
		       p.safety_stock_milli
		FROM products p
		CROSS JOIN warehouses w
		WHERE p.id = $1 AND w.id = $2`,
		productID, warehouseID,
	).Scan(&info.productName, &info.warehouseName, &info.avgDailyMilli, &info.leadTimeDays, &info.safetyStockMilli)
	if err != nil {
		return info, err
	}

	if info.avgDailyMilli > 0 {
		info.thresholdMilli = int64(info.avgDailyMilli*float64(info.leadTimeDays)) + info.safetyStockMilli
	} else {
		info.thresholdMilli = s.lowStockDefault * qtyMilliScale
	}
	return info, nil
}

// buildLowStockMessage returns a Turkmen-language low-stock alert.
func buildLowStockMessage(info reorderInfo, currentQtyMilli int64) string {
	currentQty := float64(currentQtyMilli) / float64(qtyMilliScale)
	name := info.productName
	if name == "" {
		name = "haryt"
	}
	if info.avgDailyMilli > 0 {
		daysLeft := float64(currentQtyMilli) / info.avgDailyMilli
		suggested := int64(info.avgDailyMilli*float64(info.leadTimeDays)*2) + info.safetyStockMilli - currentQtyMilli
		if suggested < 0 {
			suggested = 0
		}
		return fmt.Sprintf(
			"AZ GALDY: %s — %.1f galdy, %.1f günden gutarjak. Satyn almak maslahaty: %.1f.",
			name, currentQty, daysLeft, float64(suggested)/float64(qtyMilliScale),
		)
	}
	return fmt.Sprintf("AZ GALDY: %s — %.1f galdy.", name, currentQty)
}

// ---------------------------------------------------------------------------
// internal
// ---------------------------------------------------------------------------

// enqueueAndLog enqueues a job and logs it to the DB.
// Deduped jobs are not logged. Failed enqueues are logged with status=failed.
func (s *Service) enqueueAndLog(ctx context.Context, job SMSJob) {
	enqueued, err := s.queue.Enqueue(ctx, job, s.dedupTTL)
	if err != nil {
		log.Printf("[notification] enqueue %s failed: %v", job.Type, err)
		// Insert a failed row (no row exists yet, so we cannot UPDATE).
		s.logRepo.InsertFailed(ctx, job, s.providerName, s.smsFrom, err.Error())
		return
	}
	if !enqueued {
		log.Printf("[notification] %s deduped job=%s", job.Type, job.JobID)
		return // deduped — do not create a log row
	}

	// Successfully enqueued — persist the audit row.
	s.logRepo.UpsertQueued(ctx, job, s.providerName, s.smsFrom)
}
