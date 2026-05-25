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
// RegisterRoutes wires notification endpoints. Two tiers of access:
//
//   - SMS audit (logs, phone book) — system administration, kept on admin-only
//     via the empty RequireRoles() (admin bypass). These aren't business
//     permissions and don't appear in the /roles matrix.
//   - Runtime settings + in-app inbox — manager-level, gated on reports:view
//     so the admin can grant manager-equivalents access without giving them
//     SMS audit too.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, queue *Queue, phoneRepo *PhoneRepository, permChecker middleware.PermissionChecker) {
	logRepo := NewLogRepository(db)
	h := NewHandler(logRepo, queue)

	ph := NewPhoneHandler(phoneRepo)

	settingsRepo := NewSettingsRepository(db)
	sh := NewSettingsHandler(settingsRepo)

	inboxRepo := NewInboxRepository(db)
	ih := NewInboxHandler(inboxRepo)

	g := rg.Group("/notifications")

	// System-admin tier (admin only).
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

	// Managerial tier — settings + bell inbox.
	staffGroup := g.Group("")
	staffGroup.Use(middleware.RequirePermission(permChecker, "reports", "view"))
	{
		staffGroup.GET("/settings", sh.Get)
		staffGroup.PUT("/settings", sh.Update)

		staffGroup.GET("/inbox", middleware.PaginationMiddleware(), ih.List)
		staffGroup.GET("/inbox/unread-count", ih.UnreadCount)
		staffGroup.POST("/inbox/:id/read", ih.MarkRead)
		staffGroup.POST("/inbox/read-all", ih.MarkAllRead)
	}
}
