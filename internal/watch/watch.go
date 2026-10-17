// Package watch provides a debounced directory watcher built on inotify. It
// watches a single directory at a time (the one the user is browsing) which
// keeps the inotify watch count low even on systems with modest limits.
package watch

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Event describes a batch of changes in a watched directory.
type Event struct {
	Dir     string   `json:"dir"`
	Created []string `json:"created"`
	Removed []string `json:"removed"`
	Changed []string `json:"changed"`
	Renamed []string `json:"renamed"`
	// Any is true when at least one change of any kind was observed.
	Any bool  `json:"any"`
	At  int64 `json:"at"`
}

// Watcher watches a single directory and invokes onChange after a short
// debounce window. Calls to Watch replace the previous directory.
type Watcher struct {
	mu       sync.Mutex
	inner    *fsnotify.Watcher
	current  string
	debounce time.Duration
	pending  map[string]struct{}
	creates  map[string]struct{}
	removes  map[string]struct{}
	renames  map[string]struct{}
	timer    *time.Timer
	onChange func(Event)
	closed   bool
}

// New creates a watcher. onChange is invoked from a background goroutine.
func New(debounce time.Duration, onChange func(Event)) (*Watcher, error) {
	inner, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if debounce <= 0 {
		debounce = 300 * time.Millisecond
	}
	w := &Watcher{
		inner:    inner,
		debounce: debounce,
		pending:  map[string]struct{}{},
		creates:  map[string]struct{}{},
		removes:  map[string]struct{}{},
		renames:  map[string]struct{}{},
		onChange: onChange,
	}
	go w.loop()
	return w, nil
}

// Watch switches the watcher to a new directory.
func (w *Watcher) Watch(dir string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return os.ErrClosed
	}
	if w.current == dir {
		return nil
	}
	if w.current != "" {
		_ = w.inner.Remove(w.current)
	}
	if err := w.inner.Add(dir); err != nil {
		return err
	}
	w.current = dir
	return nil
}

// Unwatch stops watching the current directory.
func (w *Watcher) Unwatch() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.current != "" {
		_ = w.inner.Remove(w.current)
		w.current = ""
	}
}

// Current returns the directory currently being watched.
func (w *Watcher) Current() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.current
}

// Close releases the inotify handle.
func (w *Watcher) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	if w.timer != nil {
		w.timer.Stop()
	}
	w.mu.Unlock()
	return w.inner.Close()
}

// loop consumes inotify events and coalesces them into debounced batches.
func (w *Watcher) loop() {
	for {
		select {
		case ev, ok := <-w.inner.Events:
			if !ok {
				return
			}
			w.record(ev)
		case _, ok := <-w.inner.Errors:
			if !ok {
				return
			}
		}
	}
}

func (w *Watcher) record(ev fsnotify.Event) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	name := filepath.Base(ev.Name)
	w.pending[name] = struct{}{}

	switch {
	case ev.Op&fsnotify.Create != 0:
		w.creates[name] = struct{}{}
	case ev.Op&fsnotify.Remove != 0:
		w.removes[name] = struct{}{}
	case ev.Op&fsnotify.Rename != 0:
		w.renames[name] = struct{}{}
	}

	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(w.debounce, w.flush)
}

func (w *Watcher) flush() {
	w.mu.Lock()
	if w.closed || len(w.pending) == 0 {
		w.mu.Unlock()
		return
	}
	event := Event{
		Dir:     w.current,
		Any:     true,
		At:      time.Now().UnixMilli(),
		Created: keys(w.creates),
		Removed: keys(w.removes),
		Renamed: keys(w.renames),
	}
	for name := range w.pending {
		if _, ok := w.creates[name]; ok {
			continue
		}
		if _, ok := w.removes[name]; ok {
			continue
		}
		if _, ok := w.renames[name]; ok {
			continue
		}
		event.Changed = append(event.Changed, name)
	}
	w.pending = map[string]struct{}{}
	w.creates = map[string]struct{}{}
	w.removes = map[string]struct{}{}
	w.renames = map[string]struct{}{}
	onChange := w.onChange
	w.mu.Unlock()

	if onChange != nil {
		onChange(event)
	}
}

func keys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

// Supported reports whether native watching is available on this system.
func Supported() bool {
	_, err := os.Stat("/proc/sys/fs/inotify/max_user_watches")
	return err == nil
}
