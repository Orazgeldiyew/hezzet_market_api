package payroll

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/workerfinance"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, finRepo *finance.Repository) {
	repo := NewRepository(db)
	wfRepo := workerfinance.NewRepository(db)
	svc := NewService(repo, wfRepo, finRepo)
	h := NewHandler(svc)

	pr := rg.Group("/payroll")
	pr.Use(middleware.RequireRoles("manager"))

	pr.GET("", middleware.PaginationMiddleware(), h.List)
	pr.POST("/calculate", h.Calculate)
	pr.POST("/:id/pay", h.Pay)
}
