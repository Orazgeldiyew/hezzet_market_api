package server

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/redis/go-redis/v9"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	"github.com/Orazgeldiyew/hezzet_market_backend/docs"
	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/auth"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/auditlog"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/category"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/receiptsettings"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/reports"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/customer"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/notification"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/payroll"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/permissions"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/product"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/purchase"
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
	Redis      *redis.Client         // nil when Redis is not configured
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

	// Audit repo created first so it can be shared with auth routes
	auditRepo := auditlog.NewRepository(deps.DB)
	auditMW := auditlog.AuditMiddleware(auditRepo)

	// Auth routes (pass audit middleware so block/unblock/password are logged)
	getTokenVersion := auth.RegisterRoutes(r, deps.DB, deps.Cfg, auditMW)

	// Protected API
	api := r.Group("/api")
	api.Use(middleware.AuthRequired(deps.Cfg, getTokenVersion))
	api.Use(middleware.PaginationMiddleware())
	api.Use(auditMW)

	// ── Permissions module (returns repo for RequireModule middleware) ──
	permRepo := permissions.RegisterRoutes(api, deps.DB, deps.Redis)

	// helper to create a sub-group with module permission check
	mod := func(module string) *gin.RouterGroup {
		g := api.Group("")
		g.Use(middleware.RequireModule(permRepo, module))
		return g
	}

	// ── Products & Categories ──
	productsGroup := mod("products")
	category.RegisterRoutes(productsGroup, deps.DB)
	product.RegisterRoutes(productsGroup, deps.DB, deps.Cfg.UploadsDir, deps.Cfg.PublicBaseURL)

	// ── Stock & Warehouses & Suppliers ──
	stockGroup := mod("stock")
	supplier.RegisterRoutes(stockGroup, deps.DB)
	warehouse.RegisterRoutes(stockGroup, deps.DB)
	stock.RegisterRoutes(stockGroup, deps.DB, deps.NotifSvc)

	// ── Customers ──
	customer.RegisterRoutes(mod("customers"), deps.DB)

	// ── Workers & Payroll ──
	workersGroup := mod("workers")
	workers.RegisterRoutes(workersGroup, deps.DB)
	finRepo := finance.NewRepository(deps.DB)
	workerfinance.RegisterRoutes(workersGroup, deps.DB, finRepo)
	payroll.RegisterRoutes(workersGroup, deps.DB, finRepo)

	// ── Finance ──
	finance.RegisterRoutes(mod("finance"), deps.DB)

	// ── Sales ──
	receiptRepo := receiptsettings.RegisterRoutes(api, deps.DB, deps.Cfg)
	sale.RegisterRoutes(mod("sales"), deps.DB, finRepo, deps.Cfg.PublicBaseURL, receiptRepo)

	// ── Purchases ──
	purchase.RegisterRoutes(mod("purchases"), deps.DB, finRepo)

	// ── Reports & Audit ──
	reportsGroup := mod("reports")
	auditlog.RegisterRoutes(reportsGroup, auditRepo)
	reports.RegisterRoutes(reportsGroup, deps.DB, deps.Cfg.LowStockDefault)

	// ── Notifications (admin-only, no module permission needed) ──
	notification.RegisterRoutes(api, deps.DB, deps.NotifQueue)

	// ── Public receipt route: /receipt/:id?token=JWT ──
	r.GET("/receipt/:id",
		middleware.AuthFromQuery(deps.Cfg, getTokenVersion),
		sale.ReceiptHandler(deps.DB, deps.Cfg.PublicBaseURL, receiptRepo),
	)

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
