package category

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

func (s *Service) Create(ctx context.Context, req CreateRequest) (Category, error) {
	c := Category{
		Name:     req.Name,
		ParentID: req.ParentID,
		IsActive: true,
	}
	if req.IsActive != nil {
		c.IsActive = *req.IsActive
	}

	if err := s.repo.Create(ctx, &c); err != nil {
		// unique(parent_id, name) -> 23505
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Category{}, &apperr.AppError{
				Code:       "CATEGORY_ALREADY_EXISTS",
				Message:    "category already exists in this parent",
				HTTPStatus: http.StatusConflict,
				Err:        err,
			}
		}
		return Category{}, apperr.Internal(err)
	}

	return c, nil
}

func (s *Service) Get(ctx context.Context, id int) (Category, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if IsNotFound(err) {
			return Category{}, apperr.NotFound("NOT_FOUND", "category not found")
		}
		return Category{}, apperr.Internal(err)
	}
	return c, nil
}

func (s *Service) List(ctx context.Context, limit, offset int, orderBy, orderDir, q string) (ListResponse, error) {
	// Safety defaults (even if handler/middleware fails)
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
		items = []Category{}
	}

	return ListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Service) Tree(ctx context.Context) (TreeResponse, error) {
	cats, err := s.repo.ListAll(ctx)
	if err != nil {
		return TreeResponse{}, apperr.Internal(err)
	}
	return TreeResponse{Items: buildTree(cats)}, nil
}

func (s *Service) Update(ctx context.Context, id int, req UpdateRequest) (Category, error) {
	c, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if IsNotFound(err) {
			return Category{}, apperr.NotFound("NOT_FOUND", "category not found")
		}

		// unique(parent_id, name) -> 23505
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Category{}, &apperr.AppError{
				Code:       "CATEGORY_ALREADY_EXISTS",
				Message:    "category already exists in this parent",
				HTTPStatus: http.StatusConflict,
				Err:        err,
			}
		}

		return Category{}, apperr.Internal(err)
	}
	return c, nil
}

func (s *Service) Delete(ctx context.Context, id int) error {
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		if IsNotFound(err) {
			return apperr.NotFound("NOT_FOUND", "category not found")
		}
		return apperr.Internal(err)
	}
	return nil
}

func buildTree(cats []Category) []*CategoryTree {
	nodeMap := make(map[int]*CategoryTree)
	var roots []*CategoryTree

	for _, c := range cats {
		nodeMap[c.ID] = &CategoryTree{
			ID:       c.ID,
			Name:     c.Name,
			ParentID: c.ParentID,
			Children: []*CategoryTree{},
		}
	}

	for _, c := range cats {
		node := nodeMap[c.ID]
		if c.ParentID == nil {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[*c.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}

	return roots
}