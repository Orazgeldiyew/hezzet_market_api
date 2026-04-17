package printer

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

// ESC/POS commands (standard bytes that most thermal printers understand)
var (
	cmdInit      = []byte{0x1B, 0x40}             // ESC @  - initialize
	cmdCut       = []byte{0x1D, 0x56, 0x42, 0x50} // GS V 66 80 - feed 80 dots + full cut
	cmdBold      = []byte{0x1B, 0x45, 0x01}       // ESC E 1
	cmdBoldOff   = []byte{0x1B, 0x45, 0x00}       // ESC E 0
	cmdAlignLeft = []byte{0x1B, 0x61, 0x00}       // ESC a 0
	cmdAlignCtr  = []byte{0x1B, 0x61, 0x01}       // ESC a 1
	cmdLineFeed  = []byte{0x0A}                   // LF
	cmdCharset   = []byte{0x1B, 0x74, 0x10}       // ESC t 16 - WCP1252 (Western)
)

const lineWidth = 32 // 58mm/80mm printers usually fit 32 chars

// ReceiptData is the minimal info needed to print a receipt.
type ReceiptData struct {
	ShopName       string
	ShopAddress    string
	ShopPhone      string
	SaleNumber     string
	CashierName    string
	CreatedAt      time.Time
	Items          []ReceiptItem
	TotalCents     int64
	DiscountCents  int64
	DiscountPct    int
	BonusUsedCents int64
	PaidCents      int64
	Footer         string
	LogoBytes      []byte // pre-encoded ESC/POS raster image (GS v 0 command)
}

type ReceiptItem struct {
	Name           string
	QtyMilli       int64
	UnitPriceCents int64
	LineTotalCents int64
}

// BuildReceipt returns ESC/POS byte stream for the receipt.
func BuildReceipt(d ReceiptData) []byte {
	var buf bytes.Buffer

	buf.Write(cmdInit)
	buf.Write(cmdCharset)

	// ── Logo (if provided) ──
	if len(d.LogoBytes) > 0 {
		buf.Write(cmdAlignCtr)
		buf.Write(d.LogoBytes)
		buf.Write(cmdLineFeed)
	}

	// ── Header: Shop name (bold, centered) ──
	buf.Write(cmdAlignCtr)
	buf.Write(cmdBold)
	buf.WriteString(ascii(d.ShopName) + "\n")
	buf.Write(cmdBoldOff)

	if d.ShopAddress != "" {
		buf.WriteString(ascii(d.ShopAddress) + "\n")
	}
	if d.ShopPhone != "" {
		buf.WriteString(ascii(d.ShopPhone) + "\n")
	}

	buf.WriteString(dashedLine() + "\n")

	// ── Meta block (centered, matches HTML design) ──
	buf.Write(cmdAlignCtr)
	buf.Write(cmdBold)
	buf.WriteString("Chek #" + d.SaleNumber + "\n")
	buf.Write(cmdBoldOff)
	buf.WriteString("Senesi: " + d.CreatedAt.Format("02.01.2006") + "\n")
	buf.WriteString("Wagt: " + d.CreatedAt.Format("15:04") + "\n")
	if d.CashierName != "" {
		buf.WriteString("Kassir: " + ascii(d.CashierName) + "\n")
	}
	buf.Write(cmdAlignLeft)
	buf.WriteString(dashedLine() + "\n")

	// ── Items table with borders: | Haryt | Sany | Baha | Jemi | ──
	// Column widths: name=12, qty=4, price=6, total=6 — total 32 chars with | and spaces
	// Format: |name        |qty |baha  |jemi  | = 1+12+1+4+1+6+1+6+1 = 33 → adjust to 32
	// Use: |name      |qty|baha  |jemi  | = 1+10+1+3+1+6+1+6+1+... tuning
	buf.WriteString(tableBorder() + "\n")
	buf.Write(cmdBold)
	buf.WriteString(tableRow("Haryt", "Sany", "Baha", "Jemi") + "\n")
	buf.Write(cmdBoldOff)
	buf.WriteString(tableBorder() + "\n")

	for _, it := range d.Items {
		qty := float64(it.QtyMilli) / 1000.0
		qtyStr := formatQty(qty)
		name := ascii(it.Name)

		// If name too long, split: first line name, second line values
		if len(name) > 12 {
			buf.WriteString(tableRow(name[:12], "", "", "") + "\n")
			name = name[12:]
			for len(name) > 12 {
				buf.WriteString(tableRow(name[:12], "", "", "") + "\n")
				name = name[12:]
			}
			buf.WriteString(tableRow(name, qtyStr, money(it.UnitPriceCents), money(it.LineTotalCents)) + "\n")
		} else {
			buf.WriteString(tableRow(name, qtyStr, money(it.UnitPriceCents), money(it.LineTotalCents)) + "\n")
		}
	}
	buf.WriteString(tableBorder() + "\n")
	buf.WriteString("\n")

	// ── Totals block ──
	// Row 1: Umumy jemi | Arz%
	buf.Write(cmdBold)
	buf.WriteString(twoColumnRow(
		"Umumy jemi: "+money(d.TotalCents)+" TMT",
		fmt.Sprintf("Arz%%: %d", d.DiscountPct),
	) + "\n")
	buf.Write(cmdBoldOff)

	// Row 2: Tolenen | Arz Muk
	if d.PaidCents > 0 {
		buf.WriteString(twoColumnRow(
			"Tolenen: "+money(d.PaidCents)+" TMT",
			"Arz Muk: "+money(d.DiscountCents),
		) + "\n")
	}

	// Bonus (if any)
	if d.BonusUsedCents > 0 {
		buf.WriteString(padBetween("Bonus", "-"+money(d.BonusUsedCents)+" TMT") + "\n")
	}

	buf.WriteString(dashedLine() + "\n")

	// Change (Gaytargy) — big bold
	if d.PaidCents > 0 {
		change := d.PaidCents - d.TotalCents
		if change > 0 {
			buf.Write(cmdBold)
			buf.WriteString("Gaytargy: " + money(change) + " TMT\n")
			buf.Write(cmdBoldOff)
			buf.WriteString(dashedLine() + "\n")
		}
	}

	// ── Footer (centered) ──
	buf.Write(cmdAlignCtr)
	buf.WriteString("\n")
	if d.Footer != "" {
		buf.WriteString(ascii(d.Footer) + "\n")
	} else {
		buf.WriteString("Satyn alanynyz uchin sag bolun!\n")
	}

	// Feed before cut
	buf.Write(cmdLineFeed)
	buf.Write(cmdLineFeed)
	buf.Write(cmdCut)

	return buf.Bytes()
}

// dashedLine returns visual divider (like HTML border-dashed)
func dashedLine() string {
	return strings.Repeat("-", lineWidth)
}

// Table layout: |name(12)|qty(4)|baha(6)|jemi(6)| = 1+12+1+4+1+6+1+6+1 = 33 → trim to 32
// Using narrower columns to fit lineWidth=32
// |name(11)|qty(4)|baha(6)|jemi(6)| = 1+11+1+4+1+6+1+6+1 = 32 ✓

// tableBorder returns "+-----+---+------+------+"
func tableBorder() string {
	return "+" + strings.Repeat("-", 11) + "+" + strings.Repeat("-", 4) + "+" +
		strings.Repeat("-", 6) + "+" + strings.Repeat("-", 6) + "+"
}

// tableRow returns "|name       |qty |baha  |jemi  |"
func tableRow(name, qty, price, total string) string {
	return "|" + padTruncate(name, 11) + "|" +
		padTruncateRight(qty, 4) + "|" +
		padTruncateRight(price, 6) + "|" +
		padTruncateRight(total, 6) + "|"
}

// padTruncate — left-align, pad/truncate to exact width
func padTruncate(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// padTruncateRight — right-align, pad/truncate to exact width
func padTruncateRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return strings.Repeat(" ", width-len(s)) + s
}

// twoColumnRow splits lineWidth in half, left label on left, right label on right
func twoColumnRow(left, right string) string {
	spaces := lineWidth - len(left) - len(right)
	if spaces < 1 {
		spaces = 1
	}
	return left + strings.Repeat(" ", spaces) + right
}

// formatQty: show as integer if whole (e.g. "1"), else 3 decimals ("1.500")
func formatQty(qty float64) string {
	if qty == float64(int64(qty)) {
		return fmt.Sprintf("%d", int64(qty))
	}
	return fmt.Sprintf("%.3f", qty)
}

// money formats cents as "123.45"
func money(cents int64) string {
	neg := ""
	if cents < 0 {
		neg = "-"
		cents = -cents
	}
	whole := cents / 100
	frac := cents % 100
	return fmt.Sprintf("%s%d.%02d", neg, whole, frac)
}

// padBetween fills with spaces so total width equals lineWidth.
func padBetween(left, right string) string {
	spaces := lineWidth - len(left) - len(right)
	if spaces < 1 {
		spaces = 1
	}
	return left + strings.Repeat(" ", spaces) + right
}

// ascii transliterates Turkmen special characters to ASCII approximations.
func ascii(s string) string {
	replacer := strings.NewReplacer(
		"ä", "a", "Ä", "A",
		"ç", "ch", "Ç", "Ch",
		"ň", "n", "Ň", "N",
		"ö", "o", "Ö", "O",
		"ş", "sh", "Ş", "Sh",
		"ü", "u", "Ü", "U",
		"ý", "y", "Ý", "Y",
		"ž", "zh", "Ž", "Zh",
		"ı", "i", "İ", "I",
		"ğ", "g", "Ğ", "G",
		"й", "y",
		"№", "#",
	)
	return replacer.Replace(s)
}
