// modules/auth/service.go
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/Orazgeldiyew/hezzet_market_backend/config"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

// PermissionFetcher returns the permission matrix entries for given role codes.
type PermissionFetcher interface {
	MatrixForRoles(ctx context.Context, roleCodes []string) ([]MatrixEntry, error)
}

// MatrixEntry mirrors permissions.MatrixEntry to avoid import cycle.
type MatrixEntry struct {
	Module   string
	View     bool
	Create   bool
	Update   bool
	Delete   bool
	Transfer bool
	Return   bool
	Discount bool
	History  bool
}

type Service struct {
	repo    *Repository
	cfg     config.Config
	permFet PermissionFetcher // nil-safe: if nil, permissions omitted from login response
}

func NewService(repo *Repository, cfg config.Config) *Service { return &Service{repo: repo, cfg: cfg} }

// SetPermissionFetcher injects the permission fetcher after construction (breaks init cycle).
func (s *Service) SetPermissionFetcher(pf PermissionFetcher) { s.permFet = pf }

// -------------------- Roles helpers --------------------

const DefaultStaffRole = "operator"

func normalizeRoles(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, r := range in {
		if r == "" {
			continue
		}
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

func validateAdminExclusive(roles []string) error {
	for _, r := range roles {
		if r == "admin" {
			if len(roles) > 1 {
				return apperr.Validation("admin role must be exclusive (cannot combine with other roles)")
			}
			return nil
		}
	}
	return nil
}

func applyDefaultRole(roles []string) []string {
	if len(roles) == 0 {
		return []string{DefaultStaffRole}
	}
	return roles
}

// -------------------- Login --------------------

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
	if u.BlockedAt != nil {
		return LoginResponse{}, apperr.Unauthorized("account is blocked")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return LoginResponse{}, apperr.Unauthorized("invalid credentials")
	}

	// Bump token_version on every successful login — invalidates all existing sessions
	// on other devices. Only this new login's token will carry the new version.
	newVersion, err := s.repo.BumpTokenVersion(ctx, u.ID)
	if err != nil {
		return LoginResponse{}, apperr.Internal(fmt.Errorf("bump token version: %w", err))
	}

	roles, err := s.repo.GetUserRoles(ctx, u.ID)
	if err != nil {
		return LoginResponse{}, apperr.Internal(err)
	}
	roles = normalizeRoles(roles)

	payload := JWTPayload{
		UserID:       u.ID,
		Username:     u.Username,
		Role:         roles,
		TokenVersion: newVersion,
	}

	accessToken, accessExp, err := s.generateToken(payload, "access", s.getAccessExpiration(), accessSecret)
	if err != nil {
		return LoginResponse{}, apperr.Internal(err)
	}

	refreshToken, _, err := s.generateToken(payload, "refresh", s.getRefreshExpiration(), s.getRefreshSecret())
	if err != nil {
		return LoginResponse{}, apperr.Internal(err)
	}

	// Best-effort: update last_login_at
	_ = s.repo.UpdateLastLogin(ctx, u.ID)

	// Fetch permissions for the user's roles
	var perms []UserPermission
	if s.permFet != nil {
		entries, err := s.permFet.MatrixForRoles(ctx, roles)
		if err == nil {
			perms = make([]UserPermission, 0, len(entries))
			for _, e := range entries {
				perms = append(perms, UserPermission{
					Module:   e.Module,
					View:     e.View,
					Create:   e.Create,
					Update:   e.Update,
					Delete:   e.Delete,
					Transfer: e.Transfer,
					Return:   e.Return,
					Discount: e.Discount,
				History:  e.History,
				})
			}
		}
	}
	if perms == nil {
		perms = []UserPermission{}
	}

	return LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    s.cfg.TokenType,
		ExpiresIn:    int64(accessExp.Seconds()),
		User:         toDTO(u),
		Roles:        roles,
		Permissions:  perms,
	}, nil
}

// -------------------- Register (public, no roles) --------------------

func (s *Service) Register(ctx context.Context, req RegisterRequest) (UserWithRoles, error) {
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

	return UserWithRoles{UserDTO: toDTO(u), Roles: []string{}}, nil
}

// -------------------- User CRUD --------------------

func (s *Service) CreateUser(ctx context.Context, req CreateUserRequest, createdBy int64) (UserWithRoles, error) {
	roles := normalizeRoles(req.Roles)
	roles = applyDefaultRole(roles)

	if err := validateAdminExclusive(roles); err != nil {
		return UserWithRoles{}, err
	}

	// Prevent creating a second admin
	for _, r := range roles {
		if r == "admin" {
			return UserWithRoles{}, apperr.Forbidden("admin role cannot be assigned to new users")
		}
	}

	n, err := s.repo.CountRolesByCodes(ctx, roles)
	if err != nil {
		return UserWithRoles{}, apperr.Internal(err)
	}
	if n != len(roles) {
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
		CreatedBy:    &createdBy,
	}
	if req.IsActive != nil {
		u.IsActive = *req.IsActive
	}

	if err := s.repo.CreateUserWithRoles(ctx, &u, roles); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return UserWithRoles{}, &apperr.AppError{
				Code:       "USERNAME_ALREADY_EXISTS",
				Message:    "username already exists",
				HTTPStatus: http.StatusConflict,
				Err:        err,
			}
		}
		if err == pgx.ErrNoRows {
			return UserWithRoles{}, apperr.Validation("unknown role in roles[]")
		}
		return UserWithRoles{}, apperr.Internal(err)
	}

	return UserWithRoles{UserDTO: toDTO(u), Roles: roles}, nil
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
	roles = normalizeRoles(roles)

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
		roles := normalizeRoles(rolesMap[u.ID])
		items = append(items, UserWithRoles{UserDTO: toDTO(u), Roles: roles})
	}

	return ListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Service) UpdateUser(ctx context.Context, id int64, req UpdateUserRequest, updatedBy int64) (UserWithRoles, error) {
	byPtr := &updatedBy
	u, err := s.repo.Update(ctx, id, req, byPtr)
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
		roles := normalizeRoles(*req.Roles)

		if len(roles) == 0 {
			return UserWithRoles{}, apperr.Validation("roles cannot be empty")
		}
		if err := validateAdminExclusive(roles); err != nil {
			return UserWithRoles{}, err
		}

		// Prevent giving admin role to non-admin user
		currentRoles, _ := s.repo.GetUserRoles(ctx, id)
		isCurrentAdmin := false
		for _, r := range currentRoles {
			if r == "admin" { isCurrentAdmin = true; break }
		}
		for _, r := range roles {
			if r == "admin" && !isCurrentAdmin {
				return UserWithRoles{}, apperr.Forbidden("admin role cannot be assigned")
			}
		}

		n, err := s.repo.CountRolesByCodes(ctx, roles)
		if err != nil {
			return UserWithRoles{}, apperr.Internal(err)
		}
		if n != len(roles) {
			return UserWithRoles{}, apperr.Validation("unknown role in roles[]")
		}

		if err := s.repo.SetUserRoles(ctx, u.ID, roles); err != nil {
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
	roles = normalizeRoles(roles)

	return UserWithRoles{UserDTO: toDTO(u), Roles: roles}, nil
}

// DeleteUser: admin-only in routes; security: forbid self-delete
func (s *Service) DeleteUser(ctx context.Context, id int64, actorID int64) error {
	if id <= 0 {
		return apperr.Validation("invalid user id")
	}
	if actorID <= 0 {
		return apperr.Unauthorized("unauthorized")
	}
	if id == actorID {
		return apperr.Validation("cannot delete yourself")
	}

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		if isNotFound(err) {
			return apperr.NotFound("NOT_FOUND", "user not found")
		}
		return apperr.Internal(err)
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, id int64, password string, updatedBy int64) error {
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

	if err := s.repo.UpdatePasswordAndBumpVersion(ctx, id, string(hash), updatedBy); err != nil {
		if isNotFound(err) {
			return apperr.NotFound("NOT_FOUND", "user not found")
		}
		return apperr.Internal(err)
	}
	return nil
}

// -------------------- Block / Unblock --------------------

func (s *Service) BlockUser(ctx context.Context, id int64, reason string, updatedBy int64) (UserWithRoles, error) {
	if id == updatedBy {
		return UserWithRoles{}, apperr.Validation("cannot block yourself")
	}

	u, err := s.repo.BlockUser(ctx, id, reason, updatedBy)
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
	roles = normalizeRoles(roles)

	return UserWithRoles{UserDTO: toDTO(u), Roles: roles}, nil
}

func (s *Service) UnblockUser(ctx context.Context, id int64, updatedBy int64) (UserWithRoles, error) {
	// unblock self technically harmless, but keep strict symmetry with block
	if id == updatedBy {
		return UserWithRoles{}, apperr.Validation("cannot unblock yourself")
	}

	u, err := s.repo.UnblockUser(ctx, id, updatedBy)
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
	roles = normalizeRoles(roles)

	return UserWithRoles{UserDTO: toDTO(u), Roles: roles}, nil
}

// -------------------- Refresh Token --------------------

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

	// token_version: FAIL-CLOSED
	rawTV, exists := claims["token_version"]
	if !exists {
		return TokenResponse{}, apperr.Unauthorized("token missing token_version")
	}
	tvFloat, ok := rawTV.(float64)
	if !ok {
		return TokenResponse{}, apperr.Unauthorized("invalid token_version type")
	}
	claimVersion := int(tvFloat)

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
	if u.BlockedAt != nil {
		return TokenResponse{}, apperr.Unauthorized("account is blocked")
	}
	if claimVersion != u.TokenVersion {
		return TokenResponse{}, apperr.Unauthorized("token revoked")
	}

	roles, err := s.repo.GetUserRoles(ctx, u.ID)
	if err != nil {
		return TokenResponse{}, apperr.Internal(err)
	}
	roles = normalizeRoles(roles)

	payload := JWTPayload{
		UserID:       u.ID,
		Username:     u.Username,
		Role:         roles,
		TokenVersion: u.TokenVersion,
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

// -------------------- Token helpers --------------------

func (s *Service) generateToken(payload JWTPayload, tokenType string, expDuration time.Duration, secret string) (string, time.Duration, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":           strconv.FormatInt(payload.UserID, 10),
		"username":      payload.Username,
		"role":          payload.Role,
		"type":          tokenType,
		"iat":           now.Unix(),
		"exp":           now.Add(expDuration).Unix(),
		"token_version": payload.TokenVersion,
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