package purchase

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Service struct {
	repo    *Repository
	finRepo *finance.Repository
}

func NewService(repo *Repository, finRepo *finance.Repository) *Service {
	return &Service{repo: repo, finRepo: finRepo}
}

func isAppError(err error) bool {
	_, ok := err.(*apperr.AppError)
	return ok
}

func (s *Service) CreatePO(ctx context.Context, req CreatePORequest, userID int64) (PODetail, error) {
	// Validate no duplicate product IDs
	seen := make(map[int64]bool, len(req.Items))
	for _, item := range req.Items {
		if seen[item.ProductID] {
			return PODetail{}, apperr.Validation("duplicate product_id in items")
		}
		seen[item.ProductID] = true
	}

	po, items, err := s.repo.CreatePO(ctx, req, userID)
	if err != nil {
		if isAppError(err) {
			return PODetail{}, err
		}
		return PODetail{}, apperr.Internal(err)
	}
	return PODetail{PurchaseOrder: po, Items: items}, nil
}

func (s *Service) GetPO(ctx context.Context, id int64) (PODetail, error) {
	po, items, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return PODetail{}, apperr.NotFound("PO_NOT_FOUND", "purchase order not found")
		}
		return PODetail{}, apperr.Internal(err)
	}

	txnID, err := s.repo.GetTransactionID(ctx, id)
	if err != nil && err != pgx.ErrNoRows {
		return PODetail{}, apperr.Internal(err)
	}

	return PODetail{PurchaseOrder: po, Items: items, TransactionID: txnID}, nil
}

func (s *Service) ListPOs(
	ctx context.Context,
	supplierID, warehouseID *int64,
	status *string,
	dateFrom, dateTo *time.Time,
	limit, offset int,
) ([]POListItem, int, error) {
	out, total, err := s.repo.List(ctx, supplierID, warehouseID, status, dateFrom, dateTo, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	if out == nil {
		out = []POListItem{}
	}
	return out, total, nil
}

func (s *Service) ReceivePO(ctx context.Context, poID int64, userID int64) (PODetail, error) {
	_, err := s.repo.ReceivePO(ctx, poID, userID, s.finRepo)
	if err != nil {
		if isAppError(err) {
			return PODetail{}, err
		}
		return PODetail{}, apperr.Internal(err)
	}
	// Fetch full detail after commit
	return s.GetPO(ctx, poID)
}

func (s *Service) CancelPO(ctx context.Context, poID int64) error {
	err := s.repo.CancelPO(ctx, poID)
	if err != nil {
		if isAppError(err) {
			return err
		}
		return apperr.Internal(err)
	}
	return nil
}

func (s *Service) AddPayment(ctx context.Context, poID int64, req AddPaymentRequest, userID int64) (finance.Payment, error) {
	p, err := s.repo.AddPayment(ctx, poID, req, userID, s.finRepo)
	if err != nil {
		if isAppError(err) {
			return finance.Payment{}, err
		}
		return finance.Payment{}, apperr.Internal(err)
	}
	return p, nil
}

func (s *Service) DebtSummary(ctx context.Context) ([]SupplierDebtRow, error) {
	out, err := s.repo.DebtSummary(ctx)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}
