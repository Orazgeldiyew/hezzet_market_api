package notification

import (
	"context"
	"fmt"
	"log"
	"time"
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
	adminPhones     []string // fallback when phoneRepo returns nothing
	smsFrom         string
	providerName    string
	lowStockDefault int64 // threshold in regular units (not milli)
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
) *Service {
	return &Service{
		queue:           queue,
		logRepo:         logRepo,
		phoneRepo:       phoneRepo,
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
// below the threshold. Dedup key prevents spam within the TTL window.
// Safe to call on every stock-decrease operation — the dedup gate handles frequency.
func (s *Service) NotifyAdminLowStock(ctx context.Context, productID, warehouseID, currentQtyMilli int64) {
	if s == nil {
		return
	}

	thresholdMilli := s.lowStockDefault * qtyMilliScale
	if currentQtyMilli > thresholdMilli {
		return
	}

	currentQty := float64(currentQtyMilli) / float64(qtyMilliScale)
	msg := fmt.Sprintf(
		"LOW STOCK: product=%d warehouse=%d qty=%.1f (threshold=%d)",
		productID, warehouseID, currentQty, s.lowStockDefault,
	)

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
