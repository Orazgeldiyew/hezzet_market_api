package workerfinance

import (
	"context"

	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Service struct {
	repo        *Repository
	financeRepo *finance.Repository
}

func NewService(repo *Repository, financeRepo *finance.Repository) *Service {
	return &Service{repo: repo, financeRepo: financeRepo}
}

// ── compensation ────────────────────────────────────────────────────────────

func (s *Service) SetCompensation(ctx context.Context, req SetCompensationRequest) (WorkerCompensation, error) {
	c := WorkerCompensation{
		WorkerID:        req.WorkerID,
		BaseSalaryCents: req.BaseSalaryCents,
		PayDay:          req.PayDay,
	}
	if err := s.repo.UpsertCompensation(ctx, &c); err != nil {
		return WorkerCompensation{}, apperr.Internal(err)
	}
	return c, nil
}

func (s *Service) GetCompensation(ctx context.Context, workerID int64) (WorkerCompensation, error) {
	c, err := s.repo.GetCompensation(ctx, workerID)
	if err != nil {
		if isNotFound(err) {
			return WorkerCompensation{}, apperr.NotFound("COMPENSATION_NOT_FOUND", "compensation not configured for this worker")
		}
		return WorkerCompensation{}, apperr.Internal(err)
	}
	return c, nil
}

// ── fines ───────────────────────────────────────────────────────────────────

func (s *Service) CreateFine(ctx context.Context, workerID int64, req CreateFineRequest, userID int64) (WorkerFine, error) {
	uid := &userID
	f := WorkerFine{
		WorkerID:    workerID,
		AmountCents: req.AmountCents,
		Reason:      req.Reason,
		CreatedBy:   uid,
	}
	if err := s.repo.CreateFine(ctx, &f); err != nil {
		return WorkerFine{}, apperr.Internal(err)
	}
	return f, nil
}

func (s *Service) ListFines(ctx context.Context, workerID int64, limit, offset int) (FineListResult, error) {
	items, total, err := s.repo.ListFines(ctx, workerID, limit, offset)
	if err != nil {
		return FineListResult{}, apperr.Internal(err)
	}
	if items == nil {
		items = []WorkerFine{}
	}
	return FineListResult{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

// ── debts ───────────────────────────────────────────────────────────────────

func (s *Service) CreateDebt(ctx context.Context, workerID int64, req CreateDebtRequest, userID int64) (WorkerDebt, error) {
	uid := &userID

	// Look up "cash" payment type for the expense transaction
	cashPTID, err := s.financeRepo.GetPaymentTypeIDByCode(ctx, "cash")
	if err != nil {
		return WorkerDebt{}, apperr.Internal(err)
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return WorkerDebt{}, apperr.Internal(err)
	}
	defer tx.Rollback(ctx)

	// Create debt
	d := WorkerDebt{
		WorkerID:       workerID,
		AmountCents:    req.AmountCents,
		RemainingCents: req.AmountCents,
		Type:           req.Type,
		Note:           req.Note,
		CreatedBy:      uid,
	}
	if err := s.repo.CreateDebt(ctx, tx, &d); err != nil {
		return WorkerDebt{}, apperr.Internal(err)
	}

	// Create expense transaction for the advance/loan
	reason := "Worker " + req.Type
	finTxn := finance.Transaction{
		Type:         "expense",
		AmountCents:  req.AmountCents,
		RelatedTable: "worker_debt",
		RelatedID:    &d.ID,
		Status:       "paid",
		Reason:       &reason,
		CreatedBy:    uid,
		PaymentTypeID: &cashPTID,
	}
	if err := s.financeRepo.CreateTransaction(ctx, tx, &finTxn); err != nil {
		return WorkerDebt{}, apperr.Internal(err)
	}

	// Create payment record (full amount)
	finPay := finance.Payment{
		TransactionID: finTxn.ID,
		PaymentTypeID: cashPTID,
		AmountCents:   req.AmountCents,
		CreatedBy:     uid,
	}
	if err := s.financeRepo.CreatePayment(ctx, tx, &finPay); err != nil {
		return WorkerDebt{}, apperr.Internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return WorkerDebt{}, apperr.Internal(err)
	}

	return d, nil
}

func (s *Service) ListDebts(ctx context.Context, workerID int64, limit, offset int) (DebtListResult, error) {
	items, total, err := s.repo.ListDebts(ctx, workerID, limit, offset)
	if err != nil {
		return DebtListResult{}, apperr.Internal(err)
	}
	if items == nil {
		items = []WorkerDebt{}
	}
	return DebtListResult{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}
