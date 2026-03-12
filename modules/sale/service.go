package sale

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

func (s *Service) CreateSale(ctx context.Context, req CreateSaleRequest, userID int64) (SaleDetail, error) {
	// Validate: no duplicate product IDs in items
	seen := make(map[int64]bool, len(req.Items))
	for _, item := range req.Items {
		if seen[item.ProductID] {
			return SaleDetail{}, apperr.Validation("duplicate product_id in items")
		}
		seen[item.ProductID] = true
	}

	sale, items, err := s.repo.CreateSale(ctx, req, userID)
	if err != nil {
		if isAppError(err) {
			return SaleDetail{}, err
		}
		return SaleDetail{}, apperr.Internal(err)
	}

	return SaleDetail{
		Sale:  sale,
		Items: items,
		// TransactionID = 0 — no finance transaction until confirmed
	}, nil
}

func (s *Service) ConfirmSale(ctx context.Context, saleID int64, req ConfirmSaleRequest, userID int64) (SaleDetail, error) {
	if req.PaymentAmount != nil {
		if req.PaymentTypeID == nil {
			return SaleDetail{}, apperr.Validation("payment_type_id is required when payment_amount is provided")
		}
		if *req.PaymentAmount <= 0 {
			return SaleDetail{}, apperr.Validation("payment_amount must be positive")
		}
	}

	_, err := s.repo.ConfirmSale(ctx, saleID, req, userID, s.finRepo)
	if err != nil {
		if isAppError(err) {
			return SaleDetail{}, err
		}
		return SaleDetail{}, apperr.Internal(err)
	}

	// Fetch full detail (items + txn ID)
	return s.GetSale(ctx, saleID)
}

func (s *Service) CancelSale(ctx context.Context, saleID int64, userID int64) error {
	err := s.repo.CancelSale(ctx, saleID, userID)
	if err != nil {
		if isAppError(err) {
			return err
		}
		return apperr.Internal(err)
	}
	return nil
}

func (s *Service) GetSale(ctx context.Context, id int64) (SaleDetail, error) {
	sale, items, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return SaleDetail{}, apperr.NotFound("SALE_NOT_FOUND", "sale not found")
		}
		return SaleDetail{}, apperr.Internal(err)
	}

	txnID, err := s.repo.GetTransactionIDBySaleID(ctx, id)
	if err != nil && err != pgx.ErrNoRows {
		return SaleDetail{}, apperr.Internal(err)
	}

	return SaleDetail{
		Sale:          sale,
		Items:         items,
		TransactionID: txnID,
	}, nil
}

func (s *Service) ListSales(
	ctx context.Context,
	warehouseID, customerID, createdBy *int64,
	status *string,
	dateFrom, dateTo *time.Time,
	limit, offset int,
) (SaleListResult, error) {
	items, total, err := s.repo.List(ctx, warehouseID, customerID, createdBy, status, dateFrom, dateTo, limit, offset)
	if err != nil {
		return SaleListResult{}, apperr.Internal(err)
	}
	if items == nil {
		items = []SaleListItem{}
	}
	return SaleListResult{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}
