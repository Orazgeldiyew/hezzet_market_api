package notification

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes registers admin notification endpoints under the /api group.
// queue may be nil when Redis is not configured — the handler will return
// INTERNAL_ERROR on requeue attempts in that case.
// phoneRepo must be created in main.go (via NewPhoneRepository) before service
// creation, then passed here so the same instance is shared.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, queue *Queue, phoneRepo *PhoneRepository) {
	logRepo := NewLogRepository(db)
	h := NewHandler(logRepo, queue)

	ph := NewPhoneHandler(phoneRepo)

	settingsRepo := NewSettingsRepository(db)
	sh := NewSettingsHandler(settingsRepo)

	g := rg.Group("/notifications")

	// admin-only: SMS logs + phone management
	adminOnly := g.Group("")
	adminOnly.Use(middleware.RequireRoles())
	{
		adminOnly.GET("/sms", middleware.PaginationMiddleware(), h.ListSMSLogs)
		adminOnly.GET("/sms/:job_id", h.GetSMSLog)
		adminOnly.POST("/sms/:job_id/requeue", h.RequeueSMSLog)

		adminOnly.GET("/phones", ph.List)
		adminOnly.POST("/phones", ph.Add)
		adminOnly.PUT("/phones/:id", ph.Update)
		adminOnly.DELETE("/phones/:id", ph.Delete)
	}

	// admin + manager: runtime-configurable settings
	settingsGroup := g.Group("")
	settingsGroup.Use(middleware.RequireRoles("admin", "manager"))
	{
		settingsGroup.GET("/settings", sh.Get)
		settingsGroup.PUT("/settings", sh.Update)
	}
}
