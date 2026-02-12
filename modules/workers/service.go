package workers

import (
	"context"
	"time"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, req CreateRequest) (Worker, error) {
	w := Worker{
		Name:       req.Name,
		Position:   req.Position,
		Department: req.Department,
		Phone:      req.Phone,
		Email:      req.Email,
		Address:    req.Address,
		Notes:      req.Notes,
		IsActive:   true,
	}

	if req.Salary != nil {
		w.Salary = *req.Salary
	}

	if req.HireDate != nil && *req.HireDate != "" {
		t, err := time.Parse("2006-01-02", *req.HireDate)
		if err != nil {
			return Worker{}, apperr.Validation("hire_date must be in YYYY-MM-DD format")
		}
		w.HireDate = &t
	}

	if err := s.repo.Create(ctx, &w); err != nil {
		return Worker{}, apperr.Internal(err)
	}
	return w, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Worker, error) {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return Worker{}, apperr.NotFound("NOT_FOUND", "worker not found")
		}
		return Worker{}, apperr.Internal(err)
	}
	return w, nil
}

func (s *Service) List(ctx context.Context, limit, offset int, orderBy, orderDir, search string, activeOnly bool) (ListResponse, error) {
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

	items, total, err := s.repo.List(ctx, limit, offset, orderBy, orderDir, search, activeOnly)
	if err != nil {
		return ListResponse{}, apperr.Internal(err)
	}
	if items == nil {
		items = []Worker{}
	}

	return ListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateRequest) (Worker, error) {
	if req.HireDate != nil && *req.HireDate != "" {
		if _, err := time.Parse("2006-01-02", *req.HireDate); err != nil {
			return Worker{}, apperr.Validation("hire_date must be in YYYY-MM-DD format")
		}
	}

	w, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if isNotFound(err) {
			return Worker{}, apperr.NotFound("NOT_FOUND", "worker not found")
		}
		return Worker{}, apperr.Internal(err)
	}
	return w, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		if isNotFound(err) {
			return apperr.NotFound("NOT_FOUND", "worker not found")
		}
		return apperr.Internal(err)
	}
	return nil
}
