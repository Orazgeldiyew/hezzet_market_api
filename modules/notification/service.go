package notification

import (
	"context"
	"fmt"
	"log"
	"time"
)

const qtyMilliScale int64 = 1000

// Service provides high-level, fire-and-forget notification helpers.
// All methods are safe to call with a nil receiver (no-op) so callers
// don't need nil-guards.
type Service struct {
	queue           *Queue
	adminPhones     []string
	smsFrom         string
	lowStockDefault int64 // threshold in regular units (not milli)
	dedupTTL        time.Duration
}

func NewService(
	queue *Queue,
	adminPhones []string,
	smsFrom string,
	lowStockDefault int64,
	dedupTTL time.Duration,
) *Service {
	return &Service{
		queue:           queue,
		adminPhones:     adminPhones,
		smsFrom:         smsFrom,
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
		Type:        "customer_order_created",
		ToPhone:     phone,
		Message:     msg,
		MaxAttempts: 3,
		DedupKey:    fmt.Sprintf("order_created:%d", orderID),
	}

	enqueued, err := s.queue.Enqueue(ctx, job, s.dedupTTL)
	if err != nil {
		log.Printf("[notification] enqueue customer_order_created failed: %v", err)
		return
	}
	if !enqueued {
		log.Printf("[notification] customer_order_created deduped order=%d", orderID)
	}
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
		Type:        "customer_payment_received",
		ToPhone:     phone,
		Message:     msg,
		MaxAttempts: 3,
		DedupKey:    fmt.Sprintf("payment_received:%d", orderID),
	}

	enqueued, err := s.queue.Enqueue(ctx, job, s.dedupTTL)
	if err != nil {
		log.Printf("[notification] enqueue customer_payment_received failed: %v", err)
		return
	}
	if !enqueued {
		log.Printf("[notification] customer_payment_received deduped order=%d", orderID)
	}
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

	for _, phone := range s.adminPhones {
		job := SMSJob{
			Type:        "admin_low_stock",
			ToPhone:     phone,
			Message:     msg,
			MaxAttempts: 5,
			DedupKey:    fmt.Sprintf("low_stock:%d:%d", productID, warehouseID),
		}

		enqueued, err := s.queue.Enqueue(ctx, job, s.dedupTTL)
		if err != nil {
			log.Printf("[notification] enqueue admin_low_stock failed phone=%s: %v", phone, err)
			continue
		}
		if !enqueued {
			log.Printf("[notification] admin_low_stock deduped product=%d warehouse=%d", productID, warehouseID)
		}
	}
}
