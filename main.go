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
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/database"
	"github.com/Orazgeldiyew/hezzet_market_backend/server"
)

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

	// SMS audit log repository (always created if db is available).
	notifLogRepo := notification.NewLogRepository(db)

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
		)

		notification.StartWorkers(ctx, cfg.SMSWorkers, notifQueue, provider, cfg.SMSFrom, notifLogRepo)
		log.Printf("SMS workers started: count=%d provider=%s", cfg.SMSWorkers, provider.Name())
	}

	// ── Router + HTTP server ────────────────────────────────────────────────
	r := server.NewRouter(server.Deps{
		DB:         db,
		Cfg:        cfg,
		NotifSvc:   notifSvc,
		NotifQueue: notifQueue,
		Redis:      rdb,
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
