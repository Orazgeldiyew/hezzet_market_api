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
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/supplier"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/workers"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Deps struct {
	DB  *pgxpool.Pool
	Cfg config.Config
}

func NewRouter(deps Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(ErrorMiddleware())

	r.GET("/health", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ── Auth routes (public login + admin-only user management) ──
	auth.RegisterRoutes(r, deps.DB, deps.Cfg)

	// ── Protected group (all routes below require a valid JWT) ──
	protected := r.Group("")
	protected.Use(middleware.AuthRequired(deps.Cfg))

	category.RegisterRoutes(protected, deps.DB)
	product.RegisterRoutes(protected, deps.DB)
	supplier.RegisterRoutes(protected, deps.DB)
	customer.RegisterRoutes(protected, deps.DB)
	workers.RegisterRoutes(protected, deps.DB)

	return r
}
