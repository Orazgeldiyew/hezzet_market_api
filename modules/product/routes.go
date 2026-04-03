package product

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, uploadsDir, publicBaseURL string) {
	repo := NewRepository(db, publicBaseURL)
	stock := NewStockRepository(db)
	svc := NewService(repo, stock, uploadsDir)
	h := NewHandler(svc)

	// Read: operator, cashier, manager (admin bypass)
	read := rg.Group("/products")
	read.Use(middleware.RequireRoles("operator", "cashier", "manager"))
	{
		read.GET("", h.List)
		read.GET("/by-barcode/:code", h.GetByBarcode)
		read.GET("/:id", h.Get)
		read.GET("/:id/card", h.GetCard)
		read.GET("/:id/categories", h.GetCategories)
	}

	// Write: operator (admin bypass)
	write := rg.Group("/products")
	write.Use(middleware.RequireRoles("operator"))
	{
		write.POST("", h.Create)
		write.PATCH("/:id", h.Update)
		write.DELETE("/:id", h.Delete)
		write.PUT("/:id/categories", h.SetCategories)
		write.DELETE("/:id/categories/:categoryId", h.RemoveCategory)
		write.POST("/:id/photo", h.UploadPhoto)
	}
}
