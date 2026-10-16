// Package assetfs adds private HTTP routes to the Wails asset server so the
// webview can stream file contents (for previews), thumbnails and themed
// icons. Everything is mounted under the /__mura/ prefix.
package assetfs

import (
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/R0N1n-dev/mura-files/internal/fsx"
	"github.com/R0N1n-dev/mura-files/internal/shell"
	"github.com/R0N1n-dev/mura-files/internal/thumbs"
)

// Prefix is the URL prefix reserved for the private routes.
const Prefix = "/__mura/"

// Handler serves the private routes.
type Handler struct {
	thumbs *thumbs.Cache
}

// New creates the handler.
func New(cache *thumbs.Cache) *Handler {
	return &Handler{thumbs: cache}
}

// ServeHTTP dispatches the private routes and 404s everything else.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case strings.HasPrefix(path, Prefix+"file"):
		h.serveFile(w, r, false)
	case strings.HasPrefix(path, Prefix+"download"):
		h.serveFile(w, r, true)
	case strings.HasPrefix(path, Prefix+"thumb"):
		h.serveThumb(w, r)
	case strings.HasPrefix(path, Prefix+"icon"):
		h.serveIcon(w, r)
	default:
		http.NotFound(w, r)
	}
}

func queryPath(r *http.Request) string {
	raw := r.URL.Query().Get("path")
	if raw == "" {
		return ""
	}
	if decoded, err := url.QueryUnescape(raw); err == nil {
		raw = decoded
	}
	return fsx.Abs(raw)
}

// serveFile streams a file to the webview. Range requests are supported by
// http.ServeContent which is what makes video and audio scrubbing work.
func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, download bool) {
	target := queryPath(r)
	if target == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}
	info, err := os.Stat(target)
	if err != nil {
		http.Error(w, fsx.FriendlyError(err), http.StatusNotFound)
		return
	}
	if info.IsDir() {
		http.Error(w, "is a directory", http.StatusBadRequest)
		return
	}
	f, err := os.Open(target)
	if err != nil {
		http.Error(w, fsx.FriendlyError(err), http.StatusForbidden)
		return
	}
	defer f.Close()

	contentType := fsx.MimeOf(target)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if strings.HasPrefix(contentType, "text/") && !strings.Contains(contentType, "charset") {
		contentType += "; charset=utf-8"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	if download {
		w.Header().Set("Content-Disposition",
			"attachment; filename=\""+sanitiseFilename(info.Name())+"\"")
	} else {
		w.Header().Set("Content-Disposition", "inline")
	}
	// Allow the webview to render PDFs and media inline from a blob context.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

func (h *Handler) serveThumb(w http.ResponseWriter, r *http.Request) {
	target := queryPath(r)
	if target == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}
	size := thumbs.ParseSize(r.URL.Query().Get("size"))
	file, err := h.thumbs.Get(target, size)
	if err != nil {
		http.Error(w, "no thumbnail", http.StatusNotFound)
		return
	}
	f, err := os.Open(file)
	if err != nil {
		http.Error(w, "no thumbnail", http.StatusNotFound)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		http.Error(w, "no thumbnail", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeContent(w, r, filepath.Base(file), info.ModTime(), f)
}

func (h *Handler) serveIcon(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}
	path := shell.IconPath(name)
	if path == "" {
		http.Error(w, "icon not found", http.StatusNotFound)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "icon not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		http.Error(w, "icon not found", http.StatusNotFound)
		return
	}
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if contentType == "" {
		contentType = "image/png"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=604800")
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

// sanitiseFilename removes characters that would break a quoted header value.
func sanitiseFilename(name string) string {
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "\\", "")
	name = strings.ReplaceAll(name, "\n", "")
	name = strings.ReplaceAll(name, "\r", "")
	if name == "" {
		name = "download"
	}
	return name
}

// FileURL builds the private preview URL for a path.
func FileURL(path string, download bool) string {
	route := Prefix + "file"
	if download {
		route = Prefix + "download"
	}
	return route + "?path=" + url.QueryEscape(path)
}

// ThumbURL builds the private thumbnail URL for a path.
func ThumbURL(path string, size int) string {
	return Prefix + "thumb?path=" + url.QueryEscape(path) + "&size=" + strconv.Itoa(size)
}

// IconURL builds the private icon URL for a theme icon name.
func IconURL(name string) string {
	return Prefix + "icon?name=" + url.QueryEscape(name)
}
