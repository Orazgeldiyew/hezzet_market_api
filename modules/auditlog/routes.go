package auditlog

import (
	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, repo *Repository) {
	h := NewHandler(repo)

	rg.GET("/audit-logs", middleware.RequireRoles("manager"), h.List)
}
