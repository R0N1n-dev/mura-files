package assetfs

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/R0N1n-dev/mura-files/internal/thumbs"
)

func newHandler(t *testing.T) *Handler {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	return New(thumbs.New())
}

func TestServesFileWithRangeSupport(t *testing.T) {
	handler := newHandler(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	content := "hello mura files"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, Prefix+"file?path="+url.QueryEscape(path), nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if recorder.Body.String() != content {
		t.Errorf("body = %q", recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("Content-Type = %q", contentType)
	}
	if recorder.Header().Get("Content-Disposition") != "inline" {
		t.Errorf("Content-Disposition = %q", recorder.Header().Get("Content-Disposition"))
	}
}

func TestRangeRequestReturnsPartialContent(t *testing.T) {
	handler := newHandler(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "data.bin")
	if err := os.WriteFile(path, []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, Prefix+"file?path="+url.QueryEscape(path), nil)
	request.Header.Set("Range", "bytes=2-5")
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want 206", recorder.Code)
	}
	if recorder.Body.String() != "2345" {
		t.Errorf("body = %q", recorder.Body.String())
	}
}

func TestDownloadSetsAttachment(t *testing.T) {
	handler := newHandler(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "report.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4"), 0o644); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, Prefix+"download?path="+url.QueryEscape(path), nil)
	handler.ServeHTTP(recorder, request)

	disposition := recorder.Header().Get("Content-Disposition")
	if !strings.HasPrefix(disposition, "attachment") || !strings.Contains(disposition, "report.pdf") {
		t.Errorf("Content-Disposition = %q", disposition)
	}
}

func TestMissingPathIsRejected(t *testing.T) {
	handler := newHandler(t)
	for _, target := range []string{Prefix + "file", Prefix + "thumb", Prefix + "icon"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%s without a path returned %d, want 400", target, recorder.Code)
		}
	}
}

func TestNonexistentFileReturns404(t *testing.T) {
	handler := newHandler(t)
	missing := filepath.Join(t.TempDir(), "nope.txt")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, Prefix+"file?path="+url.QueryEscape(missing), nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestDirectoryIsRejected(t *testing.T) {
	handler := newHandler(t)
	dir := t.TempDir()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, Prefix+"file?path="+url.QueryEscape(dir), nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestUnknownRouteIs404(t *testing.T) {
	handler := newHandler(t)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, Prefix+"nonsense", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestThumbRouteServesImages(t *testing.T) {
	handler := newHandler(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.txt")
	if err := os.WriteFile(path, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		Prefix+"thumb?path="+url.QueryEscape(path)+"&size=64", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("non-images should yield 404, got %d", recorder.Code)
	}
}

func TestFilenameSanitising(t *testing.T) {
	cases := map[string]string{
		`evil"name.txt`: "evilname.txt",
		"new\nline.txt": "newline.txt",
		`back\slash`:    "backslash",
		"":              "download",
	}
	for input, want := range cases {
		if got := sanitiseFilename(input); got != want {
			t.Errorf("sanitiseFilename(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestURLBuilders(t *testing.T) {
	if got := FileURL("/tmp/a b.txt", false); !strings.Contains(got, "a+b.txt") && !strings.Contains(got, "a%20b.txt") {
		t.Errorf("FileURL = %q", got)
	}
	if got := FileURL("/tmp/a.txt", true); !strings.HasPrefix(got, Prefix+"download") {
		t.Errorf("FileURL(download) = %q", got)
	}
	if got := ThumbURL("/tmp/a.png", 128); !strings.Contains(got, "size=128") {
		t.Errorf("ThumbURL = %q", got)
	}
	if got := IconURL("firefox"); !strings.Contains(got, "name=firefox") {
		t.Errorf("IconURL = %q", got)
	}
}
