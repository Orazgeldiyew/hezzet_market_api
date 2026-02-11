package product

import (
	"context"
	"errors"
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

	// link categories if provided
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

	// update categories if provided
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
