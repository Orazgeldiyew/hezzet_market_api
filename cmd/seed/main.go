// package main

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"time"

// 	"github.com/jackc/pgx/v5/pgxpool"
// 	"golang.org/x/crypto/bcrypt"

// 	"github.com/Orazgeldiyew/hezzet_market_backend/config"
// 	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/database"
// )

// func main() {
// 	cfg := config.Load()

// 	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancel()

// 	db, err := database.NewPool(ctx, cfg.DBDSN)
// 	if err != nil {
// 		log.Fatalf("failed to connect to database: %v", err)
// 	}
// 	defer db.Close()

// 	if err := seedAdminRole(ctx, db); err != nil {
// 		log.Fatalf("failed to seed admin role: %v", err)
// 	}

// 	if err := seedSuperuser(ctx, db); err != nil {
// 		log.Fatalf("failed to seed superuser: %v", err)
// 	}

// 	fmt.Println("seeding completed successfully")
// }

// func seedAdminRole(ctx context.Context, db *pgxpool.Pool) error {
// 	var exists bool
// 	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM roles WHERE code = 'admin')`).Scan(&exists)
// 	if err != nil {
// 		return fmt.Errorf("check admin role: %w", err)
// 	}

// 	if exists {
// 		fmt.Println("admin role already exists, skipping")
// 		return nil
// 	}

// 	_, err = db.Exec(ctx, `INSERT INTO roles (code, name) VALUES ('admin', 'Administrator')`)
// 	if err != nil {
// 		return fmt.Errorf("insert admin role: %w", err)
// 	}

// 	fmt.Println("admin role created")
// 	return nil
// }

// func seedSuperuser(ctx context.Context, db *pgxpool.Pool) error {
// 	const (
// 		username = "admin"
// 		password = "admin123"
// 		fullName = "Super Admin"
// 	)

// 	var exists bool
// 	err := db.QueryRow(ctx,
// 		`SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND deleted_at IS NULL)`,
// 		username,
// 	).Scan(&exists)
// 	if err != nil {
// 		return fmt.Errorf("check admin user: %w", err)
// 	}

// 	if exists {
// 		fmt.Println("admin user already exists, skipping")
// 		return nil
// 	}

// 	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
// 	if err != nil {
// 		return fmt.Errorf("hash password: %w", err)
// 	}

// 	var userID int64
// 	err = db.QueryRow(ctx,
// 		`INSERT INTO users (username, password_hash, full_name, phone, email, is_active)
// 		 VALUES ($1, $2, $3, '', '', true)
// 		 RETURNING id`,
// 		username, string(hash), fullName,
// 	).Scan(&userID)
// 	if err != nil {
// 		return fmt.Errorf("insert admin user: %w", err)
// 	}

// 	fmt.Printf("admin user created (id=%d, username=%s, password=%s)\n", userID, username, password)

// 	// Assign admin role
// 	var roleID int64
// 	err = db.QueryRow(ctx, `SELECT id FROM roles WHERE code = 'admin'`).Scan(&roleID)
// 	if err != nil {
// 		return fmt.Errorf("find admin role: %w", err)
// 	}

// 	_, err = db.Exec(ctx,
// 		`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`,
// 		userID, roleID,
// 	)
// 	if err != nil {
// 		return fmt.Errorf("assign admin role: %w", err)
// 	}

// 	fmt.Println("admin role assigned to superuser")
// 	return nil
// }
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

	if err := seedOrUpdateAdmin(ctx, db); err != nil {
		log.Fatalf("failed to seed admin user: %v", err)
	}

	fmt.Println("seeding completed successfully")
}

func seedAdminRole(ctx context.Context, db *pgxpool.Pool) error {
	var exists bool
	if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM roles WHERE code='admin')`).Scan(&exists); err != nil {
		return fmt.Errorf("check admin role: %w", err)
	}
	if exists {
		fmt.Println("admin role already exists, skipping")
		return nil
	}

	_, err := db.Exec(ctx, `INSERT INTO roles (code, name) VALUES ('admin', 'Administrator')`)
	if err != nil {
		return fmt.Errorf("insert admin role: %w", err)
	}
	fmt.Println("admin role created")
	return nil
}

func seedOrUpdateAdmin(ctx context.Context, db *pgxpool.Pool) error {
	// Defaults (можешь оставить как есть)
	username := getenvDefault("SEED_ADMIN_USERNAME", "admin")
	password := getenvDefault("SEED_ADMIN_PASSWORD", "admin123")
	fullName := getenvDefault("SEED_ADMIN_FULLNAME", "Super Admin")
	reset := strings.TrimSpace(os.Getenv("SEED_ADMIN_RESET")) == "1"

	// 1) Ищем существующего admin по роли (самый правильный критерий)
	var adminUserID int64
	err := db.QueryRow(ctx, `
		SELECT u.id
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		JOIN roles r ON r.id = ur.role_id
		WHERE r.code='admin' AND u.deleted_at IS NULL
		ORDER BY u.id ASC
		LIMIT 1
	`).Scan(&adminUserID)

	if err == nil {
		// admin найден
		if !reset {
			fmt.Printf("admin user already exists (id=%d), skipping\n", adminUserID)
			return nil
		}

		// reset = 1 -> обновляем username/password (+ full_name)
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}

		_, err = db.Exec(ctx, `
			UPDATE users
			SET username = $1,
			    password_hash = $2,
			    full_name = $3,
			    is_active = true,
			    updated_at = now()
			WHERE id = $4 AND deleted_at IS NULL
		`, username, string(hash), fullName, adminUserID)
		if err != nil {
			return fmt.Errorf("update admin user: %w", err)
		}

		fmt.Printf("admin user UPDATED (id=%d, username=%s)\n", adminUserID, username)
		return nil
	}

	// Если ошибки не "no rows" — вернуть
	// pgx.ErrNoRows приходит как Scan error: no rows in result set
	if !strings.Contains(err.Error(), "no rows") {
		return fmt.Errorf("find admin user: %w", err)
	}

	// 2) Admin не найден -> создаём нового
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	var userID int64
	err = db.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, full_name, phone, email, is_active)
		VALUES ($1, $2, $3, '', '', true)
		RETURNING id
	`, username, string(hash), fullName).Scan(&userID)
	if err != nil {
		return fmt.Errorf("insert admin user: %w", err)
	}

	var roleID int64
	if err := db.QueryRow(ctx, `SELECT id FROM roles WHERE code='admin'`).Scan(&roleID); err != nil {
		return fmt.Errorf("find admin role: %w", err)
	}

	_, err = db.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, userID, roleID)
	if err != nil {
		return fmt.Errorf("assign admin role: %w", err)
	}

	fmt.Printf("admin user CREATED (id=%d, username=%s, password=%s)\n", userID, username, password)
	return nil
}

func getenvDefault(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}
