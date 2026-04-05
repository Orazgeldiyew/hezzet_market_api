package permissions

import (
	"context"
	"regexp"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

var validActions = map[string]bool{"view": true, "create": true, "update": true, "delete": true, "transfer": true, "return": true, "discount": true, "history": true}
var roleCodeRe = regexp.MustCompile(`^[a-z][a-z0-9_]{1,49}$`)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ── Roles ──

func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return roles, nil
}

func (s *Service) GetRole(ctx context.Context, id int) (Role, []Permission, error) {
	role, err := s.repo.GetRole(ctx, id)
	if err != nil {
		return Role{}, nil, apperr.NotFound("ROLE_NOT_FOUND", "role not found")
	}
	perms, err := s.repo.GetPermissionsByRole(ctx, id)
	if err != nil {
		return Role{}, nil, apperr.Internal(err)
	}
	return role, perms, nil
}

func (s *Service) CreateRole(ctx context.Context, req CreateRoleRequest) (Role, error) {
	if !roleCodeRe.MatchString(req.Code) {
		return Role{}, apperr.Validation("code must be lowercase letters, digits, underscores (2-50 chars)")
	}
	role, err := s.repo.CreateRole(ctx, req)
	if err != nil {
		return Role{}, apperr.Conflict("DUPLICATE_ROLE", "role code already exists")
	}
	return role, nil
}

func (s *Service) UpdateRole(ctx context.Context, id int, req UpdateRoleRequest) (Role, error) {
	role, err := s.repo.UpdateRole(ctx, id, req)
	if err != nil {
		return Role{}, apperr.NotFound("ROLE_NOT_FOUND", "role not found")
	}
	return role, nil
}

func (s *Service) DeleteRole(ctx context.Context, id int) error {
	role, err := s.repo.GetRole(ctx, id)
	if err != nil {
		return apperr.NotFound("ROLE_NOT_FOUND", "role not found")
	}
	if role.IsSystem {
		return apperr.Forbidden("cannot delete system role")
	}
	assigned, err := s.repo.IsRoleAssigned(ctx, id)
	if err != nil {
		return apperr.Internal(err)
	}
	if assigned {
		return apperr.Conflict("ROLE_IN_USE", "cannot delete role: users are assigned to it")
	}
	if err := s.repo.DeleteRole(ctx, id); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// ── Permissions ──

func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	perms, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return perms, nil
}

func (s *Service) Matrix(ctx context.Context) ([]MatrixEntry, error) {
	m, err := s.repo.Matrix(ctx)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return m, nil
}

func (s *Service) UpdatePermission(ctx context.Context, roleID int, module, action string, granted bool, callerRoles []string) (Permission, error) {
	if !validActions[action] {
		return Permission{}, apperr.Validation("action must be view, create, update, or delete")
	}

	role, err := s.repo.GetRole(ctx, roleID)
	if err != nil {
		return Permission{}, apperr.NotFound("ROLE_NOT_FOUND", "role not found")
	}
	if role.Code == "manager" && !sliceContains(callerRoles, "admin") {
		return Permission{}, apperr.Forbidden("only admin can change manager permissions")
	}

	p, err := s.repo.UpdatePermission(ctx, roleID, module, action, granted)
	if err != nil {
		return Permission{}, apperr.Internal(err)
	}
	return p, nil
}

func (s *Service) BulkUpdatePermissions(ctx context.Context, roleID int, req BulkPermissionRequest, callerRoles []string) ([]Permission, error) {
	for _, item := range req.Permissions {
		if !validActions[item.Action] {
			return nil, apperr.Validation("invalid action: " + item.Action)
		}
	}

	role, err := s.repo.GetRole(ctx, roleID)
	if err != nil {
		return nil, apperr.NotFound("ROLE_NOT_FOUND", "role not found")
	}
	if role.Code == "manager" && !sliceContains(callerRoles, "admin") {
		return nil, apperr.Forbidden("only admin can change manager permissions")
	}

	perms, err := s.repo.BulkUpdatePermissions(ctx, roleID, req.Permissions)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return perms, nil
}

func sliceContains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
