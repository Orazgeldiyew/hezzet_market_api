package printer

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

type Service struct {
	repo       *Repository
	db         *pgxpool.Pool
	uploadsDir string
}

func NewService(repo *Repository, db *pgxpool.Pool, uploadsDir string) *Service {
	return &Service{repo: repo, db: db, uploadsDir: uploadsDir}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Printer, error) {
	p, err := s.repo.Create(ctx, req)
	if err != nil {
		return Printer{}, apperr.Internal(err)
	}
	return p, nil
}

func (s *Service) List(ctx context.Context) ([]Printer, error) {
	out, err := s.repo.List(ctx)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateRequest) (Printer, error) {
	p, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return Printer{}, apperr.Internal(err)
	}
	return p, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// TestPrint sends a simple test page to the given printer.
func (s *Service) TestPrint(ctx context.Context, printerID int64) error {
	p, err := s.repo.GetByID(ctx, printerID)
	if err != nil {
		return apperr.NotFound("PRINTER_NOT_FOUND", "printer not found")
	}
	if !p.IsActive {
		return apperr.Conflict("PRINTER_INACTIVE", "printer is not active")
	}
	data := buildTestPage(p.Name, p.IPAddress)
	if err := SendToTCP(p.IPAddress, p.Port, data); err != nil {
		return apperr.Internal(fmt.Errorf("print failed: %w", err))
	}
	return nil
}

// PrintSale gathers sale data and prints a receipt to the printer
// bound to the given register. Returns nil if no printer bound (silent).
func (s *Service) PrintSale(ctx context.Context, saleID int64, registerID *int64) error {
	if registerID == nil {
		log.Printf("[PrintSale] registerID is nil, skipping saleID=%d", saleID)
		return nil
	}

	p, err := s.repo.GetByRegisterID(ctx, *registerID)
	if err != nil {
		log.Printf("[PrintSale] no printer for register_id=%d: %v", *registerID, err)
		return nil
	}
	log.Printf("[PrintSale] found printer id=%d name=%s ip=%s:%d active=%v", p.ID, p.Name, p.IPAddress, p.Port, p.IsActive)

	if !p.IsActive {
		log.Printf("[PrintSale] printer %d is inactive", p.ID)
		return nil
	}

	data, err := s.buildSaleReceipt(ctx, saleID)
	if err != nil {
		return fmt.Errorf("build receipt: %w", err)
	}
	log.Printf("[PrintSale] sending %d bytes to %s:%d", len(data), p.IPAddress, p.Port)

	return SendToTCP(p.IPAddress, p.Port, data)
}

// buildSaleReceipt fetches sale + items + shop info and returns ESC/POS bytes.
func (s *Service) buildSaleReceipt(ctx context.Context, saleID int64) ([]byte, error) {
	var rd ReceiptData

	// Shop info from receipt_settings (use defaults if NULL)
	var logoPath string
	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(shop_name, ''), COALESCE(shop_address, ''), COALESCE(shop_phone, ''),
		       COALESCE(footer, ''), COALESCE(logo_path, '')
		FROM receipt_settings LIMIT 1
	`).Scan(&rd.ShopName, &rd.ShopAddress, &rd.ShopPhone, &rd.Footer, &logoPath)

	if rd.ShopName == "" {
		rd.ShopName = "Hezzet Market"
	}

	// Load and encode logo if present
	if logoPath != "" && s.uploadsDir != "" {
		fullPath := filepath.Join(s.uploadsDir, logoPath)
		if logoBytes, err := EncodeLogo(fullPath, maxWidth80mm); err == nil {
			rd.LogoBytes = logoBytes
		} else {
			log.Printf("[buildSaleReceipt] logo encode failed: %v", err)
		}
	}

	// Sale header — sale_number derived from id + date
	var cashierName *string
	var saleIDInt int64
	err := s.db.QueryRow(ctx, `
		SELECT s.id, s.created_at, s.total_cents, s.discount_cents, s.discount_percent, s.bonus_used_cents,
		       u.full_name
		FROM sales s
		LEFT JOIN users u ON u.id = s.created_by
		WHERE s.id = $1
	`, saleID).Scan(&saleIDInt, &rd.CreatedAt, &rd.TotalCents, &rd.DiscountCents, &rd.DiscountPct, &rd.BonusUsedCents, &cashierName)
	if err == nil {
		rd.SaleNumber = fmt.Sprintf("%s-%06d", rd.CreatedAt.Format("20060102"), saleIDInt)
	}
	if err != nil {
		return nil, err
	}
	if cashierName != nil {
		rd.CashierName = *cashierName
	}

	// Items
	rows, err := s.db.Query(ctx, `
		SELECT p.name, si.qty_milli, si.unit_price_cents, si.line_total_cents
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		WHERE si.sale_id = $1
		ORDER BY si.id
	`, saleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var it ReceiptItem
		if err := rows.Scan(&it.Name, &it.QtyMilli, &it.UnitPriceCents, &it.LineTotalCents); err != nil {
			return nil, err
		}
		rd.Items = append(rd.Items, it)
	}

	// Paid (sum of payments for this sale)
	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(pm.amount_cents), 0)
		FROM payments pm
		JOIN transactions t ON t.id = pm.transaction_id
		WHERE t.related_table = 'sale' AND t.related_id = $1
	`, saleID).Scan(&rd.PaidCents)

	return BuildReceipt(rd), nil
}

func buildTestPage(name, ip string) []byte {
	rd := ReceiptData{
		ShopName:   "TEST PRINTER",
		SaleNumber: "TEST-001",
		Items: []ReceiptItem{
			{Name: "Test item", QtyMilli: 1000, UnitPriceCents: 1000, LineTotalCents: 1000},
		},
		TotalCents: 1000,
		Footer:     fmt.Sprintf("Printer: %s (%s)", name, ip),
	}
	return BuildReceipt(rd)
}
