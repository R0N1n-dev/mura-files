package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchReportsChanges(t *testing.T) {
	dir := t.TempDir()
	events := make(chan Event, 8)
	watcher, err := New(40*time.Millisecond, func(event Event) {
		select {
		case events <- event:
		default:
		}
	})
	if err != nil {
		t.Skipf("inotify is unavailable: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Watch(dir); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	if watcher.Current() != dir {
		t.Errorf("Current = %q, want %q", watcher.Current(), dir)
	}

	created := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(created, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-events:
		if event.Dir != dir {
			t.Errorf("event dir = %q", event.Dir)
		}
		if !event.Any {
			t.Error("event should be flagged as a change")
		}
		if len(event.Created) == 0 && len(event.Changed) == 0 && len(event.Renamed) == 0 {
			t.Errorf("expected the new file to be reported, got %+v", event)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no filesystem event was delivered")
	}
}

func TestWatchSwitchesDirectories(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	watcher, err := New(30*time.Millisecond, nil)
	if err != nil {
		t.Skipf("inotify is unavailable: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Watch(first); err != nil {
		t.Fatal(err)
	}
	if err := watcher.Watch(second); err != nil {
		t.Fatal(err)
	}
	if watcher.Current() != second {
		t.Errorf("Current = %q, want %q", watcher.Current(), second)
	}
	watcher.Unwatch()
	if watcher.Current() != "" {
		t.Errorf("Current after Unwatch = %q", watcher.Current())
	}
}

func TestWatchMissingDirectoryFails(t *testing.T) {
	watcher, err := New(30*time.Millisecond, nil)
	if err != nil {
		t.Skipf("inotify is unavailable: %v", err)
	}
	defer watcher.Close()
	if err := watcher.Watch(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("watching a missing directory should fail")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	watcher, err := New(30*time.Millisecond, nil)
	if err != nil {
		t.Skipf("inotify is unavailable: %v", err)
	}
	if err := watcher.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := watcher.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
