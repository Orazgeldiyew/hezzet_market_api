package receiptsettings

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, cfg config.Config) *Repository {
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

	// verify-delete-code is open to any authenticated staff — the cashier
	// needs it to confirm cart-item deletions. The server compares the code
	// internally so the actual code never leaves the backend.
	g.POST("/verify-delete-code",
		middleware.RequireRoles("cashier", "operator", "manager"),
		h.VerifyDeleteCode,
	)

	// Everything else (GET full settings including the code itself, PUT, logo
	// upload) stays manager + admin only.
	mgr := g.Group("")
	mgr.Use(middleware.RequireRoles("manager"))
	{
		mgr.GET("", h.GetSettings)
		mgr.PUT("", h.UpdateSettings)
		mgr.POST("/logo", h.UploadLogo)
	}

	return repo
}
