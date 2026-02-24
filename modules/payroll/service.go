package payroll

import (
	"context"
	"fmt"
	"regexp"

	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/workerfinance"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

var periodRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

type Service struct {
	repo     *Repository
	wfRepo   *workerfinance.Repository
	finRepo  *finance.Repository
}

func NewService(repo *Repository, wfRepo *workerfinance.Repository, finRepo *finance.Repository) *Service {
	return &Service{repo: repo, wfRepo: wfRepo, finRepo: finRepo}
}

// ── calculate ───────────────────────────────────────────────────────────────

func (s *Service) Calculate(ctx context.Context, req CalculateRequest, userID int64) (PayrollRun, error) {
	if !periodRe.MatchString(req.Period) {
		return PayrollRun{}, apperr.Validation("period must be YYYY-MM format")
	}

	// Get compensation
	comp, err := s.wfRepo.GetCompensation(ctx, req.WorkerID)
	if err != nil {
		if isNotFound(err) {
			return PayrollRun{}, apperr.NotFound("COMPENSATION_NOT_FOUND", "compensation not configured for this worker")
		}
		return PayrollRun{}, apperr.Internal(err)
	}

	// Sum open fines
	finesCents, err := s.wfRepo.SumOpenFines(ctx, req.WorkerID)
	if err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	// Sum open debts
	debtsCents, err := s.wfRepo.SumOpenDebts(ctx, req.WorkerID)
	if err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	// Calculate net salary
	netSalary := comp.BaseSalaryCents - finesCents - debtsCents
	if netSalary < 0 {
		netSalary = 0
	}

	run := PayrollRun{
		WorkerID:        req.WorkerID,
		Period:          req.Period,
		BaseSalaryCents: comp.BaseSalaryCents,
		FinesCents:      finesCents,
		DebtsCents:      debtsCents,
		NetSalaryCents:  netSalary,
	}

	if err := s.repo.Create(ctx, &run); err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	return run, nil
}

// ── pay ─────────────────────────────────────────────────────────────────────

func (s *Service) Pay(ctx context.Context, payrollID int64, paymentTypeCode string, userID int64) (PayrollRun, error) {
	// Resolve payment type
	ptID, err := s.finRepo.GetPaymentTypeIDByCode(ctx, paymentTypeCode)
	if err != nil {
		if isNotFound(err) {
			return PayrollRun{}, apperr.Validation("invalid payment_type_code")
		}
		return PayrollRun{}, apperr.Internal(err)
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}
	defer tx.Rollback(ctx)

	// Lock payroll run
	run, err := s.repo.LockPayrollRun(ctx, tx, payrollID)
	if err != nil {
		if isNotFound(err) {
			return PayrollRun{}, apperr.NotFound("PAYROLL_NOT_FOUND", "payroll run not found")
		}
		return PayrollRun{}, apperr.Internal(err)
	}

	if run.Status != "calculated" {
		return PayrollRun{}, apperr.Validation(
			fmt.Sprintf("payroll run is '%s', expected 'calculated'", run.Status),
		)
	}

	uid := &userID

	// Create expense transaction
	reason := fmt.Sprintf("Salary for %s", run.Period)
	finTxn := finance.Transaction{
		Type:          "expense",
		AmountCents:   run.NetSalaryCents,
		RelatedTable:  "payroll",
		RelatedID:     &run.ID,
		Status:        "paid",
		Reason:        &reason,
		CreatedBy:     uid,
		PaymentTypeID: &ptID,
	}
	if err := s.finRepo.CreateTransaction(ctx, tx, &finTxn); err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	// Create payment record (full amount)
	if run.NetSalaryCents > 0 {
		finPay := finance.Payment{
			TransactionID: finTxn.ID,
			PaymentTypeID: ptID,
			AmountCents:   run.NetSalaryCents,
			CreatedBy:     uid,
		}
		if err := s.finRepo.CreatePayment(ctx, tx, &finPay); err != nil {
			return PayrollRun{}, apperr.Internal(err)
		}
	}

	// Update payroll run
	run, err = s.repo.UpdateStatusPaid(ctx, tx, payrollID, finTxn.ID)
	if err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	// Mark fines as deducted
	if err := s.wfRepo.MarkFinesDeducted(ctx, tx, run.WorkerID); err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	// Settle debts
	if err := s.wfRepo.SettleDebts(ctx, tx, run.WorkerID); err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	return run, nil
}

// ── list ────────────────────────────────────────────────────────────────────

func (s *Service) List(ctx context.Context, workerID *int64, period *string, limit, offset int) (PayrollListResult, error) {
	items, total, err := s.repo.List(ctx, workerID, period, limit, offset)
	if err != nil {
		return PayrollListResult{}, apperr.Internal(err)
	}
	if items == nil {
		items = []PayrollRun{}
	}
	return PayrollListResult{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *Service) Get(ctx context.Context, id int64) (PayrollRun, error) {
	run, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return PayrollRun{}, apperr.NotFound("PAYROLL_NOT_FOUND", "payroll run not found")
		}
		return PayrollRun{}, apperr.Internal(err)
	}
	return run, nil
}
