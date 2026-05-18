package product

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes wires product endpoints behind per-action permissions so
// admins can configure who can edit prices, delete catalog rows, etc.
// permChecker is the same shared checker used by sales/purchases — it reads
// the role_permissions table and supports the standard admin bypass.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, uploadsDir, publicBaseURL string, permChecker middleware.PermissionChecker) {
	repo := NewRepository(db, publicBaseURL)
	stock := NewStockRepository(db)
	svc := NewService(repo, stock, uploadsDir)
	h := NewHandler(svc)

	// Reads gated on products:view (set on the parent group already; explicit
	// here for clarity and so sub-routes inherit the same check).
	read := rg.Group("/products")
	read.Use(middleware.RequirePermission(permChecker, "products", "view"))
	{
		read.GET("", h.List)
		read.GET("/by-barcode/:code", h.GetByBarcode)
		read.GET("/:id", h.Get)
		read.GET("/:id/card", h.GetCard)
		read.GET("/:id/categories", h.GetCategories)
	}

	// Per-action writes — each route requires the action that matches its
	// intent so a role can be granted, say, "update" without "delete".
	write := rg.Group("/products")
	{
		write.POST("", middleware.RequirePermission(permChecker, "products", "create"), h.Create)
		write.PATCH("/:id", middleware.RequirePermission(permChecker, "products", "update"), h.Update)
		write.DELETE("/:id", middleware.RequirePermission(permChecker, "products", "delete"), h.Delete)
		write.PUT("/:id/categories", middleware.RequirePermission(permChecker, "products", "update"), h.SetCategories)
		write.DELETE("/:id/categories/:categoryId", middleware.RequirePermission(permChecker, "products", "update"), h.RemoveCategory)
		write.POST("/:id/photo", middleware.RequirePermission(permChecker, "products", "update"), h.UploadPhoto)
	}

	// Price history is a sensitive view (shows who changed what), so it uses
	// its own "history" action — defaults to false for most roles.
	rg.GET("/products/:id/price-history",
		middleware.RequirePermission(permChecker, "products", "history"),
		h.GetPriceHistory)
}
