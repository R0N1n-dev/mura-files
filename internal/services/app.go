package services

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/R0N1n-dev/mura-files/internal/fsx"
	"github.com/R0N1n-dev/mura-files/internal/model"
	"github.com/R0N1n-dev/mura-files/internal/places"
	"github.com/R0N1n-dev/mura-files/internal/shell"
)

// AppService exposes application level features: system information, settings,
// bookmarks, trash management, native dialogs and window controls.
type AppService struct {
	rt      *Runtime
	version string
}

// NewAppService creates the service.
func NewAppService(rt *Runtime, version string) *AppService {
	return &AppService{rt: rt, version: version}
}

// ServiceName identifies the service in logs and bindings.
func (s *AppService) ServiceName() string { return "App" }

// Info returns the system description shown at boot and in the about dialog.
func (s *AppService) Info() model.SystemInfo {
	dark, _ := shell.DarkMode()
	return model.SystemInfo{
		OS:       shell.OSInfo(),
		Version:  s.version,
		DarkMode: dark,
		Accent:   shell.AccentColour(),
		Terminal: shell.DetectTerminal(),
		Tools:    shell.Tools(),
		Paths:    places.WellKnown(),
	}
}

// Theme reports the desktop colour scheme so the frontend can follow it.
func (s *AppService) Theme() model.ThemeInfo {
	dark, source := shell.DarkMode()
	return model.ThemeInfo{Dark: dark, Accent: shell.AccentColour(), Source: source}
}

// Settings returns the persisted configuration.
func (s *AppService) Settings() model.Settings { return s.rt.Config.Settings() }

// SaveSettings persists a full settings object.
func (s *AppService) SaveSettings(settings model.Settings) model.Settings {
	stored := s.rt.Config.Save(settings)
	s.rt.SetWatchEnabled(stored.WatchEnabled)
	return stored
}

// PatchSettings persists a single setting without shipping the whole object.
func (s *AppService) PatchSettings(patch map[string]any) model.Settings {
	stored := s.rt.Config.Patch(patch)
	s.rt.SetWatchEnabled(stored.WatchEnabled)
	return stored
}

// Bookmarks returns the user's saved locations.
func (s *AppService) Bookmarks() []model.Bookmark { return s.rt.Config.Bookmarks() }

// AddBookmark saves a new location.
func (s *AppService) AddBookmark(name, path, icon string) []model.Bookmark {
	if name == "" {
		name = filepath.Base(fsx.Abs(path))
	}
	return s.rt.Config.AddBookmark(name, fsx.Abs(path), icon)
}

// RemoveBookmark deletes a saved location.
func (s *AppService) RemoveBookmark(path string) []model.Bookmark {
	return s.rt.Config.RemoveBookmark(fsx.Abs(path))
}

// ReorderBookmarks rewrites the sidebar ordering.
func (s *AppService) ReorderBookmarks(paths []string) []model.Bookmark {
	return s.rt.Config.ReorderBookmarks(paths)
}

// RenameBookmark changes the label of a saved location.
func (s *AppService) RenameBookmark(path, name string) []model.Bookmark {
	return s.rt.Config.RenameBookmark(fsx.Abs(path), name)
}

// ---------------------------------------------------------------------------
// Trash
// ---------------------------------------------------------------------------

// ListTrash enumerates the contents of the freedesktop trash.
func (s *AppService) ListTrash() []model.TrashEntry { return places.ListTrash() }

// RestoreTrash puts a trashed item back where it came from.
func (s *AppService) RestoreTrash(infoPath string) (string, error) {
	return places.RestoreTrash(infoPath)
}

// DeleteTrashEntry permanently removes one trashed item.
func (s *AppService) DeleteTrashEntry(infoPath string) error {
	return places.DeleteTrashEntry(infoPath)
}

// EmptyTrash permanently removes everything in the trash.
func (s *AppService) EmptyTrash() (int, error) { return places.EmptyTrash() }

// TrashCount reports how many items are in the trash.
func (s *AppService) TrashCount() int { return places.TrashCount() }

// ---------------------------------------------------------------------------
// Drives
// ---------------------------------------------------------------------------

// Drives lists mounted filesystems.
func (s *AppService) Drives() []model.Drive { return places.Drives() }

// Eject unmounts a removable drive.
func (s *AppService) Eject(path string) error { return places.Eject(path) }

// Mount asks the system to mount a device.
func (s *AppService) Mount(path string) error { return places.Mount(path) }

// ---------------------------------------------------------------------------
// Shell integration
// ---------------------------------------------------------------------------

// Open launches a path with the desktop default handler.
func (s *AppService) Open(path string) model.OpenResult { return shell.Open(fsx.Abs(path)) }

// OpenWith launches a path with a specific application.
func (s *AppService) OpenWith(app, path string) model.OpenResult {
	return shell.OpenWith(app, fsx.Abs(path))
}

// OpenTerminal opens a terminal emulator in a directory.
func (s *AppService) OpenTerminal(dir string) model.OpenResult {
	return shell.OpenTerminal(fsx.Abs(dir))
}

// RevealInFileManager asks the desktop to show a path in its own file manager.
func (s *AppService) RevealInFileManager(path string) error {
	path = fsx.Abs(path)
	if app := s.rt.App(); app != nil {
		if err := app.Env.OpenFileManager(path, true); err == nil {
			return nil
		}
	}
	shell.Open(filepath.Dir(path))
	return nil
}

// Execute runs a custom shell command in a directory.
func (s *AppService) Execute(command, dir string) model.OpenResult {
	return shell.Execute(command, fsx.Abs(dir))
}

// CopyText puts text on the system clipboard.
func (s *AppService) CopyText(text string) error {
	app := s.rt.App()
	if app == nil {
		return fmt.Errorf("application is not ready")
	}
	if app.Clipboard.SetText(text) {
		return nil
	}
	return fmt.Errorf("could not write to the clipboard")
}

// ReadText reads the system clipboard.
func (s *AppService) ReadText() (string, error) {
	app := s.rt.App()
	if app == nil {
		return "", fmt.Errorf("application is not ready")
	}
	text, ok := app.Clipboard.Text()
	if !ok {
		return "", nil
	}
	return text, nil
}

// CopyFilesToClipboard puts file URIs on the clipboard so that other
// applications can paste the selection.
func (s *AppService) CopyFilesToClipboard(paths []string, cut bool) error {
	uris := make([]string, 0, len(paths))
	for _, p := range paths {
		uris = append(uris, "file://"+urlPathEscape(fsx.Abs(p)))
	}
	return s.CopyText(strings.Join(uris, "\n"))
}

// urlPathEscape escapes a path for use inside a file:// URI.
func urlPathEscape(path string) string {
	replacer := strings.NewReplacer(
		"%", "%25", " ", "%20", "#", "%23", "?", "%3F",
		"[", "%5B", "]", "%5D", "'", "%27",
	)
	return replacer.Replace(path)
}

// ---------------------------------------------------------------------------
// Native dialogs
// ---------------------------------------------------------------------------

// ChooseFolder opens a native folder picker.
func (s *AppService) ChooseFolder(title, start string) (string, error) {
	app := s.rt.App()
	if app == nil {
		return "", fmt.Errorf("application is not ready")
	}
	dialog := app.Dialog.OpenFile().
		SetTitle(orDefault(title, "Choose a folder")).
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		ShowHiddenFiles(true)
	if start != "" {
		dialog.SetDirectory(fsx.Abs(start))
	}
	if s.rt.MainWindow() != nil {
		dialog.AttachToWindow(s.rt.MainWindow())
	}
	return dialog.PromptForSingleSelection()
}

// ChooseFiles opens a native file picker.
func (s *AppService) ChooseFiles(title, start string, multiple bool) ([]string, error) {
	app := s.rt.App()
	if app == nil {
		return nil, fmt.Errorf("application is not ready")
	}
	dialog := app.Dialog.OpenFile().
		SetTitle(orDefault(title, "Choose files")).
		CanChooseFiles(true).
		CanChooseDirectories(false).
		ShowHiddenFiles(true)
	if start != "" {
		dialog.SetDirectory(fsx.Abs(start))
	}
	if s.rt.MainWindow() != nil {
		dialog.AttachToWindow(s.rt.MainWindow())
	}
	if multiple {
		return dialog.PromptForMultipleSelection()
	}
	file, err := dialog.PromptForSingleSelection()
	if err != nil || file == "" {
		return nil, err
	}
	return []string{file}, nil
}

// ChooseSaveFile opens a native save dialog.
func (s *AppService) ChooseSaveFile(title, start, filename string) (string, error) {
	app := s.rt.App()
	if app == nil {
		return "", fmt.Errorf("application is not ready")
	}
	dialog := app.Dialog.SaveFile().
		SetMessage(orDefault(title, "Save as")).
		ShowHiddenFiles(true)
	if start != "" {
		dialog.SetDirectory(fsx.Abs(start))
	}
	if filename != "" {
		dialog.SetFilename(filename)
	}
	if s.rt.MainWindow() != nil {
		dialog.AttachToWindow(s.rt.MainWindow())
	}
	return dialog.PromptForSingleSelection()
}

// Message shows a native message dialog. It is fire and forget: in-app
// confirmations are handled by the frontend so they match the app styling.
func (s *AppService) Message(kind, title, message string) error {
	app := s.rt.App()
	if app == nil {
		return fmt.Errorf("application is not ready")
	}
	var dialog *application.MessageDialog
	switch kind {
	case "warning":
		dialog = app.Dialog.Warning()
	case "error":
		dialog = app.Dialog.Error()
	case "question":
		dialog = app.Dialog.Question()
	default:
		dialog = app.Dialog.Info()
	}
	dialog.SetTitle(orDefault(title, "Mura Files")).SetMessage(message)
	if s.rt.MainWindow() != nil {
		dialog.AttachToWindow(s.rt.MainWindow())
	}
	dialog.Show()
	return nil
}

func orDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// ---------------------------------------------------------------------------
// Window controls (the app draws its own titlebar)
// ---------------------------------------------------------------------------

// WindowState reports the native window state.
func (s *AppService) WindowState() model.WindowState {
	if s.rt.MainWindow() == nil {
		return model.WindowState{}
	}
	width, height := s.rt.MainWindow().Size()
	return model.WindowState{
		Maximised:  s.rt.MainWindow().IsMaximised(),
		Fullscreen: s.rt.MainWindow().IsFullscreen(),
		Focused:    s.rt.MainWindow().IsFocused(),
		Width:      width,
		Height:     height,
	}
}

// Minimise sends the window to the taskbar.
func (s *AppService) Minimise() {
	if s.rt.MainWindow() != nil {
		s.rt.MainWindow().Minimise()
	}
}

// Maximise maximises the window.
func (s *AppService) Maximise() {
	if s.rt.MainWindow() != nil {
		s.rt.MainWindow().Maximise()
	}
}

// UnMaximise restores the window from its maximised state.
func (s *AppService) UnMaximise() {
	if s.rt.MainWindow() != nil {
		s.rt.MainWindow().UnMaximise()
	}
}

// ToggleMaximise flips between maximised and restored.
func (s *AppService) ToggleMaximise() {
	if s.rt.MainWindow() != nil {
		s.rt.MainWindow().ToggleMaximise()
	}
}

// ToggleFullscreen flips the fullscreen state.
func (s *AppService) ToggleFullscreen() {
	if s.rt.MainWindow() != nil {
		s.rt.MainWindow().ToggleFullscreen()
	}
}

// CloseWindow closes the main window, quitting the application.
func (s *AppService) CloseWindow() {
	if s.rt.MainWindow() != nil {
		s.rt.MainWindow().Close()
	}
}

// SetTitle updates the window title to match the active folder.
func (s *AppService) SetTitle(title string) {
	if s.rt.MainWindow() != nil && title != "" {
		s.rt.MainWindow().SetTitle(title)
	}
}

// Show brings the window to the front.
func (s *AppService) Show() {
	if s.rt.MainWindow() != nil {
		s.rt.MainWindow().Show().Focus()
	}
}

// Quit terminates the application after cancelling background work.
func (s *AppService) Quit() {
	s.rt.Shutdown()
	if app := s.rt.App(); app != nil {
		app.Quit()
	}
}

// OpenDevTools opens the web inspector.
func (s *AppService) OpenDevTools() {
	if s.rt.MainWindow() != nil {
		s.rt.MainWindow().OpenDevTools()
	}
}

// Reload reloads the frontend.
func (s *AppService) Reload() {
	if s.rt.MainWindow() != nil {
		s.rt.MainWindow().Reload()
	}
}

// Version returns the application version string.
func (s *AppService) Version() string { return s.version }

// DataPaths reports where the application stores its own files.
func (s *AppService) DataPaths() map[string]string {
	return map[string]string{
		"config": places.ConfigDir(),
		"data":   places.DataDir(),
		"cache":  places.CacheDir(),
	}
}

// CacheStats reports how many thumbnails are cached and how much space they use.
func (s *AppService) CacheStats() map[string]any {
	count, bytes := s.rt.Thumbs.Stat()
	return map[string]any{"count": count, "bytes": bytes}
}

// ClearThumbnailCache empties the thumbnail cache.
func (s *AppService) ClearThumbnailCache() error { return s.rt.Thumbs.Clear() }
