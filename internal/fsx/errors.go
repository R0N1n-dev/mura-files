package fsx

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/R0N1n-dev/mura-files/internal/model"
)

// FriendlyError converts an OS error into a message suitable for the UI.
func FriendlyError(err error) string { return friendlyError(err) }

func friendlyError(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "Location does not exist"
	case errors.Is(err, fs.ErrPermission):
		return "Permission denied"
	case errors.Is(err, fs.ErrExist):
		return "Already exists"
	case errors.Is(err, syscall.ENOSPC):
		return "No space left on device"
	}
	msg := err.Error()
	if pe, ok := err.(*os.PathError); ok {
		msg = pe.Err.Error()
	}
	if msg == "" {
		return "Unknown error"
	}
	return strings.ToUpper(msg[:1]) + msg[1:]
}

// ---------------------------------------------------------------------------
// Path helpers
// ---------------------------------------------------------------------------

// UniqueName returns a name inside dir that does not collide with an existing
// entry, using the "name (2).ext" convention preferred by desktop file managers.
func UniqueName(dir, name string) string {
	if _, err := os.Lstat(filepath.Join(dir, name)); err != nil {
		return name
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 2; i < 10000; i++ {
		candidate := stem + " (" + itoa(i) + ")" + ext
		if _, err := os.Lstat(filepath.Join(dir, candidate)); err != nil {
			return candidate
		}
	}
	return name
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

// Ancestors returns every parent directory of path, from the filesystem root
// down to path itself. The root is always included.
func Ancestors(path string) []string {
	path = Abs(path)
	var out []string
	cur := path
	for {
		out = append(out, cur)
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// Complete returns filesystem path completions for a partial path.
func Complete(partial string, limit int) []model.Completion {
	if limit <= 0 {
		limit = 40
	}
	partial = Expand(partial)
	if partial == "" {
		partial = "/"
	}

	dir := partial
	prefix := ""
	if !strings.HasSuffix(partial, string(os.PathSeparator)) {
		dir = filepath.Dir(partial)
		prefix = filepath.Base(partial)
	}
	if dir == "" {
		dir = "/"
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	lower := strings.ToLower(prefix)
	out := make([]model.Completion, 0, limit)
	for _, e := range entries {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(e.Name()), lower) {
			continue
		}
		if strings.HasPrefix(e.Name(), ".") && !strings.HasPrefix(prefix, ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		isDir := e.IsDir()
		path := filepath.Join(dir, e.Name())
		if isDir {
			path += string(os.PathSeparator)
		}
		out = append(out, model.Completion{
			Path:   path,
			Name:   e.Name(),
			IsDir:  isDir,
			IsLink: info.Mode()&os.ModeSymlink != 0,
		})
		if len(out) >= limit {
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// IsSameOrInside reports whether child equals parent or lives beneath it.
func IsSameOrInside(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if parent == child {
		return true
	}
	return strings.HasPrefix(child, parent+string(os.PathSeparator))
}
