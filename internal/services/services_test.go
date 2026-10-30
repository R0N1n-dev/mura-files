package services

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/R0N1n-dev/mura-files/internal/config"
	"github.com/R0N1n-dev/mura-files/internal/model"
	"github.com/R0N1n-dev/mura-files/internal/undo"
)

// newTestRuntime builds a Runtime with isolated config, cache and trash
// directories, and an event hook that records everything emitted.
// eventLog records emitted events from any goroutine.
type eventLog struct {
	mu    sync.Mutex
	items []captured
}

func (l *eventLog) add(event captured) {
	l.mu.Lock()
	l.items = append(l.items, event)
	l.mu.Unlock()
}

func (l *eventLog) snapshot() []captured {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]captured, len(l.items))
	copy(out, l.items)
	return out
}

func newTestRuntime(t *testing.T) (*Runtime, *eventLog) {
	t.Helper()
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(base, "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(base, "cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(base, "data"))
	t.Setenv("HOME", filepath.Join(base, "home"))

	rt := NewRuntime(config.New())
	log := &eventLog{}
	rt.SetEventHook(func(name string, data ...any) {
		log.add(captured{name: name, data: data})
	})
	return rt, log
}

type captured struct {
	name string
	data []any
}

func TestFileServiceCreatesAndUndoes(t *testing.T) {
	rt, _ := newTestRuntime(t)
	files := NewFileService(rt)
	dir := t.TempDir()

	entry, err := files.CreateDir(dir, "Project")
	if err != nil {
		t.Fatalf("CreateDir: %v", err)
	}
	if !entry.IsDir || entry.Name != "Project" {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if rt.Undo.Len() != 1 {
		t.Fatalf("creating a folder should be undoable, depth = %d", rt.Undo.Len())
	}

	file, err := files.CreateFile(dir, "notes.txt", "hello")
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if rt.Undo.Len() != 2 {
		t.Fatalf("depth = %d, want 2", rt.Undo.Len())
	}

	// Undo removes the file, then the folder.
	undoEntry, ok := rt.Undo.Pop()
	if !ok {
		t.Fatal("expected an undo entry")
	}
	if _, err := undoApply(t, undoEntry); err != nil {
		t.Fatalf("undo create: %v", err)
	}
	if _, err := os.Lstat(file.Path); !os.IsNotExist(err) {
		t.Error("the created file should be gone after undo")
	}

	undoEntry, _ = rt.Undo.Pop()
	if _, err := undoApply(t, undoEntry); err != nil {
		t.Fatalf("undo create: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "Project")); !os.IsNotExist(err) {
		t.Error("the created folder should be gone after undo")
	}
}

func TestRenameIsUndoable(t *testing.T) {
	rt, _ := newTestRuntime(t)
	files := NewFileService(rt)
	dir := t.TempDir()
	original := filepath.Join(dir, "old.txt")
	if err := os.WriteFile(original, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	renamed, err := files.Rename(original, "new.txt")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if renamed.Name != "new.txt" {
		t.Fatalf("unexpected name %q", renamed.Name)
	}
	if rt.Undo.Len() != 1 {
		t.Fatalf("rename should be undoable, depth = %d", rt.Undo.Len())
	}

	undoEntry, _ := rt.Undo.Pop()
	if _, err := undoApply(t, undoEntry); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(original); err != nil {
		t.Errorf("the original name should be restored: %v", err)
	}
}

func TestTrashIsUndoable(t *testing.T) {
	rt, _ := newTestRuntime(t)
	files := NewFileService(rt)
	dir := t.TempDir()
	victim := filepath.Join(dir, "trash-me.txt")
	if err := os.WriteFile(victim, []byte("bye"), 0o644); err != nil {
		t.Fatal(err)
	}

	moved, err := files.Trash([]string{victim})
	if err != nil {
		t.Fatalf("Trash: %v", err)
	}
	if moved != 1 {
		t.Fatalf("moved = %d", moved)
	}
	if rt.Undo.Len() != 1 {
		t.Fatalf("trashing should be undoable, depth = %d", rt.Undo.Len())
	}

	undoEntry, _ := rt.Undo.Pop()
	result, err := undoApply(t, undoEntry)
	if err != nil {
		t.Fatalf("undo trash: %v", err)
	}
	if !result.OK {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := os.Stat(victim); err != nil {
		t.Errorf("the file should be back: %v", err)
	}
}

func TestCreateLink(t *testing.T) {
	rt, _ := newTestRuntime(t)
	files := NewFileService(rt)
	dir := t.TempDir()
	target := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(target, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	link, err := files.CreateLink(target, "")
	if err != nil {
		t.Fatalf("CreateLink: %v", err)
	}
	if !link.IsSymlink {
		t.Fatalf("expected a symlink, got %+v", link)
	}
	if link.Name != "real link.txt" {
		t.Errorf("default link name = %q", link.Name)
	}
	if link.LinkBroken {
		t.Error("the link should resolve")
	}
	if link.Dir != dir {
		t.Errorf("link should live beside the target, got %q", link.Dir)
	}
	if rt.Undo.Len() != 1 {
		t.Errorf("creating a link should be undoable, depth = %d", rt.Undo.Len())
	}

	// A second link must not collide with the first.
	second, err := files.CreateLink(target, "")
	if err != nil {
		t.Fatalf("second CreateLink: %v", err)
	}
	if second.Name == link.Name {
		t.Errorf("link names must be unique, both were %q", second.Name)
	}

	if _, err := files.CreateLink(filepath.Join(dir, "missing.txt"), ""); err == nil {
		t.Error("linking a missing target should fail")
	}
}

func TestListReturnsParsedEntries(t *testing.T) {
	rt, _ := newTestRuntime(t)
	files := NewFileService(rt)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	listing := files.List(dir, model.ListOptions{SortBy: "name", DirsFirst: true})
	if listing.Error != "" {
		t.Fatalf("listing error: %s", listing.Error)
	}
	if listing.Total != 2 {
		t.Fatalf("Total = %d", listing.Total)
	}
	if !listing.Entries[0].IsDir {
		t.Error("folders should come first")
	}
	if listing.Entries[1].Kind != "image" {
		t.Errorf("Kind = %q", listing.Entries[1].Kind)
	}
}

func TestFolderSizesStreamsResults(t *testing.T) {
	rt, events := newTestRuntime(t)
	files := NewFileService(rt)

	root := t.TempDir()
	big := filepath.Join(root, "big")
	if err := os.MkdirAll(filepath.Join(big, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(big, "a.bin"), make([]byte, 1000), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(big, "nested", "b.bin"), make([]byte, 500), 0o644); err != nil {
		t.Fatal(err)
	}

	jobID := files.FolderSizes([]string{big, filepath.Join(root, "a-file.txt")})
	if jobID == "" {
		t.Fatal("expected a job identifier")
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
		for _, event := range events.snapshot() {
			if event.name != EventFolderSize {
				continue
			}
			size, ok := event.data[0].(model.FolderSize)
			if !ok {
				continue
			}
			if size.Path != big {
				continue
			}
			if !size.Done {
				t.Fatal("the first message should already be complete")
			}
			if size.Bytes != 1500 {
				t.Errorf("Bytes = %d, want 1500", size.Bytes)
			}
			if size.Files != 2 {
				t.Errorf("Files = %d, want 2", size.Files)
			}
			if !strings.HasPrefix(size.Request, "size-") {
				t.Errorf("Request = %q, want a generated size-* token", size.Request)
			}
			return
		}
	}
	t.Fatal("no folder size event was emitted")
}

func TestFolderSizesIgnoresFiles(t *testing.T) {
	rt, _ := newTestRuntime(t)
	files := NewFileService(rt)
	dir := t.TempDir()
	file := filepath.Join(dir, "plain.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if jobID := files.FolderSizes([]string{file}); jobID != "" {
		t.Error("files are not folders, no job should start")
	}
}

func TestUndoDepthAndLabel(t *testing.T) {
	rt, _ := newTestRuntime(t)
	app := NewAppService(rt, "test", NewFileService(rt))

	if app.UndoDepth() != 0 {
		t.Errorf("UndoDepth = %d on a fresh runtime", app.UndoDepth())
	}
	if app.UndoLabel() != "" {
		t.Errorf("UndoLabel = %q on a fresh runtime", app.UndoLabel())
	}

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	rt.Undo.Push(model.UndoEntry{Kind: "create", Label: "Create x",
		Items: []model.UndoItem{{To: filepath.Join(dir, "x")}}})
	if app.UndoDepth() != 1 {
		t.Errorf("UndoDepth = %d, want 1", app.UndoDepth())
	}
	if app.UndoLabel() != "Create x" {
		t.Errorf("UndoLabel = %q", app.UndoLabel())
	}

	result, err := app.Undo()
	if err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if !result.OK {
		t.Errorf("unexpected result: %+v", result)
	}
	if app.UndoDepth() != 0 {
		t.Errorf("UndoDepth = %d after undo", app.UndoDepth())
	}

	// Undoing with an empty history is a no-op, not an error.
	result, err = app.Undo()
	if err != nil {
		t.Fatalf("Undo on an empty stack: %v", err)
	}
	if result.OK {
		t.Error("nothing was undone, OK should be false")
	}
	if !strings.Contains(strings.ToLower(result.Message), "nothing") {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestUndoEmitsChangeEvent(t *testing.T) {
	rt, events := newTestRuntime(t)
	app := NewAppService(rt, "test", NewFileService(rt))
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "y"), 0o755); err != nil {
		t.Fatal(err)
	}
	rt.Undo.Push(model.UndoEntry{Kind: "create",
		Items: []model.UndoItem{{To: filepath.Join(dir, "y")}}})
	if _, err := app.Undo(); err != nil {
		t.Fatal(err)
	}

	found := false
	for _, event := range events.snapshot() {
		if event.name == EventFsChanged {
			found = true
		}
	}
	if !found {
		t.Error("undo should tell the frontend that the filesystem changed")
	}
}


// undoApply is a tiny indirection so the tests read clearly.
func undoApply(t *testing.T, entry model.UndoEntry) (model.UndoResult, error) {
	t.Helper()
	return undo.Apply(entry)
}
