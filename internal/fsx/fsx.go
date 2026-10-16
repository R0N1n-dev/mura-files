// Package fsx provides filesystem inspection helpers: path handling, stat,
// MIME and category detection, sorting and directory listing.
package fsx

import (
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// Expand resolves ~ and ~user prefixes and cleans the result.
func Expand(path string) string {
	if path == "" {
		return ""
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	if strings.HasPrefix(path, "~") {
		// ~user form: resolve against /home and /root.
		rest := path[1:]
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			user, tail := rest[:i], rest[i+1:]
			for _, base := range []string{"/home", "/root"} {
				candidate := filepath.Join(base, user)
				if st, err := os.Stat(candidate); err == nil && st.IsDir() {
					return filepath.Join(candidate, tail)
				}
			}
		}
	}
	return path
}

// Abs makes a path absolute, expanding ~ and resolving relative segments.
func Abs(path string) string {
	path = Expand(path)
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		if cwd, err := os.Getwd(); err == nil {
			path = filepath.Join(cwd, path)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(abs)
}

// IsHidden reports whether a name should be treated as hidden. On Linux this
// is the dot-file convention, matching what GTK/Nautilus do.
func IsHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

// Ext returns the lower-cased extension including the dot, or "".
func Ext(name string) string {
	e := filepath.Ext(name)
	if e == "" {
		return ""
	}
	return strings.ToLower(e)
}


// ---------------------------------------------------------------------------
// MIME and category detection
// ---------------------------------------------------------------------------

// mimeOverrides covers extensions that Go's stdlib mime package gets wrong or
// does not know about, which is common for source code and niche formats.
var mimeOverrides = map[string]string{
	".js":    "text/javascript",
	".mjs":   "text/javascript",
	".cjs":   "text/javascript",
	".jsx":   "text/jsx",
	".ts":    "text/typescript",
	".tsx":   "text/tsx",
	".vue":   "text/vue",
	".svelte": "text/svelte",
	".go":    "text/x-go",
	".rs":    "text/x-rust",
	".py":    "text/x-python",
	".rb":    "text/x-ruby",
	".php":   "text/x-php",
	".java":  "text/x-java",
	".kt":    "text/x-kotlin",
	".c":     "text/x-c",
	".h":     "text/x-c",
	".cc":    "text/x-c++",
	".cpp":   "text/x-c++",
	".hpp":   "text/x-c++",
	".cs":    "text/x-csharp",
	".sh":    "text/x-shellscript",
	".bash":  "text/x-shellscript",
	".zsh":   "text/x-shellscript",
	".fish":  "text/x-shellscript",
	".sql":   "text/x-sql",
	".lua":   "text/x-lua",
	".toml":  "text/x-toml",
	".ini":   "text/x-ini",
	".cfg":   "text/x-ini",
	".conf":  "text/x-ini",
	".env":   "text/x-ini",
	".yaml":  "text/x-yaml",
	".yml":   "text/x-yaml",
	".json":  "application/json",
	".jsonc": "application/json",
	".md":    "text/markdown",
	".markdown": "text/markdown",
	".rst":   "text/x-rst",
	".log":   "text/x-log",
	".diff":  "text/x-diff",
	".patch": "text/x-diff",
	".dockerfile": "text/x-dockerfile",
	".mk":    "text/x-makefile",
	".cmake": "text/x-cmake",
	".gradle": "text/x-gradle",
	".xml":   "text/xml",
	".svg":   "image/svg+xml",
	".heic":  "image/heic",
	".avif":  "image/avif",
	".webp":  "image/webp",
	".mkv":   "video/x-matroska",
	".m4a":   "audio/mp4",
	".flac":  "audio/flac",
	".opus":  "audio/opus",
	".ogg":   "audio/ogg",
	".wav":   "audio/wav",
	".deb":   "application/vnd.debian.binary-package",
	".rpm":   "application/x-rpm",
	".apk":   "application/vnd.android.package-archive",
	".appimage": "application/x-appimage",
	".iso":   "application/x-iso9660-image",
	".7z":    "application/x-7z-compressed",
	".rar":   "application/vnd.rar",
	".zst":   "application/zstd",
	".torrent": "application/x-bittorrent",
	".desktop": "application/x-desktop",
	".epub":  "application/epub+zip",
	".odt":   "application/vnd.oasis.opendocument.text",
	".ods":   "application/vnd.oasis.opendocument.spreadsheet",
	".odp":   "application/vnd.oasis.opendocument.presentation",
	".lock":  "text/plain",
	".ttf":   "font/ttf",
	".otf":   "font/otf",
	".woff":  "font/woff",
	".woff2": "font/woff2",
}

// specialNames maps well-known extension-less filenames to a MIME type.
var specialNames = map[string]string{
	"makefile":       "text/x-makefile",
	"dockerfile":     "text/x-dockerfile",
	"cmakelists.txt": "text/x-cmake",
	"readme":         "text/plain",
	"license":        "text/plain",
	"licence":        "text/plain",
	"copying":        "text/plain",
	"changelog":      "text/plain",
	"authors":        "text/plain",
	"todo":           "text/plain",
	".gitignore":     "text/plain",
	".gitattributes": "text/plain",
	".bashrc":        "text/x-shellscript",
	".bash_profile":  "text/x-shellscript",
	".profile":       "text/x-shellscript",
	".zshrc":         "text/x-shellscript",
	".vimrc":         "text/plain",
	".editorconfig":  "text/plain",
	".npmrc":         "text/plain",
	"pkgbuild":       "text/x-shellscript",
}

// MimeOf returns the best-effort MIME type for a path.
func MimeOf(path string) string {
	name := filepath.Base(path)
	if m, ok := specialNames[strings.ToLower(name)]; ok {
		return m
	}
	ext := BaseExt(name)
	if m, ok := mimeOverrides[ext]; ok {
		return m
	}
	if m := mime.TypeByExtension(ext); m != "" {
		if i := strings.IndexByte(m, ';'); i > 0 {
			m = strings.TrimSpace(m[:i])
		}
		return m
	}
	if ext == "" {
		return "text/plain"
	}
	return "application/octet-stream"
}

// ---------------------------------------------------------------------------
// Categories and icon keys
// ---------------------------------------------------------------------------

var extKinds = map[string]string{
	// images
	".png": "image", ".jpg": "image", ".jpeg": "image", ".gif": "image",
	".bmp": "image", ".webp": "image", ".tiff": "image", ".tif": "image",
	".svg": "image", ".ico": "image", ".heic": "image", ".avif": "image",
	".raw": "image", ".cr2": "image", ".nef": "image", ".psd": "image", ".xcf": "image",
	// video
	".mp4": "video", ".mkv": "video", ".webm": "video", ".avi": "video",
	".mov": "video", ".wmv": "video", ".flv": "video", ".m4v": "video",
	".mpg": "video", ".mpeg": "video", ".3gp": "video", ".ogv": "video",
	// audio
	".mp3": "audio", ".flac": "audio", ".wav": "audio", ".ogg": "audio",
	".oga": "audio", ".opus": "audio", ".m4a": "audio", ".aac": "audio",
	".wma": "audio", ".aiff": "audio", ".mid": "audio", ".midi": "audio",
	// archives
	".zip": "archive", ".tar": "archive", ".gz": "archive", ".bz2": "archive",
	".xz": "archive", ".7z": "archive", ".rar": "archive", ".zst": "archive",
	".lz4": "archive", ".lz": "archive", ".iso": "archive", ".cab": "archive",
	".tgz": "archive", ".tbz2": "archive", ".txz": "archive", ".jar": "archive",
	".war": "archive", ".deb": "archive", ".rpm": "archive", ".apk": "archive",
	".appimage": "archive", ".cpio": "archive", ".ar": "archive", ".whl": "archive",
	// documents
	".pdf": "document", ".doc": "document", ".docx": "document",
	".odt": "document", ".ods": "document", ".odp": "document",
	".xls": "document", ".xlsx": "document", ".ppt": "document",
	".pptx": "document", ".rtf": "document", ".epub": "document",
	".mobi": "document", ".djvu": "document", ".tex": "document",
	".csv": "document", ".tsv": "document",
	// code
	".js": "code", ".mjs": "code", ".cjs": "code", ".jsx": "code",
	".ts": "code", ".tsx": "code", ".vue": "code", ".svelte": "code",
	".go": "code", ".rs": "code", ".py": "code", ".rb": "code", ".php": "code",
	".java": "code", ".kt": "code", ".kts": "code", ".scala": "code",
	".c": "code", ".h": "code", ".cc": "code", ".cpp": "code", ".hpp": "code",
	".cs": "code", ".swift": "code", ".m": "code", ".mm": "code",
	".sh": "code", ".bash": "code", ".zsh": "code", ".fish": "code",
	".sql": "code", ".lua": "code", ".pl": "code", ".r": "code",
	".dart": "code", ".ex": "code", ".exs": "code", ".erl": "code",
	".hs": "code", ".clj": "code", ".elm": "code", ".zig": "code",
	".html": "code", ".htm": "code", ".css": "code", ".scss": "code",
	".sass": "code", ".less": "code", ".json": "code", ".jsonc": "code",
	".yaml": "code", ".yml": "code", ".toml": "code", ".xml": "code",
	".ini": "code", ".cfg": "code", ".conf": "code", ".env": "code",
	".mk": "code", ".cmake": "code", ".gradle": "code",
	// text
	".txt": "text", ".md": "text", ".markdown": "text", ".rst": "text",
	".log": "text", ".nfo": "text", ".diff": "text", ".patch": "text",
	".srt": "text", ".vtt": "text", ".sub": "text",
	// fonts
	".ttf": "font", ".otf": "font", ".woff": "font", ".woff2": "font",
	".eot": "font", ".pfb": "font",
}

// mimeKinds maps a MIME prefix to a coarse kind for unknown extensions.
var mimeKinds = map[string]string{
	"image/":   "image",
	"video/":   "video",
	"audio/":   "audio",
	"font/":    "font",
	"text/":    "text",
	"message/": "text",
}

// kindFromMime matches a MIME type against the prefix table.
func kindFromMime(mimeType string) string {
	for prefix, kind := range mimeKinds {
		if strings.HasPrefix(mimeType, prefix) {
			return kind
		}
	}
	return ""
}

// Kind classifies a path by extension and MIME type.
func Kind(path string, isDir bool) string {
	if isDir {
		return "directory"
	}
	ext := BaseExt(filepath.Base(path))
	if k, ok := extKinds[ext]; ok {
		return k
	}
	// Compound extensions (.tar.gz, .tar.xz, ...) fall back to their suffix.
	if k, ok := extKinds[Ext(path)]; ok {
		return k
	}
	m := MimeOf(path)
	if k := kindFromMime(m); k != "" {
		return k
	}
	if strings.HasSuffix(m, "+zip") || strings.HasSuffix(m, "+gzip") ||
		m == "application/zip" || m == "application/gzip" ||
		m == "application/x-tar" || m == "application/x-compressed-tar" {
		return "archive"
	}
	if strings.HasPrefix(m, "application/vnd.oasis.opendocument") ||
		strings.HasPrefix(m, "application/vnd.openxmlformats") ||
		strings.HasPrefix(m, "application/vnd.ms-") ||
		m == "application/pdf" || m == "application/epub+zip" {
		return "document"
	}
	if strings.HasPrefix(m, "application/x-executable") ||
		strings.HasPrefix(m, "application/x-sharedlib") ||
		m == "application/x-pie-executable" {
		return "executable"
	}
	return "other"
}

// iconByExt maps an extension to a frontend icon key.
var iconByExt = map[string]string{
	".pdf": "file-pdf", ".doc": "file-word", ".docx": "file-word",
	".odt": "file-word", ".rtf": "file-word",
	".xls": "file-excel", ".xlsx": "file-excel", ".ods": "file-excel",
	".csv": "file-excel", ".tsv": "file-excel",
	".ppt": "file-slides", ".pptx": "file-slides", ".odp": "file-slides",
	".zip": "file-archive", ".tar": "file-archive", ".gz": "file-archive",
	".tgz": "file-archive", ".bz2": "file-archive", ".xz": "file-archive",
	".7z": "file-archive", ".rar": "file-archive", ".zst": "file-archive",
	".deb": "file-package", ".rpm": "file-package", ".apk": "file-package",
	".appimage": "file-package", ".iso": "file-disc", ".img": "file-disc",
	".mp3": "file-audio", ".flac": "file-audio", ".wav": "file-audio",
	".ogg": "file-audio", ".opus": "file-audio", ".m4a": "file-audio",
	".mp4": "file-video", ".mkv": "file-video", ".webm": "file-video",
	".avi": "file-video", ".mov": "file-video",
	".png": "file-image", ".jpg": "file-image", ".jpeg": "file-image",
	".gif": "file-image", ".bmp": "file-image", ".webp": "file-image",
	".svg": "file-image", ".tiff": "file-image", ".ico": "file-image",
	".txt": "file-text", ".log": "file-text", ".nfo": "file-text",
	".md": "file-markdown", ".markdown": "file-markdown",
	".json": "file-json", ".jsonc": "file-json",
	".sh": "file-shell", ".bash": "file-shell", ".zsh": "file-shell",
	".fish": "file-shell", ".desktop": "file-desktop",
	".ttf": "file-font", ".otf": "file-font", ".woff": "file-font",
	".woff2": "file-font",
	".db": "file-database", ".sqlite": "file-database",
	".sqlite3": "file-database", ".sql": "file-database",
	".lock": "file-lock",
}

// IconKey returns the frontend icon key for an entry.
func IconKey(path string, isDir bool, kind string) string {
	if isDir {
		return "folder"
	}
	ext := BaseExt(filepath.Base(path))
	if k, ok := iconByExt[ext]; ok {
		return k
	}
	switch kind {
	case "image":
		return "file-image"
	case "video":
		return "file-video"
	case "audio":
		return "file-audio"
	case "archive":
		return "file-archive"
	case "document":
		return "file-document"
	case "code":
		return "file-code"
	case "text":
		return "file-text"
	case "font":
		return "file-font"
	case "executable":
		return "file-exec"
	}
	return "file"
}

// Thumbnailable reports whether the backend can build a real thumbnail.
func Thumbnailable(kind, ext string) bool {
	if kind != "image" {
		return false
	}
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp", ".tiff", ".tif":
		return true
	}
	return false
}

// IsTextKind reports whether the file is very likely human readable text.
func IsTextKind(kind string) bool {
	switch kind {
	case "text", "code":
		return true
	}
	return false
}



// BaseExt strips a single level of known archive-style double extension so that
// "backup.tar.gz" reports ".tar.gz".
func BaseExt(name string) string {
	lower := strings.ToLower(name)
	for _, de := range []string{".tar.gz", ".tar.bz2", ".tar.xz", ".tar.zst", ".tar.lz4", ".tar.lz"} {
		if strings.HasSuffix(lower, de) {
			return de
		}
	}
	return Ext(name)
}
