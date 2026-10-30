# Mura Files

A fast, elegant file manager for Linux, built with **Wails v3**, **Go** and **Vue 3** (plain JavaScript — no TypeScript anywhere).

Mura Files aims to be the file manager you actually want to keep: a calm, Nautilus-familiar layout with the details that usually get left out — a real tab bar, an in-window toolbar, live folder updates, image thumbnails, a preview pane, an operation queue with progress and cancellation, archives, a freedesktop-compliant Trash, and a properly dark theme.

<!-- Screenshots: add a couple of PNGs here once you have a desktop to take them on. -->

---

## Version 0.2 highlights

- **Fixed:** the free-space readout in the status bar. It now uses the same
  arithmetic as `df` (`f_frsize`, `f_bavail`, `used / (used + available)`) and
  refreshes the moment you change folder, after every operation, and on a slow
  poll — instead of showing another filesystem's numbers for up to 15 seconds.
- **Fixed:** mount detection reported no filesystem when the browsed path was
  not itself a mount point; it now walks up to the nearest containing mount.
- **Fixed:** a data race in the owner/group name caches that could crash the
  process under concurrent tab loads (verifiable with `-race`).
- **Fixed:** `IsSameOrInside("/", …)` always returned false, which broke mount
  attribution, recursion guards and some archive checks.
- **New:** undo history (`Ctrl+Z`), folder sizes on demand, *Make Link*,
  an rwx permission editor, an operation queue (`Ctrl+J`), reopen closed tab
  (`Ctrl+Shift+T`) and type-ahead.
- **Faster:** destination planning no longer walks the tree on the call path,
  thumbnail decodes are bounded, `.desktop` parsing is cached, and the mount
  table is cached instead of re-read on every poll.

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
- **Background operations.** Copy, move, delete, duplicate, compress, extract and folder sizing run as cancellable jobs with a progress bar, transfer speed, ETA and a per-item error log. The **operation queue** (`Ctrl+J`) shows the whole history.
- **Undo.** Trashing, moving, renaming, duplicating, extracting and creating files or folders can all be taken back with `Ctrl+Z`. Permanent deletion is deliberately *not* undoable, and says so.
- **Copy / cut / paste**, including *paste into this folder* and *move to…* / *copy to…* through a native folder picker.
- **Real Trash support.** Items go to the freedesktop trash (`~/.local/share/Trash` plus per-mount `.Trash-$uid`), and can be browsed, restored, restored elsewhere, or emptied from the app.
- **Archives.** Browse zip and tar/tar.gz/tar.bz2 archives as if they were folders, extract them safely (path traversal is refused), and create new ones. 7z and rar are supported when `7z`/`unar` are installed.
- **Symbolic links.** *Make Link* creates a relative link beside the target, named uniquely and undoable like any other creation.
- **Folder sizes on demand.** Right-click any folder for *Calculate Folder Size*, or turn on *Settings → Browsing → Show folder sizes* to measure the whole listing. Sizes are computed in a cancellable background job, cached, and invalidated when the watcher sees a change.
- **Search.** Recursive search with glob, regex, exact or substring matching; filters for size, extension, category, modification date; and optional full-text content search. Results stream in as they are found.
- **Properties dialog** with recursive file/folder counts, total size, disk usage, an **rwx permission editor** and on-demand checksums (SHA-256, SHA-1, MD5).
- **Desktop integration.** `.desktop` entries show their real application icon and launch on double click; *Open With* lists the applications registered for the file's MIME type.
- **Type-ahead.** Just start typing a name to jump to it, the way desktop file managers have always worked.
- **Reopen closed tabs** with `Ctrl+Shift+T`.

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
| `Ctrl+Shift+F` | Toggle fullscreen |
| `F2` | Rename |
| `Enter` | Open the selection |
| `Ctrl+C` / `Ctrl+X` / `Ctrl+V` | Copy / cut / paste |
| `Ctrl+Z` | Undo the last reversible operation |
| `Ctrl+D` | Duplicate |
| `Delete` / `Shift+Delete` | Move to Trash / delete permanently |
| `Ctrl+Shift+N` / `Ctrl+Alt+N` | New folder / new file |
| `Ctrl+Shift+T` | Reopen the last closed tab |
| `Ctrl+J` | Show the operation queue |
| `Alt+Enter` | Properties (including permissions) |
| `Ctrl+,` / `Ctrl+/` | Settings / shortcut list |
| `F12` | Developer tools |

Type any printable character to jump to the matching file.

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
│   ├── undo/                   bounded undo history for reversible operations
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
2. **Events.** Long-running work streams progress back over the event bus (`op:progress`, `op:finished`, `search:results`, `fs:changed`, `fs:folder-size`, `app:undo`, `app:notify`).

The webview also gets three private routes from `internal/assetfs`:

- `/__mura/file?path=…` — streams a file with HTTP range support, which is what makes video and audio scrubbing work in the preview pane.
- `/__mura/thumb?path=…&size=…` — a cached thumbnail.
- `/__mura/icon?name=…` — a themed desktop icon.

### Design notes

- **Sorting and filtering happen in Go.** Large folders never block the UI thread, and the frontend only renders what it is given.
- **Disk usage matches `df` exactly.** `statfs` block counts are multiplied by `f_frsize` (not `f_bsize`, which is the preferred I/O size and disagrees on several filesystems), the usable figure is `f_bavail` rather than `f_bfree` so blocks reserved for root are not advertised as free, and the percentage is `used / (used + available)` like `df`'s Use% column. The readout re-queries as soon as you change folder, after every operation and on a slow background poll, so it is never stale.
- **List rendering uses `content-visibility: auto`** rather than a JavaScript virtual scroller. Folders with tens of thousands of entries stay smooth without the complexity (and scroll-anchoring bugs) of windowing.
- **Conflicts are resolved before the job starts.** Destination names are chosen up front, so progress totals are accurate — but the byte estimate is computed *inside* the job goroutine, because walking a large tree on the call path used to freeze the window for seconds before anything appeared to happen.
- **Everything is cancellable.** A `context.Context` is threaded from the job manager down into the copy loop.
- **Concurrency is bounded.** Thumbnail decoding runs on a small worker pool sized to half the available cores, so scrolling quickly through large photos cannot saturate the machine.
- **Repeated work is cached with invalidation, not blindly.** `.desktop` parses are cached per file on mtime+size, the mount table is cached for two seconds, folder sizes are cached until the watcher reports a change in that subtree, and the uid/gid name lookups are cached behind a mutex.
- **Undo records only what it can faithfully reverse.** The record is committed *before* the completion event is emitted, so a listener that reacts to "done" always sees a consistent history.
- **No dependency on `gio`/`nautilus`.** Trash, mounts and launching are implemented directly against the freedesktop specifications, with `xdg-open`/`udisksctl`/`7z` used opportunistically when present.

---

## Performance

The parts that matter when a folder has 50,000 entries, or when the disk is
slow:

| Area | What is done | Why |
| --- | --- | --- |
| Listing | `readdirnames` + one `lstat` per entry, sorted in Go | Avoids a round trip per item and keeps sorting off the UI thread |
| Rendering | `content-visibility: auto` with an intrinsic row size | Off-screen rows cost no layout or paint, without a windowing library |
| Copy/move | Destination names resolved up front; byte totals estimated inside the worker | The window never blocks before a job starts |
| Thumbnails | Bounded decode pool (`cores / 2`, clamped to 2–8) + on-disk LRU cache | Fast scrolling cannot pin every core |
| Folder sizes | Only on request, or bounded to 400 folders; streamed per folder; cached until the watcher sees a change | No surprise full-tree scans |
| Search | Batched results (128 per event), early termination, `push` instead of `concat` | Linear cost as results arrive |
| `.desktop` files | Parsed once per file, cached on mtime+size (including negative results) | `/usr/share/applications` no longer re-reads hundreds of files per listing |
| Mount table | `/proc/mounts` cached for two seconds, invalidated on mount/eject | Disk-usage polling stops re-parsing it constantly |
| Name lookups | uid/gid caches behind a `RWMutex` | Correct *and* fast under concurrent tab loads |

Run the race detector to check the concurrency claims yourself:

```sh
go test -race -tags gtk3 ./internal/...
```

---

## Known limitations

- **Undo does not cover permanent deletion.** Once a file is unlinked there is
  nothing to restore, so Mura Files does not pretend otherwise.
- **Undo is per-session.** The history lives in memory and is cleared on quit.
- **7z and rar** need `p7zip-full` (or `7z`/`unar`) installed; scanning those
  archives without it reports a clear error.
- **Video and audio previews** depend on GStreamer codecs provided by the
  distribution.
- **Mount/unmount** is delegated to `udisksctl`/`umount`, so it inherits their
  policy (and polkit prompts) rather than implementing raw mount calls.

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

**The free-space number looks wrong** — it is read with the same arithmetic as `df`, so compare against `df /path`. Note that *available* is smaller than *free* on filesystems that reserve blocks for root (5% on ext4 by default); the status bar shows the available figure because that is what you can actually write.

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
