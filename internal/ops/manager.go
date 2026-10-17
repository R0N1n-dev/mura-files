// Package ops runs long lived filesystem jobs (copy, move, delete, archive)
// in the background while streaming progress to the UI and supporting
// cancellation.
package ops

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/R0N1n-dev/mura-files/internal/model"
)

// Status values reported to the frontend.
const (
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusDone      = "done"
	StatusCancelled = "cancelled"
	StatusError     = "error"
)

// Emitter receives progress updates. The service layer wires this to Wails
// events so the frontend can render a progress bar.
type Emitter func(model.OpProgress)

// Op is a single running job.
type Op struct {
	id     string
	kind   string
	cancel context.CancelFunc
	ctx    context.Context

	mu       sync.Mutex
	progress model.OpProgress
	emitter  Emitter
	// lastEmit throttles event traffic to roughly twelve updates per second.
	lastEmit time.Time
	finished bool
	lastSize int64
	lastTime time.Time
}

// Manager owns the set of running jobs.
type Manager struct {
	mu      sync.RWMutex
	ops     map[string]*Op
	order   []string
	seq     atomic.Uint64
	emitter Emitter
}

// NewManager creates an operation manager.
func NewManager(emitter Emitter) *Manager {
	return &Manager{ops: map[string]*Op{}, emitter: emitter}
}

// SetEmitter swaps the progress sink, used once the Wails app exists.
func (m *Manager) SetEmitter(emitter Emitter) {
	m.mu.Lock()
	m.emitter = emitter
	m.mu.Unlock()
}

// Worker is the body of a background job.
type Worker func(ctx context.Context, r *Reporter) error

// Start launches a job and returns its identifier.
func (m *Manager) Start(kind, title string, totalItems int, worker Worker) string {
	id := fmt.Sprintf("%s-%d", kind, m.seq.Add(1))
	ctx, cancel := context.WithCancel(context.Background())

	op := &Op{
		id:     id,
		kind:   kind,
		cancel: cancel,
		ctx:    ctx,
		progress: model.OpProgress{
			ID:         id,
			Kind:       kind,
			Status:     StatusRunning,
			Title:      title,
			TotalItems: totalItems,
			ETASeconds: -1,
			StartedAt:  time.Now().UnixMilli(),
			Errors:     []string{},
		},
		lastTime: time.Now(),
	}

	m.mu.Lock()
	op.emitter = m.emitter
	m.ops[id] = op
	m.order = append(m.order, id)
	m.mu.Unlock()

	op.emit(true)

	go func() {
		defer cancel()
		reporter := &Reporter{op: op}
		err := worker(ctx, reporter)
		op.finish(err)
	}()

	return id
}

// Cancel asks a job to stop.
func (m *Manager) Cancel(id string) bool {
	m.mu.RLock()
	op, ok := m.ops[id]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	op.cancel()
	return true
}

// CancelAll stops every running job.
func (m *Manager) CancelAll() {
	m.mu.RLock()
	list := make([]*Op, 0, len(m.ops))
	for _, op := range m.ops {
		list = append(list, op)
	}
	m.mu.RUnlock()
	for _, op := range list {
		op.cancel()
	}
}

// Active returns a snapshot of the known jobs, oldest first.
func (m *Manager) Active() []model.OpProgress {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]model.OpProgress, 0, len(m.order))
	for _, id := range m.order {
		if op, ok := m.ops[id]; ok {
			out = append(out, op.snapshot())
		}
	}
	return out
}

// Get returns a snapshot of one job.
func (m *Manager) Get(id string) (model.OpProgress, bool) {
	m.mu.RLock()
	op, ok := m.ops[id]
	m.mu.RUnlock()
	if !ok {
		return model.OpProgress{}, false
	}
	return op.snapshot(), true
}

// ClearFinished removes completed jobs from the list.
func (m *Manager) ClearFinished() {
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := m.order[:0]
	for _, id := range m.order {
		if op, ok := m.ops[id]; ok && !op.isFinished() {
			kept = append(kept, id)
			continue
		}
		delete(m.ops, id)
	}
	m.order = kept
}

// CancelAllAndWait cancels every job and waits for them to unwind.
func (m *Manager) CancelAllAndWait(timeout time.Duration) {
	m.CancelAll()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(m.Active()) == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (o *Op) isFinished() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.finished
}

func (o *Op) snapshot() model.OpProgress {
	o.mu.Lock()
	defer o.mu.Unlock()
	cp := o.progress
	cp.Errors = append([]string{}, o.progress.Errors...)
	return cp
}

func (o *Op) finish(err error) {
	o.mu.Lock()
	o.finished = true
	o.progress.EndedAt = time.Now().UnixMilli()
	switch {
	case err == nil:
		o.progress.Status = StatusDone
		o.progress.Percent = 100
	case o.ctx.Err() != nil:
		o.progress.Status = StatusCancelled
		o.progress.Message = "Cancelled"
	default:
		o.progress.Status = StatusError
		o.progress.Message = err.Error()
		o.progress.Errors = append(o.progress.Errors, err.Error())
	}
	emitter := o.emitter
	snapshot := o.progress
	snapshot.Errors = append([]string{}, o.progress.Errors...)
	o.mu.Unlock()

	if emitter != nil {
		emitter(snapshot)
	}
}

func (o *Op) emit(force bool) {
	o.mu.Lock()
	if !force && time.Since(o.lastEmit) < 80*time.Millisecond {
		o.mu.Unlock()
		return
	}
	o.lastEmit = time.Now()
	emitter := o.emitter
	snapshot := o.progress
	snapshot.Errors = append([]string{}, o.progress.Errors...)
	o.mu.Unlock()

	if emitter != nil {
		emitter(snapshot)
	}
}

// Reporter is handed to workers so they can publish progress.
type Reporter struct {
	op *Op
}

// Item records that one more item has been processed.
func (r *Reporter) Item(current string) {
	r.op.mu.Lock()
	r.op.progress.CurrentItem++
	r.op.progress.Current = current
	r.recomputeLocked()
	r.op.mu.Unlock()
	r.op.emit(false)
}

// SetCurrent updates the path being worked on without advancing the counter.
func (r *Reporter) SetCurrent(current string) {
	r.op.mu.Lock()
	r.op.progress.Current = current
	r.op.mu.Unlock()
	r.op.emit(false)
}

// SetTotal updates the discovered total number of items and bytes.
func (r *Reporter) SetTotal(items int, bytes int64) {
	r.op.mu.Lock()
	if items > 0 {
		r.op.progress.TotalItems = items
	}
	if bytes > 0 {
		r.op.progress.BytesTotal = bytes
	}
	r.recomputeLocked()
	r.op.mu.Unlock()
	r.op.emit(false)
}

// Bytes records transferred bytes, which also drives the speed estimate.
func (r *Reporter) Bytes(n int64) {
	now := time.Now()
	r.op.mu.Lock()
	r.op.progress.BytesDone += n
	elapsed := now.Sub(r.op.lastTime).Seconds()
	if elapsed > 0.25 {
		delta := r.op.progress.BytesDone - r.op.lastSize
		instant := float64(delta) / elapsed
		if r.op.progress.Speed == 0 {
			r.op.progress.Speed = instant
		} else {
			// Exponential moving average keeps the readout stable.
			r.op.progress.Speed = r.op.progress.Speed*0.7 + instant*0.3
		}
		r.op.lastSize = r.op.progress.BytesDone
		r.op.lastTime = now
	}
	r.recomputeLocked()
	r.op.mu.Unlock()
	r.op.emit(false)
}

// Message sets a free-form status message.
func (r *Reporter) Message(msg string) {
	r.op.mu.Lock()
	r.op.progress.Message = msg
	r.op.mu.Unlock()
	r.op.emit(false)
}

// Error records a non-fatal failure.
func (r *Reporter) Error(err error) {
	if err == nil {
		return
	}
	r.op.mu.Lock()
	r.op.progress.Errors = append(r.op.progress.Errors, err.Error())
	r.op.mu.Unlock()
	r.op.emit(true)
}

// Context exposes the job context so workers can honour cancellation.
func (r *Reporter) Context() context.Context { return r.op.ctx }

func (r *Reporter) recomputeLocked() {
	p := &r.op.progress
	var itemPct, bytePct float64
	if p.TotalItems > 0 {
		itemPct = float64(p.CurrentItem) / float64(p.TotalItems)
	}
	if p.BytesTotal > 0 {
		bytePct = float64(p.BytesDone) / float64(p.BytesTotal)
	}
	pct := itemPct
	if bytePct > pct {
		pct = bytePct
	}
	if pct > 1 {
		pct = 1
	}
	p.Percent = pct * 100
	if p.Speed > 0 && p.BytesTotal > 0 && p.BytesDone < p.BytesTotal {
		p.ETASeconds = int(float64(p.BytesTotal-p.BytesDone) / p.Speed)
	} else {
		p.ETASeconds = -1
	}
}
