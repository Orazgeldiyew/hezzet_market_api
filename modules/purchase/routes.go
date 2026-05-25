package purchase

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/auditlog"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/receiptsettings"
)

// RegisterRoutes wires purchase endpoints. Per-action gating via permChecker
// matches the rest of the codebase (products, categories, suppliers,
// warehouses) so admins can grant/revoke access from /roles without code
// changes. auditRepo emits PRODUCT_PRICE_CHANGE_VIA_PO when receive
// propagates prices; receiptRepo supplies the buyer block on the A4 invoice.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, finRepo *finance.Repository, auditRepo *auditlog.Repository, receiptRepo *receiptsettings.Repository, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db, auditRepo)
	svc := NewService(repo, finRepo)
	h := NewHandler(svc)

	g := rg.Group("/purchases")
	{
		// Read
		g.GET("",
			middleware.RequirePermission(permChecker, "purchases", "view"),
			h.ListPOs,
		)
		// /debt must be registered before /:id to avoid Gin routing conflict
		g.GET("/debt",
			middleware.RequirePermission(permChecker, "purchases", "view"),
			h.DebtSummary,
		)
		g.GET("/:id",
			middleware.RequirePermission(permChecker, "purchases", "view"),
			h.GetPO,
		)
		// Printable A4 invoice — share the same "view" permission since the
		// reader is just rendering data they're already allowed to read.
		g.GET("/:id/invoice",
			middleware.RequirePermission(permChecker, "purchases", "view"),
			PrintInvoiceHandler(db, receiptRepo),
		)

		// Write
		g.POST("",
			middleware.RequirePermission(permChecker, "purchases", "create"),
			h.CreatePO,
		)
		g.POST("/:id/receive",
			middleware.RequirePermission(permChecker, "purchases", "update"),
			h.ReceivePO,
		)
		g.POST("/:id/payments",
			middleware.RequirePermission(permChecker, "purchases", "update"),
			h.AddPayment,
		)
		g.POST("/:id/cancel",
			middleware.RequirePermission(permChecker, "purchases", "delete"),
			h.CancelPO,
		)
	}
}
