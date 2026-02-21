package middleware

import (
	"crypto/rand"
	"fmt"

	"github.com/gin-gonic/gin"
)

const (
	// RequestIDHeader is the canonical header name for the request ID.
	// If the client sends this header we reuse its value; otherwise we generate one.
	RequestIDHeader = "X-Request-Id"

	requestIDCtxKey = "request_id"
)

// RequestID is a middleware that attaches a unique request ID to every request.
//
//   - If the incoming request already carries X-Request-Id, that value is reused
//     (useful when a gateway / load-balancer already stamps requests).
//   - Otherwise a random UUID v4 is generated.
//
// The ID is stored in the Gin context (retrievable via GetRequestID) and echoed
// back in the X-Request-Id response header so clients can correlate log entries.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = newUUID()
		}
		c.Set(requestIDCtxKey, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// GetRequestID retrieves the request ID that was set by the RequestID middleware.
// Returns an empty string if the middleware was not registered.
func GetRequestID(c *gin.Context) string {
	v, _ := c.Get(requestIDCtxKey)
	s, _ := v.(string)
	return s
}

// newUUID returns a random UUID v4 string (RFC 4122).
// Uses crypto/rand; the _ discards the (unlikely) error from rand.Read so the
// function stays side-effect free – worst case is a lower-entropy ID.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits (RFC 4122)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
