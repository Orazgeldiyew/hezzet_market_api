package printer

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// FontSizes overrides the default thermal-receipt font sizes (in pixels)
// applied by the CSS injected before Chrome renders. Zero means "use default".
type FontSizes struct {
	Header int // shop name
	Items  int // items table cells
	Meta   int // cashier/date/receipt# table
	Total  int // change line ("Gaýtargy")
}

// Defaults bumped down by 1px each — operator wanted a tighter receipt that
// fits more lines on 80mm tape without manual font tweaking.
func (f FontSizes) headerOrDefault() int { if f.Header > 0 { return f.Header }; return 31 }
func (f FontSizes) itemsOrDefault() int  { if f.Items > 0  { return f.Items  }; return 24 }
func (f FontSizes) metaOrDefault() int   { if f.Meta > 0   { return f.Meta   }; return 22 }
func (f FontSizes) totalOrDefault() int  { if f.Total > 0  { return f.Total  }; return 29 }

// RenderHTMLToImage uses headless Chrome to render HTML to a PNG image
// sized to fit a thermal printer (widthPx pixels wide).
func RenderHTMLToImage(html string, widthPx int, fonts FontSizes) (image.Image, error) {
	tmpDir, err := os.MkdirTemp("", "receipt-*")
	if err != nil {
		return nil, fmt.Errorf("tmpdir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	htmlPath := filepath.Join(tmpDir, "receipt.html")
	pngPath := filepath.Join(tmpDir, "receipt.png")

	// Inject overriding style AFTER the template so its !important rules win.
	// Font sizes are configurable via /receipt-settings — each FontSizes field
	// of zero falls back to the historical default.
	fsHeader := fonts.headerOrDefault()
	fsItems := fonts.itemsOrDefault()
	fsMeta := fonts.metaOrDefault()
	fsTotal := fonts.totalOrDefault()
	overrideCSS := fmt.Sprintf(
		`<style>
			html, body { margin: 0 !important; padding: 0 !important; width: %dpx !important; max-width: %dpx !important; }
			body { padding: 6px 10px !important; font-size: %dpx !important; line-height: 1.3 !important; font-family: 'Courier New', monospace !important; }
			.header { font-size: %dpx !important; }
			.info, .meta, .totals, .totals-row, .extra { font-size: %dpx !important; }
			.meta-table, .meta-table td { font-size: %dpx !important; }
			.footer { font-size: %dpx !important; }
			.items-table, .items-table th, .items-table td { font-size: %dpx !important; }
			.change { font-size: %dpx !important; font-weight: bold !important; margin: 10px 0 !important; padding: 6px 0 !important; border-top: 2px solid #000 !important; border-bottom: 2px solid #000 !important; }
			img { display: none !important; }
			.items-table th, .items-table td { padding: 6px 8px !important; }
			.meta-table td { padding: 2px 0 !important; }
		</style>`,
		widthPx, widthPx,
		fsItems,  // body baseline (so totals/info inherit a reasonable size)
		fsHeader, // .header
		fsItems,  // .info/.meta/.totals/.extra share the items size
		fsMeta,   // .meta-table
		fsMeta,   // .footer reuses meta size — both are small auxiliary lines
		fsItems,  // .items-table
		fsTotal,  // .change (the change-due line, biggest by default)
	)
	wrappedHTML := html + overrideCSS
	if err := os.WriteFile(htmlPath, []byte(wrappedHTML), 0644); err != nil {
		return nil, fmt.Errorf("write html: %w", err)
	}

	cmd := exec.Command("google-chrome",
		"--headless=new",
		"--disable-gpu",
		"--no-sandbox",
		"--hide-scrollbars",
		"--default-background-color=FFFFFFFF",
		"--virtual-time-budget=2000",
		fmt.Sprintf("--window-size=%d,3000", widthPx),
		fmt.Sprintf("--screenshot=%s", pngPath),
		"file://"+htmlPath,
	)

	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()
	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("chrome render: %w", err)
		}
	case <-time.After(15 * time.Second):
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("chrome render timeout")
	}

	f, err := os.Open(pngPath)
	if err != nil {
		return nil, fmt.Errorf("open png: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode png: %w", err)
	}
	return cropWhiteBottom(img), nil
}

// cropWhiteBottom trims trailing all-white rows so the ticket is short.
func cropWhiteBottom(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	lastNonWhite := -1
	for y := h - 1; y >= 0; y-- {
		found := false
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(x+b.Min.X, y+b.Min.Y).RGBA()
			if (r+g+bl)/3 < 55000 {
				found = true
				break
			}
		}
		if found {
			lastNonWhite = y
			break
		}
	}
	if lastNonWhite < 0 {
		return img
	}
	cropH := lastNonWhite + 24
	if cropH > h {
		cropH = h
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, cropH))
	for y := 0; y < cropH; y++ {
		for x := 0; x < w; x++ {
			dst.Set(x, y, img.At(x+b.Min.X, y+b.Min.Y))
		}
	}
	return dst
}

// BuildRasterReceipt renders HTML to image via headless Chrome and wraps it
// in ESC/POS init + raster + cut commands ready for TCP send.
func BuildRasterReceipt(html string, widthPx int, fonts FontSizes) ([]byte, error) {
	return BuildFullReceipt("", html, widthPx, fonts)
}

// BuildFullReceipt prepends a logo (encoded directly from PNG for maximum clarity)
// before the HTML-rendered receipt body. The logo is sent as its own ESC/POS raster
// image, so it is not downscaled by Chrome and retains full resolution on thermal paper.
// Pass logoPath="" to skip the logo. `fonts` controls thermal font sizes.
func BuildFullReceipt(logoPath, html string, widthPx int, fonts FontSizes) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(cmdInit)

	// ── Logo (centered, rendered separately from PNG so it stays crisp) ──
	// Use ~55% of printer width — prominent without touching the edges.
	if logoPath != "" {
		logoMaxW := widthPx * 11 / 20
		if logoBytes, err := EncodeLogo(logoPath, logoMaxW); err == nil {
			buf.Write(cmdAlignCtr)
			buf.Write(logoBytes)
			buf.Write(cmdLineFeed)
		}
	}

	// ── HTML body (rendered via headless Chrome, logo hidden via CSS) ──
	img, err := RenderHTMLToImage(html, widthPx, fonts)
	if err != nil {
		return nil, err
	}
	buf.Write(cmdAlignLeft)
	buf.Write(encodeRasterImage(img))

	buf.Write(cmdLineFeed)
	buf.Write(cmdLineFeed)
	buf.Write(cmdCut)
	return buf.Bytes(), nil
}
