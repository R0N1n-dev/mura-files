package places

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/R0N1n-dev/mura-files/internal/fsop"
	"github.com/R0N1n-dev/mura-files/internal/fsx"
	"github.com/R0N1n-dev/mura-files/internal/model"
)

// trashDirFor returns the freedesktop trash directory that governs path.
//
// The specification allows two layouts for non-home filesystems:
//
//	<mount>/.Trash/<uid>   (preferred, shared with other users)
//	<mount>/.Trash-<uid>   (fallback when the shared directory is unusable)
func trashDirFor(path string) (string, error) {
	homeTrash := filepath.Join(DataDir(), "Trash")
	home := Home()

	mount := "/"
	best := 0
	for _, m := range readMounts() {
		if fsx.IsSameOrInside(m.path, path) && len(m.path) > best {
			mount = m.path
			best = len(m.path)
		}
	}
	if mount == "/" {
		// Only use the home trash when the path really is on the same device
		// as $HOME, otherwise a rename would fail.
		var a, b syscall.Stat_t
		if err := syscall.Stat(home, &a); err == nil {
			if err := syscall.Stat(path, &b); err == nil && a.Dev == b.Dev {
				return homeTrash, nil
			}
		}
	}

	uid := os.Getuid()
	shared := filepath.Join(mount, ".Trash", strconv.Itoa(uid))
	if err := ensureTrashDir(shared); err == nil {
		return shared, nil
	}
	fallback := filepath.Join(mount, ".Trash-"+strconv.Itoa(uid))
	if err := ensureTrashDir(fallback); err == nil {
		return fallback, nil
	}
	return homeTrash, nil
}

func ensureTrashDir(dir string) error {
	if err := os.MkdirAll(filepath.Join(dir, "files"), 0o700); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(dir, "info"), 0o700)
}

// TrashPath moves a single path into the freedesktop trash and returns the
// absolute path of the .trashinfo file describing the move.
func TrashPath(path string) (string, error) {
	path = fsx.Abs(path)
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	dir, err := trashDirFor(path)
	if err != nil {
		return "", err
	}
	if err := ensureTrashDir(dir); err != nil {
		return "", err
	}

	name := info.Name()
	target := filepath.Join(dir, "files", name)
	for i := 2; ; i++ {
		if _, err := os.Lstat(target); err != nil {
			break
		}
		target = filepath.Join(dir, "files", fmt.Sprintf("%s.%d", name, i))
	}

	if err := os.Rename(path, target); err != nil {
		// Cross-device rename: fall back to copy plus delete.
		if cerr := fsop.CopyTree(context.Background(), path, target, fsop.Options{Overwrite: true, SkipErrors: true}, nil, nil); cerr != nil {
			return "", err
		}
		if rerr := os.RemoveAll(path); rerr != nil {
			return "", rerr
		}
	}

	infoPath := filepath.Join(dir, "info", filepath.Base(target)+".trashinfo")
	body := "[Trash Info]\nPath=" + escapeTrashPath(path) +
		"\nDeletionDate=" + time.Now().Format("2006-01-02T15:04:05") + "\n"
	if err := os.WriteFile(infoPath, []byte(body), 0o600); err != nil {
		return "", err
	}
	return infoPath, nil
}

func escapeTrashPath(p string) string {
	repl := strings.NewReplacer("%", "%25", " ", "%20", "\n", "%0A", "\t", "%09")
	return repl.Replace(p)
}

func unescapeTrashPath(p string) string {
	repl := strings.NewReplacer("%20", " ", "%0A", "\n", "%09", "\t", "%25", "%")
	return repl.Replace(p)
}

// TrashDirs lists every trash directory we can see: the home trash plus the
// per-user trash directories on other mounted filesystems.
func TrashDirs() []string {
	out := []string{filepath.Join(DataDir(), "Trash")}
	uid := strconv.Itoa(os.Getuid())
	for _, m := range readMounts() {
		for _, candidate := range []string{
			filepath.Join(m.path, ".Trash", uid),
			filepath.Join(m.path, ".Trash-"+uid),
		} {
			if st, err := os.Stat(filepath.Join(candidate, "info")); err == nil && st.IsDir() {
				out = append(out, candidate)
			}
		}
	}
	return out
}

// ListTrash enumerates the items currently in the trash.
func ListTrash() []model.TrashEntry {
	var out []model.TrashEntry
	for _, dir := range TrashDirs() {
		infoDir := filepath.Join(dir, "info")
		entries, err := os.ReadDir(infoDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".trashinfo") {
				continue
			}
			infoPath := filepath.Join(infoDir, e.Name())
			rec, err := parseTrashInfo(infoPath)
			if err != nil {
				continue
			}
			payload := filepath.Join(dir, "files", strings.TrimSuffix(e.Name(), ".trashinfo"))
			if _, err := os.Lstat(payload); err != nil {
				continue
			}
			entry := model.TrashEntry{
				Info:      infoPath,
				Path:      rec.path,
				Name:      filepath.Base(rec.path),
				Trashed:   payload,
				DeletedAt: rec.deleted.UnixMilli(),
			}
			if st, err := os.Lstat(payload); err == nil {
				entry.IsDir = st.IsDir()
				entry.Size = st.Size()
			}
			out = append(out, entry)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].DeletedAt > out[j].DeletedAt })
	return out
}

type trashInfoRecord struct {
	path    string
	deleted time.Time
}

func parseTrashInfo(path string) (trashInfoRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return trashInfoRecord{}, err
	}
	defer f.Close()

	var rec trashInfoRecord
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if key, value, ok := strings.Cut(line, "="); ok {
			switch strings.TrimSpace(key) {
			case "Path":
				rec.path = unescapeTrashPath(strings.TrimSpace(value))
			case "DeletionDate":
				if t, err := time.ParseInLocation("2006-01-02T15:04:05",
					strings.TrimSpace(value), time.Local); err == nil {
					rec.deleted = t
				}
			}
		}
	}
	if rec.path == "" {
		return rec, fmt.Errorf("trash info %s has no Path entry", path)
	}
	return rec, nil
}

// RestoreTrash moves a trashed item back to its original location. When the
// original location is occupied the restored name is made unique.
func RestoreTrash(infoPath string) (string, error) {
	rec, err := parseTrashInfo(infoPath)
	if err != nil {
		return "", err
	}
	payload := filepath.Join(filepath.Dir(filepath.Dir(infoPath)), "files",
		strings.TrimSuffix(filepath.Base(infoPath), ".trashinfo"))
	if _, err := os.Lstat(payload); err != nil {
		return "", err
	}
	destDir := filepath.Dir(rec.path)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(destDir, filepath.Base(rec.path))
	if _, err := os.Lstat(target); err == nil {
		target = filepath.Join(destDir, fsx.UniqueName(destDir, filepath.Base(rec.path)))
	}
	if err := os.Rename(payload, target); err != nil {
		if cerr := fsop.CopyTree(context.Background(), payload, target, fsop.Options{Overwrite: true, SkipErrors: true}, nil, nil); cerr != nil {
			return "", err
		}
		if rerr := os.RemoveAll(payload); rerr != nil {
			return "", rerr
		}
	}
	_ = os.Remove(infoPath)
	return target, nil
}

// DeleteTrashEntry permanently removes a single trashed item.
func DeleteTrashEntry(infoPath string) error {
	payload := filepath.Join(filepath.Dir(filepath.Dir(infoPath)), "files",
		strings.TrimSuffix(filepath.Base(infoPath), ".trashinfo"))
	if err := os.RemoveAll(payload); err != nil {
		return err
	}
	return os.Remove(infoPath)
}

// EmptyTrash permanently removes every trashed item and reports how many
// payloads were removed.
func EmptyTrash() (int, error) {
	count := 0
	var firstErr error
	for _, dir := range TrashDirs() {
		for _, sub := range []string{"files", "info"} {
			entries, err := os.ReadDir(filepath.Join(dir, sub))
			if err != nil {
				continue
			}
			for _, e := range entries {
				if sub == "files" {
					count++
				}
				if err := os.RemoveAll(filepath.Join(dir, sub, e.Name())); err != nil && firstErr == nil {
					firstErr = err
				}
			}
		}
	}
	return count, firstErr
}

// TrashCount returns how many items are currently in the trash.
func TrashCount() int { return len(ListTrash()) }
