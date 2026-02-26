package product

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Service struct {
	repo  *Repository
	stock *StockRepository
}

func NewService(repo *Repository, stock *StockRepository) *Service {
	return &Service{repo: repo, stock: stock}
}

// ValidateUnit enforces the business rule:
//
//	piece  → unit must be "piece"
//	weight → unit must be "kg" or "g"
//	volume → unit must be "l"  or "ml"
func ValidateUnit(unitType UnitType, unit Unit) error {
	switch unitType {
	case UnitTypePiece:
		if unit != UnitPiece {
			return fmt.Errorf("unit_type 'piece' requires unit 'piece', got '%s'", unit)
		}
	case UnitTypeWeight:
		if unit != UnitKg && unit != UnitG {
			return fmt.Errorf("unit_type 'weight' requires unit 'kg' or 'g', got '%s'", unit)
		}
	case UnitTypeVolume:
		if unit != UnitL && unit != UnitML {
			return fmt.Errorf("unit_type 'volume' requires unit 'l' or 'ml', got '%s'", unit)
		}
	default:
		return fmt.Errorf("unknown unit_type '%s'", unitType)
	}
	return nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Product, error) {
	// Enforce unit_type ↔ unit compatibility before touching the DB.
	if err := ValidateUnit(req.UnitType, req.Unit); err != nil {
		return Product{}, apperr.Validation(err.Error())
	}

	barcodes := req.Barcodes
	if barcodes == nil {
		barcodes = []string{}
	}

	p := Product{
		Name:          req.Name,
		SKU:           req.SKU,
		Barcodes:      barcodes,
		UnitType:      req.UnitType,
		Unit:          req.Unit,
		PurchasePrice: req.PurchasePrice,
		SalePrice:     req.SalePrice,
		IsActive:      true,
	}
	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	}

	if err := s.repo.Create(ctx, &p); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Product{}, &apperr.AppError{
				Code:       "PRODUCT_ALREADY_EXISTS",
				Message:    "product with this sku or barcode already exists",
				HTTPStatus: http.StatusConflict,
				Err:        err,
			}
		}
		return Product{}, apperr.Internal(err)
	}

	// Link categories if provided.
	if len(req.CategoryIDs) > 0 {
		uniq := uniqueInt64(req.CategoryIDs)
		n, err := s.repo.CountActiveCategoriesByIDs(ctx, uniq)
		if err != nil {
			return p, apperr.Internal(err)
		}
		if n != len(uniq) {
			return p, apperr.Validation("some categories not found or inactive")
		}
		if err := s.repo.ReplaceCategories(ctx, p.ID, uniq); err != nil {
			return p, apperr.Internal(err)
		}
	}

	return p, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Product, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Product{}, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return Product{}, apperr.Internal(err)
	}
	return p, nil
}

func (s *Service) GetCard(ctx context.Context, id int64) (Card, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Card{}, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return Card{}, apperr.Internal(err)
	}

	stock, err := s.stock.GetStock(ctx, id)
	if err != nil {
		return Card{}, apperr.Internal(err)
	}

	cats, err := s.repo.ListCategories(ctx, id)
	if err != nil {
		return Card{}, apperr.Internal(err)
	}

	return Card{
		Product:    p,
		Stock:      stock,
		Tags:       []string{},
		Suppliers:  []int64{},
		Categories: cats,
	}, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateRequest) (Product, error) {
	// When unit_type or unit is being changed we must validate the combined
	// (effective) state: either field may be absent in the request, so we load
	// the current product and merge before validating.
	if req.UnitType != nil || req.Unit != nil {
		current, err := s.repo.GetByID(ctx, id)
		if err != nil {
			if err == pgx.ErrNoRows {
				return Product{}, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
			}
			return Product{}, apperr.Internal(err)
		}

		effectiveUnitType := current.UnitType
		if req.UnitType != nil {
			effectiveUnitType = *req.UnitType
		}

		effectiveUnit := current.Unit
		if req.Unit != nil {
			effectiveUnit = *req.Unit
		}

		if err := ValidateUnit(effectiveUnitType, effectiveUnit); err != nil {
			return Product{}, apperr.Validation(err.Error())
		}
	}

	p, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Product{}, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Product{}, &apperr.AppError{
				Code:       "PRODUCT_ALREADY_EXISTS",
				Message:    "product with this sku or barcode already exists",
				HTTPStatus: http.StatusConflict,
				Err:        err,
			}
		}
		return Product{}, apperr.Internal(err)
	}

	// Update categories if provided.
	if req.CategoryIDs != nil {
		uniq := uniqueInt64(*req.CategoryIDs)
		if len(uniq) > 0 {
			n, err := s.repo.CountActiveCategoriesByIDs(ctx, uniq)
			if err != nil {
				return p, apperr.Internal(err)
			}
			if n != len(uniq) {
				return p, apperr.Validation("some categories not found or inactive")
			}
		}
		if err := s.repo.ReplaceCategories(ctx, id, uniq); err != nil {
			return p, apperr.Internal(err)
		}
	}

	return p, nil
}

func (s *Service) List(ctx context.Context, limit, offset int, orderBy, orderDir, q string) (ListResponse, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	if orderBy == "" {
		orderBy = "created_at"
	}
	if orderDir == "" {
		orderDir = "desc"
	}

	items, total, err := s.repo.List(ctx, limit, offset, orderBy, orderDir, q)
	if err != nil {
		return ListResponse{}, apperr.Internal(err)
	}
	if items == nil {
		items = []Product{}
	}

	return ListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		if IsNoRows(err) {
			return apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return apperr.Internal(err)
	}
	return nil
}

// ─── Category helpers ────────────────────────────────────────────────────────

func (s *Service) GetCategories(ctx context.Context, productID int64) ([]CategoryBrief, error) {
	if _, err := s.repo.GetByID(ctx, productID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return nil, apperr.Internal(err)
	}
	out, err := s.repo.ListCategories(ctx, productID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}

func (s *Service) SetCategories(ctx context.Context, productID int64, ids []int64) ([]CategoryBrief, error) {
	if _, err := s.repo.GetByID(ctx, productID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return nil, apperr.Internal(err)
	}

	uniq := uniqueInt64(ids)
	if len(uniq) > 0 {
		n, err := s.repo.CountActiveCategoriesByIDs(ctx, uniq)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		if n != len(uniq) {
			return nil, apperr.Validation("some categories not found or inactive")
		}
	}

	if err := s.repo.ReplaceCategories(ctx, productID, uniq); err != nil {
		return nil, apperr.Internal(err)
	}

	out, err := s.repo.ListCategories(ctx, productID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}

func (s *Service) RemoveCategory(ctx context.Context, productID, categoryID int64) error {
	if _, err := s.repo.GetByID(ctx, productID); err != nil {
		if err == pgx.ErrNoRows {
			return apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return apperr.Internal(err)
	}

	ok, err := s.repo.RemoveCategory(ctx, productID, categoryID)
	if err != nil {
		return apperr.Internal(err)
	}
	if !ok {
		return apperr.NotFound("NOT_FOUND", "link not found")
	}
	return nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func uniqueInt64(in []int64) []int64 {
	seen := make(map[int64]struct{}, len(in))
	out := make([]int64, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
