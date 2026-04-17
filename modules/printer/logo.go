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

// EncodeLogo reads an image file, converts to monochrome bitmap,
// resizes to fit the printer, and returns ESC/POS GS v 0 command bytes.
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

	// Resize if wider than printer
	img = resizeToWidth(img, printerWidthPx)

	return encodeRasterImage(img), nil
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

// encodeRasterImage converts image to 1-bit monochrome and builds GS v 0 command.
// Format: 1D 76 30 m xL xH yL yH [data]
// m = 0 (normal), xL/xH = width in bytes, yL/yH = height in pixels.
func encodeRasterImage(img image.Image) []byte {
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()

	// Width must be multiple of 8 (pad with white)
	byteWidth := (w + 7) / 8

	data := make([]byte, byteWidth*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bVal, _ := img.At(x+b.Min.X, y+b.Min.Y).RGBA()
			// Brightness (0-65535). Threshold at 50% = 32768.
			brightness := (r + g + bVal) / 3
			if brightness < 32768 {
				// Dark pixel → set bit (print black)
				data[y*byteWidth+x/8] |= 1 << (7 - uint(x%8))
			}
		}
	}

	var buf bytes.Buffer
	// GS v 0 — print raster bitmap
	buf.Write([]byte{0x1D, 0x76, 0x30, 0x00})
	buf.WriteByte(byte(byteWidth & 0xFF))
	buf.WriteByte(byte((byteWidth >> 8) & 0xFF))
	buf.WriteByte(byte(h & 0xFF))
	buf.WriteByte(byte((h >> 8) & 0xFF))
	buf.Write(data)

	return buf.Bytes()
}
