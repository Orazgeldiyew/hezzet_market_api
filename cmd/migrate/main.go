package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	var (
		path = flag.String("path", "./migrations", "migrations folder")
		cmd  = flag.String("cmd", "up", "command: up | down | steps | version | force")
		n    = flag.Int("n", 1, "steps for steps/force cmd (can be negative)")
	)
	flag.Parse()

	// Load .env: try CWD first, then walk up to find project root
	loadEnv()

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN is not set. Check your .env file or environment variables.")
	}

	db, err := postgres.WithInstance(pgxStdlibDB(dsn), &postgres.Config{})
	if err != nil {
		log.Fatal(err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+*path,
		"postgres",
		db,
	)
	if err != nil {
		log.Fatal(err)
	}

	switch *cmd {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	case "steps":
		err = m.Steps(*n)
	case "force":
		err = m.Force(*n)
	case "version":
		v, dirty, verr := m.Version()
		if verr == migrate.ErrNilVersion {
			fmt.Println("version: nil")
			return
		}
		if verr != nil {
			log.Fatal(verr)
		}
		fmt.Printf("version: %d dirty=%v\n", v, dirty)
		return
	default:
		log.Fatalf("unknown cmd: %s", *cmd)
	}

	if err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
	}

	fmt.Println("ok")
}

// loadEnv tries to load .env from the current directory, then walks up parent
// directories until it finds one (so `go run ./cmd/migrate` works from any level).
func loadEnv() {
	// Try CWD first
	if err := godotenv.Load(); err == nil {
		return
	}

	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			_ = godotenv.Load(envPath)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

// pgxStdlibDB returns *sql.DB using pgx stdlib (needed by golang-migrate postgres driver)
func pgxStdlibDB(dsn string) *sql.DB {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		log.Fatal(err)
	}
	return stdlib.OpenDB(*cfg)
}
