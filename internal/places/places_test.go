package places

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUsageAndDrives(t *testing.T) {
	usage := Usage("/")
	if usage.Total == 0 {
		t.Fatal("root filesystem should report a non-zero size")
	}
	if usage.Used == 0 || usage.Percent <= 0 || usage.Percent > 100 {
		t.Fatalf("unexpected usage: %+v", usage)
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
