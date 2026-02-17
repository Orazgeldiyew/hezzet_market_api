package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

func main() {
	var (
		path = flag.String("path", "./migrations", "migrations folder")
		cmd  = flag.String("cmd", "up", "command: up | down | steps | version")
		n    = flag.Int("n", 1, "steps for steps cmd (can be negative)")
	)
	flag.Parse()

	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DBDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Use stdlib *sql.DB driver via pgxpool's ConnString is not available directly,
	// so simplest is open using database/sql with lib/pq OR use pgx stdlib.
	// Here we use pgx stdlib:
	db, err := postgres.WithInstance(pgxStdlibDB(cfg.DBDSN), &postgres.Config{})
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

// pgxStdlibDB returns *sql.DB using pgx stdlib (needed by golang-migrate postgres driver)
func pgxStdlibDB(dsn string) *sql.DB {
	// IMPORTANT: add imports: "database/sql" and "github.com/jackc/pgx/v5/stdlib"
	// Use stdlib.OpenDB(*pgx.ConnConfig)
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		log.Fatal(err)
	}
	return stdlib.OpenDB(*cfg)
}
