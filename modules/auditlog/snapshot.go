package auditlog

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Snapshotter fetches the current state of an entity for before/after audit
// diffing. Returns nil when the entity does not exist (e.g. before CREATE,
// after DELETE).
type Snapshotter func(ctx context.Context, db *pgxpool.Pool, entityID string) (Snapshot, error)

// snapshotRegistry maps "METHOD /full/route/pattern" → snapshotter. Routes
// that don't appear here log without before/after — the audit still records
// who/when/where, just not the diff. Add entries as needed; start with the
// fraud-sensitive entities (price changes, discounts, customers, users).
var snapshotRegistry = map[string]Snapshotter{
	// Products — primary target for price-tampering fraud detection.
	"POST /api/products":           snapshotProduct,
	"PATCH /api/products/:id":      snapshotProduct,
	"DELETE /api/products/:id":     snapshotProduct,

	// Discount rules — anyone can lower margins via these.
	"POST /api/discount-rules":         snapshotDiscountRule,
	"PATCH /api/discount-rules/:id":    snapshotDiscountRule,
	"DELETE /api/discount-rules/:id":   snapshotDiscountRule,

	// Customers — debt and bonus fields are money.
	"POST /api/customers":            snapshotCustomer,
	"PATCH /api/customers/:id":       snapshotCustomer,
	"PATCH /api/customers/:id/admin": snapshotCustomer,
	"DELETE /api/customers/:id":      snapshotCustomer,

	// Users — privilege changes, blocks.
	"POST /auth/users":          snapshotUser,
	"PATCH /auth/users/:id":     snapshotUser,
	"DELETE /auth/users/:id":    snapshotUser,
	"POST /auth/users/:id/block":   snapshotUser,
	"POST /auth/users/:id/unblock": snapshotUser,

	// Suppliers — contact info, soft-delete trail.
	"POST /api/suppliers":        snapshotSupplier,
	"PATCH /api/suppliers/:id":   snapshotSupplier,
	"DELETE /api/suppliers/:id":  snapshotSupplier,

	// Categories — rename/move can shift product reporting.
	"POST /api/categories":        snapshotCategory,
	"PATCH /api/categories/:id":   snapshotCategory,
	"DELETE /api/categories/:id":  snapshotCategory,

	// Warehouses — currently only create is exposed; entry kept for the day
	// PATCH/DELETE land.
	"POST /api/warehouses": snapshotWarehouse,

	// Receipt settings — singleton row id=1; PUT path has no :id. The
	// snapshotter ignores entityID and loads the only row.
	"PUT /api/settings/receipt": snapshotReceiptSettings,
}

// snapshotFor returns the snapshotter for a given gin route pattern, or nil.
func snapshotFor(method, fullPath string) Snapshotter {
	if s, ok := snapshotRegistry[method+" "+fullPath]; ok {
		return s
	}
	return nil
}

// queryJSONRow runs a SQL query that returns a single jsonb_build_object row.
// Returns nil if no row exists (covers "before create" and "after delete").
func queryJSONRow(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) (Snapshot, error) {
	var raw []byte
	err := db.QueryRow(ctx, sql, args...).Scan(&raw)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	var out Snapshot
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ─── Per-entity snapshotters ────────────────────────────────────────────────

func snapshotProduct(ctx context.Context, db *pgxpool.Pool, entityID string) (Snapshot, error) {
	if entityID == "" {
		return nil, nil
	}
	return queryJSONRow(ctx, db, `
		SELECT to_jsonb(p) - 'created_at' - 'updated_at'
		FROM products p
		WHERE p.id = $1::bigint AND p.deleted_at IS NULL
	`, entityID)
}

func snapshotDiscountRule(ctx context.Context, db *pgxpool.Pool, entityID string) (Snapshot, error) {
	if entityID == "" {
		return nil, nil
	}
	return queryJSONRow(ctx, db, `
		SELECT to_jsonb(d) - 'created_at' - 'updated_at'
		FROM discount_rules d
		WHERE d.id = $1::bigint
	`, entityID)
}

func snapshotCustomer(ctx context.Context, db *pgxpool.Pool, entityID string) (Snapshot, error) {
	if entityID == "" {
		return nil, nil
	}
	return queryJSONRow(ctx, db, `
		SELECT to_jsonb(c) - 'created_at' - 'updated_at'
		FROM customers c
		WHERE c.id = $1::bigint
	`, entityID)
}

func snapshotUser(ctx context.Context, db *pgxpool.Pool, entityID string) (Snapshot, error) {
	if entityID == "" {
		return nil, nil
	}
	// Strip password hash + token version — they leak credentials/session state
	// without telling us anything useful about admin intent.
	return queryJSONRow(ctx, db, `
		SELECT to_jsonb(u) - 'password_hash' - 'token_version' - 'created_at' - 'updated_at'
		FROM employees u
		WHERE u.id = $1::bigint AND u.deleted_at IS NULL
	`, entityID)
}

func snapshotSupplier(ctx context.Context, db *pgxpool.Pool, entityID string) (Snapshot, error) {
	if entityID == "" {
		return nil, nil
	}
	return queryJSONRow(ctx, db, `
		SELECT to_jsonb(s) - 'created_at' - 'updated_at'
		FROM suppliers s
		WHERE s.id = $1::bigint AND s.deleted_at IS NULL
	`, entityID)
}

func snapshotCategory(ctx context.Context, db *pgxpool.Pool, entityID string) (Snapshot, error) {
	if entityID == "" {
		return nil, nil
	}
	return queryJSONRow(ctx, db, `
		SELECT to_jsonb(c) - 'created_at' - 'updated_at'
		FROM categories c
		WHERE c.id = $1::bigint AND c.deleted_at IS NULL
	`, entityID)
}

func snapshotWarehouse(ctx context.Context, db *pgxpool.Pool, entityID string) (Snapshot, error) {
	if entityID == "" {
		return nil, nil
	}
	return queryJSONRow(ctx, db, `
		SELECT to_jsonb(w) - 'created_at' - 'updated_at'
		FROM warehouses w
		WHERE w.id = $1::bigint
	`, entityID)
}

// snapshotReceiptSettings ignores entityID — receipt_settings is a singleton
// keyed on id=1. Strips the template column (can be 10 KB of HTML) since
// surfacing that in the audit UI is noise; the rest of the row is the
// interesting part (shop name, phone, footer, delete_code, bonus_percent).
func snapshotReceiptSettings(ctx context.Context, db *pgxpool.Pool, _ string) (Snapshot, error) {
	return queryJSONRow(ctx, db, `
		SELECT to_jsonb(rs) - 'template' - 'updated_at'
		FROM receipt_settings rs
		WHERE rs.id = 1
	`)
}

// snapshotJSON marshals a Snapshot to JSON for storage. Returns nil on
// empty/nil input so the column stays NULL.
func snapshotJSON(s Snapshot) json.RawMessage {
	if s == nil {
		return nil
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	return raw
}
