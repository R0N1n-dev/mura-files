package thumbs

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// makePNG writes a small PNG and returns its path.
func makePNG(t *testing.T, dir, name string, width, height int, fill color.Color) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, fill)
		}
	}
	path := filepath.Join(dir, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func newCache(t *testing.T) *Cache {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	return New()
}

func TestScaleKeepsAspectRatio(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 400, 200))
	scaled := Scale(source, 100)
	if scaled.Bounds().Dx() != 100 || scaled.Bounds().Dy() != 50 {
		t.Fatalf("scaled to %v, want 100x50", scaled.Bounds())
	}
}

func TestScaleDoesNotUpscale(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 20, 10))
	scaled := Scale(source, 100)
	if scaled.Bounds().Dx() != 20 || scaled.Bounds().Dy() != 10 {
		t.Fatalf("small images must be left alone, got %v", scaled.Bounds())
	}
}

func TestScaleHandlesTallImages(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 100, 500))
	scaled := Scale(source, 100)
	if scaled.Bounds().Dy() != 100 {
		t.Fatalf("expected the height to become the limiting dimension, got %v", scaled.Bounds())
	}
}

func TestCropSquare(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 300, 100))
	cropped := CropSquare(source)
	if cropped.Bounds().Dx() != 100 || cropped.Bounds().Dy() != 100 {
		t.Fatalf("crop produced %v, want 100x100", cropped.Bounds())
	}
}

func TestCacheGeneratesAndReuses(t *testing.T) {
	cache := newCache(t)
	source := makePNG(t, t.TempDir(), "photo.png", 320, 240, color.RGBA{R: 200, A: 255})

	first, err := cache.Get(source, 96)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	info, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("the generated thumbnail is empty")
	}

	// The cache key is deterministic, so the second call must reuse the file.
	second, err := cache.Get(source, 96)
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatalf("expected a cache hit, got %q then %q", first, second)
	}

	count, bytes := cache.Stat()
	if count != 1 || bytes == 0 {
		t.Fatalf("Stat = (%d, %d)", count, bytes)
	}
}

func TestCacheInvalidatesOnChange(t *testing.T) {
	cache := newCache(t)
	dir := t.TempDir()
	source := makePNG(t, dir, "photo.png", 200, 200, color.RGBA{G: 200, A: 255})
	first, err := cache.Get(source, 64)
	if err != nil {
		t.Fatal(err)
	}

	// Replace the file with different content and a fresh timestamp.
	source = makePNG(t, dir, "photo.png", 400, 400, color.RGBA{B: 200, A: 255})
	future := os.Getpid()
	_ = future
	second, err := cache.Get(source, 64)
	if err != nil {
		t.Fatal(err)
	}
	if second == first {
		t.Error("a changed file should produce a new thumbnail")
	}
}

func TestCacheRejectsUnsupportedFiles(t *testing.T) {
	cache := newCache(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Get(path, 64); err == nil {
		t.Error("expected text files to be rejected")
	}
	if _, err := cache.Get(dir, 64); err == nil {
		t.Error("expected directories to be rejected")
	}
}

func TestClear(t *testing.T) {
	cache := newCache(t)
	source := makePNG(t, t.TempDir(), "a.png", 64, 64, color.RGBA{A: 255})
	if _, err := cache.Get(source, 48); err != nil {
		t.Fatal(err)
	}
	if err := cache.Clear(); err != nil {
		t.Fatal(err)
	}
	count, _ := cache.Stat()
	if count != 0 {
		t.Errorf("cache still holds %d files", count)
	}
}

func TestParseSize(t *testing.T) {
	cases := map[string]int{"": 128, "0": 128, "abc": 128, "64": 64, "4096": 1024, "128": 128}
	for input, want := range cases {
		if got := ParseSize(input); got != want {
			t.Errorf("ParseSize(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestRenderProducesJPEG(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 200, 100))
	data, err := Render(source, 64, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		t.Error("Render should emit a JPEG stream")
	}
}
