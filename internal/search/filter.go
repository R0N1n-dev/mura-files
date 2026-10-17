package search

import (
	"bufio"
	"io/fs"
	"os"
	"strings"

	"github.com/R0N1n-dev/mura-files/internal/fsx"
	"github.com/R0N1n-dev/mura-files/internal/model"
)

// buildEntry converts a walk result into a UI entry.
func buildEntry(path string, d fs.DirEntry) model.Entry {
	info, err := d.Info()
	if err != nil {
		return model.Entry{Name: d.Name(), Path: path, Dir: filepathDir(path), IsDir: d.IsDir()}
	}
	return fsx.BuildEntry(path, info)
}

func filepathDir(path string) string {
	if i := strings.LastIndexByte(path, '/'); i > 0 {
		return path[:i]
	}
	return "/"
}

// accepts applies every filter in the query to a candidate path.
//
// When content searching is enabled the pattern may match either the name or
// the file body, which is what users expect from a "search inside files"
// switch: you should not have to guess the filename.
func (m *matcher) accepts(path string, d fs.DirEntry) bool {
	q := m.query
	isDir := d.IsDir()

	if isDir {
		if !q.IncludeDirs {
			return false
		}
	} else if !q.IncludeFiles {
		return false
	}

	nameMatches := m.matchesName(d.Name())
	var info fs.FileInfo
	var haveInfo bool
	loadInfo := func() fs.FileInfo {
		if !haveInfo {
			info, _ = d.Info()
			haveInfo = true
		}
		return info
	}

	needsInfo := q.KindFilter != "" || len(q.ExtFilter) > 0 ||
		q.MinSize > 0 || q.MaxSize > 0 ||
		q.ModifiedAfter > 0 || q.ModifiedBefore > 0 || q.Content
	if !needsInfo {
		return nameMatches
	}

	fileInfo := loadInfo()
	if fileInfo == nil {
		return false
	}

	contentMatches := false
	if q.Content && !isDir {
		contentMatches = m.matchesContent(path, fileInfo.Size())
	}
	if !nameMatches && !contentMatches {
		return false
	}

	if len(q.ExtFilter) > 0 && !isDir {
		if !containsString(q.ExtFilter, fsx.BaseExt(d.Name())) {
			return false
		}
	}
	if q.KindFilter != "" && !isDir {
		if fsx.Kind(path, false) != q.KindFilter {
			return false
		}
	}
	if !isDir {
		size := fileInfo.Size()
		if q.MinSize > 0 && size < q.MinSize {
			return false
		}
		if q.MaxSize > 0 && size > q.MaxSize {
			return false
		}
	}
	if q.ModifiedAfter > 0 && fileInfo.ModTime().UnixMilli() < q.ModifiedAfter {
		return false
	}
	if q.ModifiedBefore > 0 && fileInfo.ModTime().UnixMilli() > q.ModifiedBefore {
		return false
	}
	return true
}

func containsString(list []string, needle string) bool {
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), needle) {
			return true
		}
	}
	return false
}

// matchesContent greps a text file for the search pattern.
func (m *matcher) matchesContent(path string, size int64) bool {
	if size <= 0 || size > maxContentFileSize {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	// Skip anything that looks binary in the first block.
	probe := make([]byte, 2048)
	n, _ := f.Read(probe)
	if n == 0 {
		return false
	}
	for _, b := range probe[:n] {
		if b == 0 {
			return false
		}
	}
	if _, err := f.Seek(0, 0); err != nil {
		return false
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if m.regex != nil {
			if m.regex.MatchString(line) {
				return true
			}
			continue
		}
		haystack := line
		if !m.query.MatchCase {
			haystack = strings.ToLower(line)
		}
		if m.pattern == "" || strings.Contains(haystack, m.pattern) {
			return true
		}
	}
	return false
}
