package workerfinance

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, financeRepo *finance.Repository) {
	repo := NewRepository(db)
	svc := NewService(repo, financeRepo)
	h := NewHandler(svc)

	workers := rg.Group("/workers")
	workers.Use(middleware.RequireRoles("manager"))

	// Compensation
	workers.POST("/compensation", h.SetCompensation)
	workers.GET("/:id/compensation", h.GetCompensation)

	// Fines
	workers.POST("/:id/fines", h.CreateFine)
	workers.GET("/:id/fines", middleware.PaginationMiddleware(), h.ListFines)

	// Debts
	workers.POST("/:id/debts", h.CreateDebt)
	workers.GET("/:id/debts", middleware.PaginationMiddleware(), h.ListDebts)
}
