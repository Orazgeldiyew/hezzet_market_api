package product

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Service struct {
	repo       *Repository
	stock      *StockRepository
	uploadsDir string
}

func NewService(repo *Repository, stock *StockRepository, uploadsDir string) *Service {
	return &Service{repo: repo, stock: stock, uploadsDir: uploadsDir}
}

// ValidateUnit enforces the business rule:
//
//	piece  → unit must be "piece"
//	weight → unit must be "kg" or "g"
//	volume → unit must be "l"  or "ml"
func ValidateUnit(unitType UnitType, unit Unit) error {
	switch unitType {
	case UnitTypePiece:
		if unit != UnitPiece {
			return fmt.Errorf("unit_type 'piece' requires unit 'piece', got '%s'", unit)
		}
	case UnitTypeWeight:
		if unit != UnitKg && unit != UnitG {
			return fmt.Errorf("unit_type 'weight' requires unit 'kg' or 'g', got '%s'", unit)
		}
	case UnitTypeVolume:
		if unit != UnitL && unit != UnitML {
			return fmt.Errorf("unit_type 'volume' requires unit 'l' or 'ml', got '%s'", unit)
		}
	default:
		return fmt.Errorf("unknown unit_type '%s'", unitType)
	}
	return nil
}

// SavePhoto validates, stores a photo file, updates the DB, and deletes any previous file.
// currentPhotoPath is the existing photo_path value on the product (may be nil).
// Returns the new storedPath (relative, e.g. "photos/42.jpg").
func (s *Service) SavePhoto(ctx context.Context, productID int64, fh *multipart.FileHeader, currentPhotoPath *string) (string, error) {
	if fh.Size > 5*1024*1024 {
		return "", apperr.Validation("file too large (max 5MB)")
	}

	// Open and sniff content type
	src, err := fh.Open()
	if err != nil {
		return "", apperr.Internal(fmt.Errorf("open upload: %w", err))
	}
	defer src.Close()

	buf := make([]byte, 512)
	n, err := src.Read(buf)
	if err != nil && err != io.EOF {
		return "", apperr.Internal(fmt.Errorf("read upload: %w", err))
	}
	ct := http.DetectContentType(buf[:n])

	allowed := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}
	if !allowed[ct] {
		return "", apperr.Validation("unsupported file type, allowed: jpg, jpeg, png, webp")
	}

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	allowedExt := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
	}
	if !allowedExt[ext] {
		return "", apperr.Validation("unsupported file extension")
	}

	// Create photo record → get numeric photo ID
	photoID, err := s.repo.InsertPhoto(ctx, productID, ext)
	if err != nil {
		return "", apperr.Internal(fmt.Errorf("insert photo record: %w", err))
	}

	storedPath := fmt.Sprintf("photos/%d%s", photoID, ext)
	fullDir := filepath.Join(s.uploadsDir, "photos")
	fullPath := filepath.Join(s.uploadsDir, storedPath)

	// Security: ensure path stays inside uploadsDir
	absUploads, _ := filepath.Abs(s.uploadsDir)
	absFile, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absFile, absUploads) {
		return "", apperr.Validation("invalid file path")
	}

	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", apperr.Internal(fmt.Errorf("create upload dir: %w", err))
	}

	// Seek back to beginning (we already read 512 bytes for detection)
	if seeker, ok := src.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return "", apperr.Internal(fmt.Errorf("seek upload: %w", err))
		}
	} else {
		// Re-open since the reader may not be seekable
		src.Close()
		src2, err := fh.Open()
		if err != nil {
			return "", apperr.Internal(fmt.Errorf("reopen upload: %w", err))
		}
		defer src2.Close()
		return s.writeAndFinalize(storedPath, fullPath, src2, currentPhotoPath)
	}

	return s.writeAndFinalize(storedPath, fullPath, src, currentPhotoPath)
}

func (s *Service) writeAndFinalize(storedPath, fullPath string, r io.Reader, currentPhotoPath *string) (string, error) {
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", apperr.Internal(fmt.Errorf("create file: %w", err))
	}
	defer dst.Close()

	if _, err := io.Copy(dst, r); err != nil {
		return "", apperr.Internal(fmt.Errorf("write file: %w", err))
	}

	// Delete old file (best-effort, don't fail on error)
	if currentPhotoPath != nil && *currentPhotoPath != "" {
		_ = os.Remove(filepath.Join(s.uploadsDir, *currentPhotoPath))
	}

	return storedPath, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest, fh *multipart.FileHeader) (Product, error) {
	// Enforce unit_type ↔ unit compatibility before touching the DB.
	if err := ValidateUnit(req.UnitType, req.Unit); err != nil {
		return Product{}, apperr.Validation(err.Error())
	}

	barcodes := req.Barcodes
	if barcodes == nil {
		barcodes = []string{}
	}

	p := Product{
		Name:          req.Name,
		SKU:           req.SKU,
		Barcodes:      barcodes,
		UnitType:      req.UnitType,
		Unit:          req.Unit,
		PurchasePrice: req.PurchasePrice,
		SalePrice:     req.SalePrice,
		IsActive:      true,
	}
	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	}

	if err := s.repo.Create(ctx, &p); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			code, msg := "PRODUCT_ALREADY_EXISTS", "product with this sku already exists"
			if pgErr.ConstraintName == "product_barcodes_barcode_key" {
				code, msg = "BARCODE_ALREADY_EXISTS", "barcode already exists"
			}
			return Product{}, &apperr.AppError{
				Code:    code,
				Message: msg,
				Err:     err,
			}
		}
		return Product{}, apperr.Internal(err)
	}

	// Link categories if provided.
	if len(req.CategoryIDs) > 0 {
		uniq := uniqueInt64(req.CategoryIDs)
		n, err := s.repo.CountActiveCategoriesByIDs(ctx, uniq)
		if err != nil {
			return p, apperr.Internal(err)
		}
		if n != len(uniq) {
			return p, apperr.Validation("some categories not found or inactive")
		}
		if err := s.repo.ReplaceCategories(ctx, p.ID, uniq); err != nil {
			return p, apperr.Internal(err)
		}
	}

	// Optional photo upload
	if fh != nil {
		storedPath, err := s.SavePhoto(ctx, p.ID, fh, nil)
		if err != nil {
			return p, err
		}
		if err := s.repo.UpdatePhotoPath(ctx, p.ID, &storedPath); err != nil {
			return p, apperr.Internal(err)
		}
		// Re-fetch to get the computed PhotoURL
		p, err = s.repo.GetByID(ctx, p.ID)
		if err != nil {
			return p, apperr.Internal(err)
		}
	}

	return p, nil
}

func (s *Service) GetByBarcode(ctx context.Context, barcode string) (BarcodeResult, error) {
	// 1. Try exact match first (normal barcode)
	p, err := s.repo.GetByBarcode(ctx, barcode)
	if err == nil {
		return BarcodeResult{Product: p}, nil
	}
	if err != pgx.ErrNoRows {
		return BarcodeResult{}, apperr.Internal(err)
	}

	// 2. Try parsing as weight barcode: length 13
	//    Format: PPPPPPP WWWWWW (7-digit product code + 6-digit weight in grams)
	if len(barcode) == 13 {
		productCode := barcode[0:7] // 7 digits
		weightStr := barcode[7:13]  // 6 digits (grams)

		p, err := s.repo.GetByBarcode(ctx, productCode)
		if err == nil {
			weight, _ := strconv.ParseInt(weightStr, 10, 64)
			if weight > 0 {
				return BarcodeResult{
					Product:  p,
					QtyMilli: weight, // grams = milli-kg
					IsWeight: true,
				}, nil
			}
		}
	}

	return BarcodeResult{}, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found for this barcode")
}

func (s *Service) Get(ctx context.Context, id int64) (Product, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Product{}, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return Product{}, apperr.Internal(err)
	}
	return p, nil
}

func (s *Service) GetCard(ctx context.Context, id int64) (Card, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Card{}, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return Card{}, apperr.Internal(err)
	}

	stock, err := s.stock.GetStock(ctx, id)
	if err != nil {
		return Card{}, apperr.Internal(err)
	}

	cats, err := s.repo.ListCategories(ctx, id)
	if err != nil {
		return Card{}, apperr.Internal(err)
	}

	return Card{
		Product:    p,
		Stock:      stock,
		Tags:       []string{},
		Suppliers:  []int64{},
		Categories: cats,
	}, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateRequest, userID *int64) (Product, error) {
	// Load current product for validation and price-history comparison.
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Product{}, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return Product{}, apperr.Internal(err)
	}

	// When unit_type or unit is being changed we must validate the combined
	// (effective) state: either field may be absent in the request, so we
	// merge before validating.
	if req.UnitType != nil || req.Unit != nil {
		effectiveUnitType := current.UnitType
		if req.UnitType != nil {
			effectiveUnitType = *req.UnitType
		}

		effectiveUnit := current.Unit
		if req.Unit != nil {
			effectiveUnit = *req.Unit
		}

		if err := ValidateUnit(effectiveUnitType, effectiveUnit); err != nil {
			return Product{}, apperr.Validation(err.Error())
		}
	}

	p, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Product{}, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			code, msg := "PRODUCT_ALREADY_EXISTS", "product with this sku already exists"
			if pgErr.ConstraintName == "product_barcodes_barcode_key" {
				code, msg = "BARCODE_ALREADY_EXISTS", "barcode already exists"
			}
			return Product{}, &apperr.AppError{
				Code:    code,
				Message: msg,
				Err:     err,
			}
		}
		return Product{}, apperr.Internal(err)
	}

	// Log price changes
	if req.PurchasePrice != nil && *req.PurchasePrice != current.PurchasePrice {
		_ = s.repo.LogPriceChange(ctx, id, "purchase_price", current.PurchasePrice, *req.PurchasePrice, userID)
	}
	if req.SalePrice != nil && *req.SalePrice != current.SalePrice {
		_ = s.repo.LogPriceChange(ctx, id, "sale_price", current.SalePrice, *req.SalePrice, userID)
	}

	// Update categories if provided.
	if req.CategoryIDs != nil {
		uniq := uniqueInt64(*req.CategoryIDs)
		if len(uniq) > 0 {
			n, err := s.repo.CountActiveCategoriesByIDs(ctx, uniq)
			if err != nil {
				return p, apperr.Internal(err)
			}
			if n != len(uniq) {
				return p, apperr.Validation("some categories not found or inactive")
			}
		}
		if err := s.repo.ReplaceCategories(ctx, id, uniq); err != nil {
			return p, apperr.Internal(err)
		}
	}

	return p, nil
}

// UploadPhoto validates, stores, and records a new photo for an existing product.
func (s *Service) UploadPhoto(ctx context.Context, id int64, fh *multipart.FileHeader) (Product, error) {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Product{}, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return Product{}, apperr.Internal(err)
	}

	storedPath, err := s.SavePhoto(ctx, id, fh, current.PhotoPath)
	if err != nil {
		return Product{}, err
	}

	if err := s.repo.UpdatePhotoPath(ctx, id, &storedPath); err != nil {
		return Product{}, apperr.Internal(err)
	}

	return s.repo.GetByID(ctx, id)
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
		items = []Product{}
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
		if IsNoRows(err) {
			return apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return apperr.Internal(err)
	}
	return nil
}

// ─── Category helpers ────────────────────────────────────────────────────────

func (s *Service) GetCategories(ctx context.Context, productID int64) ([]CategoryBrief, error) {
	if _, err := s.repo.GetByID(ctx, productID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return nil, apperr.Internal(err)
	}
	out, err := s.repo.ListCategories(ctx, productID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}

func (s *Service) SetCategories(ctx context.Context, productID int64, ids []int64) ([]CategoryBrief, error) {
	if _, err := s.repo.GetByID(ctx, productID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return nil, apperr.Internal(err)
	}

	uniq := uniqueInt64(ids)
	if len(uniq) > 0 {
		n, err := s.repo.CountActiveCategoriesByIDs(ctx, uniq)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		if n != len(uniq) {
			return nil, apperr.Validation("some categories not found or inactive")
		}
	}

	if err := s.repo.ReplaceCategories(ctx, productID, uniq); err != nil {
		return nil, apperr.Internal(err)
	}

	out, err := s.repo.ListCategories(ctx, productID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}

func (s *Service) RemoveCategory(ctx context.Context, productID, categoryID int64) error {
	if _, err := s.repo.GetByID(ctx, productID); err != nil {
		if err == pgx.ErrNoRows {
			return apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return apperr.Internal(err)
	}

	ok, err := s.repo.RemoveCategory(ctx, productID, categoryID)
	if err != nil {
		return apperr.Internal(err)
	}
	if !ok {
		return apperr.NotFound("NOT_FOUND", "link not found")
	}
	return nil
}

// ─── Price history ──────────────────────────────────────────────────────────

func (s *Service) GetPriceHistory(ctx context.Context, productID int64, limit, offset int) ([]PriceHistory, int, error) {
	// Verify product exists
	if _, err := s.repo.GetByID(ctx, productID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, 0, apperr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return nil, 0, apperr.Internal(err)
	}
	items, total, err := s.repo.GetPriceHistory(ctx, productID, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return items, total, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func uniqueInt64(in []int64) []int64 {
	seen := make(map[int64]struct{}, len(in))
	out := make([]int64, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
