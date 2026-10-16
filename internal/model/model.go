// Package model holds the data transfer objects shared between the Go backend
// and the Vue frontend. Every exported field carries an explicit json tag so
// that the generated JavaScript bindings stay stable and idiomatic.
package model

// ---------------------------------------------------------------------------
// Filesystem entries
// ---------------------------------------------------------------------------

// Entry describes a single file, directory or symbolic link.
type Entry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Dir  string `json:"dir"`
	Ext  string `json:"ext"`

	IsDir      bool `json:"isDir"`
	IsSymlink  bool `json:"isSymlink"`
	IsHidden   bool `json:"isHidden"`
	IsExec     bool `json:"isExec"`
	IsReadable bool `json:"isReadable"`
	IsWritable bool `json:"isWritable"`
	LinkBroken bool `json:"linkBroken"`
	Target     string `json:"target"`

	Size     int64  `json:"size"`
	Mode     string `json:"mode"`
	Perm     string `json:"perm"`
	ModeBits uint32 `json:"modeBits"`

	ModTime    int64 `json:"modTime"`
	AccessTime int64 `json:"accessTime"`
	ChangeTime int64 `json:"changeTime"`

	Owner string `json:"owner"`
	Group string `json:"group"`
	UID   uint32 `json:"uid"`
	GID   uint32 `json:"gid"`
	Nlink uint32 `json:"nlink"`

	Mime string `json:"mime"`
	Kind string `json:"kind"`
	Icon string `json:"icon"`
	Thumb bool  `json:"thumb"`
	AppIcon     string `json:"appIcon"`
	DesktopName string `json:"desktopName"`
	ItemCount   int64  `json:"itemCount"`
}

// DirListing is the result of listing a directory.
type DirListing struct {
	Path     string  `json:"path"`
	Name     string  `json:"name"`
	Parent   string  `json:"parent"`
	Entries  []Entry `json:"entries"`
	Total    int     `json:"total"`
	Filtered int     `json:"filtered"`
	Hidden   int     `json:"hidden"`

	Readable  bool   `json:"readable"`
	Writable  bool   `json:"writable"`
	Error     string `json:"error"`
	Truncated bool   `json:"truncated"`
}

// ListOptions controls how a directory is listed.
type ListOptions struct {
	ShowHidden  bool   `json:"showHidden"`
	Filter      string `json:"filter"`
	SortBy      string `json:"sortBy"`
	SortDesc    bool   `json:"sortDesc"`
	DirsFirst   bool   `json:"dirsFirst"`
	GroupByKind bool   `json:"groupByKind"`
	MaxEntries  int    `json:"maxEntries"`
	Recursive   bool   `json:"recursive"`
	Depth       int    `json:"depth"`
}

// TextPreview is a chunk of a text file together with lightweight metadata.
type TextPreview struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Size      int64  `json:"size"`
	Truncated bool   `json:"truncated"`
	Binary    bool   `json:"binary"`
	Lines     int    `json:"lines"`
	Encoding  string `json:"encoding"`
}

// Usage reports disk usage for a path.
type Usage struct {
	Path    string  `json:"path"`
	Total   uint64  `json:"total"`
	Free    uint64  `json:"free"`
	Used    uint64  `json:"used"`
	Percent float64 `json:"percent"`
	FsType  string  `json:"fsType"`
}

// CountResult aggregates statistics for a set of paths.
type CountResult struct {
	Files     int64 `json:"files"`
	Dirs      int64 `json:"dirs"`
	Symlinks  int64 `json:"symlinks"`
	Bytes     int64 `json:"bytes"`
	Truncated bool  `json:"truncated"`
}

// Completion is a single path-completion candidate.
type Completion struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	IsDir  bool   `json:"isDir"`
	IsLink bool   `json:"isLink"`
}

// ---------------------------------------------------------------------------
// Places, drives and bookmarks
// ---------------------------------------------------------------------------

// Place is a well-known location shown in the sidebar.
type Place struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Kind string `json:"kind"`
	Icon string `json:"icon"`
	// Count is the number of direct children (filled in lazily).
	Count int `json:"count"`
	// Present is false when the directory does not exist on disk.
	Present bool `json:"present"`
}

// Drive describes a mounted filesystem.
type Drive struct {
	Name      string  `json:"name"`
	Path      string  `json:"path"`
	Device    string  `json:"device"`
	FsType    string  `json:"fsType"`
	Label     string  `json:"label"`
	Total     uint64  `json:"total"`
	Free      uint64  `json:"free"`
	Used      uint64  `json:"used"`
	Percent   float64 `json:"percent"`
	Removable bool    `json:"removable"`
	ReadOnly  bool    `json:"readOnly"`
	Network   bool    `json:"network"`
	System    bool    `json:"system"`
	Mounted   bool    `json:"mounted"`
}

// Bookmark is a user-defined sidebar entry.
type Bookmark struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

// TrashEntry is an item inside the freedesktop trash.
type TrashEntry struct {
	// Info is the absolute path of the .trashinfo file (the stable handle).
	Info string `json:"info"`
	// Path is the original absolute path the item was deleted from.
	Path string `json:"path"`
	Name string `json:"name"`
	// Trashed is the absolute path of the payload inside the trash directory.
	Trashed string `json:"trashed"`
	// DeletedAt is Unix milliseconds.
	DeletedAt int64 `json:"deletedAt"`
	IsDir     bool  `json:"isDir"`
	Size      int64 `json:"size"`
}

// ---------------------------------------------------------------------------
// Operations (copy / move / delete / archive)
// ---------------------------------------------------------------------------

// OpProgress is streamed to the frontend while a long running operation runs.
type OpProgress struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	// Status is one of: queued, running, paused, done, cancelled, error.
	Status string `json:"status"`
	Title  string `json:"title"`
	// Current is the path currently being processed.
	Current     string `json:"current"`
	CurrentItem int    `json:"currentItem"`
	TotalItems  int    `json:"totalItems"`
	BytesDone   int64  `json:"bytesDone"`
	BytesTotal  int64  `json:"bytesTotal"`

	Percent float64 `json:"percent"`
	// Speed is bytes per second.
	Speed float64 `json:"speed"`
	// ETASeconds is the estimated remaining time in seconds (-1 when unknown).
	ETASeconds int    `json:"etaSeconds"`
	Message    string `json:"message"`
	Errors     []string `json:"errors"`

	StartedAt int64 `json:"startedAt"`
	EndedAt   int64 `json:"endedAt"`
}

// ConflictResolution describes how an existing destination is handled.
type ConflictResolution string

const (
	ConflictAsk      ConflictResolution = "ask"
	ConflictReplace  ConflictResolution = "replace"
	ConflictSkip     ConflictResolution = "skip"
	ConflictRename   ConflictResolution = "rename"
	ConflictKeepBoth ConflictResolution = "keepBoth"
)

// TransferOptions configures copy and move operations.
type TransferOptions struct {
	// Conflict decides what happens when a destination already exists.
	Conflict ConflictResolution `json:"conflict"`
	// Overwrite replaces existing files (implies ConflictReplace).
	Overwrite bool `json:"overwrite"`
	// PreserveAttrs keeps mode, times and ownership where permitted.
	PreserveAttrs bool `json:"preserveAttrs"`
	// FollowSymlinks dereferences links instead of recreating them.
	FollowSymlinks bool `json:"followSymlinks"`
	// RemoveSource deletes the source after a successful transfer (move).
	RemoveSource bool `json:"removeSource"`
	// SkipErrors continues after a failed item.
	SkipErrors bool `json:"skipErrors"`
}

// DeleteOptions configures delete operations.
type DeleteOptions struct {
	// Permanent bypasses the trash and unlinks immediately.
	Permanent bool `json:"permanent"`
	// SkipErrors continues after a failed item.
	SkipErrors bool `json:"skipErrors"`
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

// SearchQuery describes a recursive search.
type SearchQuery struct {
	Roots  []string `json:"roots"`
	Pattern string  `json:"pattern"`
	// Mode is one of: glob, regex, exact, contains.
	Mode      string `json:"mode"`
	MatchCase bool   `json:"matchCase"`

	IncludeFiles  bool `json:"includeFiles"`
	IncludeDirs   bool `json:"includeDirs"`
	IncludeHidden bool `json:"includeHidden"`
	FollowLinks   bool `json:"followLinks"`

	KindFilter string   `json:"kindFilter"`
	ExtFilter  []string `json:"extFilter"`

	MinSize int64 `json:"minSize"`
	MaxSize int64 `json:"maxSize"`
	// ModifiedAfter and ModifiedBefore are Unix milliseconds (0 disables).
	ModifiedAfter  int64 `json:"modifiedAfter"`
	ModifiedBefore int64 `json:"modifiedBefore"`

	// Content, when set, additionally greps file contents.
	Content bool `json:"content"`

	MaxResults int `json:"maxResults"`
	MaxDepth   int `json:"maxDepth"`
	// TimeoutSeconds aborts the search after the given time (0 = none).
	TimeoutSeconds int `json:"timeoutSeconds"`
}

// SearchResults is a streamed batch of search hits.
type SearchResults struct {
	ID      string  `json:"id"`
	Entries []Entry `json:"entries"`
	// Scanned is the number of filesystem entries inspected so far.
	Scanned int `json:"scanned"`
	Found   int `json:"found"`
	Done    bool `json:"done"`
	// Elapsed is the search duration in milliseconds.
	Elapsed int64  `json:"elapsed"`
	Error   string `json:"error"`
}

// ---------------------------------------------------------------------------
// Archives
// ---------------------------------------------------------------------------

// ArchiveEntry is a member of an archive.
type ArchiveEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Packed  int64  `json:"packed"`
	ModTime int64  `json:"modTime"`
	IsDir   bool   `json:"isDir"`
	Mode    string `json:"mode"`
}

// ArchiveInfo describes an archive file.
type ArchiveInfo struct {
	Path         string         `json:"path"`
	Format       string         `json:"format"`
	Entries      []ArchiveEntry `json:"entries"`
	Total        int            `json:"total"`
	Uncompressed int64          `json:"uncompressed"`
	Compressed   int64          `json:"compressed"`
	Truncated    bool           `json:"truncated"`
}

// ---------------------------------------------------------------------------
// System information and settings
// ---------------------------------------------------------------------------

// OSInfo describes the host operating system.
type OSInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Kernel   string `json:"kernel"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
	User     string `json:"user"`
	Home     string `json:"home"`
	Shell    string `json:"shell"`
	Desktop  string `json:"desktop"`
	Session  string `json:"session"`
}

// Tool reports whether an optional external helper is available.
type Tool struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Available bool   `json:"available"`
}

// SystemInfo is the aggregate payload returned to the frontend at boot.
type SystemInfo struct {
	OS       OSInfo  `json:"os"`
	Version  string  `json:"version"`
	DarkMode bool    `json:"darkMode"`
	Accent   string  `json:"accent"`
	Terminal string  `json:"terminal"`
	Tools    []Tool  `json:"tools"`
	Paths    []Place `json:"paths"`
}

// Settings is the persisted application configuration.
type Settings struct {
	Theme          string `json:"theme"`
	AccentColour   string `json:"accentColour"`
	ViewMode       string `json:"viewMode"`
	GridSize       int    `json:"gridSize"`
	SortBy         string `json:"sortBy"`
	SortDesc       bool   `json:"sortDesc"`
	DirsFirst      bool   `json:"dirsFirst"`
	GroupByKind    bool   `json:"groupByKind"`
	ShowHidden     bool   `json:"showHidden"`
	ShowExtensions bool   `json:"showExtensions"`
	ShowThumbnails bool   `json:"showThumbnails"`
	ShowPreview    bool   `json:"showPreview"`
	ShowSidebar    bool   `json:"showSidebar"`
	ShowStatusBar  bool   `json:"showStatusBar"`

	ConfirmDelete   bool `json:"confirmDelete"`
	ConfirmTrash    bool `json:"confirmTrash"`
	UseTrash        bool `json:"useTrash"`
	ConfirmMove     bool `json:"confirmMove"`
	SingleClickOpen bool `json:"singleClickOpen"`

	RestoreTabs    bool     `json:"restoreTabs"`
	OpenTabs       []string `json:"openTabs"`
	ActiveTab      int      `json:"activeTab"`

	DateFormat string `json:"dateFormat"`
	SizeFormat string `json:"sizeFormat"`

	TerminalCommand string `json:"terminalCommand"`
	EditorCommand   string `json:"editorCommand"`
	ThumbnailSize   int    `json:"thumbnailSize"`
	WatchEnabled    bool   `json:"watchEnabled"`

	SidebarWidth  int      `json:"sidebarWidth"`
	PreviewWidth  int      `json:"previewWidth"`
	HiddenFolders []string `json:"hiddenFolders"`

	ConfirmCloseTabs bool `json:"confirmCloseTabs"`
	FollowLinks      bool `json:"followLinks"`
	AutoExpandTree   bool `json:"autoExpandTree"`
	ShowFolderSize   bool `json:"showFolderSize"`
}

// WindowState mirrors the native window state so the frontend can render the
// correct maximise / restore affordance.
type WindowState struct {
	Maximised  bool `json:"maximised"`
	Fullscreen bool `json:"fullscreen"`
	Focused    bool `json:"focused"`
	Width      int  `json:"width"`
	Height     int  `json:"height"`
}

// AppInfo is a launchable desktop application.
type AppInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Icon string `json:"icon"`
}

// OpenResult reports the outcome of launching something.
type OpenResult struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Message string `json:"message"`
}

// GitStatus summarises the state of a repository shown in the status bar.
type GitStatus struct {
	IsRepo bool   `json:"isRepo"`
	Root   string `json:"root"`
	Branch string `json:"branch"`
	Dirty  int    `json:"dirty"`
}

// ThemeInfo carries the resolved desktop colour scheme.
type ThemeInfo struct {
	Dark   bool   `json:"dark"`
	Accent string `json:"accent"`
	Source string `json:"source"`
}
