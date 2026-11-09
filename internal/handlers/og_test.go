package handlers

import (
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestOGImage_WarmClipboardCard(t *testing.T) {
	path := filepath.Join(projectRoot(t), "web", "static", "og.png")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open og.png: %v", err)
	}
	defer f.Close()

	img, format, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode og.png: %v", err)
	}
	if format != "png" {
		t.Fatalf("format = %q want png", format)
	}
	b := img.Bounds()
	if b.Dx() != 1200 || b.Dy() != 630 {
		t.Fatalf("size = %dx%d want 1200x630", b.Dx(), b.Dy())
	}

	r16, g16, b16, _ := img.At(40, 40).RGBA()
	r, g, bl := r16>>8, g16>>8, b16>>8
	if r < 18 || bl > r || g > r+8 {
		t.Fatalf("corner rgb=%d,%d,%d want warm ink (R>=B, brown not navy scaffold)", r, g, bl)
	}
}
