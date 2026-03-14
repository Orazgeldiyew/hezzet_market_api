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

	g := rg.Group("/notifications")
	g.Use(middleware.RequireRoles()) // admin-only

	// SMS logs
	g.GET("/sms", middleware.PaginationMiddleware(), h.ListSMSLogs)
	g.GET("/sms/:job_id", h.GetSMSLog)
	g.POST("/sms/:job_id/requeue", h.RequeueSMSLog)

	// Admin phone management
	g.GET("/phones", ph.List)
	g.POST("/phones", ph.Add)
	g.PUT("/phones/:id", ph.Update)
	g.DELETE("/phones/:id", ph.Delete)

}
