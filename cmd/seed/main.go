package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/database"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	db, err := database.NewPool(ctx, cfg.DBDSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	steps := []struct {
		name string
		fn   func(context.Context, *pgxpool.Pool) error
	}{
		{"roles", seedRoles},
		{"admin user", seedAdmin},
		{"payment types", seedPaymentTypes},
		{"cash register", seedCashRegister},
		{"receipt settings", seedReceiptSettings},
		{"warehouse", seedWarehouse},	
	}

	for _, s := range steps {
		fmt.Printf("seeding %s... ", s.name)
		if err := s.fn(ctx, db); err != nil {
			log.Fatalf("FAIL: %v\n", err)
		}
		fmt.Println("OK")
	}

	fmt.Println("\nseeding completed successfully!")
}

// ── Roles ──

func seedRoles(ctx context.Context, db *pgxpool.Pool) error {
	roles := []struct{ code, name, desc string }{
		{"admin", "Administrator", "Full access"},
		{"manager", "Manager", "Store management"},
		{"operator", "Operator", "Stock operations"},
		{"cashier", "Cashier", "POS sales"},
	}
	for _, r := range roles {
		_, err := db.Exec(ctx, `
			INSERT INTO roles (code, name, description, is_system)
			VALUES ($1, $2, $3, true)
			ON CONFLICT (code) DO NOTHING
		`, r.code, r.name, r.desc)
		if err != nil {
			return err
		}
	}
	return nil
}

// ── Admin User ──

func seedAdmin(ctx context.Context, db *pgxpool.Pool) error {
	username := getenv("SEED_ADMIN_USERNAME", "admin")
	password := getenv("SEED_ADMIN_PASSWORD", "admin123")
	fullName := getenv("SEED_ADMIN_FULLNAME", "Super Admin")
	reset := strings.TrimSpace(os.Getenv("SEED_ADMIN_RESET")) == "1"

	var adminID int64
	err := db.QueryRow(ctx, `
		SELECT u.id FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		JOIN roles r ON r.id = ur.role_id
		WHERE r.code='admin' AND u.deleted_at IS NULL
		ORDER BY u.id LIMIT 1
	`).Scan(&adminID)

	if err == nil {
		if !reset {
			fmt.Printf("exists (id=%d) ", adminID)
			return nil
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		_, err = db.Exec(ctx, `
			UPDATE users SET username=$1, password_hash=$2, full_name=$3, is_active=true, deleted_at=NULL
			WHERE id=$4
		`, username, string(hash), fullName, adminID)
		if err != nil {
			return err
		}
		fmt.Printf("reset (id=%d) ", adminID)
		return nil
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	var userID int64
	err = db.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, full_name, phone, email, is_active)
		VALUES ($1, $2, $3, '', '', true)
		RETURNING id
	`, username, string(hash), fullName).Scan(&userID)
	if err != nil {
		return err
	}

	var roleID int64
	db.QueryRow(ctx, `SELECT id FROM roles WHERE code='admin'`).Scan(&roleID)
	_, err = db.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, userID, roleID)
	if err != nil {
		return err
	}
	fmt.Printf("created (id=%d, login=%s/%s) ", userID, username, password)
	return nil
}

// ── Payment Types ──

func seedPaymentTypes(ctx context.Context, db *pgxpool.Pool) error {
	types := []struct{ code, name string }{
		{"cash", "Cash"},
		{"card", "Card"},
		{"bank_transfer", "Bank Transfer"},
		{"debt", "Debt"},
		{"other", "Other"},
	}
	for _, t := range types {
		_, err := db.Exec(ctx, `
			INSERT INTO payment_types (code, name, is_active)
			VALUES ($1, $2, true)
			ON CONFLICT DO NOTHING
		`, t.code, t.name)
		if err != nil {
			return err
		}
	}
	return nil
}

// ── Cash Register ──

func seedCashRegister(ctx context.Context, db *pgxpool.Pool) error {
	var exists bool
	db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cash_registers)`).Scan(&exists)
	if exists {
		return nil
	}
	_, err := db.Exec(ctx, `INSERT INTO cash_registers (name, is_active) VALUES ('Касса 1', true)`)
	return err
}

// ── Receipt Settings ──

func seedReceiptSettings(ctx context.Context, db *pgxpool.Pool) error {
	var exists bool
	db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM receipt_settings WHERE id=1)`).Scan(&exists)
	if exists {
		return nil
	}
	_, err := db.Exec(ctx, `
		INSERT INTO receipt_settings (id, shop_name, shop_address, shop_phone, footer, delete_code, bonus_percent)
		VALUES (1, 'Hezzet Market', 'Aşgabat', '+993 12 345678', 'Satyn alanyňyz üçin sag boluň!', '0000', 1)
	`)
	return err
}

// ── Warehouse ──

func seedWarehouse(ctx context.Context, db *pgxpool.Pool) error {
	var exists bool
	db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM warehouses WHERE is_active=true)`).Scan(&exists)
	if exists {
		return nil
	}
	_, err := db.Exec(ctx, `INSERT INTO warehouses (name, address, is_active) VALUES ('Основной склад', 'Aşgabat', true)`)
	return err
}

func getenv(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}
