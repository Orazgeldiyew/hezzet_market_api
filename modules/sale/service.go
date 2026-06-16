package sale

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

// PrinterService is an optional hook for auto-printing receipts on confirm.
type PrinterService interface {
	PrintSale(ctx context.Context, saleID int64, registerID *int64) error
}

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
	if req.BonusUsedCents != nil && *req.BonusUsedCents < 0 {
		return SaleDetail{}, apperr.Validation("bonus_used_cents must be non-negative")
	}

	_, err := s.repo.ConfirmSale(ctx, saleID, req, userID, s.finRepo)
	if err != nil {
		log.Printf("[ConfirmSale] error saleID=%d: %v", saleID, err)
		if isAppError(err) {
			return SaleDetail{}, err
		}
		return SaleDetail{}, apperr.Internal(err)
	}

	// Fetch full detail (items + txn ID). The internal call is privileged —
	// we already authorized this user to confirm the sale, returning the
	// detail to them is consistent.
	detail, err := s.GetSale(ctx, saleID, userID, true)
	if err != nil {
		return detail, err
	}

	// Auto-print receipt to thermal printer via TCP (fire-and-forget — printer errors must not break the sale)
	if printerSv := s.repo.PrinterService(); printerSv != nil {
		log.Printf("[AutoPrint] START saleID=%d userID=%d", saleID, userID)

		// Get register_id from user's current open shift
		var regID *int64
		var rid int64
		err := s.repo.DB().QueryRow(ctx, `
			SELECT register_id FROM shifts
			WHERE user_id = $1 AND status = 'open'
			ORDER BY opened_at DESC LIMIT 1
		`, userID).Scan(&rid)
		if err != nil {
			log.Printf("[AutoPrint] no open shift for userID=%d: %v", userID, err)
		} else {
			regID = &rid
			log.Printf("[AutoPrint] found register_id=%d", rid)
		}

		// Pass saleID/regID by value so the goroutine doesn't depend on the
		// enclosing function's locals. context.Background() is intentional —
		// the request ctx is cancelled the moment we return the response to
		// the cashier, but the printer call must still complete after that.
		// 10s timeout keeps a stuck printer from leaking goroutines.
		go func(saleID int64, regID *int64, printerSv PrinterService) {
			printCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if perr := printerSv.PrintSale(printCtx, saleID, regID); perr != nil {
				log.Printf("[AutoPrint] FAILED saleID=%d: %v", saleID, perr)
			} else {
				log.Printf("[AutoPrint] OK saleID=%d", saleID)
			}
		}(saleID, regID, printerSv)
	} else {
		log.Printf("[AutoPrint] printer service is nil — skipping")
	}

	return detail, nil
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

func (s *Service) TransferDraft(ctx context.Context, saleID, currentUserID, newCashierID int64, callerRoles []string) error {
	err := s.repo.TransferDraft(ctx, saleID, currentUserID, newCashierID, callerRoles)
	if err != nil {
		if isAppError(err) {
			return err
		}
		return apperr.Internal(err)
	}
	return nil
}

// GetSale loads a sale with ownership enforcement: a non-privileged caller
// (cashier) can only fetch sales they themselves created. Privileged callers
// (admin/manager/operator) see everything for reporting.
func (s *Service) GetSale(ctx context.Context, id, callerID int64, privileged bool) (SaleDetail, error) {
	sale, items, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return SaleDetail{}, apperr.NotFound("SALE_NOT_FOUND", "sale not found")
		}
		return SaleDetail{}, apperr.Internal(err)
	}

	if !privileged && sale.CreatedBy != nil && *sale.CreatedBy != callerID {
		// Return a generic NotFound instead of Forbidden so a cashier can't
		// confirm that an arbitrary ID exists by getting back a different code.
		return SaleDetail{}, apperr.NotFound("SALE_NOT_FOUND", "sale not found")
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

func (s *Service) DeleteSaleItem(ctx context.Context, saleID, itemID int64) (SaleDetail, error) {
	sale, items, err := s.repo.DeleteSaleItem(ctx, saleID, itemID)
	if err != nil {
		if isAppError(err) {
			return SaleDetail{}, err
		}
		return SaleDetail{}, apperr.Internal(err)
	}
	return SaleDetail{Sale: sale, Items: items}, nil
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

func (s *Service) ReturnSale(ctx context.Context, saleID int64, req ReturnSaleRequest, userID int64) error {
	if err := s.repo.ReturnSale(ctx, saleID, req, userID); err != nil {
		if isAppError(err) {
			return err
		}
		return apperr.Internal(err)
	}
	return nil
}
func (s *Service) DecreaseItem(ctx context.Context, saleID, itemID int64) (SaleDetail, error) {
	sale, items, err := s.repo.DecreaseDraftSaleItemQty(ctx, saleID, itemID)
	if err != nil {
		if isAppError(err) {
			return SaleDetail{}, err
		}
		return SaleDetail{}, apperr.Internal(err)
	}
	return SaleDetail{Sale: sale, Items: items}, nil
}