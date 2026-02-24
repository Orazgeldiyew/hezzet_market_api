package notification

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// RegisterRoutes registers admin SMS log endpoints under the /api group.
// queue may be nil when Redis is not configured — the handler will return
// INTERNAL_ERROR on requeue attempts in that case.
func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool, queue *Queue) {
	logRepo := NewLogRepository(db)
	h := NewHandler(logRepo, queue)

	g := rg.Group("/notifications/sms")
	g.Use(middleware.RequireRoles()) // admin-only (empty roles = admin only)

	g.GET("",
		middleware.PaginationMiddleware(),
		h.ListSMSLogs,
	)
	g.GET("/:job_id", h.GetSMSLog)
	g.POST("/:job_id/requeue", h.RequeueSMSLog)
}
