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
	ShopName       string
	ShopAddress    string
	ShopPhone      string
	LogoURL        string
	Footer         string
	SaleID         int64
	Date           string
	Status         string
	CashierName    string
	WarehouseName  string
	CustomerName   string
	TotalCents     int64
	BonusUsedCents int64
	WorkerName     string
	Note           string
	Items          []ReceiptItem
}

// ReceiptItem is a line item for the receipt template.
type ReceiptItem struct {
	ProductName    string
	QtyMilli       int64
	UnitType       string
	UnitPriceCents int64
	LineTotalCents int64
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
			return "кг"
		case "volume":
			return "л"
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
    hr { border: none; border-top: 1px dashed #000; margin: 4px 0; }
    .meta { font-size: 11px; margin: 2px 0; }
    table { width: 100%; border-collapse: collapse; margin: 4px 0; }
    th, td { padding: 1px 0; font-size: 11px; text-align: left; vertical-align: top; }
    .r { text-align: right; }
    .total-line { display: flex; justify-content: space-between; font-size: 14px; font-weight: bold; margin: 4px 0; }
    .footer { font-size: 10px; color: #555; margin-top: 6px; }
    @media print {
      body { width: 80mm; margin: 0; padding: 2mm; }
      @page { size: 80mm auto; margin: 0; }
    }
  </style>
</head>
<body>
  {{if .LogoURL}}<div class="center"><img src="{{.LogoURL}}" style="max-width:50mm;max-height:20mm"></div>{{end}}
  <div class="center header">{{.ShopName}}</div>
  {{if .ShopAddress}}<div class="center info">{{.ShopAddress}}</div>{{end}}
  {{if .ShopPhone}}<div class="center info">Тел: {{.ShopPhone}}</div>{{end}}

  <hr>
  <div class="meta"><b>Чек #{{printf "%05d" .SaleID}}</b></div>
  <div class="meta">Дата: {{.Date}}</div>
  <div class="meta">Кассир: {{.CashierName}}</div>
  {{if .WarehouseName}}<div class="meta">Склад: {{.WarehouseName}}</div>{{end}}
  {{if .CustomerName}}<div class="meta">Клиент: {{.CustomerName}}</div>{{end}}

  <hr>
  <table>
    <tr><th>Товар</th><th class="r">Кол</th><th class="r">Цена</th><th class="r">Сумма</th></tr>
    {{range .Items}}
    <tr>
      <td>{{.ProductName}}</td>
      <td class="r">{{qty .QtyMilli .UnitType}}{{with unit .UnitType}} {{.}}{{end}}</td>
      <td class="r">{{money .UnitPriceCents}}</td>
      <td class="r">{{money .LineTotalCents}}</td>
    </tr>
    {{end}}
  </table>

  <hr>
  <div class="total-line"><span>ИТОГО:</span><span>{{money .TotalCents}} TMT</span></div>
  {{if gt .BonusUsedCents 0}}<div class="meta">Бонус: -{{money .BonusUsedCents}} TMT</div>{{end}}
  {{if .WorkerName}}<div class="meta">Работник (кредит): {{.WorkerName}}</div>{{end}}
  {{if .Note}}<div class="meta">Примечание: {{.Note}}</div>{{end}}

  {{if .Footer}}<hr><div class="center footer">{{.Footer}}</div>{{end}}

  <script>window.onload=function(){window.print();}</script>
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
		for i, it := range items {
			receiptItems[i] = ReceiptItem{
				ProductName:    it.ProductName,
				QtyMilli:       it.QtyMilli,
				UnitType:       it.UnitType,
				UnitPriceCents: it.UnitPriceCents,
				LineTotalCents: it.LineTotalCents,
			}
		}

		data := ReceiptData{
			ShopName:       settings.ShopName,
			ShopAddress:    settings.ShopAddress,
			ShopPhone:      settings.ShopPhone,
			LogoURL:        logoURL,
			Footer:         settings.Footer,
			SaleID:         saleRow.ID,
			Date:           saleRow.CreatedAt.Format("02.01.2006 15:04"),
			Status:         string(saleRow.Status),
			CashierName:    saleRow.CashierName,
			WarehouseName:  saleRow.WarehouseName,
			CustomerName:   saleRow.CustomerName,
			TotalCents:     saleRow.TotalCents,
			BonusUsedCents: saleRow.BonusUsedCents,
			WorkerName:     saleRow.WorkerName,
			Note:           note,
			Items:          receiptItems,
		}

		html, err := RenderReceipt(data, settings.Template)
		if err != nil {
			c.Error(apperr.Internal(err))
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	}
}
