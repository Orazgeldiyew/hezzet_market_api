package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type Pagination struct {
	Page   int
	Limit  int
	Offset int
}

const (
	DefaultPageLimit = 10
	MaxPageLimit     = 1000
)

func PaginationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		pageStr := c.DefaultQuery("page", "1")
		limitStr := c.DefaultQuery("limit", "10")

		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = DefaultPageLimit
		}
		if limit > MaxPageLimit {
			limit = MaxPageLimit
		}

		offset := (page - 1) * limit
		if offset > 100_000 {
			offset = 100_000
		}

		p := Pagination{
			Page:   page,
			Limit:  limit,
			Offset: offset,
		}

		// store in context
		c.Set("pagination", p)

		// continue to next handler
		c.Next()
	}
}
