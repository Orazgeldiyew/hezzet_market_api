package server

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/Orazgeldiyew/hezzet_market_backend/docs"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/category"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/client"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/product"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/supplier"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

type Deps struct {
	DB *pgxpool.Pool
}

func NewRouter(deps Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(ErrorMiddleware())

	r.GET("/health", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	category.RegisterRoutes(r, deps.DB)
	client.RegisterRoutes(r, deps.DB)
	product.RegisterRoutes(r, deps.DB)
	supplier.RegisterRoutes(r, deps.DB)

	return r
}
