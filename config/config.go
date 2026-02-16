package config

import (
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env                   string
	HTTPAddr              string
	DBDSN                 string
	DBHost                string
	DBPort                string
	DBUser                string
	DBPassword            string
	JWTSecret             string
	JWTExpiresIn          string
	AccessTokenSecret     string
	RefreshTokenSecret    string
	TokenType             string
	AccessTokenExpiresIn  string
	RefreshTokenExpiresIn string
}

func Load() Config {
	_ = godotenv.Load()

	cfg := Config{
		Env:                   getenv("APP_ENV", "dev"),
		HTTPAddr:              getenv("HTTP_ADDR", ":8080"),
		DBDSN:                 getenv("DB_DSN", ""),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		JWTExpiresIn:          getenv("JWT_EXPIRES_IN", "24h"),
		AccessTokenSecret:     os.Getenv("ACCESS_TOKEN_SECRET"),
		RefreshTokenSecret:    os.Getenv("REFRESH_TOKEN_SECRET"),
		TokenType:             getenv("TOKEN_TYPE", "Bearer"),
		AccessTokenExpiresIn:  getenv("ACCESS_TOKEN_EXPIRES_IN", "15m"),
		RefreshTokenExpiresIn: getenv("REFRESH_TOKEN_EXPIRES_IN", "168h"),
	}

	// Fallback: if separate secrets are empty, use JWTSecret
	if cfg.AccessTokenSecret == "" && cfg.JWTSecret != "" {
		cfg.AccessTokenSecret = cfg.JWTSecret
	}
	if cfg.RefreshTokenSecret == "" && cfg.JWTSecret != "" {
		cfg.RefreshTokenSecret = cfg.JWTSecret
	}

	return cfg
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// ParseDuration parses a duration string that supports Go's standard format
// plus "d" (days) suffix, e.g. "30d" → 720h.
func ParseDuration(s string) (time.Duration, error) {
	// Try standard Go parsing first
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Handle "Nd" format (days)
	re := regexp.MustCompile(`^(\d+)d$`)
	if matches := re.FindStringSubmatch(s); len(matches) == 2 {
		days, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// Fallback to standard parse to return proper error
	return time.ParseDuration(s)
}
