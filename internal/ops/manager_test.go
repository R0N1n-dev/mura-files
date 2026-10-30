package ops

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/R0N1n-dev/mura-files/internal/model"
)

// collector records every progress update it receives.
func collector() (*[]model.OpProgress, *sync.Mutex, Emitter) {
	var mu sync.Mutex
	var updates []model.OpProgress
	return &updates, &mu, func(progress model.OpProgress) {
		mu.Lock()
		defer mu.Unlock()
		updates = append(updates, progress)
	}
}

func TestStartEmitsProgressAndCompletion(t *testing.T) {
	updates, mu, emit := collector()
	manager := NewManager(emit)

	id := manager.Start("copy", "Copying things", 4, func(ctx context.Context, r *Reporter) error {
		r.SetTotal(4, 400)
		for i := 0; i < 4; i++ {
			r.Item("/tmp/file")
			r.Bytes(100)
		}
		return nil
	})
	if id == "" {
		t.Fatal("Start should return an identifier")
	}

	waitFor(t, func() bool {
		progress, ok := manager.Get(id)
		return ok && progress.Status == StatusDone
	})

	progress, _ := manager.Get(id)
	if progress.Status != StatusDone {
		t.Fatalf("status = %q", progress.Status)
	}
	if progress.CurrentItem != 4 {
		t.Errorf("CurrentItem = %d", progress.CurrentItem)
	}
	if progress.BytesDone != 400 {
		t.Errorf("BytesDone = %d", progress.BytesDone)
	}
	if progress.Percent != 100 {
		t.Errorf("Percent = %f", progress.Percent)
	}
	if progress.EndedAt == 0 {
		t.Error("EndedAt should be set on completion")
	}

	mu.Lock()
	count := len(*updates)
	mu.Unlock()
	if count < 2 {
		t.Errorf("expected at least a start and a finish update, got %d", count)
	}
}

func TestFailureIsReported(t *testing.T) {
	manager := NewManager(func(model.OpProgress) {})
	id := manager.Start("copy", "boom", 1, func(ctx context.Context, r *Reporter) error {
		return errors.New("disk on fire")
	})

	waitFor(t, func() bool {
		progress, ok := manager.Get(id)
		return ok && progress.Status == StatusError
	})
	progress, _ := manager.Get(id)
	if progress.Message != "disk on fire" {
		t.Errorf("message = %q", progress.Message)
	}
	if len(progress.Errors) != 1 {
		t.Errorf("errors = %v", progress.Errors)
	}
}

func TestNonFatalErrorsAccumulate(t *testing.T) {
	manager := NewManager(func(model.OpProgress) {})
	id := manager.Start("copy", "partial", 3, func(ctx context.Context, r *Reporter) error {
		r.Error(errors.New("first failure"))
		r.Error(errors.New("second failure"))
		r.Item("done")
		return nil
	})

	waitFor(t, func() bool {
		progress, ok := manager.Get(id)
		return ok && progress.Status == StatusDone
	})
	progress, _ := manager.Get(id)
	if len(progress.Errors) != 2 {
		t.Fatalf("expected 2 recorded errors, got %v", progress.Errors)
	}
}

func TestCancelStopsWorker(t *testing.T) {
	manager := NewManager(func(model.OpProgress) {})
	started := make(chan struct{})
	id := manager.Start("copy", "slow", 0, func(ctx context.Context, r *Reporter) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	<-started
	if !manager.Cancel(id) {
		t.Fatal("Cancel should find the running job")
	}
	waitFor(t, func() bool {
		progress, ok := manager.Get(id)
		return ok && progress.Status == StatusCancelled
	})
}

func TestActiveAndClearFinished(t *testing.T) {
	manager := NewManager(func(model.OpProgress) {})
	blocker := make(chan struct{})
	running := manager.Start("copy", "running", 0, func(ctx context.Context, r *Reporter) error {
		<-blocker
		return nil
	})
	done := manager.Start("copy", "quick", 0, func(ctx context.Context, r *Reporter) error { return nil })

	waitFor(t, func() bool {
		progress, ok := manager.Get(done)
		return ok && progress.Status == StatusDone
	})

	if len(manager.Active()) != 2 {
		t.Fatalf("Active should list both jobs, got %d", len(manager.Active()))
	}
	manager.ClearFinished()
	active := manager.Active()
	if len(active) != 1 || active[0].ID != running {
		t.Fatalf("ClearFinished should keep only the running job, got %+v", active)
	}

	close(blocker)
	manager.CancelAllAndWait(2 * time.Second)
	if len(manager.Active()) != 1 {
		t.Errorf("finished jobs stay listed until cleared, got %d", len(manager.Active()))
	}
}

func TestSpeedAndETA(t *testing.T) {
	manager := NewManager(func(model.OpProgress) {})
	reporterReady := make(chan *Reporter, 1)
	release := make(chan struct{})
	id := manager.Start("copy", "throughput", 0, func(ctx context.Context, r *Reporter) error {
		r.SetTotal(0, 1000)
		reporterReady <- r
		<-release
		return nil
	})
	reporter := <-reporterReady

	// The speed estimate only updates after a quarter of a second of traffic.
	time.Sleep(300 * time.Millisecond)
	reporter.Bytes(500)
	time.Sleep(300 * time.Millisecond)
	reporter.Bytes(500)

	progress, _ := manager.Get(id)
	if progress.Percent < 99 {
		t.Errorf("Percent = %f, expected the byte progress to dominate", progress.Percent)
	}
	if progress.Speed <= 0 {
		t.Errorf("Speed = %f, expected a positive estimate", progress.Speed)
	}
	close(release)
}

func TestReporterContext(t *testing.T) {
	manager := NewManager(func(model.OpProgress) {})
	seen := make(chan context.Context, 1)
	manager.Start("copy", "ctx", 0, func(ctx context.Context, r *Reporter) error {
		seen <- r.Context()
		return nil
	})
	ctx := <-seen
	if ctx == nil {
		t.Fatal("the reporter should expose the job context")
	}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before the deadline")
}

func TestSuccessfulJobRecordsUndo(t *testing.T) {
	var mu sync.Mutex
	var recorded []model.UndoEntry
	manager := NewManager(func(model.OpProgress) {})
	manager.SetUndoRecorder(func(entry model.UndoEntry) {
		mu.Lock()
		recorded = append(recorded, entry)
		mu.Unlock()
	})

	id := manager.Start("move", "Move things", 2, func(ctx context.Context, r *Reporter) error {
		r.RecordUndo("move", "Move things", model.UndoItem{From: "/a/one", To: "/b/one"})
		r.RecordUndo("move", "Move things", model.UndoItem{From: "/a/two", To: "/b/two"})
		r.Item("/a/one")
		r.Item("/a/two")
		return nil
	})

	waitFor(t, func() bool {
		progress, ok := manager.Get(id)
		return ok && progress.Status == StatusDone
	})
	mu.Lock()
	defer mu.Unlock()
	if len(recorded) != 1 {
		t.Fatalf("expected exactly one undo entry, got %d", len(recorded))
	}
	if recorded[0].Kind != "move" {
		t.Errorf("kind = %q", recorded[0].Kind)
	}
	if len(recorded[0].Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(recorded[0].Items))
	}
}

func TestFailedJobRecordsNoUndo(t *testing.T) {
	var mu sync.Mutex
	var recorded []model.UndoEntry
	manager := NewManager(func(model.OpProgress) {})
	manager.SetUndoRecorder(func(entry model.UndoEntry) {
		mu.Lock()
		recorded = append(recorded, entry)
		mu.Unlock()
	})

	id := manager.Start("move", "Move things", 1, func(ctx context.Context, r *Reporter) error {
		r.RecordUndo("move", "Move things", model.UndoItem{From: "/a/one", To: "/b/one"})
		return errors.New("permission denied")
	})

	waitFor(t, func() bool {
		progress, ok := manager.Get(id)
		return ok && progress.Status == StatusError
	})
	mu.Lock()
	defer mu.Unlock()
	if len(recorded) != 0 {
		t.Errorf("a failed job must not leave an undo entry behind, got %+v", recorded)
	}
}

func TestCancelledJobRecordsNoUndo(t *testing.T) {
	var mu sync.Mutex
	var recorded []model.UndoEntry
	manager := NewManager(func(model.OpProgress) {})
	manager.SetUndoRecorder(func(entry model.UndoEntry) {
		mu.Lock()
		recorded = append(recorded, entry)
		mu.Unlock()
	})

	started := make(chan struct{})
	id := manager.Start("move", "Move things", 1, func(ctx context.Context, r *Reporter) error {
		r.RecordUndo("move", "Move things", model.UndoItem{From: "/a", To: "/b"})
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	<-started
	manager.Cancel(id)

	waitFor(t, func() bool {
		progress, ok := manager.Get(id)
		return ok && progress.Status == StatusCancelled
	})
	mu.Lock()
	defer mu.Unlock()
	if len(recorded) != 0 {
		t.Errorf("a cancelled job must not leave an undo entry behind, got %+v", recorded)
	}
}

func TestUndoCount(t *testing.T) {
	manager := NewManager(func(model.OpProgress) {})
	id := manager.Start("create", "Create", 1, func(ctx context.Context, r *Reporter) error {
		r.RecordUndo("create", "Create", model.UndoItem{To: "/tmp/x"})
		r.RecordUndo("create", "Create", model.UndoItem{To: "/tmp/y"})
		return nil
	})
	waitFor(t, func() bool {
		progress, ok := manager.Get(id)
		return ok && progress.Status == StatusDone
	})
	if got := manager.GetUndoCountForTest(id); got != 2 {
		t.Errorf("UndoCount = %d, want 2", got)
	}
}

// GetUndoCountForTest exposes the recorded undo count for assertions.
func (m *Manager) GetUndoCountForTest(id string) int {
	m.mu.RLock()
	op, ok := m.ops[id]
	m.mu.RUnlock()
	if !ok {
		return -1
	}
	op.mu.Lock()
	defer op.mu.Unlock()
	return len(op.undo)
}
