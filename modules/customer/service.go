// modules/customer/service.go
package customer

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

func (s *Service) Create(ctx context.Context, req CreateRequest) (Customer, error) {
	c := Customer{
		Name:     req.Name,
		Phone:    req.Phone,
		Email:    req.Email,
		Type:     "regular",
		Notes:    req.Notes,
		IsActive: true,
	}
	if req.Type != nil {
		c.Type = *req.Type
	}

	if err := s.repo.Create(ctx, &c); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Customer{}, &apperr.AppError{
				Code:       "CUSTOMER_ALREADY_EXISTS",
				Message:    "customer already exists",
				HTTPStatus: http.StatusConflict,
				Err:        err,
			}
		}
		return Customer{}, apperr.Internal(err)
	}
	return c, nil
}

// Variant A: return object even if is_active=false; 404 only if deleted.
func (s *Service) Get(ctx context.Context, id int64) (Customer, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if IsNotFound(err) {
			return Customer{}, apperr.NotFound("NOT_FOUND", "customer not found")
		}
		return Customer{}, apperr.Internal(err)
	}
	return c, nil
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
		items = []Customer{}
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
		if IsNotFound(err) {
			return apperr.NotFound("NOT_FOUND", "customer not found")
		}
		return apperr.Internal(err)
	}
	return nil
}

func (s *Service) AddSpent(ctx context.Context, customerID int64, amountCents int64) (Customer, error) {
	if amountCents <= 0 {
		return Customer{}, apperr.Validation("amount must be positive")
	}
	bonusCents := amountCents / 100 // 1% bonus (integer division, no float math)

	c, err := s.repo.AddSpent(ctx, customerID, amountCents, bonusCents)
	if err != nil {
		if IsNotFound(err) {
			return Customer{}, apperr.NotFound("NOT_FOUND", "customer not found")
		}
		return Customer{}, apperr.Internal(err)
	}
	return c, nil
}
func (s *Service) UpdateContact(ctx context.Context, id int64, req UpdateContactRequest) (Customer, error) {
	c, err := s.repo.UpdateContact(ctx, id, req)
	if err != nil {
		if IsNotFound(err) {
			return Customer{}, apperr.NotFound("NOT_FOUND", "customer not found")
		}
		return Customer{}, apperr.Internal(err)
	}
	return c, nil
}
func (s *Service) UpdateAdmin(ctx context.Context, id int64, req UpdateAdminRequest) (Customer, error) {
	c, err := s.repo.UpdateAdmin(ctx, id, req)
	if err != nil {
		if IsNotFound(err) {
			return Customer{}, apperr.NotFound("NOT_FOUND", "customer not found")
		}
		return Customer{}, apperr.Internal(err)
	}
	return c, nil
}
