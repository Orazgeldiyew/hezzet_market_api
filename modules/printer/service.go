package printer

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/modules/sale"
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

// buildSaleReceipt fetches sale data, renders the HTML receipt template,
// then converts it to ESC/POS raster bytes (so the thermal print matches the on-screen design).
func (s *Service) buildSaleReceipt(ctx context.Context, saleID int64) ([]byte, error) {
	// ── Shop / receipt settings (all fields for HTML template) ──
	var shopName, shopAddress, shopPhone, footer, logoPath, customTemplate string
	var logoWidth, logoHeight *string
	var fsHeader, fsItems, fsMeta, fsTotal int
	if err := s.db.QueryRow(ctx, `
		SELECT COALESCE(shop_name, 'Hezzet Market'), COALESCE(shop_address, ''), COALESCE(shop_phone, ''),
		       COALESCE(footer, 'Satyn alanyňyz üçin sag boluň!'),
		       COALESCE(logo_path, ''), COALESCE(template, ''),
		       logo_width, logo_height,
		       font_size_header, font_size_items, font_size_meta, font_size_total
		FROM receipt_settings LIMIT 1
	`).Scan(&shopName, &shopAddress, &shopPhone, &footer, &logoPath, &customTemplate, &logoWidth, &logoHeight,
		&fsHeader, &fsItems, &fsMeta, &fsTotal); err != nil {
		// Don't fail the print on a missing settings row — fall back to
		// in-code defaults — but surface the error in the log so an operator
		// can see "why is my shop name blank on receipts".
		log.Printf("[Printer] receipt_settings query failed, using defaults: %v", err)
	}

	// Sale header
	var cashierName, warehouseName, customerName, workerName, paymentMethod, note *string
	var saleIDInt int64
	var totalCents, discountCents, bonusUsedCents, paidCents int64
	var discountPercent int
	var createdAt time.Time

	if err := s.db.QueryRow(ctx, `
		SELECT s.id, s.created_at, s.total_cents, s.discount_cents, s.discount_percent, s.bonus_used_cents,
		       u.full_name, w.name, c.name, wk.name, s.note,
		       (SELECT pt.name FROM payments p
		        JOIN transactions t ON t.id = p.transaction_id
		        JOIN payment_types pt ON pt.id = p.payment_type_id
		        WHERE t.related_table = 'sale' AND t.related_id = s.id
		        ORDER BY p.id DESC LIMIT 1),
		       COALESCE((SELECT SUM(p.amount_cents) FROM payments p
		        JOIN transactions t ON t.id = p.transaction_id
		        WHERE t.related_table = 'sale' AND t.related_id = s.id), 0)
		FROM sales s
		LEFT JOIN users u ON u.id = s.created_by
		LEFT JOIN warehouses w ON w.id = s.warehouse_id
		LEFT JOIN customers c ON c.id = s.customer_id
		LEFT JOIN workers wk ON wk.id = s.worker_id AND wk.deleted_at IS NULL
		WHERE s.id = $1
	`, saleID).Scan(&saleIDInt, &createdAt, &totalCents, &discountCents, &discountPercent, &bonusUsedCents,
		&cashierName, &warehouseName, &customerName, &workerName, &note, &paymentMethod, &paidCents); err != nil {
		return nil, fmt.Errorf("fetch sale: %w", err)
	}

	// Items
	rows, err := s.db.Query(ctx, `
		SELECT p.name, si.qty_milli, p.unit_type, si.unit_price_cents, si.line_total_cents, si.discount_percent
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		WHERE si.sale_id = $1
		ORDER BY si.id
	`, saleID)
	if err != nil {
		return nil, fmt.Errorf("fetch items: %w", err)
	}
	defer rows.Close()

	var items []sale.ReceiptItem
	var subtotalCents int64
	for rows.Next() {
		var it sale.ReceiptItem
		if err := rows.Scan(&it.ProductName, &it.QtyMilli, &it.UnitType, &it.UnitPriceCents, &it.LineTotalCents, &it.DiscountPercent); err != nil {
			return nil, err
		}
		items = append(items, it)
		subtotalCents += (it.QtyMilli*it.UnitPriceCents + 500) / 1000
	}
	totalDiscountCents := subtotalCents - totalCents - bonusUsedCents
	if totalDiscountCents < 0 {
		totalDiscountCents = 0
	}
	totalDiscountPercent := 0
	if subtotalCents > 0 {
		totalDiscountPercent = int(totalDiscountCents * 100 / subtotalCents)
	}

	// Logo URL — use file:// so headless Chrome can load it
	var logoURL string
	if logoPath != "" && s.uploadsDir != "" {
		abs, _ := filepath.Abs(filepath.Join(s.uploadsDir, logoPath))
		logoURL = "file://" + abs
	}

	changeCents := paidCents - totalCents
	if changeCents < 0 {
		changeCents = 0
	}

	data := sale.ReceiptData{
		ShopName:             shopName,
		ShopAddress:          shopAddress,
		ShopPhone:            shopPhone,
		LogoURL:              logoURL,
		LogoWidth:            derefStr(logoWidth),
		LogoHeight:           derefStr(logoHeight),
		Footer:               footer,
		SaleID:               saleIDInt,
		ReceiptNumber:        fmt.Sprintf("%s-%06d", createdAt.Format("20060102"), saleIDInt),
		Date:                 createdAt.Format("02.01.2006"),
		Time:                 createdAt.Format("15:04"),
		CashierName:          derefStr(cashierName),
		WarehouseName:        derefStr(warehouseName),
		CustomerName:         derefStr(customerName),
		WorkerName:           derefStr(workerName),
		PaymentMethod:        derefStr(paymentMethod),
		Note:                 derefStr(note),
		TotalCents:           totalCents,
		BonusUsedCents:       bonusUsedCents,
		DiscountPercent:      discountPercent,
		DiscountCents:        discountCents,
		SubtotalCents:        subtotalCents,
		TotalDiscountCents:   totalDiscountCents,
		TotalDiscountPercent: totalDiscountPercent,
		PaidCents:            paidCents,
		ChangeCents:          changeCents,
		Items:                items,
	}

	html, err := sale.RenderReceipt(data, customTemplate)
	if err != nil {
		return nil, fmt.Errorf("render html: %w", err)
	}

	// Resolve absolute logo path so the printer can embed it at full resolution
	var absLogoPath string
	if logoPath != "" && s.uploadsDir != "" {
		absLogoPath, _ = filepath.Abs(filepath.Join(s.uploadsDir, logoPath))
	}

	log.Printf("[buildSaleReceipt] rendering HTML→image for saleID=%d (html size=%d, logo=%q)", saleID, len(html), absLogoPath)
	return BuildFullReceipt(absLogoPath, html, maxWidth80mm, FontSizes{
		Header: fsHeader,
		Items:  fsItems,
		Meta:   fsMeta,
		Total:  fsTotal,
	})
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
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
