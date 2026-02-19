package server

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/Orazgeldiyew/hezzet_market_backend/docs"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/auth"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/category"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/customer"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/product"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/stock"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/supplier"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/warehouse"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/workers"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Deps struct {
	DB  *pgxpool.Pool
	Cfg config.Config
}

func NewRouter(deps Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(ErrorMiddleware())

	r.GET("/health", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok"})
	})

	// Swagger: dev only (recommended)
	if deps.Cfg.Env == "dev" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Auth routes (public login/refresh + admin-only user management inside auth/routes.go)
	auth.RegisterRoutes(r, deps.DB, deps.Cfg)

	// Protected API
	api := r.Group("/api")
	api.Use(middleware.AuthRequired(deps.Cfg))
	api.Use(middleware.PaginationMiddleware())

	category.RegisterRoutes(api, deps.DB)
	product.RegisterRoutes(api, deps.DB)
	supplier.RegisterRoutes(api, deps.DB)
	customer.RegisterRoutes(api, deps.DB)
	workers.RegisterRoutes(api, deps.DB)
	warehouse.RegisterRoutes(api, deps.DB)
	stock.RegisterRoutes(api, deps.DB)
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
