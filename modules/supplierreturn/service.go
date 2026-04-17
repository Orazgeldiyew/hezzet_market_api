package supplierreturn

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func isAppError(err error) bool {
	_, ok := err.(*apperr.AppError)
	return ok
}

func (s *Service) Create(ctx context.Context, req CreateRequest, userID int64) (ReturnDetail, error) {
	// Validate no duplicate product IDs
	seen := make(map[int64]bool, len(req.Items))
	for _, item := range req.Items {
		if seen[item.ProductID] {
			return ReturnDetail{}, apperr.Validation("duplicate product_id in items")
		}
		seen[item.ProductID] = true
	}

	ret, items, err := s.repo.Create(ctx, req, userID)
	if err != nil {
		if isAppError(err) {
			return ReturnDetail{}, err
		}
		return ReturnDetail{}, apperr.Internal(err)
	}
	return ReturnDetail{SupplierReturn: ret, Items: items}, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (ReturnDetail, error) {
	ret, items, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ReturnDetail{}, apperr.NotFound("RETURN_NOT_FOUND", "supplier return not found")
		}
		return ReturnDetail{}, apperr.Internal(err)
	}
	return ReturnDetail{SupplierReturn: ret, Items: items}, nil
}

func (s *Service) List(
	ctx context.Context,
	supplierID, warehouseID *int64,
	status *string,
	dateFrom, dateTo *time.Time,
	limit, offset int,
) ([]ReturnListItem, int, error) {
	out, total, err := s.repo.List(ctx, supplierID, warehouseID, status, dateFrom, dateTo, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return out, total, nil
}

func (s *Service) Confirm(ctx context.Context, returnID int64, userID int64) (ReturnDetail, error) {
	_, err := s.repo.Confirm(ctx, returnID, userID)
	if err != nil {
		if isAppError(err) {
			return ReturnDetail{}, err
		}
		return ReturnDetail{}, apperr.Internal(err)
	}
	return s.GetByID(ctx, returnID)
}

func (s *Service) Cancel(ctx context.Context, returnID int64) error {
	err := s.repo.Cancel(ctx, returnID)
	if err != nil {
		if isAppError(err) {
			return err
		}
		return apperr.Internal(err)
	}
	return nil
}
