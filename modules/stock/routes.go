package stock

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/notification"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, notifSvc *notification.Service) {
	repo := NewRepository(db)
	svc := NewService(repo, notifSvc)
	h := NewHandler(svc)

	s := rg.Group("/stock")

	s.POST("/in",
		middleware.RequireRoles("operator", "manager"),
		h.StockIn,
	)
	s.POST("/in/bulk",
		middleware.RequireRoles("operator", "manager"),
		h.BulkStockIn,
	)
	s.POST("/out",
		middleware.RequireRoles("operator", "manager"),
		h.StockOut,
	)
	s.POST("/transfer",
		middleware.RequireRoles("operator", "manager"),
		h.Transfer,
	)

	s.POST("/move",
		middleware.RequireRoles("operator", "manager"),
		h.Move,
	)

	s.GET("/balance",
		middleware.RequireRoles("cashier", "operator", "manager"),
		h.GetItems,
	)

	s.GET("/details",
		middleware.RequireRoles("cashier", "operator", "manager"),
		middleware.PaginationMiddleware(),
		h.GetDetails,
	)

	s.GET("/negative",
		middleware.RequireRoles("operator", "manager"),
		h.NegativeStock,
	)

}
