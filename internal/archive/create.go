package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Create writes a new archive at dest containing every path in sources. Only
// zip and tar.gz are produced natively; anything else is delegated to 7z.
func Create(ctx context.Context, dest string, sources []string, format Format, progress Progress) error {
	switch format {
	case FormatZip:
		return createZip(ctx, dest, sources, progress)
	case FormatTar, FormatTarGz:
		return createTar(ctx, dest, sources, format, progress)
	}
	return createExternal(ctx, dest, sources, format, progress)
}

// FormatForExtension maps a user supplied extension to a supported format.
func FormatForExtension(ext string) Format {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "zip":
		return FormatZip
	case "tar":
		return FormatTar
	case "tar.gz", "tgz", "gz":
		return FormatTarGz
	case "tar.xz", "txz", "xz":
		return FormatTarXz
	case "tar.bz2", "tbz2", "bz2":
		return FormatTarBz
	case "7z":
		return Format7z
	}
	return FormatZip
}

// ArchiveName builds a sensible default archive filename for a selection.
func ArchiveName(sources []string, format Format) string {
	ext := "." + string(format)
	if format == FormatTarGz {
		ext = ".tar.gz"
	}
	if len(sources) == 1 {
		base := filepath.Base(sources[0])
		base = strings.TrimSuffix(base, filepath.Ext(base))
		return base + ext
	}
	return "archive" + ext
}

func walkSources(ctx context.Context, sources []string, fn func(path string, info os.FileInfo, rel string) error) error {
	for _, src := range sources {
		if err := ctx.Err(); err != nil {
			return err
		}
		src = filepath.Clean(src)
		base := filepath.Dir(src)
		info, err := os.Lstat(src)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel := filepath.Base(src)
			if err := fn(src, info, rel); err != nil {
				return err
			}
			continue
		}
		err = filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			rel, relErr := filepath.Rel(base, path)
			if relErr != nil {
				return nil
			}
			return fn(path, fi, rel)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func createZip(ctx context.Context, dest string, sources []string, progress Progress) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	count := 0

	err = walkSources(ctx, sources, func(path string, info os.FileInfo, rel string) error {
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, in)
		in.Close()
		if copyErr != nil {
			return copyErr
		}
		count++
		if progress != nil {
			progress(path, info.Size(), 0, count, 0)
		}
		return nil
	})
	if err != nil {
		zw.Close()
		_ = os.Remove(dest)
		return err
	}
	return zw.Close()
}

func createTar(ctx context.Context, dest string, sources []string, format Format, progress Progress) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	var sink io.Writer = f
	var gz *gzip.Writer
	if format == FormatTarGz {
		gz = gzip.NewWriter(f)
		gz.Name = ""
		sink = gz
	}
	tw := tar.NewWriter(sink)
	count := 0

	writeErr := walkSources(ctx, sources, func(path string, info os.FileInfo, rel string) error {
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if info.IsDir() {
			header.Name += "/"
		}
		// Avoid embedding machine specific ownership into the archive.
		header.Uid, header.Gid = 0, 0
		header.Uname, header.Gname = "", ""
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, in)
		in.Close()
		if copyErr != nil {
			return copyErr
		}
		count++
		if progress != nil {
			progress(path, info.Size(), 0, count, 0)
		}
		return nil
	})
	if writeErr != nil {
		tw.Close()
		if gz != nil {
			gz.Close()
		}
		_ = os.Remove(dest)
		return writeErr
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if gz != nil {
		return gz.Close()
	}
	return nil
}

func createExternal(ctx context.Context, dest string, sources []string, format Format, progress Progress) error {
	bin := findTool("7z", "7za", "7zr")
	if bin == "" {
		return fmt.Errorf("creating %s archives needs the 7z tool", format)
	}
	args := []string{"a", "-y"}
	switch format {
	case Format7z:
		args = append(args, "-t7z")
	case FormatTarXz:
		args = append(args, "-ttar", "-mx=6")
	default:
		args = append(args, "-ttar")
	}
	args = append(args, dest)
	for _, s := range sources {
		args = append(args, s)
	}
	if progress != nil {
		progress(dest, 0, 0, 0, 0)
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
