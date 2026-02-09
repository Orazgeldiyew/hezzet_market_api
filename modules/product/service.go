package product

import (
	"context"

	"github.com/jackc/pgx/v5"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Service struct {
	repo  *Repository
	stock *StockRepository
}

func NewService(repo *Repository, stock *StockRepository) *Service {
	return &Service{repo: repo, stock: stock}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Product, error) {
	p := Product{
		Name:          req.Name,
		SKU:           req.SKU,
		Barcode:       req.Barcode,
		Unit:          req.Unit,
		PurchasePrice: req.PurchasePrice,
		SalePrice:     req.SalePrice,
		IsActive:      true,
	}
	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	}
	if err := s.repo.Create(ctx, &p); err != nil {
		return Product{}, apperr.Internal(err)
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
	p, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Product{}, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return Product{}, apperr.Internal(err)
	}
	return p, nil
}

func (s *Service) List(ctx context.Context, limit, offset int, q string) ([]Product, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	out, err := s.repo.List(ctx, limit, offset, q)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}

//
// ===== Product ↔ Categories endpoints support =====
//

func (s *Service) GetCategories(ctx context.Context, productID int64) ([]CategoryBrief, error) {
	// product must exist
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
	// product must exist
	if _, err := s.repo.GetByID(ctx, productID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return nil, apperr.Internal(err)
	}

	uniq := uniqueInt64(ids)

	// allow empty => clear all
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
	// product must exist
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
