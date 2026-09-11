package sale

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/customer"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/receiptsettings"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, finRepo *finance.Repository, publicBaseURL string, receiptRepo *receiptsettings.Repository, customerRepo *customer.Repository, permChecker middleware.PermissionChecker) *Repository {
	repo := NewRepository(db, publicBaseURL, customerRepo)
	repo.receiptRepo = receiptRepo
	svc := NewService(repo, finRepo)
	h := NewHandler(svc, receiptRepo, publicBaseURL)

	sales := rg.Group("/sales")

	sales.POST("",
		middleware.RequirePermission(permChecker, "sales", "create"),
		h.CreateSale,
	)

	sales.GET("",
		middleware.RequirePermission(permChecker, "sales", "view"),
		h.ListSales,
	)

	sales.GET("/:id",
		middleware.RequirePermission(permChecker, "sales", "view"),
		h.GetSale,
	)

	sales.GET("/:id/receipt",
		middleware.RequirePermission(permChecker, "sales", "view"),
		h.GetReceipt,
	)

	sales.POST("/:id/print",
		middleware.RequirePermission(permChecker, "sales", "view"),
		h.Reprint,
	)

	sales.POST("/:id/confirm",
		middleware.RequirePermission(permChecker, "sales", "update"),
		h.ConfirmSale,
	)

	sales.POST("/:id/cancel",
		middleware.RequirePermission(permChecker, "sales", "update"),
		h.CancelSale,
	)

	sales.POST("/:id/transfer",
		middleware.RequirePermission(permChecker, "sales", "transfer"),
		h.TransferDraft,
	)

	sales.POST("/:id/return",
		middleware.RequirePermission(permChecker, "sales", "return"),
		h.ReturnSale,
	)
	sales.PATCH("/:id/items/:item_id/decrease",
		middleware.RequirePermission(permChecker, "sales", "update"),
		h.DecreaseItem,
	)

	sales.DELETE("/:id/items/:item_id",
		middleware.RequirePermission(permChecker, "sales", "update"),
		h.DeleteSaleItem,
	)

	return repo
}
