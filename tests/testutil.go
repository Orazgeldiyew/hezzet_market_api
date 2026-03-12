package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	"github.com/Orazgeldiyew/hezzet_market_backend/server"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/database"
)

var (
	testRouter *gin.Engine
	testDB     *pgxpool.Pool
	adminToken string
)

func testDSN() string {
	if dsn := os.Getenv("TEST_DB_DSN"); dsn != "" {
		return dsn
	}
	return "postgres://postgres:postgres123@localhost:5432/market_test?sslmode=disable"
}

// SetupSuite initializes the test router and DB once per test run.
func SetupSuite(t *testing.T) (*gin.Engine, *pgxpool.Pool) {
	t.Helper()
	if testRouter != nil {
		return testRouter, testDB
	}

	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	db, err := database.NewPool(ctx, testDSN())
	if err != nil {
		t.Fatalf("connect to test DB: %v", err)
	}

	// Set test env vars for config
	os.Setenv("DB_DSN", testDSN())
	os.Setenv("APP_ENV", "dev")
	os.Setenv("ACCESS_TOKEN_SECRET", "test-secret-key-for-jwt")
	os.Setenv("REFRESH_TOKEN_SECRET", "test-refresh-secret-key")
	os.Setenv("JWT_SECRET", "test-secret-key-for-jwt")
	os.Setenv("RECEIPT_SHOP_NAME", "Test Shop")
	os.Setenv("RECEIPT_FOOTER", "Thanks!")

	cfg := config.Load()

	r := server.NewRouter(server.Deps{
		DB:  db,
		Cfg: cfg,
	})

	testRouter = r
	testDB = db
	return r, db
}

// CleanupDB truncates all test data in correct FK order.
func CleanupDB(t *testing.T, db *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	tables := []string{
		"payments",
		"transactions",
		"stock_reservations",
		"sale_items",
		"sales",
		"purchase_items",
		"purchase_orders",
		"warehouse_item_details",
		"warehouse_items",
		"product_categories",
		"product_barcodes",
		"products",
		"categories",
		"customers",
		"suppliers",
		"warehouses",
		"sms_logs",
		"audit_logs",
		"receipt_settings",
	}
	for _, tbl := range tables {
		_, _ = db.Exec(ctx, fmt.Sprintf("TRUNCATE %s CASCADE", tbl))
	}
	// Re-insert receipt_settings singleton
	_, _ = db.Exec(ctx, "INSERT INTO receipt_settings (id) VALUES (1) ON CONFLICT DO NOTHING")
}

// GetAdminToken creates an admin user and returns a valid JWT token.
func GetAdminToken(t *testing.T, r *gin.Engine, db *pgxpool.Pool) string {
	t.Helper()
	if adminToken != "" {
		return adminToken
	}

	ctx := context.Background()

	// Create admin user via DB (register endpoint creates regular user, we need admin role)
	_, _ = db.Exec(ctx, `DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE username = 'testadmin')`)
	_, _ = db.Exec(ctx, `DELETE FROM users WHERE username = 'testadmin'`)

	// Register via API
	resp := DoRequest(t, r, "POST", "/auth/register", map[string]any{
		"username":  "testadmin",
		"password":  "admin123",
		"full_name": "Test Admin",
	}, "")

	if resp.Code != http.StatusCreated && resp.Code != http.StatusOK {
		t.Fatalf("register failed: %d %s", resp.Code, resp.Body.String())
	}

	// Grant admin role directly in DB
	_, err := db.Exec(ctx, `
		INSERT INTO user_roles (user_id, role)
		SELECT id, 'admin' FROM users WHERE username = 'testadmin'
		ON CONFLICT DO NOTHING
	`)
	if err != nil {
		t.Fatalf("grant admin role: %v", err)
	}

	// Also grant manager, operator, cashier for broader access
	for _, role := range []string{"manager", "operator", "cashier"} {
		_, _ = db.Exec(ctx, `
			INSERT INTO user_roles (user_id, role)
			SELECT id, $1 FROM users WHERE username = 'testadmin'
			ON CONFLICT DO NOTHING
		`, role)
	}

	// Login
	loginResp := DoRequest(t, r, "POST", "/auth/login", map[string]any{
		"username": "testadmin",
		"password": "admin123",
	}, "")

	if loginResp.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", loginResp.Code, loginResp.Body.String())
	}

	var result struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &result); err != nil {
		t.Fatalf("parse login response: %v", err)
	}

	adminToken = result.Data.AccessToken
	return adminToken
}

// DoRequest makes an HTTP request against the test router.
func DoRequest(t *testing.T, r *gin.Engine, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, path, reqBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ParseResponse unmarshals the JSON response data field.
func ParseResponse(t *testing.T, w *httptest.ResponseRecorder, target any) {
	t.Helper()
	var envelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("parse response envelope: %v (body: %s)", err, w.Body.String())
	}
	if target != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, target); err != nil {
			t.Fatalf("parse response data: %v", err)
		}
	}
}

// DoMultipartRequest sends a multipart/form-data request with a "data" JSON field (for product creation).
func DoMultipartRequest(t *testing.T, r *gin.Engine, method, path string, data any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	dataJSON, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	if err := writer.WriteField("data", string(dataJSON)); err != nil {
		t.Fatalf("write field: %v", err)
	}
	writer.Close()

	req, err := http.NewRequest(method, path, &body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// Seed helpers

func SeedWarehouse(t *testing.T, db *pgxpool.Pool) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(context.Background(),
		`INSERT INTO warehouses (name, address) VALUES ('Test Warehouse', 'Test Address') RETURNING id`,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed warehouse: %v", err)
	}
	return id
}

func SeedProduct(t *testing.T, db *pgxpool.Pool) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(context.Background(),
		`INSERT INTO products (name, sku, unit_type, unit, unit_scale, purchase_price, sale_price, is_active)
		 VALUES ('Test Product', 'TST-001', 'piece', 'piece', 1000, 5000, 10000, true) RETURNING id`,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}
	return id
}

func SeedProductWithStock(t *testing.T, db *pgxpool.Pool, warehouseID int64) int64 {
	t.Helper()
	productID := SeedProduct(t, db)
	_, err := db.Exec(context.Background(),
		`INSERT INTO warehouse_items (warehouse_id, product_id, qty_milli, avg_cost_cents, total_cost_cents)
		 VALUES ($1, $2, 100000, 5000, 500000)`, // 100 units
		warehouseID, productID,
	)
	if err != nil {
		t.Fatalf("seed stock: %v", err)
	}
	return productID
}

func SeedSupplier(t *testing.T, db *pgxpool.Pool) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(context.Background(),
		`INSERT INTO suppliers (name, phone) VALUES ('Test Supplier', '+99312345') RETURNING id`,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed supplier: %v", err)
	}
	return id
}

func SeedCustomer(t *testing.T, db *pgxpool.Pool) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(context.Background(),
		`INSERT INTO customers (name, phone) VALUES ('Test Customer', '+99354321') RETURNING id`,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	return id
}

func SeedPaymentType(t *testing.T, db *pgxpool.Pool) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(context.Background(),
		`INSERT INTO payment_types (code, name, is_active) VALUES ('cash', 'Cash', true) RETURNING id`,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed payment type: %v", err)
	}
	return id
}
