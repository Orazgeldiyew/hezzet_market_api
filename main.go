package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/database"
	"github.com/Orazgeldiyew/hezzet_market_backend/server"
)

// @title           Hezzet Market API
// @version         1.0
// @description     Market backend (products, stock, income, sales)
// @BasePath        /
// @schemes         http
func main() {
	cfg := config.Load()

	if cfg.Env == "dev" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.NewPool(context.Background(), cfg.DBDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	r := server.NewRouter(server.Deps{DB: db})

	log.Println("listening on", cfg.HTTPAddr)
	log.Fatal(r.Run(cfg.HTTPAddr))
}
