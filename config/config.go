package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Env          string
	HTTPAddr     string
	DBDSN        string
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	JWTSecret    string
	JWTExpiresIn string
}

func Load() Config {
	_ = godotenv.Load()
	return Config{
		Env:          getenv("APP_ENV", "dev"),
		HTTPAddr:     getenv("HTTP_ADDR", ":8080"),
		DBDSN:        getenv("DB_DSN", ""),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		JWTExpiresIn: getenv("JWT_EXPIRES_IN", "24h"),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
