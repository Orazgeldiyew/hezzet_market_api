package warehouse

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, req CreateRequest) (Warehouse, error) {
	w := Warehouse{
		Name:    req.Name,
		Address: req.Address,
	}
	if err := s.repo.Create(ctx, &w); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Warehouse{}, &apperr.AppError{
				Code:       "WAREHOUSE_ALREADY_EXISTS",
				Message:    "warehouse with this name already exists",
				HTTPStatus: http.StatusConflict,
				Err:        err,
			}
		}
		return Warehouse{}, apperr.Internal(err)
	}
	return w, nil
}

func (s *Service) List(ctx context.Context, limit, offset int) (ListResponse, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	items, total, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return ListResponse{}, apperr.Internal(err)
	}
	if items == nil {
		items = []Warehouse{}
	}
	return ListResponse{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Warehouse, error) {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return Warehouse{}, apperr.NotFound("WAREHOUSE_NOT_FOUND", "warehouse not found")
		}
		return Warehouse{}, apperr.Internal(err)
	}
	return w, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateRequest) (Warehouse, error) {
	w, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if isNotFound(err) {
			return Warehouse{}, apperr.NotFound("WAREHOUSE_NOT_FOUND", "warehouse not found")
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Warehouse{}, &apperr.AppError{
				Code:       "WAREHOUSE_ALREADY_EXISTS",
				Message:    "warehouse with this name already exists",
				HTTPStatus: http.StatusConflict,
				Err:        err,
			}
		}
		return Warehouse{}, apperr.Internal(err)
	}
	return w, nil
}

// Delete soft-deletes the warehouse. Blocked when the warehouse still holds
// stock or has active reservations — the caller must move inventory out first.
func (s *Service) Delete(ctx context.Context, id int64) error {
	hasInv, err := s.repo.HasInventory(ctx, id)
	if err != nil {
		return apperr.Internal(err)
	}
	if hasInv {
		return &apperr.AppError{
			Code:       "WAREHOUSE_NOT_EMPTY",
			Message:    "warehouse has stock or active reservations — move them out before deleting",
			HTTPStatus: http.StatusConflict,
		}
	}
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		if isNotFound(err) {
			return apperr.NotFound("WAREHOUSE_NOT_FOUND", "warehouse not found")
		}
		return apperr.Internal(err)
	}
	return nil
}
