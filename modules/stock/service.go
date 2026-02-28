package stock

import (
	"context"
	"time"

	"github.com/Orazgeldiyew/hezzet_market_backend/modules/notification"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Service struct {
	repo     *Repository
	notifSvc *notification.Service // nil-safe
}

func NewService(repo *Repository, notifSvc *notification.Service) *Service {
	return &Service{repo: repo, notifSvc: notifSvc}
}

func (s *Service) StockIn(ctx context.Context, req InRequest, userID int64) (MovementResult, error) {
	d, it, err := s.repo.StockIn(ctx, req, userID)
	if err != nil {
		if isAppError(err) {
			return MovementResult{}, err
		}
		return MovementResult{}, apperr.Internal(err)
	}
	return MovementResult{Detail: d, Item: it}, nil
}

func (s *Service) BulkStockIn(ctx context.Context, req BulkInRequest, userID int64) (BulkInResult, error) {
	results, err := s.repo.BulkStockIn(ctx, req.WarehouseID, req.Items, userID)
	if err != nil {
		if isAppError(err) {
			return BulkInResult{}, err
		}
		return BulkInResult{}, apperr.Internal(err)
	}
	return BulkInResult{Results: results}, nil
}

func (s *Service) StockOut(ctx context.Context, req OutRequest, userID int64) (MovementResult, error) {
	d, it, err := s.repo.StockOut(ctx, req, userID)
	if err != nil {
		if isAppError(err) {
			return MovementResult{}, err
		}
		return MovementResult{}, apperr.Internal(err)
	}

	// Async low-stock check (non-blocking, best-effort)
	go s.notifSvc.NotifyAdminLowStock(context.Background(), req.ProductID, req.WarehouseID, it.QtyMilli)

	return MovementResult{Detail: d, Item: it}, nil
}

func (s *Service) Transfer(ctx context.Context, req TransferRequest, userID int64) (TransferResult, error) {
	if req.FromWarehouseID == req.ToWarehouseID {
		return TransferResult{}, apperr.Validation("source and destination warehouses must differ")
	}
	res, err := s.repo.Transfer(ctx, req, userID)
	if err != nil {
		if isAppError(err) {
			return TransferResult{}, err
		}
		return TransferResult{}, apperr.Internal(err)
	}

	// Async low-stock check on source warehouse
	go s.notifSvc.NotifyAdminLowStock(context.Background(), req.ProductID, req.FromWarehouseID, res.FromItem.QtyMilli)

	return res, nil
}

func (s *Service) Move(ctx context.Context, req MoveRequest, userID int64) (MovementResult, error) {
	// Business rules for return_to_supplier
	if normalizeType(req.Type) == "return_to_supplier" {
		if req.DeltaMilli >= 0 {
			return MovementResult{}, apperr.Validation("delta_milli must be negative for return_to_supplier")
		}
		if req.SupplierID == nil {
			return MovementResult{}, apperr.Validation("supplier_id is required for return_to_supplier")
		}
	}

	d, it, err := s.repo.Move(ctx, req, userID)
	if err != nil {
		if isAppError(err) {
			return MovementResult{}, err
		}
		return MovementResult{}, apperr.Internal(err)
	}

	// Async low-stock check only for stock-decreasing moves
	if req.DeltaMilli < 0 {
		go s.notifSvc.NotifyAdminLowStock(context.Background(), req.ProductID, req.WarehouseID, it.QtyMilli)
	}

	return MovementResult{Detail: d, Item: it}, nil
}

func (s *Service) GetItems(ctx context.Context, warehouseID, productID *int64) ([]WarehouseItem, error) {
	items, err := s.repo.GetItems(ctx, warehouseID, productID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if items == nil {
		items = []WarehouseItem{}
	}
	return items, nil
}

func (s *Service) GetDetails(
	ctx context.Context,
	warehouseID, productID *int64,
	mType *string,
	supplierID *int64,
	dateFrom, dateTo *time.Time,
	limit, offset int,
) (DetailsListResult, error) {

	items, total, err := s.repo.GetDetails(ctx, warehouseID, productID, mType, supplierID, dateFrom, dateTo, limit, offset)
	if err != nil {
		if isAppError(err) {
			return DetailsListResult{}, err
		}
		return DetailsListResult{}, apperr.Internal(err)
	}
	if items == nil {
		items = []WarehouseItemDetail{}
	}
	return DetailsListResult{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func isAppError(err error) bool {
	_, ok := err.(*apperr.AppError)
	return ok
}
func (s *Service) OpeningBalance(ctx context.Context, req OpeningBalanceRequest, userID int64) (MovementResult, error) {
	d, it, err := s.repo.OpeningBalance(ctx, req, userID)
	if err != nil {
		if isAppError(err) {
			return MovementResult{}, err
		}
		return MovementResult{}, apperr.Internal(err)
	}
	return MovementResult{Detail: d, Item: it}, nil
}

func (s *Service) GetNegativeItems(ctx context.Context) ([]NegativeStockRow, error) {
	out, err := s.repo.GetNegativeItems(ctx)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}
