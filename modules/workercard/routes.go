package workercard

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes wires worker-card endpoints under the `worker-cards`
// permission module. Lookup-by-code is used at the cashier till — it's a
// view operation; full list and card CRUD are management ops.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db)
	h := NewHandler(repo)

	g := rg.Group("/worker-cards")

	g.GET("/by-card/:code",
		middleware.RequirePermission(permChecker, "worker-cards", "view"),
		h.GetByCard,
	)
	g.GET("",
		middleware.RequirePermission(permChecker, "worker-cards", "view"),
		h.List,
	)
	g.POST("",
		middleware.RequirePermission(permChecker, "worker-cards", "create"),
		h.Add,
	)
	g.PUT("/:id",
		middleware.RequirePermission(permChecker, "worker-cards", "update"),
		h.Update,
	)
	g.DELETE("/:id",
		middleware.RequirePermission(permChecker, "worker-cards", "delete"),
		h.Delete,
	)
}
