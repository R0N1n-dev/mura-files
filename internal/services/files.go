package services

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/R0N1n-dev/mura-files/internal/archive"
	"github.com/R0N1n-dev/mura-files/internal/fsop"
	"github.com/R0N1n-dev/mura-files/internal/fsx"
	"github.com/R0N1n-dev/mura-files/internal/model"
	"github.com/R0N1n-dev/mura-files/internal/places"
	"github.com/R0N1n-dev/mura-files/internal/shell"
)

// FileService exposes directory listing and single item operations.
type FileService struct {
	rt *Runtime
}

// NewFileService creates the service.
func NewFileService(rt *Runtime) *FileService { return &FileService{rt: rt} }

// ServiceName identifies the service in logs and bindings.
func (s *FileService) ServiceName() string { return "Files" }

// List reads a directory and returns the entries already sorted server side.
func (s *FileService) List(path string, opts model.ListOptions) model.DirListing {
	if path == "" || path == "~" {
		path = places.Home()
	}
	return fsx.List(path, opts)
}

// Stat returns metadata for a single path.
func (s *FileService) Stat(path string) (model.Entry, error) {
	path = fsx.Abs(path)
	if path == "" {
		return model.Entry{}, fmt.Errorf("no path supplied")
	}
	return fsx.Stat(path)
}

// StatMany returns metadata for several paths, skipping unreadable ones.
func (s *FileService) StatMany(paths []string) []model.Entry {
	out := make([]model.Entry, 0, len(paths))
	for _, p := range paths {
		if entry, err := fsx.Stat(fsx.Abs(p)); err == nil {
			out = append(out, entry)
		}
	}
	return out
}

// Home returns the current user's home directory.
func (s *FileService) Home() string { return places.Home() }

// Resolve expands ~ and relative segments into an absolute path.
func (s *FileService) Resolve(path string) string { return fsx.Abs(path) }

// Parent returns the parent directory of a path.
func (s *FileService) Parent(path string) string {
	abs := fsx.Abs(path)
	parent := filepath.Dir(abs)
	if parent == abs {
		return ""
	}
	return parent
}

// Exists reports whether a path exists.
func (s *FileService) Exists(path string) bool {
	_, err := os.Lstat(fsx.Abs(path))
	return err == nil
}

// Complete returns path completions for the location bar.
func (s *FileService) Complete(prefix string, limit int) []model.Completion {
	return fsx.Complete(prefix, limit)
}

// CreateDir creates a new directory inside parent.
func (s *FileService) CreateDir(parent, name string) (model.Entry, error) {
	parent = fsx.Abs(parent)
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Entry{}, fmt.Errorf("folder name cannot be empty")
	}
	if strings.ContainsAny(name, "/\x00") {
		return model.Entry{}, fmt.Errorf("folder name cannot contain a slash")
	}
	target := filepath.Join(parent, name)
	if _, err := os.Lstat(target); err == nil {
		target = filepath.Join(parent, fsx.UniqueName(parent, name))
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return model.Entry{}, fmt.Errorf("%s", fsx.FriendlyError(err))
	}
	return fsx.Stat(target)
}

// CreateFile creates a new empty file, optionally with initial content.
func (s *FileService) CreateFile(parent, name, content string) (model.Entry, error) {
	parent = fsx.Abs(parent)
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Entry{}, fmt.Errorf("file name cannot be empty")
	}
	if strings.ContainsAny(name, "/\x00") {
		return model.Entry{}, fmt.Errorf("file name cannot contain a slash")
	}
	target := filepath.Join(parent, name)
	if _, err := os.Lstat(target); err == nil {
		target = filepath.Join(parent, fsx.UniqueName(parent, name))
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return model.Entry{}, fmt.Errorf("%s", fsx.FriendlyError(err))
	}
	return fsx.Stat(target)
}

// Rename renames a path within its own directory.
func (s *FileService) Rename(path, newName string) (model.Entry, error) {
	path = fsx.Abs(path)
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return model.Entry{}, fmt.Errorf("new name cannot be empty")
	}
	if strings.ContainsAny(newName, "/\x00") {
		return model.Entry{}, fmt.Errorf("new name cannot contain a slash")
	}
	dir := filepath.Dir(path)
	target := filepath.Join(dir, newName)
	if target == path {
		return fsx.Stat(path)
	}
	if _, err := os.Lstat(target); err == nil {
		return model.Entry{}, fmt.Errorf("%q already exists", newName)
	}
	if err := os.Rename(path, target); err != nil {
		return model.Entry{}, fmt.Errorf("%s", fsx.FriendlyError(err))
	}
	return fsx.Stat(target)
}

// Trash moves paths into the freedesktop trash.
func (s *FileService) Trash(paths []string) (int, error) {
	moved := 0
	var firstErr error
	for _, p := range paths {
		if _, err := places.TrashPath(fsx.Abs(p)); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		moved++
	}
	return moved, firstErr
}

// Chmod applies a permission mask.
func (s *FileService) Chmod(path string, mode uint32) error {
	return os.Chmod(fsx.Abs(path), os.FileMode(mode&0o7777))
}

// ReadLink returns the destination of a symbolic link.
func (s *FileService) ReadLink(path string) (string, error) {
	return os.Readlink(fsx.Abs(path))
}

// Checksum computes a hash for a file. Algorithm is one of md5, sha1, sha256.
func (s *FileService) Checksum(path, algorithm string) (string, error) {
	path = fsx.Abs(path)
	var h hash.Hash
	switch strings.ToLower(algorithm) {
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	default:
		h = sha256.New()
	}
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("%s", fsx.FriendlyError(err))
	}
	defer f.Close()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("%s", fsx.FriendlyError(err))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ReadText returns a preview of a text file.
func (s *FileService) ReadText(path string, maxBytes int) (model.TextPreview, error) {
	path = fsx.Abs(path)
	if maxBytes <= 0 {
		maxBytes = 256 * 1024
	}
	if maxBytes > 4<<20 {
		maxBytes = 4 << 20
	}
	info, err := os.Stat(path)
	if err != nil {
		return model.TextPreview{}, fmt.Errorf("%s", fsx.FriendlyError(err))
	}
	preview := model.TextPreview{Path: path, Size: info.Size(), Encoding: "utf-8"}
	f, err := os.Open(path)
	if err != nil {
		return preview, fmt.Errorf("%s", fsx.FriendlyError(err))
	}
	defer f.Close()

	buf := make([]byte, maxBytes+1)
	n, _ := io.ReadFull(f, buf)
	if n > maxBytes {
		n = maxBytes
		preview.Truncated = true
	}
	raw := buf[:n]
	if len(raw) == 0 {
		return preview, nil
	}
	// Detect binary content by looking for NUL bytes in the sample.
	check := raw
	if len(check) > 8192 {
		check = check[:8192]
	}
	for _, b := range check {
		if b == 0 {
			preview.Binary = true
			return preview, nil
		}
	}
	content := strings.ToValidUTF8(string(raw), "\uFFFD")
	if !utf8.Valid(raw) {
		preview.Encoding = "latin-1"
		content = decodeLatin1(raw)
	}
	preview.Content = content
	preview.Lines = strings.Count(content, "\n") + 1
	return preview, nil
}

func decodeLatin1(raw []byte) string {
	var b strings.Builder
	b.Grow(len(raw))
	for _, c := range raw {
		b.WriteRune(rune(c))
	}
	return b.String()
}

// WriteText overwrites a file with the supplied text.
func (s *FileService) WriteText(path, content string) error {
	path = fsx.Abs(path)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("%s", fsx.FriendlyError(err))
	}
	return nil
}

// Count aggregates statistics for a selection, used by the properties dialog.
func (s *FileService) Count(paths []string, limit int64) model.CountResult {
	var out model.CountResult
	if limit <= 0 {
		limit = 200000
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, p := range paths {
		files, dirs, links, bytes, truncated := fsop.TreeSize(ctx, fsx.Abs(p), limit)
		out.Files += files
		out.Dirs += dirs
		out.Symlinks += links
		out.Bytes += bytes
		out.Truncated = out.Truncated || truncated
	}
	return out
}

// Usage reports free space for the filesystem containing a path.
func (s *FileService) Usage(path string) model.Usage {
	return places.Usage(fsx.Abs(path))
}

// Drives lists the mounted filesystems for the sidebar.
func (s *FileService) Drives() []model.Drive { return places.Drives() }

// WellKnown returns the sidebar places.
func (s *FileService) WellKnown() []model.Place { return places.WellKnown() }

// ArchiveInfo describes an archive so the UI can show a folder-like listing.
func (s *FileService) ArchiveInfo(path string) (model.ArchiveInfo, error) {
	return archive.List(fsx.Abs(path))
}

// IsArchive reports whether a path is a supported archive.
func (s *FileService) IsArchive(path string) bool { return archive.IsArchive(path) }

// Applications lists installed desktop applications for "open with".
func (s *FileService) Applications(path string) []model.AppInfo {
	if path != "" {
		if list := shell.ApplicationsFor(fsx.MimeOf(fsx.Abs(path))); len(list) > 0 {
			return list
		}
	}
	return shell.Applications()
}

// LaunchDesktop starts an application described by a .desktop file.
func (s *FileService) LaunchDesktop(path string) model.OpenResult {
	return shell.LaunchDesktopEntry(fsx.Abs(path))
}

// DesktopMeta reads the display name and icon for a .desktop file.
func (s *FileService) DesktopMeta(path string) (string, string) {
	return shell.DesktopMeta(fsx.Abs(path))
}

// SortEntries sorts a slice of entries, used by the flat search results view.
func (s *FileService) SortEntries(entries []model.Entry, opts model.ListOptions) []model.Entry {
	fsx.Sort(entries, opts)
	return entries
}
