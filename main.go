// Mura Files - a fast, elegant file manager for Linux.
//
// The application is built with Wails v3. The Go side owns everything that
// touches the filesystem, while the Vue frontend renders the interface.
package main

import (
	"embed"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/R0N1n-dev/mura-files/internal/assetfs"
	"github.com/R0N1n-dev/mura-files/internal/config"
	"github.com/R0N1n-dev/mura-files/internal/model"
	"github.com/R0N1n-dev/mura-files/internal/places"
	"github.com/R0N1n-dev/mura-files/internal/services"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "0.1.0"

// The compiled frontend is embedded into the binary.
//
//go:embed all:frontend/dist
var assets embed.FS

// The window icon is embedded so the taskbar entry looks right without
// depending on the installed .desktop file.
//
//go:embed build/appicon.png
var appIcon []byte

func init() {
	// Registering the event payload types gives the binding generator enough
	// information to emit typed helpers for them.
	application.RegisterEvent[model.OpProgress](services.EventOpProgress)
	application.RegisterEvent[model.OpProgress](services.EventOpFinished)
	application.RegisterEvent[model.SearchResults](services.EventSearchResults)
	application.RegisterEvent[services.Notification](services.EventNotify)
	application.RegisterEvent[string](services.EventOpenPath)
	application.RegisterEvent[model.ThemeInfo](services.EventTheme)
}

func main() {
	store := config.New()
	settings := store.Load()
	rt := services.NewRuntime(store)

	fileService := services.NewFileService(rt)
	operationService := services.NewOperationService(rt)
	searchService := services.NewSearchService(rt)
	watchService := services.NewWatchService(rt)
	appService := services.NewAppService(rt, version)

	assetHandler := assetfs.New(rt.Thumbs)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	app := application.New(application.Options{
		Name:        "Mura Files",
		Description: "A fast, elegant file manager for Linux",
		Icon:        appIcon,
		Logger:      logger,
		LogLevel:    slog.LevelWarn,
		Services: []application.Service{
			application.NewService(fileService),
			application.NewService(operationService),
			application.NewService(searchService),
			application.NewService(watchService),
			application.NewService(appService),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.URL.Path, assetfs.Prefix) {
						assetHandler.ServeHTTP(w, r)
						return
					}
					next.ServeHTTP(w, r)
				})
			},
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "dev.mura.files",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				// Focus the existing window and, when the second launch carried
				// paths, ask the frontend to reveal them.
				rt.Emit(services.EventOpenPath, strings.Join(data.Args, "\n"))
				appService.Show()
			},
		},
		OnShutdown: func() {
			rt.Shutdown()
		},
	})

	rt.SetApp(app)
	if err := rt.StartWatcher(); err != nil {
		logger.Warn("live folder updates are unavailable", "error", err)
	}

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "Mura Files",
		Width:     1180,
		Height:    740,
		MinWidth:  760,
		MinHeight: 460,
		// The interface draws its own titlebar, so the native frame is hidden.
		Frameless:         true,
		BackgroundColour:  application.NewRGB(24, 25, 31),
		URL:               "/",
		EnableFileDrop:    true,
		DevToolsEnabled:   true,
		DefaultContextMenuDisabled: true,
		Linux: application.LinuxWindow{
			Icon:               appIcon,
			WebviewGpuPolicy:   application.WebviewGpuPolicyOnDemand,
		},
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
			Backdrop: application.MacBackdropTranslucent,
		},
	})

	rt.SetMainWindow(window)

	// Live folder updates are started from the frontend, so restore the last
	// used state here.
	rt.SetWatchEnabled(settings.WatchEnabled)

	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		files := event.Context().DroppedFiles()
		if len(files) == 0 {
			return
		}
		target := ""
		if details := event.Context().DropTargetDetails(); details != nil {
			target = details.ElementID
		}
		rt.Emit(services.EventNotify, services.Notification{
			Kind:    "drop",
			Title:   target,
			Message: strings.Join(files, "\n"),
		})
	})

	window.OnWindowEvent(events.Common.WindowMaximise, func(*application.WindowEvent) {
		rt.Emit(services.EventTheme, model.ThemeInfo{Dark: true, Source: "window"})
	})

	app.Event.OnApplicationEvent(events.Common.ThemeChanged, func(*application.ApplicationEvent) {
		// The desktop colour scheme changed: tell the frontend to re-resolve
		// the "follow system" theme option.
		rt.Emit(services.EventNotify, services.Notification{Kind: "theme", Title: "system"})
	})

	// Make sure the data directories exist before the frontend asks for them.
	_ = os.MkdirAll(places.ConfigDir(), 0o755)
	_ = os.MkdirAll(places.CacheDir(), 0o755)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
