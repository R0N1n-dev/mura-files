package config

import (
	"os"
	"path/filepath"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return New()
}

func TestDefaultsAreSensible(t *testing.T) {
	defaults := DefaultSettings()
	if defaults.Theme != "system" {
		t.Errorf("default theme = %q", defaults.Theme)
	}
	if !defaults.UseTrash {
		t.Error("deletes should go to the trash by default")
	}
	if !defaults.ConfirmDelete {
		t.Error("deleting should ask for confirmation by default")
	}
	if defaults.ViewMode != "list" {
		t.Errorf("default view = %q", defaults.ViewMode)
	}
	if !defaults.ShowThumbnails || !defaults.ShowSidebar || !defaults.ShowStatusBar {
		t.Error("the default layout should show thumbnails, the sidebar and the status bar")
	}
	if defaults.SidebarWidth < 150 || defaults.PreviewWidth < 200 {
		t.Error("default panel widths look wrong")
	}
}

func TestLoadReturnsDefaultsWhenMissing(t *testing.T) {
	store := newStore(t)
	settings := store.Load()
	if settings.Theme != "system" {
		t.Errorf("expected defaults, got %+v", settings)
	}
	if _, err := os.Stat(filepath.Join(store.dir, "settings.json")); err == nil {
		t.Error("Load should not write a file on its own")
	}
}

func TestSaveAndReload(t *testing.T) {
	store := newStore(t)
	settings := store.Load()
	settings.Theme = "dark"
	settings.ViewMode = "grid"
	settings.ShowHidden = true
	store.Save(settings)

	if _, err := os.Stat(store.settingsPath()); err != nil {
		t.Fatalf("settings file was not written: %v", err)
	}

	fresh := New()
	fresh.dir = store.dir
	reloaded := fresh.Load()
	if reloaded.Theme != "dark" || reloaded.ViewMode != "grid" || !reloaded.ShowHidden {
		t.Fatalf("settings did not survive a reload: %+v", reloaded)
	}
	// Untouched fields must keep their defaults.
	if !reloaded.UseTrash {
		t.Error("unrelated fields should fall back to their defaults")
	}
}

func TestPatchMergesSingleField(t *testing.T) {
	store := newStore(t)
	store.Load()
	store.Patch(map[string]any{"theme": "light"})
	store.Patch(map[string]any{"gridSize": 160})

	current := store.Settings()
	if current.Theme != "light" {
		t.Errorf("theme = %q", current.Theme)
	}
	if current.GridSize != 160 {
		t.Errorf("gridSize = %d", current.GridSize)
	}
	if !current.ShowSidebar {
		t.Error("a patch must not clear unrelated settings")
	}
}

func TestPatchIgnoresUnknownKeys(t *testing.T) {
	store := newStore(t)
	store.Load()
	store.Patch(map[string]any{"nonsense": true, "theme": "dark"})
	if store.Settings().Theme != "dark" {
		t.Error("known keys should still apply")
	}
}

func TestBookmarksCRUD(t *testing.T) {
	store := newStore(t)
	if len(store.Bookmarks()) != 0 {
		t.Fatal("a fresh store should have no bookmarks")
	}

	list := store.AddBookmark("Projects", "/home/user/projects", "folder")
	if len(list) != 1 || list[0].Name != "Projects" {
		t.Fatalf("AddBookmark returned %+v", list)
	}

	// Adding the same path twice must not duplicate it.
	list = store.AddBookmark("Projects again", "/home/user/projects", "folder")
	if len(list) != 1 {
		t.Fatalf("duplicate paths should be ignored, got %d", len(list))
	}

	store.AddBookmark("Music", "/home/user/music", "music")
	store.AddBookmark("Work", "/home/user/work", "folder")

	store.RenameBookmark("/home/user/music", "Tracks")
	byPath := map[string]string{}
	for _, bookmark := range store.Bookmarks() {
		byPath[bookmark.Path] = bookmark.Name
	}
	if byPath["/home/user/music"] != "Tracks" {
		t.Errorf("rename failed: %+v", byPath)
	}

	store.ReorderBookmarks([]string{"/home/user/work", "/home/user/projects", "/home/user/music"})
	ordered := store.Bookmarks()
	if len(ordered) != 3 || ordered[0].Name != "Work" {
		t.Fatalf("reorder failed: %+v", ordered)
	}

	store.RemoveBookmark("/home/user/projects")
	if len(store.Bookmarks()) != 2 {
		t.Errorf("remove failed: %+v", store.Bookmarks())
	}
}

func TestBookmarksPersist(t *testing.T) {
	store := newStore(t)
	store.AddBookmark("Saved", "/tmp/saved", "folder")

	fresh := New()
	fresh.dir = store.dir
	list := fresh.Bookmarks()
	if len(list) != 1 || list[0].Path != "/tmp/saved" {
		t.Fatalf("bookmarks did not persist: %+v", list)
	}
}

func TestSetBookmarksWithNilClears(t *testing.T) {
	store := newStore(t)
	store.AddBookmark("One", "/tmp/one", "folder")
	store.SetBookmarks(nil)
	if len(store.Bookmarks()) != 0 {
		t.Error("passing nil should clear the list")
	}
}
