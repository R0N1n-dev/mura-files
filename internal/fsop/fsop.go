// Package fsop implements the low level filesystem primitives used by the
// operation engine: recursive copy, move, delete and permission handling.
//
// Everything in this package is blocking and cancellation aware; the higher
// level operation service adds progress reporting and event streaming on top.
package fsop

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/R0N1n-dev/mura-files/internal/fsx"
)

// Options tunes the behaviour of the transfer primitives.
type Options struct {
	// PreserveAttrs restores mode, timestamps and (where permitted) ownership.
	PreserveAttrs bool
	// FollowSymlinks copies the link target instead of recreating the link.
	FollowSymlinks bool
	// Overwrite replaces an existing destination.
	Overwrite bool
	// RenameOnConflict picks a free name instead of overwriting.
	RenameOnConflict bool
	// SkipErrors records failures and keeps going.
	SkipErrors bool
	// Exclude is consulted for every path; returning true skips it.
	Exclude func(path string) bool
}

// Progress is invoked after each successfully transferred item.
type Progress func(current string, bytesDone, bytesTotal int64, item, total int)

// Result summarises a completed operation.
type Result struct {
	Items     int
	Bytes     int64
	Skipped   int
	Errors    []string
	Cancelled bool
}

// ErrCancelled is returned when the context is cancelled mid operation.
var ErrCancelled = errors.New("operation cancelled")

// ErrExists is returned when a destination already exists and neither
// overwriting nor renaming on conflict was requested.
var ErrExists = errors.New("destination already exists")

func checkCancel(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ErrCancelled
	default:
		return nil
	}
}

// CopyFile copies a single regular file, streaming with a modest buffer so
// that very large files do not exhaust memory.
func CopyFile(src, dst string, info os.FileInfo, opts Options) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return 0, err
	}
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer in.Close()

	perm := os.FileMode(0o644)
	if info != nil {
		perm = info.Mode().Perm()
	}
	if opts.Overwrite {
		_ = os.Remove(dst)
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return 0, err
	}
	written, err := io.Copy(out, in)
	closeErr := out.Close()
	if err != nil {
		return written, err
	}
	if closeErr != nil {
		return written, closeErr
	}
	if opts.PreserveAttrs && info != nil {
		_ = os.Chtimes(dst, time.Now(), info.ModTime())
		if st, ok := info.Sys().(*syscall.Stat_t); ok {
			_ = os.Chown(dst, int(st.Uid), int(st.Gid))
		}
	}
	return written, nil
}

// CopyTree copies a file, directory or symlink recursively. The destination is
// always the full target path (not the parent directory).
func CopyTree(ctx context.Context, src, dst string, opts Options, progress Progress, acc *Result) error {
	if err := checkCancel(ctx); err != nil {
		return err
	}
	if opts.Exclude != nil && opts.Exclude(src) {
		return nil
	}
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	switch {
	case info.Mode()&os.ModeSymlink != 0 && !opts.FollowSymlinks:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if opts.Overwrite {
			_ = os.Remove(dst)
		}
		if err := os.Symlink(target, dst); err != nil && !os.IsExist(err) {
			return err
		}
		if acc != nil {
			acc.Items++
		}
		if progress != nil {
			progress(src, 0, 0, 0, 0)
		}
		return nil

	case info.IsDir():
		perm := info.Mode().Perm()
		if err := os.MkdirAll(dst, perm); err != nil {
			return err
		}
		if opts.PreserveAttrs {
			_ = os.Chtimes(dst, time.Now(), info.ModTime())
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		if acc != nil {
			acc.Items++
		}
		if progress != nil {
			progress(src, 0, 0, 0, 0)
		}
		for _, e := range entries {
			childSrc := filepath.Join(src, e.Name())
			childDst := filepath.Join(dst, e.Name())
			if err := CopyTree(ctx, childSrc, childDst, opts, progress, acc); err != nil {
				if err == ErrCancelled {
					return err
				}
				if acc != nil {
					acc.Errors = append(acc.Errors, err.Error())
				}
				if !opts.SkipErrors {
					return err
				}
			}
		}
		return nil

	default:
		if _, err := os.Lstat(dst); err == nil && !opts.Overwrite {
			if !opts.RenameOnConflict {
				return ErrExists
			}
			dir := filepath.Dir(dst)
			dst = filepath.Join(dir, fsx.UniqueName(dir, filepath.Base(dst)))
		}
		n, err := CopyFile(src, dst, info, opts)
		if err != nil {
			return err
		}
		if acc != nil {
			acc.Items++
			acc.Bytes += n
		}
		if progress != nil {
			progress(src, n, info.Size(), 0, 0)
		}
		return nil
	}
}

// MovePath renames src to dst, falling back to a copy plus delete when the two
// paths live on different filesystems.
func MovePath(ctx context.Context, src, dst string, opts Options, progress Progress, acc *Result) error {
	if err := checkCancel(ctx); err != nil {
		return err
	}
	if opts.Exclude != nil && opts.Exclude(src) {
		return nil
	}
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if opts.Overwrite {
		_ = os.RemoveAll(dst)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		if acc != nil {
			acc.Items++
			if !info.IsDir() {
				acc.Bytes += info.Size()
			}
		}
		if progress != nil {
			progress(src, info.Size(), info.Size(), 0, 0)
		}
		return nil
	}

	// Different filesystem (EXDEV) or a directory merge: copy then remove.
	copyOpts := opts
	copyOpts.Overwrite = true
	if err := CopyTree(ctx, src, dst, copyOpts, progress, acc); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

// RemovePath deletes a path. When permanent is false the item is moved to the
// freedesktop trash instead. Callers that need trash support pass a mover.
func RemovePath(path string, permanent bool, mover func(string) (string, error)) error {
	if !permanent && mover != nil {
		_, err := mover(path)
		return err
	}
	return os.RemoveAll(path)
}

// Chmod applies a permission mask to a path.
func Chmod(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}

// TreeSize walks a tree and reports the number of files, directories and bytes.
func TreeSize(ctx context.Context, path string, limit int64) (files, dirs, links int64, bytes int64, truncated bool) {
	root, err := os.Lstat(path)
	if err != nil {
		return 0, 0, 0, 0, false
	}
	if !root.IsDir() {
		if root.Mode()&os.ModeSymlink != 0 {
			return 0, 0, 1, 0, false
		}
		return 1, 0, 0, root.Size(), false
	}
	dirs = 1
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return filepath.SkipAll
		}
		if p == path {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			links++
		case d.IsDir():
			dirs++
		default:
			files++
			bytes += info.Size()
		}
		if limit > 0 && files+dirs+links > limit {
			truncated = true
			return filepath.SkipAll
		}
		return nil
	})
	return files, dirs, links, bytes, truncated
}
