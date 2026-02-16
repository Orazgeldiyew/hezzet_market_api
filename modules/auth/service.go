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
	accessSecret := s.getAccessSecret()
	if accessSecret == "" {
		return LoginResponse{}, apperr.Internal(errors.New("ACCESS_TOKEN_SECRET is empty"))
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

	payload := JWTPayload{
		UserID:   u.ID,
		Username: u.Username,
		Role:     roles,
	}

	accessToken, accessExp, err := s.generateToken(payload, "access", s.getAccessExpiration(), accessSecret)
	if err != nil {
		return LoginResponse{}, apperr.Internal(err)
	}

	refreshToken, _, err := s.generateToken(payload, "refresh", s.getRefreshExpiration(), s.getRefreshSecret())
	if err != nil {
		return LoginResponse{}, apperr.Internal(err)
	}

	return LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    s.cfg.TokenType,
		ExpiresIn:    int64(accessExp.Seconds()),
		User:         toDTO(u),
		Roles:        roles,
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

// ---------- Refresh Token ----------

func (s *Service) RefreshToken(ctx context.Context, refreshTokenStr string) (TokenResponse, error) {
	refreshSecret := s.getRefreshSecret()
	if refreshSecret == "" {
		return TokenResponse{}, apperr.Internal(errors.New("REFRESH_TOKEN_SECRET is empty"))
	}

	token, err := jwt.Parse(refreshTokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(refreshSecret), nil
	})
	if err != nil || !token.Valid {
		return TokenResponse{}, apperr.Unauthorized("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return TokenResponse{}, apperr.Unauthorized("invalid refresh token")
	}

	tokenType, _ := claims["type"].(string)
	if tokenType != "refresh" {
		return TokenResponse{}, apperr.Unauthorized("invalid token type")
	}

	sub, _ := claims.GetSubject()
	userID, err := strconv.ParseInt(sub, 10, 64)
	if err != nil || userID <= 0 {
		return TokenResponse{}, apperr.Unauthorized("invalid refresh token")
	}

	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return TokenResponse{}, apperr.Unauthorized("user not found")
		}
		return TokenResponse{}, apperr.Internal(err)
	}

	if !u.IsActive {
		return TokenResponse{}, apperr.Unauthorized("account is disabled")
	}

	roles, err := s.repo.GetUserRoles(ctx, u.ID)
	if err != nil {
		return TokenResponse{}, apperr.Internal(err)
	}
	if roles == nil {
		roles = []string{}
	}

	payload := JWTPayload{
		UserID:   u.ID,
		Username: u.Username,
		Role:     roles,
	}

	newAccessToken, accessExp, err := s.generateToken(payload, "access", s.getAccessExpiration(), s.getAccessSecret())
	if err != nil {
		return TokenResponse{}, apperr.Internal(err)
	}

	newRefreshToken, _, err := s.generateToken(payload, "refresh", s.getRefreshExpiration(), refreshSecret)
	if err != nil {
		return TokenResponse{}, apperr.Internal(err)
	}

	return TokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    s.cfg.TokenType,
		ExpiresIn:    int64(accessExp.Seconds()),
	}, nil
}

// ---------- Token helpers ----------

func (s *Service) generateToken(payload JWTPayload, tokenType string, expDuration time.Duration, secret string) (string, time.Duration, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      strconv.FormatInt(payload.UserID, 10),
		"username": payload.Username,
		"role":     payload.Role,
		"type":     tokenType,
		"iat":      now.Unix(),
		"exp":      now.Add(expDuration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", 0, err
	}
	return tokenStr, expDuration, nil
}

func (s *Service) getAccessSecret() string {
	if s.cfg.AccessTokenSecret != "" {
		return s.cfg.AccessTokenSecret
	}
	return s.cfg.JWTSecret
}

func (s *Service) getRefreshSecret() string {
	if s.cfg.RefreshTokenSecret != "" {
		return s.cfg.RefreshTokenSecret
	}
	return s.cfg.JWTSecret
}

func (s *Service) getAccessExpiration() time.Duration {
	exp := s.cfg.AccessTokenExpiresIn
	if exp == "" {
		exp = "15m"
	}
	d, err := config.ParseDuration(exp)
	if err != nil {
		return 15 * time.Minute
	}
	return d
}

func (s *Service) getRefreshExpiration() time.Duration {
	exp := s.cfg.RefreshTokenExpiresIn
	if exp == "" {
		exp = "168h"
	}
	d, err := config.ParseDuration(exp)
	if err != nil {
		return 7 * 24 * time.Hour
	}
	return d
}
