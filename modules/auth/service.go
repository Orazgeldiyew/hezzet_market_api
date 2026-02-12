// modules/auth/service.go
package auth

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Service struct {
	repo *Repository
	cfg  config.Config
}

func NewService(repo *Repository, cfg config.Config) *Service { return &Service{repo: repo, cfg: cfg} }

// ---------- Login ----------

func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	if s.cfg.JWTSecret == "" {
		return LoginResponse{}, apperr.Internal(errors.New("JWT_SECRET is empty"))
	}

	u, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		if isNotFound(err) {
			return LoginResponse{}, apperr.Unauthorized("invalid credentials")
		}
		return LoginResponse{}, apperr.Internal(err)
	}

	if !u.IsActive {
		return LoginResponse{}, apperr.Unauthorized("account is disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return LoginResponse{}, apperr.Unauthorized("invalid credentials")
	}

	roles, err := s.repo.GetUserRoles(ctx, u.ID)
	if err != nil {
		return LoginResponse{}, apperr.Internal(err)
	}
	if roles == nil {
		roles = []string{}
	}

	expDuration := s.getExpiration()
	now := time.Now()
	exp := now.Add(expDuration)

	claims := jwt.MapClaims{
		"sub":      strconv.FormatInt(u.ID, 10),
		"username": u.Username,
		"roles":    roles,
		"iat":      now.Unix(),
		"exp":      exp.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return LoginResponse{}, apperr.Internal(err)
	}

	return LoginResponse{
		AccessToken: tokenStr,
		TokenType:   "Bearer",
		ExpiresIn:   int64(expDuration.Seconds()),
		User:        toDTO(u),
		Roles:       roles,
	}, nil
}

// ---------- User CRUD ----------

func (s *Service) CreateUser(ctx context.Context, req CreateUserRequest) (UserWithRoles, error) {
	n, err := s.repo.CountRolesByCodes(ctx, req.Roles)
	if err != nil {
		return UserWithRoles{}, apperr.Internal(err)
	}
	if n != len(req.Roles) {
		return UserWithRoles{}, apperr.Validation("unknown role in roles[]")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return UserWithRoles{}, apperr.Internal(err)
	}

	u := User{
		Username:     req.Username,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		Phone:        req.Phone,
		Email:        req.Email,
		IsActive:     true,
	}
	if req.IsActive != nil {
		u.IsActive = *req.IsActive
	}

	if err := s.repo.CreateUser(ctx, &u); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return UserWithRoles{}, &apperr.AppError{
				Code:       "USERNAME_ALREADY_EXISTS",
				Message:    "username already exists",
				HTTPStatus: http.StatusConflict,
				Err:        err,
			}
		}
		return UserWithRoles{}, apperr.Internal(err)
	}

	if err := s.repo.SetUserRoles(ctx, u.ID, req.Roles); err != nil {
		if err == pgx.ErrNoRows {
			return UserWithRoles{}, apperr.Validation("unknown role in roles[]")
		}
		return UserWithRoles{}, apperr.Internal(err)
	}

	return UserWithRoles{UserDTO: toDTO(u), Roles: req.Roles}, nil
}

func (s *Service) GetUser(ctx context.Context, id int64) (UserWithRoles, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return UserWithRoles{}, apperr.NotFound("NOT_FOUND", "user not found")
		}
		return UserWithRoles{}, apperr.Internal(err)
	}

	roles, err := s.repo.GetUserRoles(ctx, u.ID)
	if err != nil {
		return UserWithRoles{}, apperr.Internal(err)
	}
	if roles == nil {
		roles = []string{}
	}

	return UserWithRoles{UserDTO: toDTO(u), Roles: roles}, nil
}

func (s *Service) ListUsers(ctx context.Context, limit, offset int, orderBy, orderDir, search string) (ListResponse, error) {
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

	users, total, err := s.repo.List(ctx, limit, offset, orderBy, orderDir, search)
	if err != nil {
		return ListResponse{}, apperr.Internal(err)
	}

	userIDs := make([]int64, 0, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
	}

	rolesMap, err := s.repo.GetRolesByUserIDs(ctx, userIDs)
	if err != nil {
		return ListResponse{}, apperr.Internal(err)
	}

	items := make([]UserWithRoles, 0, len(users))
	for _, u := range users {
		roles := rolesMap[u.ID]
		if roles == nil {
			roles = []string{}
		}
		items = append(items, UserWithRoles{UserDTO: toDTO(u), Roles: roles})
	}

	return ListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Service) UpdateUser(ctx context.Context, id int64, req UpdateUserRequest) (UserWithRoles, error) {
	u, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if isNotFound(err) {
			return UserWithRoles{}, apperr.NotFound("NOT_FOUND", "user not found")
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return UserWithRoles{}, &apperr.AppError{
				Code:       "USERNAME_ALREADY_EXISTS",
				Message:    "username already exists",
				HTTPStatus: http.StatusConflict,
				Err:        err,
			}
		}
		return UserWithRoles{}, apperr.Internal(err)
	}

	if req.Roles != nil {
		n, err := s.repo.CountRolesByCodes(ctx, req.Roles)
		if err != nil {
			return UserWithRoles{}, apperr.Internal(err)
		}
		if n != len(req.Roles) {
			return UserWithRoles{}, apperr.Validation("unknown role in roles[]")
		}
		if err := s.repo.SetUserRoles(ctx, u.ID, req.Roles); err != nil {
			if err == pgx.ErrNoRows {
				return UserWithRoles{}, apperr.Validation("unknown role in roles[]")
			}
			return UserWithRoles{}, apperr.Internal(err)
		}
	}

	roles, err := s.repo.GetUserRoles(ctx, u.ID)
	if err != nil {
		return UserWithRoles{}, apperr.Internal(err)
	}
	if roles == nil {
		roles = []string{}
	}

	return UserWithRoles{UserDTO: toDTO(u), Roles: roles}, nil
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		if isNotFound(err) {
			return apperr.NotFound("NOT_FOUND", "user not found")
		}
		return apperr.Internal(err)
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, id int64, password string) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return apperr.NotFound("NOT_FOUND", "user not found")
		}
		return apperr.Internal(err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return apperr.Internal(err)
	}

	if err := s.repo.UpdatePassword(ctx, id, string(hash)); err != nil {
		if isNotFound(err) {
			return apperr.NotFound("NOT_FOUND", "user not found")
		}
		return apperr.Internal(err)
	}
	return nil
}

func (s *Service) getExpiration() time.Duration {
	exp := s.cfg.JWTExpiresIn
	if exp == "" {
		exp = "24h"
	}
	d, err := time.ParseDuration(exp)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}
