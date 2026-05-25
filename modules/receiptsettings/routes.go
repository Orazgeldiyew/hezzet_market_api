package receiptsettings

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes wires receipt-settings endpoints under the `reports`
// permission module (settings = managerial config). verify-delete-code is
// the one exception: it's a cashier-side check at the till, so it uses
// sales:view (every cashier already has it).
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, cfg config.Config, permChecker middleware.PermissionChecker) *Repository {
	defaults := Defaults{
		ShopName:    cfg.ReceiptShopName,
		ShopAddress: cfg.ReceiptShopAddress,
		ShopPhone:   cfg.ReceiptShopPhone,
		Footer:      cfg.ReceiptFooter,
	}

	uploadsDir := cfg.UploadsDir
	if uploadsDir == "" {
		uploadsDir = "./uploads"
	}

	repo := NewRepository(db, defaults)
	h := NewHandler(repo, uploadsDir, cfg.PublicBaseURL)

	g := rg.Group("/settings/receipt")

	// Server compares the code internally so the stored value never leaves
	// the backend — safe to expose to cashier-level staff at the till.
	g.POST("/verify-delete-code",
		middleware.RequirePermission(permChecker, "sales", "view"),
		h.VerifyDeleteCode,
	)

	// Full settings (incl. delete_code itself) — manager/admin level.
	mgr := g.Group("")
	mgr.Use(middleware.RequirePermission(permChecker, "reports", "view"))
	{
		mgr.GET("", h.GetSettings)
		mgr.PUT("", h.UpdateSettings)
		mgr.POST("/logo", h.UploadLogo)
	}

	return repo
}
