package permissions

import (
	"context"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

var validRoles = map[string]bool{"cashier": true, "operator": true, "manager": true}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]Permission, error) {
	perms, err := s.repo.List(ctx)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return perms, nil
}

// Update changes a permission. callerIsAdmin must be true when the caller has the admin role.
// Manager permissions can only be changed by admin.
func (s *Service) Update(ctx context.Context, role, module string, enabled bool, callerIsAdmin bool) (Permission, error) {
	if !validRoles[role] {
		return Permission{}, apperr.Validation("invalid role: must be cashier, operator, or manager")
	}
	if role == "manager" && !callerIsAdmin {
		return Permission{}, apperr.Forbidden("only admin can change manager permissions")
	}

	p, err := s.repo.Update(ctx, role, module, enabled)
	if err != nil {
		return Permission{}, apperr.Internal(err)
	}
	return p, nil
}
