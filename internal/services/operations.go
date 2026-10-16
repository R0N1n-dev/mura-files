package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/R0N1n-dev/mura-files/internal/archive"
	"github.com/R0N1n-dev/mura-files/internal/fsop"
	"github.com/R0N1n-dev/mura-files/internal/fsx"
	"github.com/R0N1n-dev/mura-files/internal/model"
	"github.com/R0N1n-dev/mura-files/internal/ops"
	"github.com/R0N1n-dev/mura-files/internal/places"
)

// OperationService runs copy, move, delete and archive jobs.
type OperationService struct {
	rt  *Runtime
	seq atomic.Uint64
}

// NewOperationService creates the service.
func NewOperationService(rt *Runtime) *OperationService { return &OperationService{rt: rt} }

// ServiceName identifies the service in logs and bindings.
func (s *OperationService) ServiceName() string { return "Operations" }

// Copy starts a background copy job and returns its identifier.
func (s *OperationService) Copy(sources []string, dest string, opts model.TransferOptions) (string, error) {
	if len(sources) == 0 {
		return "", fmt.Errorf("nothing to copy")
	}
	dest = fsx.Abs(dest)
	if info, err := os.Stat(dest); err != nil || !info.IsDir() {
		return "", fmt.Errorf("destination must be an existing folder")
	}
	if reason := validateTransfer(sources, dest); reason != "" {
		return "", fmt.Errorf("%s", reason)
	}
	opts.RemoveSource = false
	title := fmt.Sprintf("Copying %s", describeSources(sources))
	return s.rt.Ops.Start("copy", title, len(sources), func(ctx context.Context, r *ops.Reporter) error {
		return s.transfer(ctx, r, sources, dest, opts, false)
	}), nil
}

// Move starts a background move job and returns its identifier.
func (s *OperationService) Move(sources []string, dest string, opts model.TransferOptions) (string, error) {
	if len(sources) == 0 {
		return "", fmt.Errorf("nothing to move")
	}
	dest = fsx.Abs(dest)
	if info, err := os.Stat(dest); err != nil || !info.IsDir() {
		return "", fmt.Errorf("destination must be an existing folder")
	}
	if reason := validateTransfer(sources, dest); reason != "" {
		return "", fmt.Errorf("%s", reason)
	}
	opts.RemoveSource = true
	title := fmt.Sprintf("Moving %s", describeSources(sources))
	return s.rt.Ops.Start("move", title, len(sources), func(ctx context.Context, r *ops.Reporter) error {
		return s.transfer(ctx, r, sources, dest, opts, true)
	}), nil
}

// Delete starts a background delete job. When opts.Permanent is false the
// items go to the trash instead of being unlinked.
func (s *OperationService) Delete(paths []string, opts model.DeleteOptions) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("nothing to delete")
	}
	kind := "delete"
	title := fmt.Sprintf("Deleting %s", describeSources(paths))
	if !opts.Permanent {
		kind = "trash"
		title = fmt.Sprintf("Moving %s to Trash", describeSources(paths))
	}
	return s.rt.Ops.Start(kind, title, len(paths), func(ctx context.Context, r *ops.Reporter) error {
		var firstErr error
		for _, p := range paths {
			if err := ctx.Err(); err != nil {
				return err
			}
			abs := fsx.Abs(p)
			r.SetCurrent(abs)
			if !opts.Permanent {
				if _, err := places.TrashPath(abs); err != nil {
					r.Error(fmt.Errorf("%s: %s", filepath.Base(abs), fsx.FriendlyError(err)))
					if firstErr == nil {
						firstErr = err
					}
					if !opts.SkipErrors {
						return err
					}
				}
			} else if err := os.RemoveAll(abs); err != nil {
				r.Error(fmt.Errorf("%s: %s", filepath.Base(abs), fsx.FriendlyError(err)))
				if firstErr == nil {
					firstErr = err
				}
				if !opts.SkipErrors {
					return err
				}
			}
			r.Item(abs)
		}
		return firstErr
	}), nil
}

// Duplicate creates "name (copy).ext" style duplicates next to the originals.
func (s *OperationService) Duplicate(paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("nothing to duplicate")
	}
	return s.rt.Ops.Start("duplicate", fmt.Sprintf("Duplicating %s", describeSources(paths)),
		len(paths), func(ctx context.Context, r *ops.Reporter) error {
			var firstErr error
			for _, p := range paths {
				if err := ctx.Err(); err != nil {
					return err
				}
				abs := fsx.Abs(p)
				dir := filepath.Dir(abs)
				base := filepath.Base(abs)
				ext := filepath.Ext(base)
				stem := strings.TrimSuffix(base, ext)
				candidate := fsx.UniqueName(dir, stem+" copy"+ext)
				r.SetCurrent(abs)
				err := fsop.CopyTree(ctx, abs, filepath.Join(dir, candidate),
					fsop.Options{PreserveAttrs: true, SkipErrors: true}, nil, nil)
				if err != nil {
					r.Error(err)
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				r.Item(abs)
			}
			return firstErr
		}), nil
}

// Compress packs the selection into a new archive.
func (s *OperationService) Compress(sources []string, dest string, format string) (string, error) {
	if len(sources) == 0 {
		return "", fmt.Errorf("nothing to compress")
	}
	f := archive.FormatForExtension(format)
	dest = fsx.Abs(dest)
	return s.rt.Ops.Start("compress", fmt.Sprintf("Creating %s", filepath.Base(dest)),
		len(sources), func(ctx context.Context, r *ops.Reporter) error {
			r.SetTotal(len(sources), 0)
			return archive.Create(ctx, dest, sources, f,
				func(current string, done, total int64, items, totalItems int) {
					r.SetCurrent(current)
					if done > 0 {
						r.Bytes(done)
					}
					if items > 0 {
						r.Item(current)
					}
				})
		}), nil
}

// Extract unpacks archives into dest. When dest is empty each archive is
// extracted into a sibling folder named after the archive.
func (s *OperationService) Extract(archives []string, dest string) (string, error) {
	if len(archives) == 0 {
		return "", fmt.Errorf("nothing to extract")
	}
	return s.rt.Ops.Start("extract", fmt.Sprintf("Extracting %s", describeSources(archives)),
		len(archives), func(ctx context.Context, r *ops.Reporter) error {
			var firstErr error
			for _, a := range archives {
				if err := ctx.Err(); err != nil {
					return err
				}
				abs := fsx.Abs(a)
				target := dest
				if target == "" {
					base := filepath.Base(abs)
					for _, suffix := range []string{".tar.gz", ".tar.bz2", ".tar.xz", ".tar.zst", ".tgz", ".tbz2", ".txz"} {
						base = strings.TrimSuffix(base, suffix)
					}
					base = strings.TrimSuffix(base, filepath.Ext(base))
					parent := filepath.Dir(abs)
					target = filepath.Join(parent, fsx.UniqueName(parent, base))
				} else {
					target = fsx.Abs(target)
				}
				r.SetCurrent(abs)
				err := archive.Extract(ctx, abs, target,
					func(current string, done, total int64, items, totalItems int) {
						r.SetCurrent(current)
						if items > 0 {
							r.Item(current)
						}
					})
				if err != nil {
					r.Error(fmt.Errorf("%s: %s", filepath.Base(abs), err.Error()))
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				r.Item(abs)
			}
			return firstErr
		}), nil
}

// Cancel stops a running job.
func (s *OperationService) Cancel(id string) bool { return s.rt.Ops.Cancel(id) }

// CancelAll stops every running job.
func (s *OperationService) CancelAll() { s.rt.Ops.CancelAll() }

// Active lists the running and recently finished jobs.
func (s *OperationService) Active() []model.OpProgress { return s.rt.Ops.Active() }

// ClearFinished drops completed jobs from the list.
func (s *OperationService) ClearFinished() { s.rt.Ops.ClearFinished() }

// ---------------------------------------------------------------------------
// Internals
// ---------------------------------------------------------------------------

// transfer performs the copy or move work for a batch of sources.
func (s *OperationService) transfer(ctx context.Context, r *ops.Reporter, sources []string,
	dest string, opts model.TransferOptions, move bool) error {

	plan, totalBytes, totalItems := buildPlan(sources, dest)
	r.SetTotal(totalItems, totalBytes)

	var firstErr error
	for _, item := range plan {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.SetCurrent(item.src)

		if item.skip {
			r.Item(item.src)
			continue
		}

		transferOpts := fsop.Options{
			PreserveAttrs:    opts.PreserveAttrs,
			FollowSymlinks:   opts.FollowSymlinks,
			Overwrite:        item.overwrite,
			RenameOnConflict: false,
			SkipErrors:       opts.SkipErrors,
		}
		progress := func(current string, done, doneTotal int64, items, total int) {
			r.SetCurrent(current)
			if done > 0 {
				r.Bytes(done)
			}
		}
		var err error
		if move {
			err = fsop.MovePath(ctx, item.src, item.dst, transferOpts, progress, nil)
		} else {
			err = fsop.CopyTree(ctx, item.src, item.dst, transferOpts, progress, nil)
		}
		if err != nil {
			r.Error(fmt.Errorf("%s: %s", filepath.Base(item.src), fsx.FriendlyError(err)))
			if firstErr == nil {
				firstErr = err
			}
			if !opts.SkipErrors {
				return err
			}
			continue
		}
		r.Item(item.src)
	}
	return firstErr
}

type transferItem struct {
	src       string
	dst       string
	overwrite bool
	skip      bool
}

// buildPlan resolves destination names and conflict handling up front so the
// UI can show an accurate item and byte total.
func buildPlan(sources []string, dest string) ([]transferItem, int64, int) {
	plan := make([]transferItem, 0, len(sources))
	var totalBytes int64
	totalItems := 0

	// Track names already claimed in this batch to avoid collisions between
	// two sources with the same basename.
	claimed := map[string]bool{}
	for _, src := range sources {
		src = fsx.Abs(src)
		base := filepath.Base(src)
		target := filepath.Join(dest, base)
		item := transferItem{src: src, dst: target}

		if fsx.IsSameOrInside(src, dest) && filepath.Dir(src) == dest {
			// Moving a file onto itself inside the same folder is a no-op.
			item.skip = true
			plan = append(plan, item)
			continue
		}
		if _, err := os.Lstat(target); err == nil || claimed[target] {
			unique := filepath.Join(dest, fsx.UniqueName(dest, base))
			item.dst = unique
		}
		claimed[item.dst] = true

		files, dirs, _, bytes, _ := fsop.TreeSize(context.Background(), src, 50000)
		totalItems += int(files + dirs + 1)
		totalBytes += bytes
		plan = append(plan, item)
	}
	return plan, totalBytes, totalItems
}

// validateTransfer rejects obviously wrong transfers before starting a job.
func validateTransfer(sources []string, dest string) string {
	for _, src := range sources {
		abs := fsx.Abs(src)
		if abs == dest {
			return "a folder cannot be copied into itself"
		}
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			if fsx.IsSameOrInside(abs, dest) {
				return "a folder cannot be copied into one of its own subfolders"
			}
		}
	}
	return ""
}

// describeSources renders a short human readable label for a selection.
func describeSources(paths []string) string {
	switch len(paths) {
	case 0:
		return "nothing"
	case 1:
		return filepath.Base(paths[0])
	case 2:
		return filepath.Base(paths[0]) + " and " + filepath.Base(paths[1])
	default:
		return fmt.Sprintf("%d items", len(paths))
	}
}
