// Package services exposes the backend functionality to the frontend through
// Wails service bindings. Each service is a small, focused facade over the
// internal packages.
package services

import (
	"context"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/R0N1n-dev/mura-files/internal/config"
	"github.com/R0N1n-dev/mura-files/internal/model"
	"github.com/R0N1n-dev/mura-files/internal/ops"
	"github.com/R0N1n-dev/mura-files/internal/search"
	"github.com/R0N1n-dev/mura-files/internal/thumbs"
	"github.com/R0N1n-dev/mura-files/internal/watch"
)

// Event names emitted by the backend.
const (
	EventOpProgress    = "op:progress"
	EventSearchResults = "search:results"
	EventFsChanged     = "fs:changed"
	EventTheme         = "app:theme"
	EventNotify        = "app:notify"
	EventOpenPath      = "app:open-path"
	EventOpFinished    = "op:finished"
)

// Notification is a small toast message pushed from the backend.
type Notification struct {
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

// Runtime holds the shared backend state used by every service.
type Runtime struct {
	mu      sync.RWMutex
	app     *application.App
	Config  *config.Store
	Thumbs  *thumbs.Cache
	Ops     *ops.Manager
	Search  *search.Engine
	Watcher *watch.Watcher

	// watchEnabled mirrors the user setting so the watcher can be paused.
	watchEnabled bool
	// mainWindow is the primary webview window, used by the window controls.
	mainWindow *application.WebviewWindow
}

// NewRuntime wires the shared state together.
func NewRuntime(store *config.Store) *Runtime {
	rt := &Runtime{
		Config:       store,
		Thumbs:       thumbs.New(),
		Search:       search.New(),
		watchEnabled: true,
	}
	rt.Ops = ops.NewManager(rt.emitOpProgress)
	return rt
}

// SetApp records the Wails application so services can emit events and drive
// the window. It must be called right after application.New.
func (rt *Runtime) SetApp(app *application.App) {
	rt.mu.Lock()
	rt.app = app
	rt.mu.Unlock()
}

// App returns the Wails application, which may be nil during early startup.
func (rt *Runtime) App() *application.App {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.app
}

// SetMainWindow records the primary window so services can drive it.
func (rt *Runtime) SetMainWindow(w *application.WebviewWindow) {
	rt.mu.Lock()
	rt.mainWindow = w
	rt.mu.Unlock()
}

// MainWindow returns the primary window, which may be nil before it is created.
func (rt *Runtime) MainWindow() *application.WebviewWindow {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.mainWindow
}

// Emit sends a custom event to the frontend.
func (rt *Runtime) Emit(name string, data ...any) {
	if app := rt.App(); app != nil {
		app.Event.Emit(name, data...)
	}
}

func (rt *Runtime) emitOpProgress(progress model.OpProgress) {
	rt.Emit(EventOpProgress, progress)
	if progress.Status == ops.StatusDone || progress.Status == ops.StatusError ||
		progress.Status == ops.StatusCancelled {
		rt.Emit(EventOpFinished, progress)
	}
}

// StartWatcher boots the directory watcher. Failures are non-fatal because the
// file manager stays fully usable without live updates.
func (rt *Runtime) StartWatcher() error {
	w, err := watch.New(280*time.Millisecond, func(ev watch.Event) {
		rt.mu.RLock()
		enabled := rt.watchEnabled
		rt.mu.RUnlock()
		if !enabled {
			return
		}
		rt.Emit(EventFsChanged, ev)
	})
	if err != nil {
		return err
	}
	rt.mu.Lock()
	rt.Watcher = w
	rt.mu.Unlock()
	return nil
}

// SetWatchEnabled toggles live directory updates.
func (rt *Runtime) SetWatchEnabled(enabled bool) {
	rt.mu.Lock()
	rt.watchEnabled = enabled
	w := rt.Watcher
	rt.mu.Unlock()
	if !enabled && w != nil {
		w.Unwatch()
	}
}

// Shutdown stops background workers.
func (rt *Runtime) Shutdown() {
	rt.mu.RLock()
	w := rt.Watcher
	rt.mu.RUnlock()
	if w != nil {
		_ = w.Close()
	}
	rt.Search.CancelAll()
	rt.Ops.CancelAllAndWait(2 * time.Second)
}

// Guard is a helper that returns a context bounded by a timeout, used by the
// service methods that shell out to external tools.
func (rt *Runtime) Guard() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Minute)
}
