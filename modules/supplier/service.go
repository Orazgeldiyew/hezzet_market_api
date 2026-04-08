package supplier

import (
	"context"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Supplier, error) {
	sup := Supplier{
		UserID:   req.UserID,
		Name:     req.Name,
		Phone:    req.Phone,
		Email:    req.Email,
		Address:  req.Address,
		IsActive: true,
	}

	if err := s.repo.Create(ctx, &sup); err != nil {
		return Supplier{}, apperr.Internal(err)
	}
	return sup, nil
}

func (s *Service) Get(ctx context.Context, id int) (Supplier, error) {
	sup, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if IsNotFound(err) {
			return Supplier{}, apperr.NotFound("NOT_FOUND", "supplier not found")
		}
		return Supplier{}, apperr.Internal(err)
	}
	return sup, nil
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
		items = []Supplier{}
	}
	return ListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Service) Update(ctx context.Context, id int, req UpdateRequest) (Supplier, error) {
	sup, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if IsNotFound(err) {
			return Supplier{}, apperr.NotFound("NOT_FOUND", "supplier not found")
		}
		return Supplier{}, apperr.Internal(err)
	}
	return sup, nil
}

func (s *Service) Delete(ctx context.Context, id int) error {
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		if IsNotFound(err) {
			return apperr.NotFound("NOT_FOUND", "supplier not found")
		}
		return apperr.Internal(err)
	}
	return nil
}
