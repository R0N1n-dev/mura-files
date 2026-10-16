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
