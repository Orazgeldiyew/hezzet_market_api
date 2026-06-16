package auditlog

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/middleware"
)

// The Repository carries the *pgxpool.Pool we need for snapshotting. Pulling
// it out of the repo keeps the middleware signature stable and avoids weaving
// another dependency through the call sites.

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
	"POST /api/sales/:id/transfer":                       "SALE_TRANSFER",
	"POST /api/sales/:id/return":                         "SALE_RETURN",
	"POST /api/sales/:id/print":                          "SALE_PRINT",
	"POST /api/purchases/:id/receive":                    "PO_RECEIVE",
	"POST /api/purchases/:id/cancel":                     "PO_CANCEL",
	"POST /api/purchases/:id/payments":                   "PO_PAYMENT",
	"POST /api/shifts/open":                              "SHIFT_OPEN",
	"POST /api/shifts/:id/close":                         "SHIFT_CLOSE",
	"POST /api/inventory/:id/confirm":                    "INVENTORY_CONFIRM",
	"POST /api/inventory/:id/cancel":                     "INVENTORY_CANCEL",
	"POST /api/customer-debts/:id/pay":                   "CUSTOMER_DEBT_PAY",
	"POST /api/supplier-debts/:id/pay":                   "SUPPLIER_DEBT_PAY",
	"POST /api/workers/debts/:debt_id/pay":               "WORKER_DEBT_PAY",
	"POST /api/supplier-returns/:id/confirm":             "SUPPLIER_RETURN_CONFIRM",
	"POST /api/supplier-returns/:id/cancel":              "SUPPLIER_RETURN_CANCEL",
	"POST /api/products/:id/photo":                       "PRODUCT_PHOTO_UPLOAD",
	"PUT /api/settings/receipt":                          "RECEIPT_SETTINGS_UPDATE",
	"POST /api/settings/receipt/logo":                    "RECEIPT_LOGO_UPLOAD",
}

// publicAuditPaths maps "METHOD /full/gin/route/pattern" of public-auth endpoints
// (login, refresh, register) to (successAction, failedAction). These are logged
// even when no user_id is in context — the handler is expected to set
// "audit_username" so failed attempts still carry the attempted username.
var publicAuditPaths = map[string][2]string{
	"POST /api/auth/login":    {"LOGIN", "LOGIN_FAILED"},
	"POST /api/auth/refresh":  {"TOKEN_REFRESH", "TOKEN_REFRESH_FAILED"},
	"POST /api/auth/register": {"REGISTER", "REGISTER_FAILED"},
}

// AuditMiddleware records write operations to audit_logs. It logs:
//   - Successful authenticated writes (2xx with user_id in context)
//   - Failed authenticated writes (4xx/5xx with user_id) — action gets a _FAILED suffix
//   - Public auth attempts (login/refresh/register) — both success and failure,
//     using attempted username from "audit_username" context key
//
// For routes registered in snapshotRegistry, it also captures BEFORE/AFTER
// snapshots of the entity so reviewers can see exactly what changed.
//
// Other public routes (no user_id, no audit_username) are still silently skipped.
// All writes go through a goroutine so audit never blocks the request.
func AuditMiddleware(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Take the "before" snapshot before the handler runs. We need it now
		// because the handler will mutate the row; afterwards the old state
		// is unrecoverable.
		method := c.Request.Method
		fullPath := c.FullPath()
		snapper := snapshotFor(method, fullPath)
		entityID := c.Param("id")

		var oldSnap Snapshot
		if snapper != nil {
			// Use a fresh context — request context may already be cancelled
			// once we read it later in the goroutine. Errors are swallowed:
			// missing snapshot is better than blocking the request.
			if snap, err := snapper(c.Request.Context(), repo.DB(), entityID); err == nil {
				oldSnap = snap
			}
		}

		c.Next()

		// Only write methods
		if method != "POST" && method != "PATCH" && method != "PUT" && method != "DELETE" {
			return
		}

		// ErrorMiddleware runs AFTER us in the middleware chain, so by the time
		// audit fires the response body and status code may not be written yet.
		// Trust c.Errors (handler-reported failures) over c.Writer.Status() —
		// otherwise a handler that calls c.Error(...) and returns shows up with
		// status=200 here and misleads the audit log.
		status := c.Writer.Status()
		isSuccess := status >= 200 && status < 300 && len(c.Errors) == 0

		uidVal, _ := c.Get("user_id")
		uid, _ := uidVal.(int64)

		usernameVal, _ := c.Get("username")
		username, _ := usernameVal.(string)

		// Fallback for public auth attempts: the handler stashes the attempted
		// username on the context before calling the service so failed logins
		// still capture WHO tried.
		if username == "" {
			if v, ok := c.Get("audit_username"); ok {
				username, _ = v.(string)
			}
		}

		// Decide whether to log this request at all.
		routeKey := c.Request.Method + " " + c.FullPath()
		publicAuth, isPublicAuth := publicAuditPaths[routeKey]
		switch {
		case isPublicAuth:
			// Always log login/refresh/register attempts.
		case uid > 0:
			// Authenticated route — log success and failure both.
		default:
			// Unauthenticated request to a non-auth route — skip silently.
			return
		}

		action := resolveAction(c)
		if isPublicAuth {
			if isSuccess {
				action = publicAuth[0]
			} else {
				action = publicAuth[1]
			}
		} else if !isSuccess {
			// Distinguish failed attempts from successful ones for the same route.
			action = action + "_FAILED"
		}

		entityType := resolveEntityType(c.FullPath())
		entityIDPtr := entityIDFromParams(c)
		ip := c.ClientIP()
		reqID := middleware.GetRequestID(c)

		// "After" snapshot: same lookup, but the row may now be updated or gone.
		// Only do it on success — on failure the state didn't change, no diff.
		// For CREATE the URL has no :id, so the handler may set
		// "audit_entity_id" on the context to expose the newly-created ID.
		afterID := entityID
		if v, ok := c.Get("audit_entity_id"); ok {
			if s, ok := v.(string); ok && s != "" {
				afterID = s
				if entityIDPtr == nil {
					entityIDPtr = &afterID
				}
			}
		}

		var newSnap Snapshot
		if isSuccess && snapper != nil {
			if snap, err := snapper(c.Request.Context(), repo.DB(), afterID); err == nil {
				newSnap = snap
			}
		}

		entry := &AuditLog{
			UserID:     uid,
			Username:   username,
			Action:     action,
			EntityType: entityType,
			EntityID:   entityIDPtr,
			Method:     method,
			Path:       c.Request.URL.Path,
			IPAddress:  &ip,
			RequestID:  &reqID,
			OldValue:   snapshotJSON(oldSnap),
			NewValue:   snapshotJSON(newSnap),
		}

		// Bound the audit-log write to 5 seconds. context.Background() is
		// deliberate — the request's own ctx is already cancelled by the time
		// gin runs deferred middleware code, so any descendant ctx would
		// short-circuit immediately. NOTE: in-flight goroutines aren't waited
		// for on graceful shutdown, so a few audit entries can still be lost.
		// Mitigation if that matters: switch to a buffered channel drained by
		// a background worker.
		go func(entry *AuditLog) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			repo.Create(ctx, entry)
		}(entry)
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
