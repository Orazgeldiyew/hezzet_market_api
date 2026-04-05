package shift

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) {
	repo := NewRepository(db)
	h := NewHandler(repo)

	// Cash registers
	rg.GET("/registers",
		middleware.RequireRoles("cashier", "operator", "manager"),
		h.ListRegisters,
	)

	// Shifts
	shifts := rg.Group("/shifts")

	shifts.POST("/open",
		middleware.RequireRoles("cashier", "operator", "manager"),
		h.OpenShift,
	)

	shifts.GET("/current",
		middleware.RequireRoles("cashier", "operator", "manager"),
		h.GetCurrent,
	)

	shifts.POST("/:id/close",
		middleware.RequireRoles("cashier", "operator", "manager"),
		h.CloseShift,
	)

	shifts.GET("",
		middleware.RequireRoles("manager"),
		h.ListShifts,
	)

	shifts.GET("/:id",
		middleware.RequireRoles("manager"),
		h.GetShift,
	)
}
