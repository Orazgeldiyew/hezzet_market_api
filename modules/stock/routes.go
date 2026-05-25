package stock

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/notification"
)

// RegisterRoutes wires stock endpoints behind the `stock` permission module so
// admins can grant operators "in only" or "view + transfer" combos via /roles.
// Granularity: stock:view = read balance/details, stock:create = stock-in,
// stock:update = stock-out + transfer + move (warehouse movements that don't
// add new inventory), stock:history = negative-stock view (audit-like).
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, notifSvc *notification.Service, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db)
	svc := NewService(repo, notifSvc)
	h := NewHandler(svc)

	s := rg.Group("/stock")

	// Inbound (stock-in) — gets new inventory into the warehouse.
	s.POST("/in",
		middleware.RequirePermission(permChecker, "stock", "create"),
		h.StockIn,
	)
	s.POST("/in/bulk",
		middleware.RequirePermission(permChecker, "stock", "create"),
		h.BulkStockIn,
	)

	// Adjustments — write-off, between-warehouse transfer, in-warehouse move.
	s.POST("/out",
		middleware.RequirePermission(permChecker, "stock", "update"),
		h.StockOut,
	)
	s.POST("/transfer",
		middleware.RequirePermission(permChecker, "stock", "update"),
		h.Transfer,
	)
	s.POST("/move",
		middleware.RequirePermission(permChecker, "stock", "update"),
		h.Move,
	)

	// Reads.
	s.GET("/balance",
		middleware.RequirePermission(permChecker, "stock", "view"),
		h.GetItems,
	)
	s.GET("/details",
		middleware.RequirePermission(permChecker, "stock", "view"),
		middleware.PaginationMiddleware(),
		h.GetDetails,
	)

	// Negative-stock surfaces accounting issues — gate behind stock:history so
	// it can be reserved for managers without giving them the write actions.
	s.GET("/negative",
		middleware.RequirePermission(permChecker, "stock", "history"),
		h.NegativeStock,
	)
}
