// Package search implements recursive filename and content search with
// streaming result batches so the UI can stay responsive on huge trees.
package search

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/R0N1n-dev/mura-files/internal/fsx"
	"github.com/R0N1n-dev/mura-files/internal/model"
	"github.com/R0N1n-dev/mura-files/internal/places"
)

// BatchSize is how many hits are flushed to the caller at once.
const BatchSize = 128

// maxContentFileSize caps how large a file may be before we skip grepping it.
const maxContentFileSize = 16 << 20

// Engine runs searches. A single Engine can run several searches concurrently.
type Engine struct {
	running map[string]context.CancelFunc
	seq     atomic.Uint64
}

// New creates a search engine.
func New() *Engine {
	return &Engine{running: map[string]context.CancelFunc{}}
}

// Start begins a search and streams batches through onBatch. It returns
// immediately; the caller receives the final batch with Done set to true.
func (e *Engine) Start(query model.SearchQuery, onBatch func(model.SearchResults)) string {
	id := "search-" + time.Now().Format("150405") + "-" + itoa(int(e.seq.Add(1)))
	ctx, cancel := context.WithCancel(context.Background())
	if query.TimeoutSeconds > 0 {
		var timeoutCancel context.CancelFunc
		ctx, timeoutCancel = context.WithTimeout(ctx, time.Duration(query.TimeoutSeconds)*time.Second)
		go func() {
			<-ctx.Done()
			timeoutCancel()
		}()
	}
	e.running[id] = cancel

	go func() {
		defer func() {
			delete(e.running, id)
			cancel()
		}()
		e.run(ctx, id, query, onBatch)
	}()
	return id
}

// Cancel stops a running search.
func (e *Engine) Cancel(id string) bool {
	if cancel, ok := e.running[id]; ok {
		cancel()
		return true
	}
	return false
}

// CancelAll stops every running search.
func (e *Engine) CancelAll() {
	for _, cancel := range e.running {
		cancel()
	}
}

// Running reports the identifiers of the active searches.
func (e *Engine) Running() []string {
	out := make([]string, 0, len(e.running))
	for id := range e.running {
		out = append(out, id)
	}
	return out
}

type matcher struct {
	query   model.SearchQuery
	regex   *regexp.Regexp
	pattern string
}

func newMatcher(query model.SearchQuery) (*matcher, error) {
	m := &matcher{query: query}
	pattern := query.Pattern
	switch query.Mode {
	case "regex":
		expr := pattern
		if !query.MatchCase {
			expr = "(?i)" + expr
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil, err
		}
		m.regex = re
	case "glob":
		expr := globToRegex(pattern)
		if !query.MatchCase {
			expr = "(?i)" + expr
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil, err
		}
		m.regex = re
	default:
		m.pattern = pattern
		if !query.MatchCase {
			m.pattern = strings.ToLower(pattern)
		}
	}
	return m, nil
}

func (m *matcher) matchesName(name string) bool {
	if m.regex != nil {
		return m.regex.MatchString(name)
	}
	if m.pattern == "" {
		return true
	}
	haystack := name
	if !m.query.MatchCase {
		haystack = strings.ToLower(name)
	}
	switch m.query.Mode {
	case "exact":
		return haystack == m.pattern
	default: // contains
		return strings.Contains(haystack, m.pattern)
	}
}

// globToRegex converts a shell glob into an anchored regular expression.
func globToRegex(glob string) string {
	var b strings.Builder
	b.WriteString("^")
	for _, r := range glob {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		case '.', '+', '(', ')', '|', '^', '$', '[', ']', '{', '}', '\\':
			b.WriteRune('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteString("$")
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// run walks the query roots and streams matching entries.
func (e *Engine) run(ctx context.Context, id string, query model.SearchQuery, onBatch func(model.SearchResults)) {
	start := time.Now()
	m, err := newMatcher(query)
	if err != nil {
		onBatch(model.SearchResults{ID: id, Done: true, Error: err.Error(),
			Elapsed: time.Since(start).Milliseconds()})
		return
	}
	if !query.IncludeFiles && !query.IncludeDirs {
		query.IncludeFiles = true
	}
	maxResults := query.MaxResults
	if maxResults <= 0 {
		maxResults = 20000
	}

	roots := query.Roots
	if len(roots) == 0 {
		roots = []string{places.Home()}
	}

	var (
		batch   = model.SearchResults{ID: id}
		scanned int
		found   int
		aborted string
	)

	flush := func(final bool) {
		batch.Scanned = scanned
		batch.Found = found
		batch.Done = final
		batch.Elapsed = time.Since(start).Milliseconds()
		if final {
			batch.Error = aborted
		}
		onBatch(batch)
		batch = model.SearchResults{ID: id}
	}

outer:
	for _, root := range roots {
		if ctx.Err() != nil {
			break
		}
		root = fsx.Abs(root)
		rootDepth := strings.Count(strings.TrimSuffix(root, "/"), "/")

		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			select {
			case <-ctx.Done():
				return filepath.SkipAll
			default:
			}
			if path == root {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") && !query.IncludeHidden {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if query.MaxDepth > 0 && d.IsDir() {
				depth := strings.Count(strings.TrimSuffix(path, "/"), "/") - rootDepth
				if depth >= query.MaxDepth {
					// Still test the directory itself, then stop descending.
					if m.accepts(path, d) {
						scanned++
						found++
						batch.Entries = append(batch.Entries, buildEntry(path, d))
					}
					return filepath.SkipDir
				}
			}
			info, ierr := d.Info()
			if ierr != nil {
				return nil
			}
			scanned++
			if !query.FollowLinks && info.Mode()&os.ModeSymlink != 0 && d.IsDir() {
				return filepath.SkipDir
			}
			if !m.accepts(path, d) {
				return nil
			}
			found++
			batch.Entries = append(batch.Entries, buildEntry(path, d))
			if len(batch.Entries) >= BatchSize {
				flush(false)
			}
			if found >= maxResults {
				aborted = "Result limit reached"
				return filepath.SkipAll
			}
			return nil
		})
		if walkErr != nil && walkErr != filepath.SkipAll {
			aborted = walkErr.Error()
		}
		if found >= maxResults || ctx.Err() != nil {
			break outer
		}
	}

	if ctx.Err() != nil && aborted == "" {
		aborted = "Search cancelled"
	}
	if len(batch.Entries) > 0 || true {
		flush(true)
	}
}
