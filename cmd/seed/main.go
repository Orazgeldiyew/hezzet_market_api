package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/database"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.NewPool(ctx, cfg.DBDSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := seedAdminRole(ctx, db); err != nil {
		log.Fatalf("failed to seed admin role: %v", err)
	}

	if err := seedSuperuser(ctx, db); err != nil {
		log.Fatalf("failed to seed superuser: %v", err)
	}

	fmt.Println("seeding completed successfully")
}

func seedAdminRole(ctx context.Context, db *pgxpool.Pool) error {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM roles WHERE code = 'admin')`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check admin role: %w", err)
	}

	if exists {
		fmt.Println("admin role already exists, skipping")
		return nil
	}

	_, err = db.Exec(ctx, `INSERT INTO roles (code, name) VALUES ('admin', 'Administrator')`)
	if err != nil {
		return fmt.Errorf("insert admin role: %w", err)
	}

	fmt.Println("admin role created")
	return nil
}

func seedSuperuser(ctx context.Context, db *pgxpool.Pool) error {
	const (
		username = "admin"
		password = "admin123"
		fullName = "Super Admin"
	)

	var exists bool
	err := db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND deleted_at IS NULL)`,
		username,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check admin user: %w", err)
	}

	if exists {
		fmt.Println("admin user already exists, skipping")
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	var userID int64
	err = db.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, full_name, phone, email, is_active)
		 VALUES ($1, $2, $3, '', '', true)
		 RETURNING id`,
		username, string(hash), fullName,
	).Scan(&userID)
	if err != nil {
		return fmt.Errorf("insert admin user: %w", err)
	}

	fmt.Printf("admin user created (id=%d, username=%s, password=%s)\n", userID, username, password)

	// Assign admin role
	var roleID int64
	err = db.QueryRow(ctx, `SELECT id FROM roles WHERE code = 'admin'`).Scan(&roleID)
	if err != nil {
		return fmt.Errorf("find admin role: %w", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`,
		userID, roleID,
	)
	if err != nil {
		return fmt.Errorf("assign admin role: %w", err)
	}

	fmt.Println("admin role assigned to superuser")
	return nil
}
