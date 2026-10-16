package fsx

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/R0N1n-dev/mura-files/internal/model"
)

// List reads a directory according to the supplied options.
func List(dir string, opts model.ListOptions) model.DirListing {
	dir = Abs(dir)
	listing := model.DirListing{
		Path:   dir,
		Name:   filepath.Base(dir),
		Parent: filepath.Dir(dir),
	}
	if listing.Parent == dir {
		listing.Parent = ""
	}

	info, err := os.Stat(dir)
	if err != nil {
		listing.Error = friendlyError(err)
		return listing
	}
	if !info.IsDir() {
		// Listing a file yields a single-entry listing so the UI can show it.
		entry := BuildEntry(dir, info)
		listing.Entries = []model.Entry{entry}
		listing.Total = 1
		listing.Filtered = 1
		listing.Readable = true
		listing.Writable = true
		return listing
	}
	listing.Readable = access(info, false)
	listing.Writable = access(info, true)

	var entries []model.Entry
	var hidden int
	var truncated bool

	if opts.Recursive {
		entries, truncated, err = listRecursive(dir, opts.ShowHidden, opts.Depth, opts.MaxEntries)
		hidden = 0
	} else {
		entries, hidden, err = listOneDir(dir, opts.ShowHidden)
	}
	if err != nil {
		listing.Error = friendlyError(err)
	}
	listing.Hidden = hidden
	listing.Truncated = truncated

	total := len(entries)
	if opts.Filter != "" {
		needle := strings.ToLower(opts.Filter)
		filtered := entries[:0:0]
		for _, e := range entries {
			if strings.Contains(strings.ToLower(e.Name), needle) {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	Sort(entries, opts)
	if opts.MaxEntries > 0 && len(entries) > opts.MaxEntries {
		entries = entries[:opts.MaxEntries]
		listing.Truncated = true
	}

	listing.Entries = entries
	listing.Total = total
	listing.Filtered = len(entries)
	return listing
}

// Sort orders entries according to the supplied options. Sorting happens on
// the Go side so that huge directories do not block the UI thread.
func Sort(entries []model.Entry, opts model.ListOptions) {
	by := opts.SortBy
	if by == "" {
		by = "name"
	}
	desc := opts.SortDesc

	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if opts.DirsFirst && a.IsDir != b.IsDir {
			return a.IsDir
		}
		if opts.GroupByKind {
			if ka, kb := kindRank(a.Kind), kindRank(b.Kind); ka != kb {
				return ka < kb
			}
		}
		var less bool
		switch by {
		case "size":
			less = a.Size < b.Size
		case "modified":
			less = a.ModTime < b.ModTime
		case "created":
			less = a.ChangeTime < b.ChangeTime
		case "accessed":
			less = a.AccessTime < b.AccessTime
		case "type":
			if a.Kind != b.Kind {
				less = a.Kind < b.Kind
			} else if a.Ext != b.Ext {
				less = a.Ext < b.Ext
			} else {
				less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case "ext":
			if a.Ext != b.Ext {
				less = a.Ext < b.Ext
			} else {
				less = naturalLess(a.Name, b.Name)
			}
		case "perm":
			less = a.Mode < b.Mode
		case "owner":
			less = a.Owner < b.Owner
		default: // name
			less = naturalLess(a.Name, b.Name)
		}
		if desc {
			return !less
		}
		return less
	})
}

func kindRank(kind string) int {
	switch kind {
	case "directory":
		return 0
	case "image":
		return 1
	case "video":
		return 2
	case "audio":
		return 3
	case "document":
		return 4
	case "archive":
		return 5
	case "code":
		return 6
	case "text":
		return 7
	default:
		return 8
	}
}

// naturalLess compares two names treating embedded digit runs numerically so
// that "file2" sorts before "file10", matching what users expect.
func naturalLess(a, b string) bool {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	i, j := 0, 0
	for i < len(la) && j < len(lb) {
		ca, cb := la[i], lb[j]
		da, db := ca >= '0' && ca <= '9', cb >= '0' && cb <= '9'
		if da && db {
			si, sj := i, j
			for i < len(la) && la[i] >= '0' && la[i] <= '9' {
				i++
			}
			for j < len(lb) && lb[j] >= '0' && lb[j] <= '9' {
				j++
			}
			na := strings.TrimLeft(la[si:i], "0")
			nb := strings.TrimLeft(lb[sj:j], "0")
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			if na != nb {
				return na < nb
			}
			continue
		}
		if ca != cb {
			return ca < cb
		}
		i++
		j++
	}
	return len(la) < len(lb)
}
