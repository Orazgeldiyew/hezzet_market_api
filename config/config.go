package config

import (
	"os"
	"regexp"
	"strconv"
	"strings"
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

	// Swagger (prod-safe)
	SwaggerUser     string
	SwaggerPass     string
	SwaggerAllowIPs string // comma-separated CIDRs
	SwaggerHost     string // e.g. localhost:5000
	SwaggerSchemes  string // e.g. http or https

	// File uploads
	UploadsDir    string // local directory for uploaded files, e.g. "./uploads"
	PublicBaseURL string // public base URL for building file URLs, e.g. "http://localhost:8080"

	// CORS
	CORSAllowedOrigins []string

	// Receipt defaults
	ReceiptShopName    string
	ReceiptShopAddress string
	ReceiptShopPhone   string
	ReceiptFooter      string

	// Notification / SMS
	RedisURL            string
	SMSProvider         string // "log" | "twilio"
	SMSFrom             string
	SMSWorkers          int
	LowStockDefault     int64 // threshold in regular units (not milli)
	LowStockDedupTTL    time.Duration
	SMSRateLimitPerHour int
	AdminPhones         []string
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
		SwaggerUser:           getenv("SWAGGER_USER", ""),
		SwaggerPass:           getenv("SWAGGER_PASS", ""),
		SwaggerAllowIPs:       getenv("SWAGGER_ALLOW_IPS", ""),
		SwaggerHost:           getenv("SWAGGER_HOST", ""),
		SwaggerSchemes:        getenv("SWAGGER_SCHEMES", ""),
		
		
	}

	// Fallback: if separate secrets are empty, use JWTSecret
	if cfg.AccessTokenSecret == "" && cfg.JWTSecret != "" {
		cfg.AccessTokenSecret = cfg.JWTSecret
	}
	if cfg.RefreshTokenSecret == "" && cfg.JWTSecret != "" {
		cfg.RefreshTokenSecret = cfg.JWTSecret
	}

	// ── File uploads ──
	cfg.UploadsDir    = getenv("UPLOADS_DIR", "./uploads")
	cfg.PublicBaseURL = getenv("PUBLIC_BASE_URL", "http://localhost:8080")

	// ── CORS ──
	cfg.CORSAllowedOrigins = splitEnvCSV(getenv("CORS_ALLOWED_ORIGINS", ""))
	if cfg.Env == "dev" && len(cfg.CORSAllowedOrigins) == 0 {
		cfg.CORSAllowedOrigins = []string{"http://localhost:5000"}
	}

	// ── Receipt defaults ──
	cfg.ReceiptShopName = getenv("RECEIPT_SHOP_NAME", "Hezzet Market")
	cfg.ReceiptShopAddress = getenv("RECEIPT_SHOP_ADDRESS", "")
	cfg.ReceiptShopPhone = getenv("RECEIPT_SHOP_PHONE", "")
	cfg.ReceiptFooter = getenv("RECEIPT_FOOTER", "Satyn alanyňyz üçin sag boluň!")

	// ── Notification / SMS ──
	cfg.RedisURL = getenv("REDIS_URL", "")
	cfg.SMSProvider = getenv("SMS_PROVIDER", "log")
	cfg.SMSFrom = getenv("SMS_FROM", "HezzetMarket")
	cfg.AdminPhones = splitEnvCSV(getenv("ADMIN_PHONES", ""))

	if n, err := strconv.Atoi(getenv("SMS_WORKERS", "1")); err == nil && n > 0 {
		cfg.SMSWorkers = n
	} else {
		cfg.SMSWorkers = 1
	}
	if n, err := strconv.ParseInt(getenv("LOW_STOCK_DEFAULT", "10"), 10, 64); err == nil && n > 0 {
		cfg.LowStockDefault = n
	} else {
		cfg.LowStockDefault = 10
	}
	if n, err := strconv.Atoi(getenv("SMS_RATE_LIMIT_PER_HOUR", "3")); err == nil && n > 0 {
		cfg.SMSRateLimitPerHour = n
	} else {
		cfg.SMSRateLimitPerHour = 3
	}
	if d, err := ParseDuration(getenv("LOW_STOCK_DEDUP_TTL", "6h")); err == nil {
		cfg.LowStockDedupTTL = d
	} else {
		cfg.LowStockDedupTTL = 6 * time.Hour
	}

	return cfg
}

func splitEnvCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
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
