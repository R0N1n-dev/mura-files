package undo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/R0N1n-dev/mura-files/internal/model"
	"github.com/R0N1n-dev/mura-files/internal/places"
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

func TestStackPushPopAndLimit(t *testing.T) {
	stack := New()
	if stack.Len() != 0 {
		t.Fatal("a new stack should be empty")
	}
	if _, ok := stack.Pop(); ok {
		t.Fatal("Pop on an empty stack should report nothing")
	}

	stack.Push(model.UndoEntry{Kind: KindRename, Label: "first",
		Items: []model.UndoItem{{From: "/a", To: "/b"}}})
	stack.Push(model.UndoEntry{Kind: KindRename, Label: "second",
		Items: []model.UndoItem{{From: "/c", To: "/d"}}})
	if stack.Len() != 2 {
		t.Fatalf("Len = %d, want 2", stack.Len())
	}

	peeked, ok := stack.Peek()
	if !ok || peeked.Label != "second" {
		t.Fatalf("Peek = %+v, want the newest entry", peeked)
	}
	if stack.Len() != 2 {
		t.Errorf("Peek must not remove the entry, Len = %d", stack.Len())
	}

	popped, ok := stack.Pop()
	if !ok || popped.Label != "second" {
		t.Fatalf("Pop = %+v", popped)
	}
	if stack.Len() != 1 {
		t.Errorf("Len = %d after pop", stack.Len())
	}

	stack.Clear()
	if stack.Len() != 0 {
		t.Error("Clear should forget the history")
	}
}

func TestStackIgnoresEmptyEntries(t *testing.T) {
	stack := New()
	stack.Push(model.UndoEntry{Kind: KindCreate, Label: "nothing"})
	if stack.Len() != 0 {
		t.Error("an entry without items must not be recorded")
	}
}

func TestStackIsBounded(t *testing.T) {
	stack := New()
	for i := 0; i < defaultDepth+15; i++ {
		stack.Push(model.UndoEntry{Kind: KindCreate,
			Items: []model.UndoItem{{To: "/tmp/x"}}})
	}
	if stack.Len() != defaultDepth {
		t.Fatalf("Len = %d, want the cap of %d", stack.Len(), defaultDepth)
	}
}

func TestApplyTrashRestores(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	source := filepath.Join(t.TempDir(), "notes.txt")
	write(t, source, "keep me")
	infoPath, err := places.TrashPath(source)
	if err != nil {
		t.Fatalf("TrashPath: %v", err)
	}

	result, err := Apply(model.UndoEntry{
		Kind:  KindTrash,
		Label: "Trash notes.txt",
		Items: []model.UndoItem{{From: source, To: infoPath}},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !result.OK || result.Count != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("the file should be back: %v", err)
	}
	if string(data) != "keep me" {
		t.Errorf("content = %q", data)
	}
	if len(places.ListTrash()) != 0 {
		t.Error("the trash should be empty after the undo")
	}
}

func TestApplyMoveMovesBack(t *testing.T) {
	source := filepath.Join(t.TempDir(), "moved.txt")
	write(t, source, "payload")
	destination := filepath.Join(t.TempDir(), "moved.txt")
	if err := os.Rename(source, destination); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(model.UndoEntry{
		Kind:  KindMove,
		Label: "Move moved.txt",
		Items: []model.UndoItem{{From: source, To: destination}},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !result.OK || result.Count != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Error("the moved file should be gone from its new location")
	}
	if _, err := os.Stat(source); err != nil {
		t.Errorf("the original location should be restored: %v", err)
	}
}

func TestApplyRenameReverts(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "before.txt")
	renamed := filepath.Join(dir, "after.txt")
	write(t, renamed, "content")

	result, err := Apply(model.UndoEntry{
		Kind:  KindRename,
		Items: []model.UndoItem{{From: original, To: renamed}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := os.Stat(original); err != nil {
		t.Errorf("the original name should be back: %v", err)
	}
}

func TestApplyMoveDoesNotClobber(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "target.txt")
	moved := filepath.Join(t.TempDir(), "target.txt")
	write(t, moved, "incoming")
	// Something new now occupies the original path.
	write(t, original, "resident")

	result, err := Apply(model.UndoEntry{
		Kind:  KindMove,
		Items: []model.UndoItem{{From: original, To: moved}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("unexpected result: %+v", result)
	}

	resident, _ := os.ReadFile(original)
	if string(resident) != "resident" {
		t.Fatal("undo must never overwrite an existing file")
	}
	// The restored file lands beside it under a unique name.
	restored := filepath.Join(dir, "target (2).txt")
	content, err := os.ReadFile(restored)
	if err != nil {
		t.Fatalf("expected the restored copy at %q: %v", restored, err)
	}
	if string(content) != "incoming" {
		t.Errorf("restored content = %q", content)
	}
}

func TestApplyCreateRemoves(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	dir := t.TempDir()
	created := filepath.Join(dir, "new-folder")
	if err := os.MkdirAll(filepath.Join(created, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(model.UndoEntry{
		Kind:  KindCreate,
		Items: []model.UndoItem{{To: created}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Count != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := os.Stat(created); !os.IsNotExist(err) {
		t.Error("the created folder should be gone")
	}
}

func TestApplyRejectsUnknownKind(t *testing.T) {
	if _, err := Apply(model.UndoEntry{Kind: "teleport", Items: []model.UndoItem{{To: "/x"}}}); err == nil {
		t.Error("an unknown operation kind should be rejected")
	}
}

func TestApplyMissingPathIsReportedButNotFatal(t *testing.T) {
	result, err := Apply(model.UndoEntry{
		Kind:  KindMove,
		Items: []model.UndoItem{{From: "/tmp/a", To: filepath.Join(t.TempDir(), "gone")}},
	})
	if err == nil {
		t.Error("a missing source should be reported")
	}
	if result.OK {
		t.Error("nothing was undone, so OK should be false")
	}
}
