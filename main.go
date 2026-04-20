package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/notification"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/reports"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/database"
	"github.com/Orazgeldiyew/hezzet_market_backend/server"
)

// reorderDigestFetcher adapts reports.Repository to notification.ReorderFetcher
// so notification has no direct dependency on reports.
type reorderDigestFetcher struct {
	repo *reports.Repository
}

func (f *reorderDigestFetcher) Fetch(ctx context.Context) ([]notification.ReorderSuggestionBrief, error) {
	rows, err := f.repo.ReorderSuggestions(ctx, nil)
	if err != nil {
		return nil, err
	}
	out := make([]notification.ReorderSuggestionBrief, len(rows))
	for i, r := range rows {
		out[i] = notification.ReorderSuggestionBrief{
			ProductName:         r.Name,
			WarehouseName:       r.WarehouseName,
			CurrentQtyMilli:     r.CurrentQtyMilli,
			AvgDailyMilli:       r.AvgDailyMilli,
			SuggestedOrderMilli: r.SuggestedOrderMilli,
			DaysUntilStockout:   r.DaysUntilStockout,
		}
	}
	return out, nil
}

// @title           Hezzet Market API
// @version         1.0
// @description     Market backend (products, stock, income, sales)
// @BasePath        /
// @schemes         http

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter the token with the `Bearer ` prefix, e.g. "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
func main() {
	cfg := config.Load()

	// Gin mode
	if cfg.Env == "dev" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// DB
	db, err := database.NewPool(context.Background(), cfg.DBDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// ── Redis + notification ────────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var rdb *redis.Client
	var notifSvc *notification.Service

	if cfg.RedisURL != "" {
		opt, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			log.Printf("WARNING: invalid REDIS_URL: %v — notifications disabled", err)
		} else {
			rdb = redis.NewClient(opt)

			pingCtx, pingCancel := context.WithTimeout(ctx, 3*time.Second)
			err = rdb.Ping(pingCtx).Err()
			pingCancel()

			if err != nil {
				log.Printf("WARNING: Redis ping failed: %v — notifications disabled", err)
				rdb.Close()
				rdb = nil
			}
		}
	}

	// SMS repositories (always created — DB is always available).
	notifLogRepo := notification.NewLogRepository(db)
	notifPhoneRepo := notification.NewPhoneRepository(db)

	var notifQueue *notification.Queue

	if rdb != nil {
		defer rdb.Close()

		notifQueue = notification.NewQueue(rdb, cfg.SMSRateLimitPerHour)

		// Choose SMS provider
		var provider notification.SMSProvider
		switch cfg.SMSProvider {
		case "twilio":
			provider = notification.NewTwilioProvider(
				os.Getenv("TWILIO_ACCOUNT_SID"),
				os.Getenv("TWILIO_AUTH_TOKEN"),
			)
		default:
			provider = &notification.LogProvider{}
		}

		notifSvc = notification.NewService(
			notifQueue, notifLogRepo, cfg.AdminPhones, cfg.SMSFrom,
			provider.Name(), cfg.LowStockDefault, cfg.LowStockDedupTTL,
			notifPhoneRepo, db,
		)

		notification.StartWorkers(ctx, cfg.SMSWorkers, notifQueue, provider, cfg.SMSFrom, notifLogRepo)
		log.Printf("SMS workers started: count=%d provider=%s", cfg.SMSWorkers, provider.Name())

		// Daily reorder digest — one SMS per day at the configured hour listing
		// products that should be ordered. Hour is read from notification_settings
		// on each tick so admins can change it live.
		reportsRepo := reports.NewRepository(db, cfg.LowStockDefault)
		settingsRepo := notification.NewSettingsRepository(db)
		notifSvc.StartReorderDigestCron(ctx, cfg.ReorderDigestHour, &reorderDigestFetcher{repo: reportsRepo}, settingsRepo)
		log.Printf("Reorder digest scheduled: default hour=%d (overridable via /api/notifications/settings)", cfg.ReorderDigestHour)
	}

	// ── Draft cleanup cron (cancel drafts older than 2 hours) ──────────────
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		log.Println("[draft-cleanup] started (every 1h, max age 2h)")
		for {
			select {
			case <-ctx.Done():
				log.Println("[draft-cleanup] shutting down")
				return
			case <-ticker.C:
				cutoff := time.Now().Add(-2 * time.Hour)
				tag, err := db.Exec(ctx, `
					UPDATE stock_reservations SET status = 'released', released_at = now()
					WHERE status = 'active'
					AND sale_id IN (SELECT id FROM sales WHERE status = 'draft' AND created_at < $1)
				`, cutoff)
				if err != nil {
					log.Printf("[draft-cleanup] release reservations error: %v", err)
					continue
				}
				tag2, err := db.Exec(ctx, `
					UPDATE sales SET status = 'cancelled' WHERE status = 'draft' AND created_at < $1
				`, cutoff)
				if err != nil {
					log.Printf("[draft-cleanup] cancel drafts error: %v", err)
					continue
				}
				if tag.RowsAffected() > 0 || tag2.RowsAffected() > 0 {
					log.Printf("[draft-cleanup] released %d reservations, cancelled %d drafts",
						tag.RowsAffected(), tag2.RowsAffected())
				}
			}
		}
	}()

	// ── Router + HTTP server ────────────────────────────────────────────────
	r := server.NewRouter(server.Deps{
		DB:             db,
		Cfg:            cfg,
		NotifSvc:       notifSvc,
		NotifQueue:     notifQueue,
		Redis:          rdb,
		NotifPhoneRepo: notifPhoneRepo,
	})

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: r,
	}

	// Graceful shutdown on SIGINT / SIGTERM
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("received shutdown signal")
		cancel() // stop notification workers

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
	}()

	log.Println("listening on", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	log.Println("server stopped")
}
