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
	g.Use(middleware.RequireRoles("manager")) // admin + manager

	g.GET("", h.GetSettings)
	g.PUT("", h.UpdateSettings)
	g.POST("/logo", h.UploadLogo)

	return repo
}
