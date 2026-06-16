package employees

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, f ListFilter, limit, offset int) (ListResponse, error) {
	items, total, err := s.repo.List(ctx, f, limit, offset)
	if err != nil {
		return ListResponse{}, apperr.Internal(err)
	}
	if items == nil {
		items = []Employee{}
	}
	return ListResponse{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Employee, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return Employee{}, apperr.NotFound("EMPLOYEE_NOT_FOUND", "employee not found")
		}
		return Employee{}, apperr.Internal(err)
	}
	return e, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest, createdBy *int64) (Employee, error) {
	if req.HasAccount {
		if req.Username == nil || *req.Username == "" {
			return Employee{}, apperr.Validation("username is required when has_account=true")
		}
		if req.Password == nil || *req.Password == "" {
			return Employee{}, apperr.Validation("password is required when has_account=true")
		}
	}

	var hash *string
	if req.HasAccount {
		h, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return Employee{}, apperr.Internal(err)
		}
		s := string(h)
		hash = &s
	}

	e, err := s.repo.Create(ctx, req, hash, createdBy)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Employee{}, &apperr.AppError{
				Code:       "USERNAME_TAKEN",
				Message:    "username already in use",
				HTTPStatus: http.StatusConflict,
				Err:        err,
			}
		}
		return Employee{}, apperr.Internal(err)
	}
	return e, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateRequest, updatedBy *int64) (Employee, error) {
	e, err := s.repo.Update(ctx, id, req, updatedBy)
	if err != nil {
		if isNotFound(err) {
			return Employee{}, apperr.NotFound("EMPLOYEE_NOT_FOUND", "employee not found")
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Employee{}, &apperr.AppError{
				Code:       "USERNAME_TAKEN",
				Message:    "username already in use",
				HTTPStatus: http.StatusConflict,
				Err:        err,
			}
		}
		return Employee{}, apperr.Internal(err)
	}
	return e, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id == 1 {
		// Belt-and-braces: never let the super-admin row be soft-deleted via
		// the employees UI. The token-version check would already block any
		// further logins, but accidental deletion here would lose the only
		// admin login.
		return apperr.Conflict("CANNOT_DELETE_SUPER_ADMIN", "cannot delete super admin")
	}
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		if isNotFound(err) {
			return apperr.NotFound("EMPLOYEE_NOT_FOUND", "employee not found")
		}
		return apperr.Internal(err)
	}
	return nil
}
