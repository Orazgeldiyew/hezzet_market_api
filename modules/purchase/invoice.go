package purchase

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Orazgeldiyew/hezzet_market_backend/modules/receiptsettings"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
)

// InvoiceData is the view model fed into the A4 invoice template.
type InvoiceData struct {
	// Document metadata
	InvoiceNumber string // PO-YYYY-NNNNNN
	Date          string // dd.mm.yyyy
	Status        string // draft, received, cancelled

	// Buyer (the shop)
	BuyerName    string
	BuyerLegal   string
	BuyerTaxID   string
	BuyerAddress string
	BuyerPhone   string

	// Supplier
	SupplierName    string
	SupplierLegal   string
	SupplierTaxID   string
	SupplierAddress string
	SupplierPhone   string
	SupplierEmail   string

	// Receiving warehouse + people
	WarehouseName string
	CreatedBy     string
	ReceivedBy    string
	Note          string

	// Items
	Items      []InvoiceItem
	TotalCents int64
}

type InvoiceItem struct {
	Idx            int
	ProductName    string
	QtyMilli       int64
	UnitType       string
	UnitCostCents  int64
	LineTotalCents int64
}

// invoiceTmplFuncs gives the template the same money/qty helpers as the receipt.
var invoiceTmplFuncs = template.FuncMap{
	"money": func(c int64) string {
		whole := c / 100
		frac := c % 100
		if frac < 0 {
			frac = -frac
		}
		return fmt.Sprintf("%d.%02d", whole, frac)
	},
	"qty": func(m int64, unit string) string {
		switch unit {
		case "weight", "volume":
			whole := m / 1000
			frac := m % 1000
			if frac < 0 {
				frac = -frac
			}
			return fmt.Sprintf("%d.%03d", whole, frac)
		default:
			return fmt.Sprintf("%d", m/1000)
		}
	},
	"unit": func(unit string) string {
		switch unit {
		case "weight":
			return "kg"
		case "volume":
			return "l"
		default:
			return "şt"
		}
	},
}

const invoiceTemplate = `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>{{.InvoiceNumber}}</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: Arial, sans-serif; font-size: 12pt; color: #000; padding: 20mm 15mm; }
.title { text-align: center; font-size: 18pt; font-weight: bold; margin-bottom: 4mm; }
.meta { text-align: center; font-size: 11pt; margin-bottom: 8mm; }
.parties { display: flex; gap: 8mm; margin-bottom: 8mm; }
.party { flex: 1; border: 1px solid #000; padding: 4mm; }
.party h3 { font-size: 11pt; margin-bottom: 2mm; text-transform: uppercase; }
.party .line { margin: 1mm 0; }
.party .label { color: #555; font-size: 9pt; }
.party .val { font-weight: bold; }
table.items { width: 100%; border-collapse: collapse; margin-top: 4mm; }
table.items th, table.items td { border: 1px solid #000; padding: 3mm 2mm; font-size: 10pt; }
table.items th { background: #f0f0f0; }
table.items td.num { text-align: right; white-space: nowrap; }
table.items td.idx { text-align: center; width: 8mm; }
.total-row { font-weight: bold; background: #f8f8f8; }
.signatures { display: flex; gap: 20mm; margin-top: 15mm; }
.signature { flex: 1; }
.signature .name { margin-bottom: 8mm; font-size: 10pt; }
.signature .line { border-top: 1px solid #000; padding-top: 1mm; font-size: 9pt; color: #555; text-align: center; }
.footer-note { font-size: 9pt; color: #555; margin-top: 10mm; }
.status-badge { display: inline-block; padding: 1mm 3mm; border-radius: 2mm; font-size: 9pt; font-weight: bold; text-transform: uppercase; }
.status-received { background: #d4f7d4; color: #060; border: 1px solid #060; }
.status-draft { background: #fff3cd; color: #856404; border: 1px solid #856404; }
.status-cancelled { background: #f8d7da; color: #721c24; border: 1px solid #721c24; }
@media print {
  body { padding: 15mm 12mm; }
  @page { size: A4; margin: 0; }
}
</style>
</head>
<body>

<div class="title">PRIHODNAÝA NAKLADNAÝA</div>
<div class="meta">
  № <b>{{.InvoiceNumber}}</b> &nbsp;|&nbsp; {{.Date}}
  &nbsp;|&nbsp;
  <span class="status-badge status-{{.Status}}">{{.Status}}</span>
</div>

<div class="parties">
  <div class="party">
    <h3>Iberiji (Supplier)</h3>
    <div class="line"><span class="val">{{.SupplierName}}</span></div>
    {{if .SupplierLegal}}<div class="line">{{.SupplierLegal}}</div>{{end}}
    {{if .SupplierTaxID}}<div class="line"><span class="label">TIN:</span> {{.SupplierTaxID}}</div>{{end}}
    {{if .SupplierPhone}}<div class="line"><span class="label">Tel:</span> {{.SupplierPhone}}</div>{{end}}
    {{if .SupplierEmail}}<div class="line"><span class="label">Email:</span> {{.SupplierEmail}}</div>{{end}}
    {{if .SupplierAddress}}<div class="line"><span class="label">Salgy:</span> {{.SupplierAddress}}</div>{{end}}
  </div>

  <div class="party">
    <h3>Alyjy (Buyer)</h3>
    <div class="line"><span class="val">{{.BuyerName}}</span></div>
    {{if .BuyerLegal}}<div class="line">{{.BuyerLegal}}</div>{{end}}
    {{if .BuyerTaxID}}<div class="line"><span class="label">TIN:</span> {{.BuyerTaxID}}</div>{{end}}
    {{if .BuyerPhone}}<div class="line"><span class="label">Tel:</span> {{.BuyerPhone}}</div>{{end}}
    {{if .BuyerAddress}}<div class="line"><span class="label">Salgy:</span> {{.BuyerAddress}}</div>{{end}}
    {{if .WarehouseName}}<div class="line"><span class="label">Ammar:</span> {{.WarehouseName}}</div>{{end}}
  </div>
</div>

<table class="items">
  <thead>
    <tr>
      <th>№</th>
      <th>Harydyň ady</th>
      <th>Mukdary</th>
      <th>Birlik</th>
      <th>Bahasy</th>
      <th>Jemi</th>
    </tr>
  </thead>
  <tbody>
    {{range .Items}}
    <tr>
      <td class="idx">{{.Idx}}</td>
      <td>{{.ProductName}}</td>
      <td class="num">{{qty .QtyMilli .UnitType}}</td>
      <td>{{unit .UnitType}}</td>
      <td class="num">{{money .UnitCostCents}}</td>
      <td class="num">{{money .LineTotalCents}}</td>
    </tr>
    {{end}}
    <tr class="total-row">
      <td colspan="5" style="text-align:right;">JEMI:</td>
      <td class="num">{{money .TotalCents}} TMT</td>
    </tr>
  </tbody>
</table>

<div class="signatures">
  <div class="signature">
    <div class="name">{{if .CreatedBy}}{{.CreatedBy}}{{else}}—{{end}}</div>
    <div class="line">Iberen (Передал)</div>
  </div>
  <div class="signature">
    <div class="name">{{if .ReceivedBy}}{{.ReceivedBy}}{{else}}_______________{{end}}</div>
    <div class="line">Kabul eden (Принял)</div>
  </div>
</div>

{{if .Note}}<div class="footer-note"><b>Bellik:</b> {{.Note}}</div>{{end}}

<script>window.onload=function(){window.print();}</script>
</body>
</html>`

// PrintInvoiceHandler renders the A4 invoice HTML for a given PO. Designed to
// be opened in a new browser tab — window.print() fires on load.
func PrintInvoiceHandler(db *pgxpool.Pool, receiptRepo *receiptsettings.Repository) gin.HandlerFunc {
	tmpl, err := template.New("invoice").Funcs(invoiceTmplFuncs).Parse(invoiceTemplate)
	if err != nil {
		panic(fmt.Errorf("invoice template parse: %w", err))
	}

	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.Error(apperr.Validation("invalid purchase id"))
			return
		}

		data, err := loadInvoiceData(c.Request.Context(), db, receiptRepo, id)
		if err != nil {
			if err == pgx.ErrNoRows {
				c.Error(apperr.NotFound("PO_NOT_FOUND", "purchase order not found"))
				return
			}
			c.Error(apperr.Internal(err))
			return
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			c.Error(apperr.Internal(err))
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
	}
}

func loadInvoiceData(ctx context.Context, db *pgxpool.Pool, receiptRepo *receiptsettings.Repository, poID int64) (InvoiceData, error) {
	var d InvoiceData

	// PO header + supplier + warehouse + creator/receiver names
	var createdAt time.Time
	var status string
	var totalCents int64
	var note, supName, supLegal, supTaxID, supPhone, supEmail, supAddr, whName, creatorName, receiverName *string

	err := db.QueryRow(ctx, `
		SELECT po.created_at, po.status, po.total_cents, po.note,
		       s.name, COALESCE(s.legal_name,''), COALESCE(s.tax_id,''), s.phone, s.email, s.address,
		       w.name,
		       creator.name,
		       receiver.name
		FROM purchase_orders po
		LEFT JOIN suppliers s        ON s.id = po.supplier_id
		LEFT JOIN warehouses w       ON w.id = po.warehouse_id
		LEFT JOIN employees creator      ON creator.id = po.created_by
		LEFT JOIN employees receiver     ON receiver.id = po.received_by
		WHERE po.id = $1
	`, poID).Scan(&createdAt, &status, &totalCents, &note,
		&supName, &supLegal, &supTaxID, &supPhone, &supEmail, &supAddr,
		&whName, &creatorName, &receiverName)
	if err != nil {
		return d, err
	}

	// Items
	rows, err := db.Query(ctx, `
		SELECT p.name, p.unit_type, poi.qty_milli, poi.unit_cost_cents, poi.line_total_cents
		FROM purchase_order_items poi
		JOIN products p ON p.id = poi.product_id
		WHERE poi.po_id = $1
		ORDER BY poi.id
	`, poID)
	if err != nil {
		return d, err
	}
	defer rows.Close()

	var items []InvoiceItem
	idx := 1
	for rows.Next() {
		var it InvoiceItem
		if err := rows.Scan(&it.ProductName, &it.UnitType, &it.QtyMilli, &it.UnitCostCents, &it.LineTotalCents); err != nil {
			return d, err
		}
		it.Idx = idx
		idx++
		items = append(items, it)
	}

	// Receipt settings → buyer block on the invoice.
	settings, err := receiptRepo.Get(ctx)
	if err != nil {
		return d, err
	}

	d.InvoiceNumber = fmt.Sprintf("PO-%d-%06d", createdAt.Year(), poID)
	d.Date = createdAt.Format("02.01.2006")
	d.Status = status
	d.TotalCents = totalCents
	d.Items = items
	d.Note = derefS(note)

	d.SupplierName = derefS(supName)
	d.SupplierLegal = derefS(supLegal)
	d.SupplierTaxID = derefS(supTaxID)
	d.SupplierPhone = derefS(supPhone)
	d.SupplierEmail = derefS(supEmail)
	d.SupplierAddress = derefS(supAddr)
	d.WarehouseName = derefS(whName)
	d.CreatedBy = derefS(creatorName)
	d.ReceivedBy = derefS(receiverName)

	d.BuyerName = settings.ShopName
	d.BuyerLegal = settings.LegalName
	d.BuyerTaxID = settings.TaxID
	d.BuyerPhone = settings.ShopPhone
	d.BuyerAddress = settings.ShopAddress

	return d, nil
}

func derefS(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
