package discountrule

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes wires discount-rule endpoints under the `sales:discount`
// permission (rules govern auto-applied discounts at checkout). Listing is
// done under sales:view so cashiers can also fetch them for client-side
// preview if needed; CRUD stays behind discount.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, permChecker middleware.PermissionChecker) *Repository {
	repo := NewRepository(db)
	h := NewHandler(repo)

	g := rg.Group("/discount-rules")
	{
		g.GET("",
			middleware.RequirePermission(permChecker, "sales", "view"),
			h.List,
		)
		g.GET("/:id",
			middleware.RequirePermission(permChecker, "sales", "view"),
			h.GetByID,
		)
		g.POST("",
			middleware.RequirePermission(permChecker, "sales", "discount"),
			h.Create,
		)
		g.PATCH("/:id",
			middleware.RequirePermission(permChecker, "sales", "discount"),
			h.Update,
		)
		g.DELETE("/:id",
			middleware.RequirePermission(permChecker, "sales", "discount"),
			h.Delete,
		)
	}

	return repo
}
