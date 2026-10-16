// Package thumbs renders and caches image thumbnails for the grid and preview
// views. Thumbnails are written to the XDG cache directory and invalidated
// whenever the source file changes.
package thumbs

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	xdraw "golang.org/x/image/draw"

	"github.com/R0N1n-dev/mura-files/internal/fsx"
	"github.com/R0N1n-dev/mura-files/internal/places"
)

// maxSourceBytes caps the size of images we are willing to decode.
const maxSourceBytes = 96 << 20

// maxCacheBytes caps the on-disk thumbnail cache.
const maxCacheBytes = 512 << 20

// Cache renders and stores thumbnails.
type Cache struct {
	dir string

	mu      sync.Mutex
	pending map[string]*sync.WaitGroup
}

// New creates a thumbnail cache in the application cache directory.
func New() *Cache {
	dir := filepath.Join(places.CacheDir(), "thumbnails")
	_ = os.MkdirAll(dir, 0o755)
	c := &Cache{dir: dir, pending: map[string]*sync.WaitGroup{}}
	go c.trim()
	return c
}

// Dir exposes the cache directory, used by diagnostics.
func (c *Cache) Dir() string { return c.dir }

func decodeImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	switch fsx.BaseExt(path) {
	case ".jpg", ".jpeg":
		return jpeg.Decode(f)
	case ".png":
		return png.Decode(f)
	case ".gif":
		return gif.Decode(f)
	}
	// Fall back to content sniffing which also covers formats registered by
	// additional decoders.
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	if _, _, err := image.DecodeConfig(bytes.NewReader(head[:n])); err != nil {
		return nil, fmt.Errorf("unsupported image format")
	}
	img, _, err := image.Decode(f)
	return img, err
}

// Scale produces a thumbnail that fits inside a box of the given size while
// preserving the aspect ratio. Images smaller than the box are left alone.
func Scale(src image.Image, size int) *image.RGBA {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	if w <= size && h <= size {
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		xdraw.Draw(dst, dst.Bounds(), src, bounds.Min, xdraw.Src)
		return dst
	}
	scale := float64(size) / float64(w)
	if sh := float64(size) / float64(h); sh < scale {
		scale = sh
	}
	nw := int(float64(w)*scale + 0.5)
	nh := int(float64(h)*scale + 0.5)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	// Fill with white so transparent PNGs do not render as black boxes.
	for i := range dst.Pix {
		dst.Pix[i] = 0xff
	}
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, xdraw.Over, nil)
	return dst
}

func (c *Cache) key(path string, size int, modTime int64, fileSize int64) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%d|%d", path, size, modTime, fileSize)))
	return hex.EncodeToString(sum[:]) + ".jpg"
}

// Get returns a cached thumbnail for path, generating it when necessary.
func (c *Cache) Get(path string, size int) (string, error) {
	if size <= 0 {
		size = 128
	}
	// Bucket sizes so that a small number of cached variants cover the UI.
	size = bucketSize(size)

	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() || info.Size() == 0 || info.Size() > maxSourceBytes {
		return "", fmt.Errorf("not thumbnailable")
	}
	key := c.key(path, size, info.ModTime().UnixNano(), info.Size())
	target := filepath.Join(c.dir, key)
	if st, err := os.Stat(target); err == nil && st.Size() > 0 {
		return target, nil
	}

	// Collapse concurrent requests for the same thumbnail.
	c.mu.Lock()
	if wg, ok := c.pending[key]; ok {
		c.mu.Unlock()
		wg.Wait()
		if st, err := os.Stat(target); err == nil && st.Size() > 0 {
			return target, nil
		}
		return "", fmt.Errorf("thumbnail generation failed")
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	c.pending[key] = wg
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, key)
		c.mu.Unlock()
		wg.Done()
	}()

	src, err := decodeImage(path)
	if err != nil {
		return "", err
	}
	thumb := Scale(src, size)
	tmp := target + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	err = jpeg.Encode(f, thumb, &jpeg.Options{Quality: 84})
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		_ = os.Remove(tmp)
		if err != nil {
			return "", err
		}
		return "", closeErr
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return target, nil
}

// GetBytes renders a thumbnail directly into memory.
func (c *Cache) GetBytes(path string, size int) ([]byte, error) {
	file, err := c.Get(path, size)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(file)
}

// Render encodes an in-memory image to JPEG, applying an optional square crop.
func Render(src image.Image, size int, square bool) ([]byte, error) {
	if square {
		src = CropSquare(src)
	}
	scaled := Scale(src, size)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, scaled, &jpeg.Options{Quality: 86}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// CropSquare centre-crops an image to a square, used by the icon grid.
func CropSquare(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == h {
		return src
	}
	side := w
	if h < side {
		side = h
	}
	offX := b.Min.X + (w-side)/2
	offY := b.Min.Y + (h-side)/2
	cropped := image.NewRGBA(image.Rect(0, 0, side, side))
	xdraw.Draw(cropped, cropped.Bounds(), src, image.Pt(offX, offY), xdraw.Src)
	return cropped
}

// Placeholder returns a flat colour image used when no thumbnail can be made.
func Placeholder(size int, col color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, col)
		}
	}
	return img
}

func bucketSize(size int) int {
	for _, b := range []int{48, 64, 96, 128, 192, 256, 384, 512, 768, 1024} {
		if size <= b {
			return b
		}
	}
	return 1024
}

// trim keeps the cache under maxCacheBytes by deleting the oldest entries.
func (c *Cache) trim() {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}
	type item struct {
		path string
		size int64
		mod  time.Time
	}
	items := make([]item, 0, len(entries))
	var total int64
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".tmp") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		items = append(items, item{filepath.Join(c.dir, e.Name()), info.Size(), info.ModTime()})
		total += info.Size()
	}
	if total <= maxCacheBytes {
		return
	}
	sort.Slice(items, func(i, j int) bool { return items[i].mod.Before(items[j].mod) })
	for _, it := range items {
		if total <= maxCacheBytes/2 {
			break
		}
		if err := os.Remove(it.path); err == nil {
			total -= it.size
		}
	}
}

// Clear removes every cached thumbnail.
func (c *Cache) Clear() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		_ = os.RemoveAll(filepath.Join(c.dir, e.Name()))
	}
	return nil
}

// Stat reports the number of cached thumbnails and their total size.
func (c *Cache) Stat() (count int, bytes int64) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return 0, 0
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if info, err := e.Info(); err == nil {
			count++
			bytes += info.Size()
		}
	}
	return count, bytes
}

// ParseSize converts a query string thumbnail size, clamped to sane bounds.
func ParseSize(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 128
	}
	if n > 1024 {
		return 1024
	}
	return n
}
