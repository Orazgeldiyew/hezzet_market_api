package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinanceTransactions(t *testing.T) {
	r, db := SetupSuite(t)
	CleanupDB(t, db)
	token := GetAdminToken(t, r, db)

	paymentTypeID := SeedPaymentType(t, db)

	var incomeTxnID float64
	var expenseTxnID float64

	t.Run("CreateManualIncome", func(t *testing.T) {
		resp := DoRequest(t, r, "POST", "/api/transactions/manual", map[string]any{
			"type":         "income",
			"amount_cents": 50000,
			"reason":       "Test income",
		}, token)
		require.Equal(t, http.StatusCreated, resp.Code, "body: %s", resp.Body.String())

		var txn map[string]any
		ParseResponse(t, resp, &txn)
		assert.Equal(t, "income", txn["type"])
		assert.Equal(t, float64(50000), txn["amount_cents"])
		assert.Equal(t, "pending", txn["status"])
		incomeTxnID = txn["id"].(float64)
	})

	t.Run("CreateManualExpense", func(t *testing.T) {
		resp := DoRequest(t, r, "POST", "/api/transactions/manual", map[string]any{
			"type":         "expense",
			"amount_cents": 20000,
			"reason":       "Test expense",
		}, token)
		require.Equal(t, http.StatusCreated, resp.Code, "body: %s", resp.Body.String())

		var txn map[string]any
		ParseResponse(t, resp, &txn)
		assert.Equal(t, "expense", txn["type"])
		assert.Equal(t, float64(20000), txn["amount_cents"])
		expenseTxnID = txn["id"].(float64)
	})

	t.Run("AddPaymentPartial", func(t *testing.T) {
		path := fmt.Sprintf("/api/transactions/%d/payments", int64(incomeTxnID))
		resp := DoRequest(t, r, "POST", path, map[string]any{
			"payment_type_id": paymentTypeID,
			"amount_cents":    25000,
			"note":            "partial payment",
		}, token)
		require.Equal(t, http.StatusCreated, resp.Code, "body: %s", resp.Body.String())

		var txn map[string]any
		ParseResponse(t, resp, &txn)
		assert.Equal(t, "partial", txn["status"])
		assert.Equal(t, float64(25000), txn["paid_cents"])
		assert.Equal(t, float64(25000), txn["debt_cents"])
	})

	t.Run("AddPaymentFull", func(t *testing.T) {
		path := fmt.Sprintf("/api/transactions/%d/payments", int64(incomeTxnID))
		resp := DoRequest(t, r, "POST", path, map[string]any{
			"payment_type_id": paymentTypeID,
			"amount_cents":    25000,
		}, token)
		require.Equal(t, http.StatusCreated, resp.Code, "body: %s", resp.Body.String())

		var txn map[string]any
		ParseResponse(t, resp, &txn)
		assert.Equal(t, "paid", txn["status"])
		assert.Equal(t, float64(50000), txn["paid_cents"])
		assert.Equal(t, float64(0), txn["debt_cents"])
	})

	t.Run("GetTransaction", func(t *testing.T) {
		path := fmt.Sprintf("/api/transactions/%d", int64(incomeTxnID))
		resp := DoRequest(t, r, "GET", path, nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())

		var txn map[string]any
		ParseResponse(t, resp, &txn)
		assert.Equal(t, "paid", txn["status"])
		payments := txn["payments"].([]any)
		assert.Len(t, payments, 2)
	})

	t.Run("CancelTransaction", func(t *testing.T) {
		path := fmt.Sprintf("/api/transactions/%d/cancel", int64(expenseTxnID))
		resp := DoRequest(t, r, "POST", path, nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())

		var txn map[string]any
		ParseResponse(t, resp, &txn)
		assert.Equal(t, "canceled", txn["status"])
	})

	t.Run("ListTransactions", func(t *testing.T) {
		resp := DoRequest(t, r, "GET", "/api/transactions", nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())
	})

	t.Run("ListTransactionsFilterByType", func(t *testing.T) {
		resp := DoRequest(t, r, "GET", "/api/transactions?type=income", nil, token)
		require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())
	})
}
