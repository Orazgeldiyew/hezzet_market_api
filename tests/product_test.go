package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductCRUD(t *testing.T) {
	r, db := SetupSuite(t)
	CleanupDB(t, db)
	token := GetAdminToken(t, r, db)

	var productID float64

	t.Run("CreateProduct", func(t *testing.T) {
		resp := DoMultipartRequest(t, r, "POST", "/api/products", map[string]any{
			"name":           "Integration Test Product",
			"sku":            "INT-001",
			"unit_type":      "piece",
			"unit":           "piece",
			"purchase_price": 5000,
			"sale_price":     10000,
		}, token)

		require.Equal(t, http.StatusCreated, resp.Code, "body: %s", resp.Body.String())

		var product map[string]any
		ParseResponse(t, resp, &product)
		assert.NotZero(t, product["id"])
		assert.Equal(t, "Integration Test Product", product["name"])
		assert.Equal(t, "INT-001", product["sku"])
		assert.Equal(t, "piece", product["unit_type"])
		productID = product["id"].(float64)
	})

	t.Run("ListProducts", func(t *testing.T) {
		resp := DoRequest(t, r, "GET", "/api/products", nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())

		var list map[string]any
		ParseResponse(t, resp, &list)
		items := list["items"].([]any)
		assert.GreaterOrEqual(t, len(items), 1)
	})

	t.Run("GetProduct", func(t *testing.T) {
		path := fmt.Sprintf("/api/products/%d", int64(productID))
		resp := DoRequest(t, r, "GET", path, nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())

		var product map[string]any
		ParseResponse(t, resp, &product)
		assert.Equal(t, "Integration Test Product", product["name"])
	})

	t.Run("UpdateProduct", func(t *testing.T) {
		path := fmt.Sprintf("/api/products/%d", int64(productID))
		resp := DoRequest(t, r, "PATCH", path, map[string]any{
			"name":       "Updated Product Name",
			"sale_price": 15000,
		}, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())

		var product map[string]any
		ParseResponse(t, resp, &product)
		assert.Equal(t, "Updated Product Name", product["name"])
	})

	t.Run("DeleteProduct", func(t *testing.T) {
		path := fmt.Sprintf("/api/products/%d", int64(productID))
		resp := DoRequest(t, r, "DELETE", path, nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())

		// Verify product is soft-deleted (GET should return 404 or inactive)
		getResp := DoRequest(t, r, "GET", path, nil, token)
		// Product might still be accessible but marked inactive, or return 404
		if getResp.Code == http.StatusOK {
			var product map[string]any
			ParseResponse(t, getResp, &product)
			assert.Equal(t, false, product["is_active"])
		}
	})
}
