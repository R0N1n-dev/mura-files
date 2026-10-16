package fsx

import (
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/R0N1n-dev/mura-files/internal/model"
)

// ownerCache avoids repeated lookups of the same uid/gid.
var (
	uidCache = map[uint32]string{}
	gidCache = map[uint32]string{}
)

func ownerName(uid uint32) string {
	if n, ok := uidCache[uid]; ok {
		return n
	}
	name := strconv.FormatUint(uint64(uid), 10)
	if u, err := user.LookupId(name); err == nil && u.Username != "" {
		name = u.Username
	}
	uidCache[uid] = name
	return name
}

func groupName(gid uint32) string {
	if n, ok := gidCache[gid]; ok {
		return n
	}
	name := strconv.FormatUint(uint64(gid), 10)
	if g, err := user.LookupGroupId(name); err == nil && g.Name != "" {
		name = g.Name
	}
	gidCache[gid] = name
	return name
}

// PermString renders an ls-style permission string such as "-rw-r--r--".
func PermString(mode os.FileMode) string {
	buf := []byte("----------")
	switch {
	case mode&os.ModeDir != 0:
		buf[0] = 'd'
	case mode&os.ModeSymlink != 0:
		buf[0] = 'l'
	case mode&os.ModeNamedPipe != 0:
		buf[0] = 'p'
	case mode&os.ModeSocket != 0:
		buf[0] = 's'
	case mode&os.ModeDevice != 0:
		if mode&os.ModeCharDevice != 0 {
			buf[0] = 'c'
		} else {
			buf[0] = 'b'
		}
	}
	const rwx = "rwxrwxrwx"
	for i := 0; i < 9; i++ {
		if mode&(1<<uint(8-i)) != 0 {
			buf[i+1] = rwx[i]
		}
	}
	if mode&os.ModeSetuid != 0 {
		if buf[3] == 'x' {
			buf[3] = 's'
		} else {
			buf[3] = 'S'
		}
	}
	if mode&os.ModeSetgid != 0 {
		if buf[6] == 'x' {
			buf[6] = 's'
		} else {
			buf[6] = 'S'
		}
	}
	if mode&os.ModeSticky != 0 {
		if buf[9] == 'x' {
			buf[9] = 't'
		} else {
			buf[9] = 'T'
		}
	}
	return string(buf)
}

// access checks whether the current process may read or write a path by
// consulting the permission bits. It is a fast approximation that matches what
// a file manager needs to decide whether to grey out an action.
func access(info os.FileInfo, write bool) bool {
	mode := info.Mode().Perm()
	if write {
		return mode&0200 != 0 || mode&0020 != 0 || mode&0002 != 0
	}
	return mode&0400 != 0 || mode&0040 != 0 || mode&0004 != 0
}

// BuildEntry converts an os.FileInfo into a model.Entry without extra syscalls.
func BuildEntry(path string, info os.FileInfo) model.Entry {
	name := info.Name()
	isDir := info.IsDir()
	ext := BaseExt(name)
	kind := Kind(path, isDir)

	e := model.Entry{
		Name:       name,
		Path:       path,
		Dir:        filepath.Dir(path),
		Ext:        ext,
		IsDir:      isDir,
		IsSymlink:  info.Mode()&os.ModeSymlink != 0,
		IsHidden:   IsHidden(name),
		IsExec:     info.Mode().Perm()&0111 != 0,
		IsReadable: access(info, false),
		IsWritable: access(info, true),
		Size:       info.Size(),
		Mode:       "0" + strconv.FormatUint(uint64(info.Mode().Perm()), 8),
		Perm:       PermString(info.Mode()),
		ModeBits:   uint32(info.Mode().Perm()),
		ModTime:    info.ModTime().UnixMilli(),
		Mime:       MimeOf(path),
		Kind:       kind,
		Icon:       IconKey(path, isDir, kind),
		Thumb:      Thumbnailable(kind, ext),
	}
	if isDir {
		e.Mime = "inode/directory"
	}
	uid, gid, nlink, atime, ctime, ok := statExtra(info)
	if ok {
		e.UID = uid
		e.GID = gid
		e.Nlink = nlink
		e.AccessTime = atime
		e.ChangeTime = ctime
		e.Owner = ownerName(uid)
		e.Group = groupName(gid)
	}
	if e.IsSymlink {
		if target, err := os.Readlink(path); err == nil {
			e.Target = target
			if _, err := os.Stat(path); err != nil {
				e.LinkBroken = true
			}
		}
	}
	return e
}

// Stat builds an Entry for a single path, following symlinks for the type
// information but reporting the link itself.
func Stat(path string) (model.Entry, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return model.Entry{}, err
	}
	return BuildEntry(path, info), nil
}

// listOneDir reads a single directory level (non-recursive).
func listOneDir(dir string, showHidden bool) ([]model.Entry, int, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	names, err := f.Readdirnames(-1)
	if err != nil && len(names) == 0 {
		return nil, 0, err
	}
	entries := make([]model.Entry, 0, len(names))
	hidden := 0
	for _, name := range names {
		if IsHidden(name) {
			hidden++
			if !showHidden {
				continue
			}
		}
		full := filepath.Join(dir, name)
		info, err := os.Lstat(full)
		if err != nil {
			continue
		}
		entries = append(entries, BuildEntry(full, info))
	}
	return entries, hidden, nil
}

// listRecursive walks a tree up to depth levels deep, flattening the result.
func listRecursive(root string, showHidden bool, depth, maxEntries int) ([]model.Entry, bool, error) {
	var out []model.Entry
	truncated := false
	rootDepth := strings.Count(strings.TrimSuffix(root, "/"), "/")

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable branches
		}
		if path == root {
			return nil
		}
		name := d.Name()
		if IsHidden(name) && !showHidden {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		cur := strings.Count(strings.TrimSuffix(path, "/"), "/") - rootDepth
		if depth > 0 && cur > depth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		out = append(out, BuildEntry(path, info))
		if maxEntries > 0 && len(out) >= maxEntries {
			truncated = true
			return filepath.SkipAll
		}
		// Include the boundary level, but do not descend past it.
		if depth > 0 && d.IsDir() && cur >= depth {
			return filepath.SkipDir
		}
		return nil
	})
	return out, truncated, err
}
