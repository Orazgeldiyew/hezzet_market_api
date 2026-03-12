package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaleLifecycle(t *testing.T) {
	r, db := SetupSuite(t)
	CleanupDB(t, db)
	token := GetAdminToken(t, r, db)

	warehouseID := SeedWarehouse(t, db)
	productID := SeedProductWithStock(t, db, warehouseID)
	paymentTypeID := SeedPaymentType(t, db)

	var draftSaleID float64
	var confirmSaleID float64

	t.Run("CreateDraftSale", func(t *testing.T) {
		resp := DoRequest(t, r, "POST", "/api/sales", map[string]any{
			"warehouse_id": warehouseID,
			"items": []map[string]any{
				{"product_id": productID, "qty_milli": 5000}, // 5 units
			},
			"note": "test draft sale",
		}, token)

		require.Equal(t, http.StatusCreated, resp.Code, "body: %s", resp.Body.String())

		var detail map[string]any
		ParseResponse(t, resp, &detail)
		sale := detail["sale"].(map[string]any)
		assert.Equal(t, "draft", sale["status"])
		assert.NotZero(t, sale["id"])
		draftSaleID = sale["id"].(float64)
	})

	t.Run("GetSale", func(t *testing.T) {
		path := fmt.Sprintf("/api/sales/%d", int64(draftSaleID))
		resp := DoRequest(t, r, "GET", path, nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())

		var detail map[string]any
		ParseResponse(t, resp, &detail)
		sale := detail["sale"].(map[string]any)
		assert.Equal(t, "draft", sale["status"])
		items := detail["items"].([]any)
		assert.Len(t, items, 1)
	})

	t.Run("CancelDraftSale", func(t *testing.T) {
		path := fmt.Sprintf("/api/sales/%d/cancel", int64(draftSaleID))
		resp := DoRequest(t, r, "POST", path, nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())
	})

	t.Run("CreateAndConfirmSale", func(t *testing.T) {
		// Create a new sale
		resp := DoRequest(t, r, "POST", "/api/sales", map[string]any{
			"warehouse_id": warehouseID,
			"items": []map[string]any{
				{"product_id": productID, "qty_milli": 3000}, // 3 units
			},
		}, token)
		require.Equal(t, http.StatusCreated, resp.Code, "body: %s", resp.Body.String())

		var detail map[string]any
		ParseResponse(t, resp, &detail)
		sale := detail["sale"].(map[string]any)
		confirmSaleID = sale["id"].(float64)

		// Confirm the sale
		confirmPath := fmt.Sprintf("/api/sales/%d/confirm", int64(confirmSaleID))
		confirmResp := DoRequest(t, r, "POST", confirmPath, nil, token)
		require.Equal(t, http.StatusOK, confirmResp.Code, "body: %s", confirmResp.Body.String())

		var confirmDetail map[string]any
		ParseResponse(t, confirmResp, &confirmDetail)
		confirmedSale := confirmDetail["sale"].(map[string]any)
		assert.Equal(t, "confirmed", confirmedSale["status"])
	})

	t.Run("ConfirmSaleWithPayment", func(t *testing.T) {
		// Create another sale
		resp := DoRequest(t, r, "POST", "/api/sales", map[string]any{
			"warehouse_id": warehouseID,
			"items": []map[string]any{
				{"product_id": productID, "qty_milli": 2000}, // 2 units
			},
		}, token)
		require.Equal(t, http.StatusCreated, resp.Code, "body: %s", resp.Body.String())

		var detail map[string]any
		ParseResponse(t, resp, &detail)
		sale := detail["sale"].(map[string]any)
		saleID := sale["id"].(float64)
		totalCents := sale["total_cents"].(float64)

		// Confirm with payment
		confirmPath := fmt.Sprintf("/api/sales/%d/confirm", int64(saleID))
		confirmResp := DoRequest(t, r, "POST", confirmPath, map[string]any{
			"payment_type_id": paymentTypeID,
			"payment_amount":  int64(totalCents),
		}, token)
		require.Equal(t, http.StatusOK, confirmResp.Code, "body: %s", confirmResp.Body.String())

		var confirmDetail map[string]any
		ParseResponse(t, confirmResp, &confirmDetail)
		confirmedSale := confirmDetail["sale"].(map[string]any)
		assert.Equal(t, "confirmed", confirmedSale["status"])
		// Should have a transaction_id
		assert.NotNil(t, confirmDetail["transaction_id"])
	})

	t.Run("CancelConfirmedSale", func(t *testing.T) {
		path := fmt.Sprintf("/api/sales/%d/cancel", int64(confirmSaleID))
		resp := DoRequest(t, r, "POST", path, nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())
	})

	t.Run("ListSales", func(t *testing.T) {
		resp := DoRequest(t, r, "GET", "/api/sales", nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())
	})

	t.Run("GetReceipt", func(t *testing.T) {
		// Create and confirm a sale for receipt
		resp := DoRequest(t, r, "POST", "/api/sales", map[string]any{
			"warehouse_id": warehouseID,
			"items": []map[string]any{
				{"product_id": productID, "qty_milli": 1000},
			},
		}, token)
		require.Equal(t, http.StatusCreated, resp.Code, "body: %s", resp.Body.String())

		var detail map[string]any
		ParseResponse(t, resp, &detail)
		sale := detail["sale"].(map[string]any)
		saleID := int64(sale["id"].(float64))

		confirmPath := fmt.Sprintf("/api/sales/%d/confirm", saleID)
		confirmResp := DoRequest(t, r, "POST", confirmPath, nil, token)
		require.Equal(t, http.StatusOK, confirmResp.Code, "body: %s", confirmResp.Body.String())

		// Get receipt HTML via API
		receiptPath := fmt.Sprintf("/api/sales/%d/receipt", saleID)
		receiptResp := DoRequest(t, r, "GET", receiptPath, nil, token)
		require.Equal(t, http.StatusOK, receiptResp.Code, "body: %s", receiptResp.Body.String())
		assert.Contains(t, receiptResp.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, receiptResp.Body.String(), "Test Shop")
	})
}
