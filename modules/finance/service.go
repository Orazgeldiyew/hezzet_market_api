package finance

import (
	"context"
	"fmt"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func isAppError(err error) bool {
	_, ok := err.(*apperr.AppError)
	return ok
}

// ── payment types ───────────────────────────────────────────────────────────

func (s *Service) ListPaymentTypes(ctx context.Context) ([]PaymentType, error) {
	pts, err := s.repo.ListPaymentTypes(ctx)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if pts == nil {
		pts = []PaymentType{}
	}
	return pts, nil
}

// ── transactions ────────────────────────────────────────────────────────────

func (s *Service) ListTransactions(ctx context.Context, f TransactionFilter, limit, offset int) (TransactionListResult, error) {
	items, total, err := s.repo.ListTransactions(ctx, f, limit, offset)
	if err != nil {
		if isAppError(err) {
			return TransactionListResult{}, err
		}
		return TransactionListResult{}, apperr.Internal(err)
	}
	if items == nil {
		items = []Transaction{}
	}
	return TransactionListResult{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Service) GetTransaction(ctx context.Context, id int64) (TransactionDetail, error) {
	txn, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return TransactionDetail{}, apperr.NotFound("TRANSACTION_NOT_FOUND", "transaction not found")
		}
		return TransactionDetail{}, apperr.Internal(err)
	}

	payments, err := s.repo.GetPayments(ctx, id)
	if err != nil {
		return TransactionDetail{}, apperr.Internal(err)
	}
	if payments == nil {
		payments = []Payment{}
	}

	var paidCents int64
	for _, p := range payments {
		paidCents += p.AmountCents
	}

	return TransactionDetail{
		Transaction: txn,
		Payments:    payments,
		PaidCents:   paidCents,
		DebtCents:   txn.AmountCents - paidCents,
	}, nil
}

func (s *Service) CreateManual(ctx context.Context, req CreateManualRequest, userID int64) (TransactionDetail, error) {
	// Validation for optional initial payment
	if req.PaymentAmount != nil {
		if req.PaymentTypeID == nil {
			return TransactionDetail{}, apperr.Validation("payment_type_id is required when payment_amount is provided")
		}
		if *req.PaymentAmount <= 0 {
			return TransactionDetail{}, apperr.Validation("payment_amount must be positive")
		}
		if *req.PaymentAmount > req.AmountCents {
			return TransactionDetail{}, apperr.Validation("initial payment cannot exceed transaction amount")
		}
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return TransactionDetail{}, apperr.Internal(err)
	}
	defer tx.Rollback(ctx)

	uid := &userID
	txn := Transaction{
		Type:         req.Type,
		AmountCents:  req.AmountCents,
		Reason:       req.Reason,
		RelatedTable: "manual",
		Status:       "pending",
		CreatedBy:    uid,
	}

	if err := s.repo.CreateTransaction(ctx, tx, &txn); err != nil {
		return TransactionDetail{}, apperr.Internal(err)
	}

	var payments []Payment

	if req.PaymentAmount != nil {
		p := Payment{
			TransactionID: txn.ID,
			PaymentTypeID: *req.PaymentTypeID,
			AmountCents:   *req.PaymentAmount,
			Note:          req.PaymentNote,
			CreatedBy:     uid,
		}
		if err := s.repo.CreatePayment(ctx, tx, &p); err != nil {
			return TransactionDetail{}, apperr.Internal(err)
		}
		payments = append(payments, p)

		// Compute status
		status := "partial"
		if *req.PaymentAmount >= req.AmountCents {
			status = "paid"
		}
		txn, err = s.repo.UpdateTransactionStatus(ctx, tx, txn.ID, status)
		if err != nil {
			return TransactionDetail{}, apperr.Internal(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return TransactionDetail{}, apperr.Internal(err)
	}

	if payments == nil {
		payments = []Payment{}
	}

	var paidCents int64
	for _, p := range payments {
		paidCents += p.AmountCents
	}

	return TransactionDetail{
		Transaction: txn,
		Payments:    payments,
		PaidCents:   paidCents,
		DebtCents:   txn.AmountCents - paidCents,
	}, nil
}

func (s *Service) AddPayment(ctx context.Context, transactionID int64, req AddPaymentRequest, userID int64) (TransactionDetail, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return TransactionDetail{}, apperr.Internal(err)
	}
	defer tx.Rollback(ctx)

	// Lock the transaction row
	txn, err := s.repo.LockTransaction(ctx, tx, transactionID)
	if err != nil {
		if isNotFound(err) {
			return TransactionDetail{}, apperr.NotFound("TRANSACTION_NOT_FOUND", "transaction not found")
		}
		return TransactionDetail{}, apperr.Internal(err)
	}

	if txn.Status == "canceled" {
		return TransactionDetail{}, apperr.Validation("cannot add payment to a canceled transaction")
	}
	if txn.Status == "paid" {
		return TransactionDetail{}, apperr.Validation("transaction is already fully paid")
	}

	paidSoFar, err := s.repo.GetPaidSum(ctx, tx, transactionID)
	if err != nil {
		return TransactionDetail{}, apperr.Internal(err)
	}

	remaining := txn.AmountCents - paidSoFar
	if req.AmountCents > remaining {
		return TransactionDetail{}, apperr.Validation(
			fmt.Sprintf("payment exceeds remaining amount of %d cents", remaining),
		)
	}

	uid := &userID
	p := Payment{
		TransactionID: transactionID,
		PaymentTypeID: req.PaymentTypeID,
		AmountCents:   req.AmountCents,
		Note:          req.Note,
		CreatedBy:     uid,
	}
	if err := s.repo.CreatePayment(ctx, tx, &p); err != nil {
		return TransactionDetail{}, apperr.Internal(err)
	}

	newPaid := paidSoFar + req.AmountCents
	newStatus := "partial"
	if newPaid >= txn.AmountCents {
		newStatus = "paid"
	}

	txn, err = s.repo.UpdateTransactionStatus(ctx, tx, transactionID, newStatus)
	if err != nil {
		return TransactionDetail{}, apperr.Internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return TransactionDetail{}, apperr.Internal(err)
	}

	// Re-fetch all payments for the response
	payments, err := s.repo.GetPayments(ctx, transactionID)
	if err != nil {
		return TransactionDetail{}, apperr.Internal(err)
	}
	if payments == nil {
		payments = []Payment{}
	}

	var totalPaid int64
	for _, pay := range payments {
		totalPaid += pay.AmountCents
	}

	return TransactionDetail{
		Transaction: txn,
		Payments:    payments,
		PaidCents:   totalPaid,
		DebtCents:   txn.AmountCents - totalPaid,
	}, nil
}

func (s *Service) CancelTransaction(ctx context.Context, id int64) (Transaction, error) {
	txn, err := s.repo.SetCanceled(ctx, id)
	if err != nil {
		if isAppError(err) {
			return Transaction{}, err
		}
		return Transaction{}, apperr.Internal(err)
	}
	return txn, nil
}
