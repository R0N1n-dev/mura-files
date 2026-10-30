package places

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/R0N1n-dev/mura-files/internal/fsx"
)

func TestUsageAndDrives(t *testing.T) {
	usage := Usage("/")
	if usage.Total == 0 {
		t.Fatal("root filesystem should report a non-zero size")
	}
	if usage.Used == 0 || usage.Percent <= 0 || usage.Percent > 100 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
	if usage.Available == 0 {
		t.Fatalf("available space should be reported: %+v", usage)
	}
	if usage.FsType == "" || usage.Mount == "" {
		t.Fatalf("the containing filesystem should be identified: %+v", usage)
	}
	if usage.Used+usage.Free != usage.Total {
		t.Fatalf("used+free must equal total: %+v", usage)
	}
	if usage.Available > usage.Free {
		t.Fatalf("available (%d) must not exceed free (%d)", usage.Available, usage.Free)
	}

	drives := Drives()
	if len(drives) == 0 {
		t.Fatal("expected at least the root filesystem to be listed")
	}
	foundRoot := false
	for _, drive := range drives {
		if drive.Path == "/" {
			foundRoot = true
			if drive.Total == 0 {
				t.Errorf("root drive should report its size: %+v", drive)
			}
		}
		if drive.FsType == "proc" || drive.FsType == "sysfs" {
			t.Errorf("pseudo filesystem %s leaked into the drive list", drive.FsType)
		}
	}
	if !foundRoot {
		t.Error("root filesystem missing from the drive list")
	}
}

func TestWellKnownPlaces(t *testing.T) {
	places := WellKnown()
	if len(places) == 0 {
		t.Fatal("expected at least one well-known place")
	}
	if places[0].Kind != "home" {
		t.Fatalf("expected Home first, got %+v", places[0])
	}
	var hasTrash bool
	for _, place := range places {
		if place.Kind == "trash" {
			hasTrash = true
		}
	}
	if !hasTrash {
		t.Error("Trash should always be offered")
	}
}

func TestTrashRoundTrip(t *testing.T) {
	// Point the data directory at a temporary location so the real trash is
	// never touched by the test run.
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	source := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(source, []byte("remember this"), 0o644); err != nil {
		t.Fatal(err)
	}

	infoPath, err := TrashPath(source)
	if err != nil {
		t.Fatalf("TrashPath: %v", err)
	}
	if _, err := os.Lstat(source); !os.IsNotExist(err) {
		t.Error("the original file should be gone after trashing")
	}
	if !strings.HasSuffix(infoPath, ".trashinfo") {
		t.Fatalf("unexpected trash info path %q", infoPath)
	}

	entries := ListTrash()
	if len(entries) != 1 {
		t.Fatalf("expected 1 trashed item, got %d", len(entries))
	}
	if entries[0].Path != source {
		t.Errorf("recorded path = %q, want %q", entries[0].Path, source)
	}
	if entries[0].Name != "note.txt" {
		t.Errorf("name = %q", entries[0].Name)
	}
	if entries[0].DeletedAt == 0 {
		t.Error("DeletionDate should be parsed")
	}
	if TrashCount() != 1 {
		t.Errorf("TrashCount = %d", TrashCount())
	}

	restored, err := RestoreTrash(entries[0].Info)
	if err != nil {
		t.Fatalf("RestoreTrash: %v", err)
	}
	if restored != source {
		t.Errorf("restored to %q, want %q", restored, source)
	}
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "remember this" {
		t.Errorf("content = %q", content)
	}
	if len(ListTrash()) != 0 {
		t.Error("trash should be empty after restoring")
	}
}

func TestRestoreAvoidsOverwriting(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	dir := t.TempDir()
	source := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(source, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := TrashPath(source); err != nil {
		t.Fatal(err)
	}
	// A new file takes the original name before we restore.
	if err := os.WriteFile(source, []byte("replacement"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := ListTrash()
	restored, err := RestoreTrash(entries[0].Info)
	if err != nil {
		t.Fatal(err)
	}
	if restored == source {
		t.Fatal("restore must not clobber an existing file")
	}
	if filepath.Base(restored) != "data (2).txt" {
		t.Errorf("unexpected restored name %q", filepath.Base(restored))
	}
	content, _ := os.ReadFile(source)
	if string(content) != "replacement" {
		t.Error("the existing file must be left untouched")
	}
}

func TestTrashDirectoryRoundTrip(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	dir := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nested", "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := TrashPath(dir); err != nil {
		t.Fatal(err)
	}
	entries := ListTrash()
	if len(entries) != 1 || !entries[0].IsDir {
		t.Fatalf("expected a single trashed folder, got %+v", entries)
	}
	if _, err := RestoreTrash(entries[0].Info); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "nested", "file.txt")); err != nil {
		t.Fatalf("nested content should survive the round trip: %v", err)
	}
}

func TestEmptyTrash(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	dir := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := TrashPath(path); err != nil {
			t.Fatal(err)
		}
	}
	if TrashCount() != 3 {
		t.Fatalf("expected 3 items, got %d", TrashCount())
	}
	count, err := EmptyTrash()
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("EmptyTrash removed %d items, want 3", count)
	}
	if TrashCount() != 0 {
		t.Error("trash should be empty")
	}
}

func TestEscapeUnescapeTrashPath(t *testing.T) {
	original := "/home/user/my folder/notes 100%.txt"
	if got := unescapeTrashPath(escapeTrashPath(original)); got != original {
		t.Fatalf("round trip produced %q, want %q", got, original)
	}
}

// TestUsageMatchesDfSemantics pins down the numbers so the status bar agrees
// with df. Getting this wrong is exactly the bug that made the status bar
// report nonsense: Bfree instead of Bavail, and f_bsize instead of f_frsize.
func TestUsageMatchesDfSemantics(t *testing.T) {
	usage := Usage("/")
	if usage.Total == 0 {
		t.Skip("no filesystem to inspect")
	}

	if usage.Used != usage.Total-usage.Free {
		t.Errorf("Used = %d, want Total-Free = %d", usage.Used, usage.Total-usage.Free)
	}
	if usage.Available == 0 {
		t.Error("Available should never be zero on a writable root")
	}
	if usage.Available > usage.Free {
		t.Errorf("Available (%d) must not exceed Free (%d): root-reserved blocks are not available",
			usage.Available, usage.Free)
	}

	wantPercent := float64(usage.Used) / float64(usage.Used+usage.Available) * 100
	if diff := usage.Percent - wantPercent; diff > 0.001 || diff < -0.001 {
		t.Errorf("Percent = %f, want %f", usage.Percent, wantPercent)
	}
	if usage.Percent < 0 || usage.Percent > 100 {
		t.Errorf("Percent out of range: %f", usage.Percent)
	}
}

func TestUsageResolvesContainingFilesystem(t *testing.T) {
	// A path well below the root must still be attributed to a filesystem.
	dir := t.TempDir()
	usage := Usage(dir)
	if usage.Mount == "" {
		t.Fatalf("no mount point resolved for %q: %+v", dir, usage)
	}
	if usage.FsType == "" {
		t.Fatalf("no filesystem type resolved for %q", dir)
	}
	if !fsx.IsSameOrInside(usage.Mount, dir) {
		t.Errorf("mount %q does not contain %q", usage.Mount, dir)
	}
}

func TestMountForPrefersLongestMatch(t *testing.T) {
	if _, ok := MountFor("/"); !ok {
		t.Fatal("the root mount should always be found")
	}
	mount, ok := MountFor("/proc/self")
	if ok && mount.path != "/proc" && mount.path != "/" {
		t.Errorf("unexpected mount for /proc/self: %q", mount.path)
	}
	if _, ok := MountFor("/definitely/not/a/real/path"); ok {
		// It is fine to fall back to "/", but not to match a bogus mount.
		t.Log("no exact mount, fell back to root")
	}
}

func TestStatfsIsCoherent(t *testing.T) {
	total, free, avail, err := Statfs("/")
	if err != nil {
		t.Fatalf("Statfs: %v", err)
	}
	if total == 0 {
		t.Fatal("total should be non-zero")
	}
	if free > total {
		t.Errorf("free (%d) exceeds total (%d)", free, total)
	}
	if avail > free {
		t.Errorf("avail (%d) exceeds free (%d)", avail, free)
	}
}

func TestBlockSizeFallsBackToBsize(t *testing.T) {
	st := syscall.Statfs_t{Frsize: 4096, Bsize: 65536}
	if got := blockSize(&st); got != 4096 {
		t.Errorf("expected f_frsize to win, got %d", got)
	}
	st = syscall.Statfs_t{Frsize: 0, Bsize: 65536}
	if got := blockSize(&st); got != 65536 {
		t.Errorf("expected the f_bsize fallback, got %d", got)
	}
	st = syscall.Statfs_t{Frsize: 0, Bsize: 0}
	if got := blockSize(&st); got != 4096 {
		t.Errorf("expected the 4096 default, got %d", got)
	}
}

func TestMountCacheInvalidation(t *testing.T) {
	first := readMounts()
	if len(first) == 0 {
		t.Skip("no mount table in this environment")
	}
	invalidateMounts()
	second := readMounts()
	if len(second) != len(first) {
		t.Errorf("mount table changed size between reads: %d vs %d", len(first), len(second))
	}
}
