package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const localeKey = "locale"

// Locale parses the Accept-Language header and stores the locale in the context.
// Supported: "ru" (default), "tk".
func Locale() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := parseAcceptLanguage(c.GetHeader("Accept-Language"))
		c.Set(localeKey, lang)
		c.Next()
	}
}

// GetLocale returns the locale from the Gin context, defaulting to "ru".
func GetLocale(c *gin.Context) string {
	if v, ok := c.Get(localeKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "ru"
}

func parseAcceptLanguage(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return "ru"
	}
	if idx := strings.Index(header, ","); idx >= 0 {
		header = header[:idx]
	}
	if idx := strings.Index(header, ";"); idx >= 0 {
		header = header[:idx]
	}
	lang := strings.TrimSpace(strings.ToLower(header))
	if lang == "tk" {
		return "tk"
	}
	return "ru"
}
