package video

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func encodePatternPNG(t *testing.T, pixel func(x, y int) color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, pixel(x, y))
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestPerceptionHashPNG_DeterministicAndIdempotent(t *testing.T) {
	raw := encodePatternPNG(t, func(x, y int) color.Color {
		if x/8%2 == 0 {
			return color.RGBA{R: 20, G: 220, B: 40, A: 255}
		}
		return color.RGBA{R: 20, G: 20, B: 20, A: 255}
	})
	a, err := perceptionHashPNG(raw)
	if err != nil {
		t.Fatalf("hash a: %v", err)
	}
	b, err := perceptionHashPNG(raw)
	if err != nil {
		t.Fatalf("hash b: %v", err)
	}
	dist, err := a.Distance(b)
	if err != nil {
		t.Fatalf("distance: %v", err)
	}
	if dist != 0 {
		t.Fatalf("same bytes must hash to distance 0, got %d", dist)
	}
}

func TestPerceptionHashPNG_DifferentScenes(t *testing.T) {
	stripes := encodePatternPNG(t, func(x, y int) color.Color {
		if x/8%2 == 0 {
			return color.RGBA{R: 20, G: 220, B: 40, A: 255}
		}
		return color.RGBA{R: 20, G: 20, B: 20, A: 255}
	})
	bands := encodePatternPNG(t, func(x, y int) color.Color {
		if y/8%2 == 0 {
			return color.RGBA{R: 40, G: 80, B: 240, A: 255}
		}
		return color.RGBA{R: 20, G: 20, B: 20, A: 255}
	})
	a, err := perceptionHashPNG(stripes)
	if err != nil {
		t.Fatalf("hash stripes: %v", err)
	}
	b, err := perceptionHashPNG(bands)
	if err != nil {
		t.Fatalf("hash bands: %v", err)
	}
	dist, err := a.Distance(b)
	if err != nil {
		t.Fatalf("distance: %v", err)
	}
	if dist == 0 {
		t.Fatalf("distinct scenes must not hash identically, got %d", dist)
	}
}
