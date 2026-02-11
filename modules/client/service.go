package client

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

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Client, error) {
	c := Client{
		Name:     req.Name,
		Phone:    req.Phone,
		Email:    req.Email,
		IsActive: true,
	}

	if err := s.repo.Create(ctx, &c); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Client{}, &apperr.AppError{
				Code:       "PHONE_ALREADY_EXISTS",
				Message:    "phone number already exists",
				HTTPStatus: http.StatusConflict,
				Err:        err,
			}
		}
		return Client{}, apperr.Internal(err)
	}
	return c, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Client, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if IsNotFound(err) {
			return Client{}, apperr.NotFound("NOT_FOUND", "client not found")
		}
		return Client{}, apperr.Internal(err)
	}
	// Return 404 if inactive
	if !c.IsActive {
		return Client{}, apperr.NotFound("NOT_FOUND", "client not found")
	}
	return c, nil
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
		items = []Client{}
	}
	return ListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateRequest) (Client, error) {
	c, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if IsNotFound(err) {
			return Client{}, apperr.NotFound("NOT_FOUND", "client not found")
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Client{}, &apperr.AppError{
				Code:       "PHONE_ALREADY_EXISTS",
				Message:    "phone number already exists",
				HTTPStatus: http.StatusConflict,
				Err:        err,
			}
		}
		return Client{}, apperr.Internal(err)
	}
	return c, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		if IsNotFound(err) {
			return apperr.NotFound("NOT_FOUND", "client not found")
		}
		return apperr.Internal(err)
	}
	return nil
}
