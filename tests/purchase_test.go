package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPurchaseLifecycle(t *testing.T) {
	r, db := SetupSuite(t)
	CleanupDB(t, db)
	token := GetAdminToken(t, r, db)

	warehouseID := SeedWarehouse(t, db)
	productID := SeedProduct(t, db)
	supplierID := SeedSupplier(t, db)
	paymentTypeID := SeedPaymentType(t, db)

	var draftPOID float64
	var receivePOID float64

	t.Run("CreatePO", func(t *testing.T) {
		resp := DoRequest(t, r, "POST", "/api/purchases", map[string]any{
			"supplier_id":  supplierID,
			"warehouse_id": warehouseID,
			"items": []map[string]any{
				{
					"product_id":     productID,
					"qty_milli":      10000, // 10 units
					"unit_cost_cents": 5000,
				},
			},
			"note": "test purchase order",
		}, token)

		require.Equal(t, http.StatusCreated, resp.Code, "body: %s", resp.Body.String())

		var po map[string]any
		ParseResponse(t, resp, &po)
		assert.Equal(t, "draft", po["status"])
		assert.NotZero(t, po["id"])
		draftPOID = po["id"].(float64)
	})

	t.Run("CreatePOForReceive", func(t *testing.T) {
		resp := DoRequest(t, r, "POST", "/api/purchases", map[string]any{
			"supplier_id":  supplierID,
			"warehouse_id": warehouseID,
			"items": []map[string]any{
				{
					"product_id":     productID,
					"qty_milli":      5000, // 5 units
					"unit_cost_cents": 6000,
				},
			},
		}, token)
		require.Equal(t, http.StatusCreated, resp.Code, "body: %s", resp.Body.String())

		var po map[string]any
		ParseResponse(t, resp, &po)
		receivePOID = po["id"].(float64)
	})

	t.Run("GetPO", func(t *testing.T) {
		path := fmt.Sprintf("/api/purchases/%d", int64(draftPOID))
		resp := DoRequest(t, r, "GET", path, nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())

		var detail map[string]any
		ParseResponse(t, resp, &detail)
		assert.Equal(t, "draft", detail["status"])
		items := detail["items"].([]any)
		assert.Len(t, items, 1)
	})

	t.Run("ReceivePO", func(t *testing.T) {
		path := fmt.Sprintf("/api/purchases/%d/receive", int64(receivePOID))
		resp := DoRequest(t, r, "POST", path, nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())

		var po map[string]any
		ParseResponse(t, resp, &po)
		assert.Equal(t, "received", po["status"])
	})

	t.Run("AddPurchasePayment", func(t *testing.T) {
		path := fmt.Sprintf("/api/purchases/%d/payments", int64(receivePOID))
		resp := DoRequest(t, r, "POST", path, map[string]any{
			"payment_type_id": paymentTypeID,
			"amount_cents":    15000,
			"note":            "partial payment for PO",
		}, token)
		require.Equal(t, http.StatusCreated, resp.Code, "body: %s", resp.Body.String())
	})

	t.Run("CancelDraftPO", func(t *testing.T) {
		path := fmt.Sprintf("/api/purchases/%d/cancel", int64(draftPOID))
		resp := DoRequest(t, r, "POST", path, nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())
	})

	t.Run("ListPurchases", func(t *testing.T) {
		resp := DoRequest(t, r, "GET", "/api/purchases", nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())
	})

	t.Run("DebtSummary", func(t *testing.T) {
		resp := DoRequest(t, r, "GET", "/api/purchases/debt", nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())
	})
}
