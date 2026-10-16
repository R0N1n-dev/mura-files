package fsop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestCopyTreePreservesStructure(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source")
	write(t, filepath.Join(src, "a.txt"), "alpha")
	write(t, filepath.Join(src, "nested", "b.txt"), "beta")
	write(t, filepath.Join(src, "nested", "deeper", "c.txt"), "gamma")

	dst := filepath.Join(t.TempDir(), "target")
	acc := &Result{}
	if err := CopyTree(context.Background(), src, dst, Options{PreserveAttrs: true}, nil, acc); err != nil {
		t.Fatalf("CopyTree: %v", err)
	}

	if got := read(t, filepath.Join(dst, "a.txt")); got != "alpha" {
		t.Errorf("a.txt = %q", got)
	}
	if got := read(t, filepath.Join(dst, "nested", "deeper", "c.txt")); got != "gamma" {
		t.Errorf("c.txt = %q", got)
	}
	if acc.Bytes != int64(len("alpha"+"beta"+"gamma")) {
		t.Errorf("reported %d bytes, want %d", acc.Bytes, len("alpha"+"beta"+"gamma"))
	}
	if acc.Items < 4 {
		t.Errorf("expected at least 4 items, got %d", acc.Items)
	}
}

func TestCopyTreeConflictHandling(t *testing.T) {
	src := filepath.Join(t.TempDir(), "file.txt")
	write(t, src, "new")
	dir := t.TempDir()
	dst := filepath.Join(dir, "file.txt")
	write(t, dst, "old")

	// Refusing is the safe default: an existing file must never be clobbered
	// unless the caller explicitly asked for it.
	if err := CopyTree(context.Background(), src, dst, Options{}, nil, nil); err != ErrExists {
		t.Fatalf("expected ErrExists, got %v", err)
	}
	if got := read(t, dst); got != "old" {
		t.Errorf("existing file was modified without Overwrite: %q", got)
	}

	// RenameOnConflict keeps both files, following the desktop convention.
	if err := CopyTree(context.Background(), src, dst, Options{RenameOnConflict: true}, nil, nil); err != nil {
		t.Fatalf("CopyTree with RenameOnConflict: %v", err)
	}
	if got := read(t, filepath.Join(dir, "file (2).txt")); got != "new" {
		t.Errorf("renamed copy = %q, want %q", got, "new")
	}
	if got := read(t, dst); got != "old" {
		t.Errorf("the original should be untouched, got %q", got)
	}

	// Overwrite replaces the destination in place.
	if err := CopyTree(context.Background(), src, dst, Options{Overwrite: true}, nil, nil); err != nil {
		t.Fatalf("CopyTree with overwrite: %v", err)
	}
	if got := read(t, dst); got != "new" {
		t.Errorf("overwrite did not replace the file: %q", got)
	}
}

func TestCopyTreeRecreatesSymlinks(t *testing.T) {
	src := t.TempDir()
	write(t, filepath.Join(src, "real.txt"), "content")
	if err := os.Symlink("real.txt", filepath.Join(src, "link.txt")); err != nil {
		t.Skip("symlinks unsupported")
	}

	dst := filepath.Join(t.TempDir(), "copy")
	if err := CopyTree(context.Background(), src, dst, Options{}, nil, nil); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(dst, "link.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlink should have been recreated, not dereferenced")
	}
}

func TestMovePathRenames(t *testing.T) {
	src := filepath.Join(t.TempDir(), "file.txt")
	write(t, src, "payload")
	dst := filepath.Join(t.TempDir(), "moved.txt")

	if err := MovePath(context.Background(), src, dst, Options{}, nil, nil); err != nil {
		t.Fatalf("MovePath: %v", err)
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Error("the source should be gone after a move")
	}
	if got := read(t, dst); got != "payload" {
		t.Errorf("moved content = %q", got)
	}
}

func TestMovePathAcrossDirectories(t *testing.T) {
	src := filepath.Join(t.TempDir(), "tree")
	write(t, filepath.Join(src, "a", "one.txt"), "1")
	write(t, filepath.Join(src, "b", "two.txt"), "2")
	dst := filepath.Join(t.TempDir(), "moved")

	if err := MovePath(context.Background(), src, dst, Options{Overwrite: true}, nil, nil); err != nil {
		t.Fatalf("MovePath: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "b", "two.txt")); err != nil {
		t.Fatalf("nested file missing after move: %v", err)
	}
}

func TestCancellationStopsCopy(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source")
	for i := 0; i < 20; i++ {
		write(t, filepath.Join(src, string(rune('a'+i)), "file.txt"), "x")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := CopyTree(ctx, src, filepath.Join(t.TempDir(), "dst"), Options{}, nil, nil)
	if err != ErrCancelled {
		t.Fatalf("expected ErrCancelled, got %v", err)
	}
}

func TestTreeSize(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "a.txt"), "12345")
	write(t, filepath.Join(root, "sub", "b.txt"), "123")

	files, dirs, links, bytes, truncated := TreeSize(context.Background(), root, 0)
	if files != 2 || dirs != 2 || links != 0 {
		t.Errorf("files=%d dirs=%d links=%d", files, dirs, links)
	}
	if bytes != 8 {
		t.Errorf("bytes = %d, want 8", bytes)
	}
	if truncated {
		t.Error("small tree should not be truncated")
	}
}

func TestTreeSizeLimit(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 30; i++ {
		write(t, filepath.Join(root, fmt.Sprintf("f%02d.txt", i)), "x")
	}
	_, _, _, _, truncated := TreeSize(context.Background(), root, 5)
	if !truncated {
		t.Error("expected the walk to stop at the limit")
	}
}

func TestRemovePathWithMover(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "victim.txt")
	write(t, file, "bye")

	moved := ""
	mover := func(path string) (string, error) {
		moved = path
		return "info.trashinfo", nil
	}
	if err := RemovePath(file, false, mover); err != nil {
		t.Fatal(err)
	}
	if moved != file {
		t.Errorf("mover received %q, want %q", moved, file)
	}
	if _, err := os.Lstat(file); err != nil {
		t.Error("the mover is responsible for the file, it should still exist")
	}

	if err := RemovePath(file, true, mover); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(file); !os.IsNotExist(err) {
		t.Error("permanent removal should unlink the file")
	}
}
