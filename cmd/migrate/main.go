// cmd/migrate/main.go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: migrate [up|down|drop|force|version]")
	}
	cmd := os.Args[1]

	cfg := config.Load()
	dsn := cfg.DBDSN

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		log.Fatalf("migrate init: %v", err)
	}
	defer func() { _, _ = m.Close() }()

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("migrate up: %v", err)
		}
		fmt.Println("migrate up: OK")

	case "down":
		// по умолчанию откатываем 1 шаг
		if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("migrate down: %v", err)
		}
		fmt.Println("migrate down 1: OK")

	case "drop":
		if err := m.Drop(); err != nil {
			log.Fatalf("migrate drop: %v", err)
		}
		fmt.Println("migrate drop: OK")

	case "force":
		if len(os.Args) < 3 {
			log.Fatalf("usage: migrate force <version>")
		}
		var v int
		_, err := fmt.Sscanf(os.Args[2], "%d", &v)
		if err != nil {
			log.Fatalf("force parse version: %v", err)
		}
		if err := m.Force(v); err != nil {
			log.Fatalf("migrate force: %v", err)
		}
		fmt.Println("migrate force: OK")

	case "version":
		v, dirty, err := m.Version()
		if err == migrate.ErrNilVersion {
			fmt.Println("version: none (0), dirty=false")
			return
		}
		if err != nil {
			log.Fatalf("migrate version: %v", err)
		}
		fmt.Printf("version: %d, dirty=%v\n", v, dirty)

	default:
		log.Fatalf("unknown command: %s", cmd)
	}
}
