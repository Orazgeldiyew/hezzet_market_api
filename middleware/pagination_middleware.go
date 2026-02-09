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

func PaginationMiddleware() gin.HandlerFunc {
  return func(c *gin.Context) {
    pageStr := c.DefaultQuery("page", "1")
    limitStr := c.DefaultQuery("limit", "10")

    page, err := strconv.Atoi(pageStr)
    if err != nil || page < 1 {
      page = 1
    }

    limit, err := strconv.Atoi(limitStr)
    if err != nil || limit <= 0 || limit > 100 {
      limit = 10
    }

    offset := (page - 1) * limit

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