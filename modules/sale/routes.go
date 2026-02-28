package sale

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, finRepo *finance.Repository, publicBaseURL string) {
	repo := NewRepository(db, publicBaseURL)
	svc := NewService(repo, finRepo)
	h := NewHandler(svc)

	sales := rg.Group("/sales")

	sales.POST("",
		middleware.RequireRoles("cashier", "operator", "manager"),
		h.CreateSale,
	)

	sales.GET("",
		middleware.RequireRoles("cashier", "operator", "manager"),
		h.ListSales,
	)

	sales.GET("/:id",
		middleware.RequireRoles("cashier", "operator", "manager"),
		h.GetSale,
	)

	sales.POST("/:id/confirm",
		middleware.RequireRoles("cashier", "operator", "manager"),
		h.ConfirmSale,
	)

	sales.POST("/:id/cancel",
		middleware.RequireRoles("cashier", "operator", "manager"),
		h.CancelSale,
	)
}
