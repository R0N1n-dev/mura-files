# Mura Files

A fast, elegant file manager for Linux, built with **Wails v3**, **Go** and **Vue 3** (plain JavaScript — no TypeScript anywhere).

Mura Files aims to be the file manager you actually want to keep: a calm, Nautilus-familiar layout with the details that usually get left out — a real tab bar, an in-window toolbar, live folder updates, image thumbnails, a preview pane, an operation queue with progress and cancellation, archives, a freedesktop-compliant Trash, and a properly dark theme.

<!-- Screenshots: add a couple of PNGs here once you have a desktop to take them on. -->

---

## Highlights

### Interface
- **Frameless window with its own chrome.** A custom title bar, tab strip and toolbar, so nothing depends on the desktop's window decorations. Window dragging and edge resizing are wired through the Wails runtime.
- **Tabs.** Drag to reorder, middle-click to close, context menu for *close others* / *close to the right*, and session restore across restarts.
- **Three view modes.** A detailed list with sortable columns, an icon grid with thumbnails, and a compact view.
- **Preview pane.** Inline images, video, audio and PDF, plus syntax-free text previews, a metadata table and quick actions.
- **Dark and light themes** with eight accent colours, following the desktop preference by default (`gsettings` / `GTK_THEME`).
- **Command palette** (`Ctrl+P`) that searches every action, place, bookmark and drive — and opens a typed path directly.
- **Rubber-band selection, inline rename, keyboard-first navigation**, and full multi-select with `Ctrl`/`Shift`.

### File handling
- **Live updates.** An inotify watcher refreshes the view when files change on disk (toggleable).
- **Background operations.** Copy, move, delete, duplicate, compress and extract run as cancellable jobs with a progress bar, transfer speed, ETA and a per-item error log.
- **Copy / cut / paste**, including *paste into this folder* and *move to…* / *copy to…* through a native folder picker.
- **Real Trash support.** Items go to the freedesktop trash (`~/.local/share/Trash` plus per-mount `.Trash-$uid`), and can be browsed, restored, restored elsewhere, or emptied from the app.
- **Archives.** Browse zip and tar/tar.gz/tar.bz2 archives as if they were folders, extract them safely (path traversal is refused), and create new ones. 7z and rar are supported when `7z`/`unar` are installed.
- **Search.** Recursive search with glob, regex, exact or substring matching; filters for size, extension, category, modification date; and optional full-text content search. Results stream in as they are found.
- **Properties dialog** with recursive file/folder counts, total size, disk usage and on-demand checksums (SHA-256, SHA-1, MD5).
- **Desktop integration.** `.desktop` entries show their real application icon and launch on double click; *Open With* lists the applications registered for the file's MIME type.

### Safety and comfort
- Deletes go to the Trash by default, with confirmation before anything permanent.
- Archives are extracted with traversal protection; nothing can write outside the chosen folder.
- A single D-Bus-unavailable environment degrades gracefully instead of refusing to start.
- Settings, bookmarks and session state live in `~/.config/mura-files` as readable JSON.

---

## Requirements

Mura Files targets **GTK 3 + WebKit2GTK 4.1**, which is what Debian 12, Ubuntu 22.04–24.04, Linux Mint, Pop!\_OS and most other current distributions ship.

| Distribution | Packages |
| --- | --- |
| Debian 12 / Ubuntu 22.04+ / Mint | `libgtk-3-dev libwebkit2gtk-4.1-dev` |
| Fedora | `gtk3-devel webkit2gtk4.1-devel` |
| Arch | `gtk3 webkit2gtk-4.1` |

You also need **Go 1.24+**, **Node.js 20+** and the **Wails v3 CLI**:

```sh
# The CLI is not on the package managers yet; install it from source.
git clone https://github.com/wailsapp/wails
cd wails/v3 && go install ./cmd/wails3
```

> **Why the `gtk3` build tag?** Wails v3 defaults to GTK 4 + WebKitGTK 6.0, which is only available on very recent distributions. `build/linux/Taskfile.yml` therefore enables the `gtk3` tag by default, so a plain `wails3 build` does the right thing. If you are on a GTK 4 system, build with `wails3 build EXTRA_TAGS=`.

---

## Building

```sh
# 1. Generate the JavaScript bindings from the Go services.
wails3 generate bindings -d frontend/bindings -f "-tags gtk3" .

# 2. Install the frontend dependencies.
cd frontend && npm install && cd ..

# 3. Build the application (frontend + Go, single binary).
wails3 build

# The result is bin/mura-files.
./bin/mura-files
```

Day to day development:

```sh
wails3 dev          # hot reload for both the Go and the Vue side
```

Packaging:

```sh
wails3 package                      # AppImage
wails3 task linux:create:deb        # .deb via nfpm
wails3 task linux:create:rpm        # .rpm
```

### Running the tests

```sh
# Backend (the gtk3 tag is needed because the services package imports Wails).
go test -tags gtk3 ./internal/...

# Frontend build check.
cd frontend && npx vite build --mode production
```

---

## Keyboard reference

| Keys | Action |
| --- | --- |
| `Ctrl+T` / `Ctrl+W` | New tab / close tab |
| `Ctrl+Tab` | Next tab |
| `Alt+Left` / `Alt+Right` | Back / forward |
| `Alt+Up` / `Backspace` | Parent folder |
| `Ctrl+L` | Edit the location bar (with `Tab` completion) |
| `Ctrl+F` | Search this folder |
| `Ctrl+P` | Command palette |
| `Ctrl+1/2/3` | List / grid / compact view |
| `Ctrl+H` | Show hidden files |
| `F11` or `Space` | Toggle the preview pane |
| `F2` | Rename |
| `Enter` | Open the selection |
| `Ctrl+C` / `Ctrl+X` / `Ctrl+V` | Copy / cut / paste |
| `Ctrl+D` | Duplicate |
| `Delete` / `Shift+Delete` | Move to Trash / delete permanently |
| `Ctrl+Shift+N` / `Ctrl+Alt+N` | New folder / new file |
| `Alt+Enter` | Properties |
| `Ctrl+,` / `Ctrl+/` | Settings / shortcut list |
| `F12` | Developer tools |

---

## Architecture

```
.
├── main.go                     application bootstrap: window, services, events
├── internal/
│   ├── model/                  shared DTOs (the frontend contract)
│   ├── fsx/                    path handling, stat, MIME, categories, listing, sorting
│   ├── fsop/                   copy / move / delete primitives
│   ├── ops/                    background job manager with progress and cancellation
│   ├── search/                 streaming recursive search
│   ├── watch/                  debounced inotify watcher
│   ├── archive/                zip and tar reading/writing, safe extraction
│   ├── thumbs/                 image thumbnail rendering with an LRU disk cache
│   ├── places/                 XDG places, mounts, freedesktop trash
│   ├── shell/                  launching apps, terminals, .desktop parsing, icon themes
│   ├── config/                 settings, bookmarks and session persistence
│   ├── assetfs/                private HTTP routes for previews and thumbnails
│   └── services/               the Wails service layer exposed to JavaScript
└── frontend/
    ├── bindings/               generated from the Go services (do not edit)
    └── src/
        ├── lib/                formatting, icons, the backend facade
        ├── store/app.js        the single reactive store
        ├── styles/             design tokens, base styles, component primitives
        └── components/         the interface
```

The Go side owns everything that touches the filesystem. The Vue side owns presentation and interaction. They meet at two places:

1. **Service bindings.** `internal/services` exposes small, focused facades (`FileService`, `OperationService`, `SearchService`, `WatchService`, `AppService`). `wails3 generate bindings` turns them into JavaScript modules under `frontend/bindings/`.
2. **Events.** Long-running work streams progress back over the event bus (`op:progress`, `search:results`, `fs:changed`, `app:notify`).

The webview also gets three private routes from `internal/assetfs`:

- `/__mura/file?path=…` — streams a file with HTTP range support, which is what makes video and audio scrubbing work in the preview pane.
- `/__mura/thumb?path=…&size=…` — a cached thumbnail.
- `/__mura/icon?name=…` — a themed desktop icon.

### Design notes

- **Sorting and filtering happen in Go.** Large folders never block the UI thread, and the frontend only renders what it is given.
- **List rendering uses `content-visibility: auto`** rather than a JavaScript virtual scroller. Folders with tens of thousands of entries stay smooth without the complexity (and scroll-anchoring bugs) of windowing.
- **Conflicts are resolved before the job starts.** `buildPlan` in `internal/services/operations.go` picks final destination names up front, so progress totals are accurate.
- **Everything is cancellable.** A `context.Context` is threaded from the job manager down into the copy loop.
- **No dependency on `gio`/`nautilus`.** Trash, mounts and launching are implemented directly against the freedesktop specifications, with `xdg-open`/`udisksctl`/`7z` used opportunistically when present.

---

## Configuration

All state lives under `~/.config/mura-files`:

| File | Contents |
| --- | --- |
| `settings.json` | Every preference, written atomically |
| `bookmarks.json` | The sidebar's saved locations |

Thumbnails are cached in `~/.cache/mura-files/thumbnails` and trimmed to 512 MiB.

---

## Troubleshooting

**`Package gtk4 was not found`** — you are building without the `gtk3` tag on a distribution that only has GTK 3. Use `wails3 build EXTRA_TAGS=gtk3`, or just `wails3 build`, which enables it by default.

**`libwebkit2gtk-4.1` not found** — install the WebKit2GTK 4.1 development package for your distribution (see [Requirements](#requirements)).

**No thumbnails in the grid** — check that the file type is supported (PNG, JPEG, GIF, BMP, WebP, TIFF) and that *Settings → Appearance → Thumbnails in grid view* is on.

**Video or audio will not play in the preview pane** — WebKitGTK needs GStreamer codecs: install `gstreamer1.0-plugins-good` and `gstreamer1.0-plugins-bad` (or your distribution's equivalent).

**`7z` archives cannot be opened** — install `p7zip-full` (Debian/Ubuntu) or `p7zip` (Arch).

**The app exits immediately with a D-Bus error** — it should not; the single-instance hand-off is skipped when no session bus is reachable. If you hit this, please report it with the terminal output.

---

## Licence

The application code in this repository is provided as-is for you to use and adapt.

Bundled third-party material:

- The Wails v3 runtime (`github.com/wailsapp/wails/v3`) — MIT.
- The Lucide-derived icon paths in `frontend/src/lib/icons.js` — ISC.
- Inter, if you install it, is licensed under the SIL Open Font License.
