package purchase

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/auditlog"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
)

// RegisterRoutes wires purchase endpoints. auditRepo is used by ReceivePO to
// emit per-product PRICE_CHANGE_VIA_PO audit entries when receive propagates
// new prices to the products table.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, finRepo *finance.Repository, auditRepo *auditlog.Repository) {
	repo := NewRepository(db, auditRepo)
	svc := NewService(repo, finRepo)
	h := NewHandler(svc)

	g := rg.Group("/purchases")

	g.POST("",
		middleware.RequireRoles("operator", "manager"),
		h.CreatePO,
	)

	g.GET("",
		middleware.RequireRoles("operator", "manager"),
		h.ListPOs,
	)

	// /debt must be registered before /:id to avoid Gin routing conflict
	g.GET("/debt",
		middleware.RequireRoles("operator", "manager"),
		h.DebtSummary,
	)

	g.GET("/:id",
		middleware.RequireRoles("operator", "manager"),
		h.GetPO,
	)

	g.POST("/:id/receive",
		middleware.RequireRoles("operator", "manager"),
		h.ReceivePO,
	)

	g.POST("/:id/cancel",
		middleware.RequireRoles("operator", "manager"),
		h.CancelPO,
	)

	g.POST("/:id/payments",
		middleware.RequireRoles("operator", "manager"),
		h.AddPayment,
	)
}
