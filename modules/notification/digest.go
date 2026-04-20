package notification

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// ReorderSuggestionBrief is the minimal data the digest needs from reports.
// Kept package-local so notification does not depend on reports directly.
type ReorderSuggestionBrief struct {
	ProductName       string
	WarehouseName     string
	CurrentQtyMilli   int64
	AvgDailyMilli     float64
	SuggestedOrderMilli int64
	DaysUntilStockout *float64
}

// ReorderFetcher is implemented by whoever can produce today's reorder list
// (in practice reports.Repository). Keeping it as an interface avoids an
// import cycle and lets tests inject fake data.
type ReorderFetcher interface {
	Fetch(ctx context.Context) ([]ReorderSuggestionBrief, error)
}

// StartReorderDigestCron launches a goroutine that once per day fetches
// reorder suggestions and sends a single summary SMS to all admin phones.
// The trigger hour + on/off flag are read from notification_settings on
// every tick so admins can change them live from the UI.
//
// Safe to call when s == nil — then it's a no-op. defaultHour is used only
// as a fallback when settingsRepo is nil or errors.
func (s *Service) StartReorderDigestCron(
	ctx context.Context,
	defaultHour int,
	fetcher ReorderFetcher,
	settingsRepo *SettingsRepository,
) {
	if s == nil || fetcher == nil {
		return
	}
	if defaultHour < 0 || defaultHour > 23 {
		defaultHour = 9
	}

	go func() {
		log.Printf("[reorder-digest] started, default hour=%d", defaultHour)
		// Check every 30 minutes; dedup key guarantees at-most-once-per-day.
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		// Run once immediately if we're already past the trigger hour — useful
		// for testing and ensures we don't miss today's window on startup.
		s.maybeSendDigest(ctx, defaultHour, fetcher, settingsRepo)

		for {
			select {
			case <-ctx.Done():
				log.Println("[reorder-digest] shutting down")
				return
			case <-ticker.C:
				s.maybeSendDigest(ctx, defaultHour, fetcher, settingsRepo)
			}
		}
	}()
}

// maybeSendDigest sends the digest if we've crossed the target hour today.
// The dedup key (scoped by date) prevents multiple sends per day.
func (s *Service) maybeSendDigest(
	ctx context.Context,
	defaultHour int,
	fetcher ReorderFetcher,
	settingsRepo *SettingsRepository,
) {
	hour := defaultHour
	enabled := true
	if settingsRepo != nil {
		if cur, err := settingsRepo.Get(ctx); err == nil {
			hour = cur.ReorderDigestHour
			enabled = cur.ReorderDigestEnabled
		}
	}
	if !enabled {
		return
	}

	now := time.Now()
	if now.Hour() < hour {
		return
	}

	items, err := fetcher.Fetch(ctx)
	if err != nil {
		log.Printf("[reorder-digest] fetch error: %v", err)
		return
	}
	if len(items) == 0 {
		return
	}

	msg := buildDigestMessage(items)
	dedupKey := fmt.Sprintf("reorder_digest:%s", now.Format("2006-01-02"))

	phones := s.adminPhones
	if s.phoneRepo != nil {
		if dbPhones, err := s.phoneRepo.GetActivePhones(ctx); err == nil && len(dbPhones) > 0 {
			phones = dbPhones
		}
	}

	for _, phone := range phones {
		job := SMSJob{
			JobID:       generateUUID(),
			Type:        "admin_reorder_digest",
			ToPhone:     phone,
			Message:     msg,
			MaxAttempts: 3,
			DedupKey:    dedupKey + ":" + phone,
		}
		s.enqueueAndLog(ctx, job)
	}
}

// buildDigestMessage formats the daily digest in Turkmen. Caps at 5 lines so
// a single SMS stays readable; appends "+N more" when truncated.
func buildDigestMessage(items []ReorderSuggestionBrief) string {
	const maxLines = 5
	var b strings.Builder
	b.WriteString("Satyn alynmaga maslahat berilýän harytlar:\n")

	shown := len(items)
	if shown > maxLines {
		shown = maxLines
	}
	for i := 0; i < shown; i++ {
		it := items[i]
		suggested := float64(it.SuggestedOrderMilli) / float64(qtyMilliScale)
		line := fmt.Sprintf("%d. %s (%s) — %.1f", i+1, it.ProductName, it.WarehouseName, suggested)
		if it.DaysUntilStockout != nil {
			line += fmt.Sprintf(", %.0f günden gutarjak", *it.DaysUntilStockout)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	if len(items) > maxLines {
		fmt.Fprintf(&b, "... ýene %d haryt", len(items)-maxLines)
	}
	return b.String()
}
