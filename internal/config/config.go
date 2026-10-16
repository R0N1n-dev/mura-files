// Package config persists user settings, bookmarks and window geometry to the
// XDG configuration directory as small, human readable JSON files.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/R0N1n-dev/mura-files/internal/model"
	"github.com/R0N1n-dev/mura-files/internal/places"
)

// Store is a concurrency safe, lazily loaded settings store.
type Store struct {
	mu        sync.RWMutex
	dir       string
	settings  model.Settings
	bookmarks []model.Bookmark
	loaded    bool
}

// New creates a store rooted at the application config directory.
func New() *Store {
	return &Store{dir: places.ConfigDir()}
}

// DefaultSettings mirrors the out of the box experience: a modern, uncluttered
// file manager with a dark theme, thumbnails enabled and trash-protected deletes.
func DefaultSettings() model.Settings {
	return model.Settings{
		Theme:          "system",
		AccentColour:   "",
		ViewMode:       "list",
		GridSize:       112,
		SortBy:         "name",
		SortDesc:       false,
		DirsFirst:      true,
		GroupByKind:    false,
		ShowHidden:     false,
		ShowExtensions: true,
		ShowThumbnails: true,
		ShowPreview:    false,
		ShowSidebar:    true,
		ShowStatusBar:  true,

		ConfirmDelete: true,
		ConfirmTrash:  false,
		UseTrash:      true,

		RestoreTabs: true,
		OpenTabs:    nil,
		ActiveTab:   0,

		DateFormat: "auto",
		SizeFormat: "binary",

		ThumbnailSize: 128,
		WatchEnabled:   true,

		SidebarWidth: 244,
		PreviewWidth: 320,

		ConfirmCloseTabs: false,
		FollowLinks:      false,
		AutoExpandTree:   true,
		ShowFolderSize:   false,
	}
}

func (s *Store) settingsPath() string  { return filepath.Join(s.dir, "settings.json") }
func (s *Store) bookmarksPath() string { return filepath.Join(s.dir, "bookmarks.json") }

// Load reads the settings and bookmarks from disk, creating defaults when the
// files do not exist yet. It is safe to call repeatedly.
func (s *Store) Load() model.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loaded {
		return s.settings
	}
	s.loaded = true
	s.settings = DefaultSettings()

	if data, err := os.ReadFile(s.settingsPath()); err == nil {
		// Start from the defaults so that newly added fields keep sane values.
		merged := DefaultSettings()
		if err := json.Unmarshal(data, &merged); err == nil {
			s.settings = merged
		}
	}
	if data, err := os.ReadFile(s.bookmarksPath()); err == nil {
		var list []model.Bookmark
		if err := json.Unmarshal(data, &list); err == nil {
			s.bookmarks = list
		}
	}
	return s.settings
}

// Settings returns the current settings, loading them when necessary.
func (s *Store) Settings() model.Settings {
	s.mu.RLock()
	if s.loaded {
		defer s.mu.RUnlock()
		return s.settings
	}
	s.mu.RUnlock()
	return s.Load()
}

// Save merges and persists the supplied settings, returning the stored value.
func (s *Store) Save(in model.Settings) model.Settings {
	s.Load()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = in
	s.writeLocked(s.settingsPath(), in)
	return s.settings
}

// Patch applies a partial update expressed as a JSON object on top of the
// current settings. This lets the frontend persist a single field without
// shipping the whole object.
func (s *Store) Patch(patch map[string]any) model.Settings {
	s.Load()
	raw, err := json.Marshal(s.Settings())
	if err != nil {
		return s.Settings()
	}
	var merged map[string]any
	if err := json.Unmarshal(raw, &merged); err != nil {
		return s.Settings()
	}
	for k, v := range patch {
		merged[k] = v
	}
	combined, err := json.Marshal(merged)
	if err != nil {
		return s.Settings()
	}
	var out model.Settings
	if err := json.Unmarshal(combined, &out); err != nil {
		return s.Settings()
	}
	return s.Save(out)
}

func (s *Store) writeLocked(path string, value any) {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return
	}
	// Write to a temporary file first so a crash cannot corrupt the settings.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

// Bookmarks returns the user's saved locations.
func (s *Store) Bookmarks() []model.Bookmark {
	s.Load()
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Bookmark, len(s.bookmarks))
	copy(out, s.bookmarks)
	return out
}

// SetBookmarks replaces the saved locations and persists them.
func (s *Store) SetBookmarks(list []model.Bookmark) []model.Bookmark {
	s.Load()
	s.mu.Lock()
	defer s.mu.Unlock()
	if list == nil {
		list = []model.Bookmark{}
	}
	s.bookmarks = list
	s.writeLocked(s.bookmarksPath(), list)
	return s.bookmarks
}

// AddBookmark appends a bookmark, ignoring duplicates by path.
func (s *Store) AddBookmark(name, path, icon string) []model.Bookmark {
	list := s.Bookmarks()
	for _, b := range list {
		if b.Path == path {
			return list
		}
	}
	id := path
	if len(id) > 64 {
		id = id[:64]
	}
	list = append(list, model.Bookmark{ID: id, Name: name, Path: path, Icon: icon})
	return s.SetBookmarks(list)
}

// RemoveBookmark deletes a bookmark by path.
func (s *Store) RemoveBookmark(path string) []model.Bookmark {
	list := s.Bookmarks()
	out := list[:0]
	for _, b := range list {
		if b.Path != path {
			out = append(out, b)
		}
	}
	return s.SetBookmarks(out)
}

// RenameBookmark changes the label of a saved location.
func (s *Store) RenameBookmark(path, name string) []model.Bookmark {
	list := s.Bookmarks()
	for i := range list {
		if list[i].Path == path {
			list[i].Name = name
		}
	}
	return s.SetBookmarks(list)
}

// ReorderBookmarks rewrites the ordering from a list of paths.
func (s *Store) ReorderBookmarks(paths []string) []model.Bookmark {
	byPath := map[string]model.Bookmark{}
	for _, b := range s.Bookmarks() {
		byPath[b.Path] = b
	}
	out := make([]model.Bookmark, 0, len(paths))
	for _, p := range paths {
		if b, ok := byPath[p]; ok {
			out = append(out, b)
		}
	}
	return s.SetBookmarks(out)
}
