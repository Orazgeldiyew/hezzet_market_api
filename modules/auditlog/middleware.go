package auditlog

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// actionOverrides maps "METHOD /full/gin/route/pattern" → action name.
// Paths not found here fall back to method-based defaults:
//
//	POST   → CREATE
//	PATCH  → UPDATE
//	PUT    → UPDATE
//	DELETE → DELETE
var actionOverrides = map[string]string{
	"POST /api/payroll/calculate":                        "CALCULATE",
	"POST /api/payroll/:id/pay":                          "PAY",
	"POST /api/transactions/:id/cancel":                  "CANCEL",
	"POST /api/transactions/:id/payments":                "PAYMENT_ADD",
	"POST /api/stock/in":                                 "STOCK_IN",
	"POST /api/stock/in/bulk":                            "STOCK_IN_BULK",
	"POST /api/stock/out":                                "STOCK_OUT",
	"POST /api/stock/transfer":                           "STOCK_TRANSFER",
	"POST /api/stock/move":                               "STOCK_MOVE",
	"POST /api/workers/:id/fines":                        "FINE_CREATE",
	"POST /api/workers/:id/debts":                        "DEBT_CREATE",
	"POST /api/workers/compensation":                     "COMPENSATION_SET",
	"POST /auth/users/:id/block":                         "USER_BLOCK",
	"POST /auth/users/:id/unblock":                       "USER_UNBLOCK",
	"POST /auth/users/:id/password":                      "PASSWORD_CHANGE",
	"POST /api/customers/:id/spent":                      "CUSTOMER_SPENT",
	"PUT /api/products/:id/categories":                   "CATEGORY_SET",
	"DELETE /api/products/:id/categories/:categoryId":    "CATEGORY_REMOVE",
	"POST /api/sales/:id/confirm":                        "SALE_CONFIRM",
	"POST /api/sales/:id/cancel":                         "SALE_CANCEL",
	"POST /api/purchases/:id/receive":                    "PO_RECEIVE",
	"POST /api/purchases/:id/cancel":                     "PO_CANCEL",
	"POST /api/purchases/:id/payments":                   "PO_PAYMENT",
}

// AuditMiddleware logs every successful write operation (POST/PATCH/PUT/DELETE
// with a 2xx response) to the audit_logs table. The write is non-blocking.
// Public routes that have no user_id in context are silently skipped.
func AuditMiddleware(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only write methods
		method := c.Request.Method
		if method != "POST" && method != "PATCH" && method != "PUT" && method != "DELETE" {
			return
		}

		// Only successful responses
		status := c.Writer.Status()
		if status < 200 || status >= 300 {
			return
		}

		// Skip if no authenticated user (public routes)
		uidVal, exists := c.Get("user_id")
		if !exists {
			return
		}
		uid, _ := uidVal.(int64)
		if uid == 0 {
			return
		}

		usernameVal, _ := c.Get("username")
		username, _ := usernameVal.(string)

		action := resolveAction(c)
		entityType := resolveEntityType(c.FullPath())
		entityID := entityIDFromParams(c)
		ip := c.ClientIP()
		reqID := middleware.GetRequestID(c)

		entry := &AuditLog{
			UserID:     uid,
			Username:   username,
			Action:     action,
			EntityType: entityType,
			EntityID:   entityID,
			Method:     method,
			Path:       c.Request.URL.Path,
			IPAddress:  &ip,
			RequestID:  &reqID,
		}

		go repo.Create(context.Background(), entry)
	}
}

func resolveAction(c *gin.Context) string {
	key := c.Request.Method + " " + c.FullPath()
	if action, ok := actionOverrides[key]; ok {
		return action
	}
	switch c.Request.Method {
	case "POST":
		return "CREATE"
	case "PATCH", "PUT":
		return "UPDATE"
	case "DELETE":
		return "DELETE"
	default:
		return c.Request.Method
	}
}

// resolveEntityType extracts the entity name from the gin route pattern.
// "/api/products/:id"  → "product"
// "/auth/users/:id"    → "user"
// "/api/stock/in"      → "stock"
func resolveEntityType(fullPath string) string {
	path := strings.TrimPrefix(fullPath, "/api/")
	path = strings.TrimPrefix(path, "/auth/")

	seg := strings.SplitN(path, "/", 2)[0]
	if seg == "" {
		return "unknown"
	}

	exceptions := map[string]string{
		"categories": "category",
	}
	if mapped, ok := exceptions[seg]; ok {
		return mapped
	}
	if strings.HasSuffix(seg, "s") && len(seg) > 3 {
		return seg[:len(seg)-1]
	}
	return seg
}

func entityIDFromParams(c *gin.Context) *string {
	id := c.Param("id")
	if id == "" {
		return nil
	}
	return &id
}
