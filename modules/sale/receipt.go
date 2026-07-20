package sale

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/modules/receiptsettings"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

// ReceiptData holds all variables available in receipt templates.
type ReceiptData struct {
	ShopName        string
	ShopAddress     string
	ShopPhone       string
	LogoURL         string
	LogoWidth       string
	LogoHeight      string
	Footer          string
	SaleID          int64
	ReceiptNumber   string // YYYYMMDD-NNNNNN
	Date            string
	Time            string
	Status          string
	CashierName     string
	WarehouseName   string
	CustomerName    string
	TotalCents           int64
	BonusUsedCents       int64
	DiscountPercent      int   // sale-level only (legacy)
	DiscountCents        int64 // sale-level only (legacy)
	SubtotalCents        int64 // before any discount (item + sale-level)
	TotalDiscountCents   int64 // combined item + sale discount amount
	TotalDiscountPercent int   // combined discount as a percentage of subtotal
	PaymentMethod        string
	PaidCents            int64
	ChangeCents          int64
	WorkerName      string
	Note            string
	Items           []ReceiptItem
}

// ReceiptItem is a line item for the receipt template.
type ReceiptItem struct {
	ProductName     string
	QtyMilli        int64
	UnitType        string
	UnitPriceCents  int64
	LineTotalCents  int64
	DiscountPercent int
}

// templateFuncs provides money/qty/unit helpers for receipt templates.
var templateFuncs = template.FuncMap{
	"money": func(cents int64) string {
		whole := cents / 100
		frac := cents % 100
		if frac < 0 {
			frac = -frac
		}
		return fmt.Sprintf("%d.%02d", whole, frac)
	},
	"qty": func(milli int64, unitType string) string {
		switch unitType {
		case "weight", "volume":
			whole := milli / 1000
			frac := milli % 1000
			if frac < 0 {
				frac = -frac
			}
			return fmt.Sprintf("%d.%03d", whole, frac)
		default: // piece
			return fmt.Sprintf("%d", milli/1000)
		}
	},
	"unit": func(unitType string) string {
		switch unitType {
		case "weight":
			return "kg"
		case "volume":
			return "l"
		default:
			return ""
		}
	},
}

// defaultReceiptTemplate is an 80mm thermal printer receipt.
const defaultReceiptTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: 'Courier New', monospace; width: 80mm; margin: 0 auto; padding: 3mm; font-size: 12px; }
    .center { text-align: center; }
    .header { font-size: 16px; font-weight: bold; margin-bottom: 2px; }
    .info { font-size: 11px; color: #333; }
    .sep { border-top: 1px solid #000; margin: 6px 0; }
    .meta-table { width: 100%; font-size: 11px; margin: 4px 0; border-collapse: collapse; }
    .meta-table td { padding: 1px 0; }
    .meta-table td:last-child { text-align: right; font-weight: bold; }
    .items-table { width: 100%; border-collapse: collapse; font-size: 11px; margin: 6px 0; }
    .items-table th, .items-table td { border: 1px solid #000; padding: 4px 6px; }
    .items-table th { text-align: left; font-weight: bold; }
    .items-table td.num { text-align: right; white-space: nowrap; }
    .totals { font-size: 11px; margin: 6px 0; }
    .totals-row { display: flex; justify-content: space-between; margin: 2px 0; }
    .change { font-size: 13px; font-weight: bold; margin: 6px 0; }
    .extra { font-size: 11px; margin: 2px 0; }
    .footer { font-size: 11px; color: #333; margin-top: 8px; }
    @media print {
      body { width: 80mm; margin: 0; padding: 2mm; }
      @page { size: 80mm auto; margin: 0; }
    }
  </style>
</head>
<body>
  {{if .LogoURL}}<div class="center"><img src="{{.LogoURL}}" style="max-width:{{if .LogoWidth}}{{.LogoWidth}}{{else}}70px{{end}};max-height:{{if .LogoHeight}}{{.LogoHeight}}{{else}}30px{{end}}"></div>{{end}}
  <div class="center header">{{.ShopName}}</div>
  {{if .ShopAddress}}<div class="center info">{{.ShopAddress}}</div>{{end}}
  {{if .ShopPhone}}<div class="center info">{{.ShopPhone}}</div>{{end}}

  <div class="sep"></div>

  <table class="meta-table">
    <tr><td>Çek №</td><td>{{.ReceiptNumber}}</td></tr>
    <tr><td>Senesi</td><td>{{.Date}}</td></tr>
    <tr><td>Wagt</td><td>{{.Time}}</td></tr>
    <tr><td>Kassir</td><td>{{.CashierName}}</td></tr>
    {{if .WarehouseName}}<tr><td>Ammar</td><td>{{.WarehouseName}}</td></tr>{{end}}
    {{if .CustomerName}}<tr><td>Müşderi</td><td>{{.CustomerName}}</td></tr>{{end}}
  </table>

  <table class="items-table">
    <tr><th>Haryt</th><th>Sany</th><th>Baha</th><th>Jemi</th></tr>
    {{range .Items}}
    <tr>
      <td>{{.ProductName}}{{if gt .DiscountPercent 0}} (-{{.DiscountPercent}}%){{end}}</td>
      <td class="num">{{qty .QtyMilli .UnitType}}{{with unit .UnitType}} {{.}}{{end}}</td>
      <td class="num">{{money .UnitPriceCents}}</td>
      <td class="num">{{money .LineTotalCents}}</td>
    </tr>
    {{end}}
  </table>

  <div class="totals">
    <div class="totals-row"><span><b>Umumy jemi: {{money .TotalCents}} TMT</b></span><span>Arz%: {{.TotalDiscountPercent}}</span></div>
    <div class="totals-row"><span>Tölenen: {{money .PaidCents}} TMT</span><span>Arz Muk: {{money .TotalDiscountCents}} TMT</span></div>
  </div>

  <div class="change">Gaýtargy: {{money .ChangeCents}} TMT</div>

  {{if .PaymentMethod}}<div class="extra">Töleg: {{.PaymentMethod}}</div>{{end}}
  {{if gt .BonusUsedCents 0}}<div class="extra">Bonus: -{{money .BonusUsedCents}} TMT</div>{{end}}
  {{if .WorkerName}}<div class="extra">Işgär (karz): {{.WorkerName}}</div>{{end}}
  {{if .Note}}<div class="extra">Bellik: {{.Note}}</div>{{end}}

  {{if .Footer}}<div class="sep"></div><div class="center footer">{{.Footer}}</div>{{end}}

  <script>
    // Auto-print only when opened directly with ?autoprint=1 (the frontend's
    // window.open fallback). When the HTML is embedded by the till app —
    // Electron silent print or the hidden-iframe path — the frontend drives
    // the print itself and this must stay quiet, otherwise a second print
    // dialog pops over the silent job.
    if (window.location.search.indexOf('autoprint') !== -1) {
      window.onload = function () { window.print() }
    }
  </script>
</body>
</html>`

// RenderReceipt renders receipt HTML using the custom template or the default one.
func RenderReceipt(data ReceiptData, customTemplate string) (string, error) {
	tmplStr := defaultReceiptTemplate
	if strings.TrimSpace(customTemplate) != "" {
		tmplStr = customTemplate
	}

	t, err := template.New("receipt").Funcs(templateFuncs).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("template parse: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execute: %w", err)
	}
	return buf.String(), nil
}

// ReceiptHandler returns a gin.HandlerFunc for the public /receipt/:id route.
func ReceiptHandler(db *pgxpool.Pool, baseURL string, receiptRepo *receiptsettings.Repository) gin.HandlerFunc {
	repo := NewRepository(db, baseURL, nil)

	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.Error(apperr.Validation("invalid sale id"))
			return
		}

		ctx := c.Request.Context()

		saleRow, items, err := repo.GetReceiptData(ctx, id)
		if err != nil {
			if err == pgx.ErrNoRows {
				c.Error(apperr.NotFound("SALE_NOT_FOUND", "sale not found"))
				return
			}
			c.Error(apperr.Internal(err))
			return
		}

		settings, err := receiptRepo.Get(ctx)
		if err != nil {
			c.Error(apperr.Internal(err))
			return
		}

		var logoURL string
		if settings.LogoPath != "" && baseURL != "" {
			logoURL = baseURL + "/uploads/" + settings.LogoPath
		}

		var note string
		if saleRow.Note != nil {
			note = *saleRow.Note
		}

		receiptItems := make([]ReceiptItem, len(items))
		var subtotalCents int64
		for i, it := range items {
			receiptItems[i] = ReceiptItem{
				ProductName:     it.ProductName,
				QtyMilli:        it.QtyMilli,
				UnitType:        it.UnitType,
				UnitPriceCents:  it.UnitPriceCents,
				LineTotalCents:  it.LineTotalCents,
				DiscountPercent: it.DiscountPercent,
			}
			subtotalCents += (it.QtyMilli*it.UnitPriceCents + 500) / 1000
		}
		totalDiscountCents := subtotalCents - saleRow.TotalCents - saleRow.BonusUsedCents
		if totalDiscountCents < 0 {
			totalDiscountCents = 0
		}
		totalDiscountPercent := 0
		if subtotalCents > 0 {
			totalDiscountPercent = int(totalDiscountCents * 100 / subtotalCents)
		}

		receiptNumber := saleRow.CreatedAt.Format("20060102") + "-" + fmt.Sprintf("%06d", saleRow.ID)

		data := ReceiptData{
			ShopName:             settings.ShopName,
			ShopAddress:          settings.ShopAddress,
			ShopPhone:            settings.ShopPhone,
			LogoURL:              logoURL,
			LogoWidth:            settings.LogoWidth,
			LogoHeight:           settings.LogoHeight,
			Footer:               settings.Footer,
			SaleID:               saleRow.ID,
			ReceiptNumber:        receiptNumber,
			Date:                 saleRow.CreatedAt.Format("02.01.2006"),
			Time:                 saleRow.CreatedAt.Format("15:04"),
			Status:               string(saleRow.Status),
			CashierName:          saleRow.CashierName,
			WarehouseName:        saleRow.WarehouseName,
			CustomerName:         saleRow.CustomerName,
			TotalCents:           saleRow.TotalCents,
			BonusUsedCents:       saleRow.BonusUsedCents,
			DiscountPercent:      saleRow.DiscountPercent,
			DiscountCents:        saleRow.DiscountCents,
			SubtotalCents:        subtotalCents,
			TotalDiscountCents:   totalDiscountCents,
			TotalDiscountPercent: totalDiscountPercent,
			PaymentMethod:        saleRow.PaymentMethod,
			PaidCents:            saleRow.PaidCents,
			ChangeCents:          max(saleRow.PaidCents-saleRow.TotalCents, 0),
			WorkerName:           saleRow.WorkerName,
			Note:                 note,
			Items:                receiptItems,
		}

		html, err := RenderReceipt(data, settings.Template)
		if err != nil {
			c.Error(apperr.Internal(err))
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	}
}
