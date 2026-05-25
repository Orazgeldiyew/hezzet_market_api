package customer

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes wires customer routes behind per-action permissions so admins
// can grant/revoke "kassir создает клиента", "manager меняет бонусы" etc. via
// the /roles UI without redeploying. Default seed matches the previous role-
// based gates; admins still bypass via middleware.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	customers := rg.Group("/customers")

	// Read (any staff with customers:view)
	customers.GET("",
		middleware.RequirePermission(permChecker, "customers", "view"),
		middleware.PaginationMiddleware(),
		h.List,
	)
	customers.GET("/by-card/:code",
		middleware.RequirePermission(permChecker, "customers", "view"),
		h.GetByCard,
	)
	customers.GET("/:id",
		middleware.RequirePermission(permChecker, "customers", "view"),
		h.Get,
	)

	// Create — same as view-creator pattern in other modules.
	customers.POST("",
		middleware.RequirePermission(permChecker, "customers", "create"),
		h.Create,
	)

	// AddSpent is a routine bonus-points operation: treat it as an update.
	customers.POST("/:id/spent",
		middleware.RequirePermission(permChecker, "customers", "update"),
		h.AddSpent,
	)

	// Contact update (phone/address etc.) — same update permission.
	customers.PATCH("/:id/contact",
		middleware.RequirePermission(permChecker, "customers", "update"),
		h.UpdateContact,
	)

	// Admin-level update changes financial fields (bonus_cents, etc.) — guard
	// behind history action so it can be gated separately from routine updates.
	customers.PATCH("/:id/admin",
		middleware.RequirePermission(permChecker, "customers", "history"),
		h.UpdateAdmin,
	)

	// Delete stays delete (admins bypass).
	customers.DELETE("/:id",
		middleware.RequirePermission(permChecker, "customers", "delete"),
		h.Delete,
	)
}
