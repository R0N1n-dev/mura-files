// Package undo records reversible filesystem changes so the user can take back
// a mistake without leaving the application.
//
// Only operations with a faithful inverse are recorded: trashing, moving,
// renaming and creating (including duplicates and extracted folders). Permanent
// deletion is deliberately absent - it cannot be undone, and pretending
// otherwise would be worse than offering nothing.
package undo

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/R0N1n-dev/mura-files/internal/fsx"
	"github.com/R0N1n-dev/mura-files/internal/model"
	"github.com/R0N1n-dev/mura-files/internal/places"
)

// Kinds of reversible operation.
const (
	KindTrash     = "trash"
	KindMove      = "move"
	KindRename    = "rename"
	KindCreate    = "create"
	KindDuplicate = "duplicate"
)

// defaultDepth is how many operations are remembered.
const defaultDepth = 40

// Stack is a bounded, concurrency safe undo history.
type Stack struct {
	mu      sync.Mutex
	entries []model.UndoEntry
	depth   int
}

// New creates an empty stack.
func New() *Stack {
	return &Stack{depth: defaultDepth}
}

// Push records a new entry, discarding the oldest when the stack is full.
func (s *Stack) Push(entry model.UndoEntry) {
	if len(entry.Items) == 0 {
		return
	}
	if entry.At == 0 {
		entry.At = time.Now().UnixMilli()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entry)
	if len(s.entries) > s.depth {
		s.entries = s.entries[len(s.entries)-s.depth:]
	}
}

// Pop removes and returns the most recent entry.
func (s *Stack) Pop() (model.UndoEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.entries) == 0 {
		return model.UndoEntry{}, false
	}
	last := s.entries[len(s.entries)-1]
	s.entries = s.entries[:len(s.entries)-1]
	return last, true
}

// Peek returns the most recent entry without removing it.
func (s *Stack) Peek() (model.UndoEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.entries) == 0 {
		return model.UndoEntry{}, false
	}
	return s.entries[len(s.entries)-1], true
}

// Len reports how many operations can be undone.
func (s *Stack) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

// Clear forgets the whole history.
func (s *Stack) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = nil
}

// Apply reverses one recorded entry. It is best effort: every item is
// attempted, and the first failure is reported while the rest still run.
func Apply(entry model.UndoEntry) (model.UndoResult, error) {
	result := model.UndoResult{Label: entry.Label}
	var firstErr error

	switch entry.Kind {
	case KindTrash:
		restored := 0
		for _, item := range entry.Items {
			if _, err := places.RestoreTrash(item.To); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			restored++
		}
		result.Count = restored
		result.Message = fmt.Sprintf("Restored %d item(s) from the Trash", restored)
		result.OK = restored > 0

	case KindMove, KindRename:
		reverted := 0
		for _, item := range entry.Items {
			if item.From == "" || item.To == "" {
				continue
			}
			if _, err := os.Lstat(item.To); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			// Make sure the destination is free; a file may have been created
			// in the meantime, in which case pick a unique name rather than
			// clobbering it.
			target := item.From
			if _, err := os.Lstat(target); err == nil {
				dir := filepath.Dir(target)
				target = filepath.Join(dir, fsx.UniqueName(dir, filepath.Base(target)))
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			if err := os.Rename(item.To, target); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			reverted++
		}
		verb := "Moved back"
		if entry.Kind == KindRename {
			verb = "Renamed back"
		}
		result.Count = reverted
		result.Message = fmt.Sprintf("%s %d item(s)", verb, reverted)
		result.OK = reverted > 0

	case KindCreate, KindDuplicate:
		removed := 0
		for _, item := range entry.Items {
			if item.To == "" {
				continue
			}
			if _, err := os.Lstat(item.To); err != nil {
				continue
			}
			// Send it to the trash rather than unlinking: even an undo should
			// be recoverable if the user had second thoughts.
			if _, err := places.TrashPath(item.To); err != nil {
				if err := os.RemoveAll(item.To); err != nil {
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
			}
			removed++
		}
		result.Count = removed
		result.Message = fmt.Sprintf("Removed %d newly created item(s)", removed)
		result.OK = removed > 0

	default:
		return model.UndoResult{}, fmt.Errorf("operation %q cannot be undone", entry.Kind)
	}

	if firstErr != nil {
		result.OK = result.Count > 0
		return result, firstErr
	}
	return result, nil
}
