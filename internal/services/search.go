package services

import (
	"github.com/R0N1n-dev/mura-files/internal/model"
	"github.com/R0N1n-dev/mura-files/internal/watch"
)

// SearchService streams recursive search results to the frontend.
type SearchService struct {
	rt *Runtime
}

// NewSearchService creates the service.
func NewSearchService(rt *Runtime) *SearchService { return &SearchService{rt: rt} }

// ServiceName identifies the service in logs and bindings.
func (s *SearchService) ServiceName() string { return "Search" }

// Start begins a search and returns its identifier. Results arrive on the
// "search:results" event as incremental batches.
func (s *SearchService) Start(query model.SearchQuery) string {
	return s.rt.Search.Start(query, func(batch model.SearchResults) {
		s.rt.Emit(EventSearchResults, batch)
	})
}

// Cancel stops a running search.
func (s *SearchService) Cancel(id string) bool { return s.rt.Search.Cancel(id) }

// CancelAll stops every running search.
func (s *SearchService) CancelAll() { s.rt.Search.CancelAll() }

// Running lists the identifiers of the active searches.
func (s *SearchService) Running() []string { return s.rt.Search.Running() }

// WatchService manages the live directory watcher.
type WatchService struct {
	rt *Runtime
}

// NewWatchService creates the service.
func NewWatchService(rt *Runtime) *WatchService { return &WatchService{rt: rt} }

// ServiceName identifies the service in logs and bindings.
func (s *WatchService) ServiceName() string { return "Watch" }

// Watch starts watching a directory, replacing any previous target.
func (s *WatchService) Watch(path string) error {
	s.rt.mu.RLock()
	w := s.rt.Watcher
	s.rt.mu.RUnlock()
	if w == nil {
		return nil
	}
	return w.Watch(path)
}

// Unwatch stops watching.
func (s *WatchService) Unwatch() {
	s.rt.mu.RLock()
	w := s.rt.Watcher
	s.rt.mu.RUnlock()
	if w != nil {
		w.Unwatch()
	}
}

// Current reports the directory currently being watched.
func (s *WatchService) Current() string {
	s.rt.mu.RLock()
	w := s.rt.Watcher
	s.rt.mu.RUnlock()
	if w == nil {
		return ""
	}
	return w.Current()
}

// Enabled reports whether live updates are switched on and available.
func (s *WatchService) Enabled() bool {
	s.rt.mu.RLock()
	defer s.rt.mu.RUnlock()
	return s.rt.watchEnabled && s.rt.Watcher != nil
}

// SetEnabled turns live updates on or off.
func (s *WatchService) SetEnabled(enabled bool) { s.rt.SetWatchEnabled(enabled) }

// Supported reports whether inotify is available on this system.
func (s *WatchService) Supported() bool { return watch.Supported() }
