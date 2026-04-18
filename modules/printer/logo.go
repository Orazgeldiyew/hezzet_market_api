package printer

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"os"
)

const (
	// Max printer widths in pixels
	maxWidth58mm = 384 // 58mm printer — 384 dots at 203 DPI
	maxWidth80mm = 576 // 80mm printer — 576 dots at 203 DPI
)

// EncodeLogo reads an image file, trims surrounding white borders,
// converts to monochrome bitmap, resizes to fit the printer, and returns
// ESC/POS GS v 0 command bytes.
func EncodeLogo(path string, printerWidthPx int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open logo: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	// Trim white/transparent borders so the logo fills the available width
	img = trimWhiteBorders(img)

	// Resize if wider than printer
	img = resizeToWidth(img, printerWidthPx)

	return encodeRasterImage(img), nil
}

// trimWhiteBorders removes outer rows/columns that are effectively blank.
// A row/column is treated as "blank" if fewer than ~0.8% of its pixels are dark —
// this drops rows containing only soft anti-aliased edges that would otherwise
// leave a visible empty margin above/below the printed logo.
func trimWhiteBorders(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	isDark := func(x, y int) bool {
		r, g, bl, a := src.At(x+b.Min.X, y+b.Min.Y).RGBA()
		if a < 32768 {
			return false // transparent = treat as white
		}
		return (r+g+bl)/3 < 46000 // brightness < ~180/255
	}

	// Rows/columns need at least this many dark pixels to count as content
	rowThreshold := (w * 20) / 1000 // 2.0% of width
	colThreshold := (h * 20) / 1000 // 2.0% of height
	if rowThreshold < 3 {
		rowThreshold = 3
	}
	if colThreshold < 3 {
		colThreshold = 3
	}

	// Find content bounds
	top, bottom, left, right := -1, -1, -1, -1
	for y := 0; y < h; y++ {
		darkInRow := 0
		for x := 0; x < w; x++ {
			if isDark(x, y) {
				darkInRow++
			}
		}
		if darkInRow >= rowThreshold {
			if top < 0 {
				top = y
			}
			bottom = y
		}
	}
	for x := 0; x < w; x++ {
		darkInCol := 0
		for y := 0; y < h; y++ {
			if isDark(x, y) {
				darkInCol++
			}
		}
		if darkInCol >= colThreshold {
			if left < 0 {
				left = x
			}
			right = x
		}
	}
	if top < 0 || left < 0 {
		return src // entirely blank
	}

	// Copy the cropped region to a fresh RGBA so subsequent processing works consistently.
	cropW := right - left + 1
	cropH := bottom - top + 1
	dst := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
	for y := 0; y < cropH; y++ {
		for x := 0; x < cropW; x++ {
			dst.Set(x, y, src.At(b.Min.X+left+x, b.Min.Y+top+y))
		}
	}
	return dst
}

// resizeToWidth scales image down (nearest-neighbor) to fit printer width.
// Keeps aspect ratio. If already smaller — returns as-is.
func resizeToWidth(src image.Image, maxW int) image.Image {
	b := src.Bounds()
	oldW, oldH := b.Dx(), b.Dy()
	if oldW <= maxW {
		return src
	}
	newW := maxW
	newH := oldH * newW / oldW

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			srcX := x * oldW / newW
			srcY := y * oldH / newH
			dst.Set(x, y, src.At(srcX+b.Min.X, srcY+b.Min.Y))
		}
	}
	return dst
}

// encodeRasterImage converts image to 1-bit monochrome (with Floyd-Steinberg dithering)
// and builds the GS v 0 command. Dithering preserves gradients/anti-aliased edges
// which would otherwise disappear with a hard threshold.
// Format: 1D 76 30 m xL xH yL yH [data]
func encodeRasterImage(img image.Image) []byte {
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()

	// Build a grayscale buffer (float for error diffusion)
	gray := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bVal, _ := img.At(x+b.Min.X, y+b.Min.Y).RGBA()
			gray[y*w+x] = float64(r+g+bVal) / 3.0 / 65535.0 // normalized 0..1
		}
	}

	// Floyd-Steinberg dithering
	byteWidth := (w + 7) / 8
	data := make([]byte, byteWidth*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			old := gray[y*w+x]
			var newPx float64
			if old < 0.5 {
				newPx = 0
				data[y*byteWidth+x/8] |= 1 << (7 - uint(x%8))
			} else {
				newPx = 1
			}
			err := old - newPx
			// Distribute error: right 7/16, below-left 3/16, below 5/16, below-right 1/16
			if x+1 < w {
				gray[y*w+(x+1)] += err * 7 / 16
			}
			if y+1 < h {
				if x > 0 {
					gray[(y+1)*w+(x-1)] += err * 3 / 16
				}
				gray[(y+1)*w+x] += err * 5 / 16
				if x+1 < w {
					gray[(y+1)*w+(x+1)] += err * 1 / 16
				}
			}
		}
	}

	var buf bytes.Buffer
	buf.Write([]byte{0x1D, 0x76, 0x30, 0x00})
	buf.WriteByte(byte(byteWidth & 0xFF))
	buf.WriteByte(byte((byteWidth >> 8) & 0xFF))
	buf.WriteByte(byte(h & 0xFF))
	buf.WriteByte(byte((h >> 8) & 0xFF))
	buf.Write(data)

	return buf.Bytes()
}
