package server

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	"github.com/Orazgeldiyew/hezzet_market_backend/docs"
	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/auth"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/category"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/customer"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/notification"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/payroll"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/product"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/sale"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/stock"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/supplier"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/warehouse"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/workerfinance"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/workers"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Deps struct {
	DB         *pgxpool.Pool
	Cfg        config.Config
	NotifSvc   *notification.Service // nil-safe — notifications disabled when nil
	NotifQueue *notification.Queue   // nil when Redis is not configured
}

func NewRouter(deps Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// ── Request ID ── (must be before ErrorMiddleware so the ID is in context)
	r.Use(middleware.RequestID())

	// ── CORS ──
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		MaxAge:          12 * time.Hour,
	}))

	r.Use(ErrorMiddleware())

	r.GET("/health", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok"})
	})

	// ── Static file serving for uploaded photos (no auth required) ──
	uploadsDir := deps.Cfg.UploadsDir
	if uploadsDir == "" {
		uploadsDir = "./uploads"
	}
	r.Static("/uploads", uploadsDir)

	// ---- Swagger route (host/scheme from config or dynamic per request) ----
	swaggerHandler := func(c *gin.Context) {
		if deps.Cfg.SwaggerHost != "" {
			docs.SwaggerInfo.Host = deps.Cfg.SwaggerHost
		} else {
			docs.SwaggerInfo.Host = c.Request.Host
		}

		if deps.Cfg.SwaggerSchemes != "" {
			schemes := strings.Split(deps.Cfg.SwaggerSchemes, ",")
			for i := range schemes {
				schemes[i] = strings.TrimSpace(schemes[i])
			}
			docs.SwaggerInfo.Schemes = schemes
		} else {
			scheme := "http"
			if xf := c.GetHeader("X-Forwarded-Proto"); xf != "" {
				scheme = strings.ToLower(strings.TrimSpace(xf))
			} else if c.Request.TLS != nil {
				scheme = "https"
			}
			docs.SwaggerInfo.Schemes = []string{scheme}
		}

		ginSwagger.WrapHandler(swaggerFiles.Handler)(c)
	}

	if deps.Cfg.Env == "dev" {
		// dev: open
		r.GET("/swagger/*any", swaggerHandler)
	} else {
		// prod/stage: guarded (IP allowlist + basic auth)
		allow := splitCSV(deps.Cfg.SwaggerAllowIPs)
		user := deps.Cfg.SwaggerUser
		pass := deps.Cfg.SwaggerPass

		r.GET("/swagger/*any",
			middleware.SwaggerGuard(allow, user, pass),
			swaggerHandler,
		)
	}

	// Auth routes
	getTokenVersion := auth.RegisterRoutes(r, deps.DB, deps.Cfg)

	// Protected API
	api := r.Group("/api")
	api.Use(middleware.AuthRequired(deps.Cfg, getTokenVersion))
	api.Use(middleware.PaginationMiddleware())

	category.RegisterRoutes(api, deps.DB)
	product.RegisterRoutes(api, deps.DB, deps.Cfg.UploadsDir, deps.Cfg.PublicBaseURL)
	supplier.RegisterRoutes(api, deps.DB)
	customer.RegisterRoutes(api, deps.DB)
	workers.RegisterRoutes(api, deps.DB)
	warehouse.RegisterRoutes(api, deps.DB)
	stock.RegisterRoutes(api, deps.DB, deps.NotifSvc)
	notification.RegisterRoutes(api, deps.DB, deps.NotifQueue)
	finRepo := finance.NewRepository(deps.DB)
	finance.RegisterRoutes(api, deps.DB)
	workerfinance.RegisterRoutes(api, deps.DB, finRepo)
	payroll.RegisterRoutes(api, deps.DB, finRepo)
	sale.RegisterRoutes(api, deps.DB, finRepo, deps.Cfg.PublicBaseURL)

	r.GET("/debug/routes", func(c *gin.Context) {
		type R struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		}
		rs := r.Routes()
		out := make([]R, 0, len(rs))
		for _, x := range rs {
			out = append(out, R{Method: x.Method, Path: x.Path})
		}
		c.JSON(200, out)
	})

	return r
}

func splitCSV(s string) []string {
	out := []string{}
	for _, x := range strings.Split(s, ",") {
		x = strings.TrimSpace(x)
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}
