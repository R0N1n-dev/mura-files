// Package archive reads and writes the archive formats that can be handled
// with the Go standard library, and shells out to `7z`/`unrar` for the
// formats that cannot.
package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/R0N1n-dev/mura-files/internal/fsx"
	"github.com/R0N1n-dev/mura-files/internal/model"
)

// Format describes an archive container.
type Format string

const (
	FormatZip     Format = "zip"
	FormatTar     Format = "tar"
	FormatTarGz   Format = "tar.gz"
	FormatTarBz   Format = "tar.bz2"
	FormatTarXz   Format = "tar.xz"
	FormatTarZst  Format = "tar.zst"
	FormatGz      Format = "gz"
	Format7z      Format = "7z"
	FormatRar     Format = "rar"
	FormatDeb     Format = "deb"
	FormatRpm     Format = "rpm"
	FormatUnknown Format = ""
)

// Detect guesses the archive format from a filename.
func Detect(path string) Format {
	lower := strings.ToLower(filepath.Base(path))
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return FormatTarGz
	case strings.HasSuffix(lower, ".tar.bz2"), strings.HasSuffix(lower, ".tbz2"),
		strings.HasSuffix(lower, ".tbz"):
		return FormatTarBz
	case strings.HasSuffix(lower, ".tar.xz"), strings.HasSuffix(lower, ".txz"):
		return FormatTarXz
	case strings.HasSuffix(lower, ".tar.zst"), strings.HasSuffix(lower, ".tzst"):
		return FormatTarZst
	case strings.HasSuffix(lower, ".tar"):
		return FormatTar
	case strings.HasSuffix(lower, ".zip"), strings.HasSuffix(lower, ".jar"),
		strings.HasSuffix(lower, ".war"), strings.HasSuffix(lower, ".whl"),
		strings.HasSuffix(lower, ".apk"), strings.HasSuffix(lower, ".cbz"),
		strings.HasSuffix(lower, ".epub"), strings.HasSuffix(lower, ".odt"),
		strings.HasSuffix(lower, ".ods"), strings.HasSuffix(lower, ".docx"),
		strings.HasSuffix(lower, ".xlsx"), strings.HasSuffix(lower, ".pptx"):
		return FormatZip
	case strings.HasSuffix(lower, ".gz"):
		return FormatGz
	case strings.HasSuffix(lower, ".7z"):
		return Format7z
	case strings.HasSuffix(lower, ".rar"):
		return FormatRar
	case strings.HasSuffix(lower, ".deb"):
		return FormatDeb
	case strings.HasSuffix(lower, ".rpm"):
		return FormatRpm
	}
	return FormatUnknown
}

// IsArchive reports whether a path looks like something we can open.
func IsArchive(path string) bool { return Detect(path) != FormatUnknown }

// MaxEntries caps how many members we enumerate for the UI.
const MaxEntries = 5000

// List enumerates an archive.
func List(path string) (model.ArchiveInfo, error) {
	format := Detect(path)
	info := model.ArchiveInfo{Path: path, Format: string(format)}
	if format == FormatUnknown {
		return info, fmt.Errorf("unsupported archive format")
	}
	if st, err := os.Stat(path); err == nil {
		info.Compressed = st.Size()
	}
	switch format {
	case FormatZip:
		return listZip(path, info)
	case FormatTar, FormatTarGz, FormatTarBz:
		return listTar(path, format, info)
	}
	if out, err := listWith7z(path, info); err == nil {
		return out, nil
	}
	return info, fmt.Errorf("%s archives need the 7z tool", format)
}

func listZip(path string, info model.ArchiveInfo) (model.ArchiveInfo, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return info, err
	}
	defer reader.Close()
	for _, f := range reader.File {
		info.Total++
		info.Uncompressed += int64(f.UncompressedSize64)
		if len(info.Entries) < MaxEntries {
			info.Entries = append(info.Entries, model.ArchiveEntry{
				Name:    filepath.Base(f.Name),
				Path:    f.Name,
				Size:    int64(f.UncompressedSize64),
				Packed:  int64(f.CompressedSize64),
				ModTime: f.Modified.UnixMilli(),
				IsDir:   f.FileInfo().IsDir(),
				Mode:    f.Mode().String(),
			})
		} else {
			info.Truncated = true
		}
	}
	sortEntries(info.Entries)
	return info, nil
}

func listTar(path string, format Format, info model.ArchiveInfo) (model.ArchiveInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return info, err
	}
	defer f.Close()

	var reader io.Reader = f
	switch format {
	case FormatTarGz:
		gz, err := gzip.NewReader(f)
		if err != nil {
			return info, err
		}
		defer gz.Close()
		reader = gz
	case FormatTarBz:
		reader = bzip2.NewReader(f)
	}
	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return info, err
		}
		info.Total++
		info.Uncompressed += header.Size
		if len(info.Entries) < MaxEntries {
			info.Entries = append(info.Entries, model.ArchiveEntry{
				Name:    filepath.Base(header.Name),
				Path:    header.Name,
				Size:    header.Size,
				ModTime: header.ModTime.UnixMilli(),
				IsDir:   header.Typeflag == tar.TypeDir,
				Mode:    os.FileMode(header.Mode).String(),
			})
		} else {
			info.Truncated = true
		}
	}
	sortEntries(info.Entries)
	return info, nil
}

func sortEntries(entries []model.ArchiveEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Path < entries[j].Path
	})
}

// listWith7z falls back to the 7z CLI for exotic containers.
func listWith7z(path string, info model.ArchiveInfo) (model.ArchiveInfo, error) {
	bin := findTool("7z", "7za", "7zr")
	if bin == "" {
		return info, fmt.Errorf("7z is not installed")
	}
	out, err := exec.Command(bin, "l", "-slt", "--", path).Output()
	if err != nil {
		return info, err
	}
	var current model.ArchiveEntry
	flush := func() {
		if current.Path == "" {
			return
		}
		info.Total++
		info.Uncompressed += current.Size
		if len(info.Entries) < MaxEntries {
			info.Entries = append(info.Entries, current)
		} else {
			info.Truncated = true
		}
		current = model.ArchiveEntry{}
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			flush()
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "Path":
			if current.Path != "" {
				flush()
			}
			current.Path = value
			current.Name = filepath.Base(value)
		case "Size":
			current.Size, _ = strconv.ParseInt(value, 10, 64)
		case "Packed Size":
			current.Packed, _ = strconv.ParseInt(value, 10, 64)
		case "Modified":
			if t, err := parseLooseTime(value); err == nil {
				current.ModTime = t
			}
		case "Folder":
			current.IsDir = value == "+"
		case "Attributes":
			if strings.HasPrefix(value, "D") {
				current.IsDir = true
			}
		}
	}
	flush()
	sortEntries(info.Entries)
	return info, nil
}

func parseLooseTime(value string) (int64, error) {
	for _, layout := range []string{
		"2006-01-02 15:04:05.0000000",
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t.UnixMilli(), nil
		}
	}
	return 0, fmt.Errorf("unrecognised timestamp %q", value)
}

func findTool(names ...string) string {
	for _, n := range names {
		if p, err := exec.LookPath(n); err == nil {
			return p
		}
	}
	return ""
}

// Progress is invoked as extraction or creation proceeds.
type Progress func(current string, done, total int64, items, totalItems int)

// Extract unpacks an archive into destDir. It reports progress through the
// supplied callback and honours context cancellation.
func Extract(ctx context.Context, path, destDir string, progress Progress) error {
	format := Detect(path)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	switch format {
	case FormatZip:
		return extractZip(ctx, path, destDir, progress)
	case FormatTar, FormatTarGz, FormatTarBz:
		return extractTar(ctx, path, format, destDir, progress)
	case FormatGz:
		return extractSingleGz(ctx, path, destDir, progress)
	}
	return extractExternal(ctx, path, destDir, progress)
}

func extractZip(ctx context.Context, path, destDir string, progress Progress) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()

	total := len(reader.File)
	for i, f := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		target, err := safeJoin(destDir, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode().Perm()|0o600)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if progress != nil {
			progress(f.Name, int64(f.UncompressedSize64), 0, i+1, total)
		}
	}
	return nil
}

func extractTar(ctx context.Context, path string, format Format, destDir string, progress Progress) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var reader io.Reader = f
	switch format {
	case FormatTarGz:
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		reader = gz
	case FormatTarBz:
		reader = bzip2.NewReader(f)
	}
	tr := tar.NewReader(reader)
	index := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		index++
		target, err := safeJoin(destDir, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode).Perm()|0o700); err != nil {
				return err
			}
		case tar.TypeSymlink:
			_ = os.MkdirAll(filepath.Dir(target), 0o755)
			_ = os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		case tar.TypeLink:
			_ = os.MkdirAll(filepath.Dir(target), 0o755)
			if source, err := safeJoin(destDir, header.Linkname); err == nil {
				_ = os.Link(source, target)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
				os.FileMode(header.Mode).Perm()|0o600)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			out.Close()
			if copyErr != nil {
				return copyErr
			}
		}
		if progress != nil {
			progress(header.Name, header.Size, 0, index, 0)
		}
	}
}

func extractSingleGz(ctx context.Context, path, destDir string, progress Progress) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	name := gz.Name
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), ".gz")
	}
	target, err := safeJoin(destDir, name)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, gz)
	if progress != nil {
		progress(name, 0, 0, 1, 1)
	}
	return err
}

// extractExternal shells out to 7z or unar for formats the stdlib cannot read.
func extractExternal(ctx context.Context, path, destDir string, progress Progress) error {
	bin := findTool("7z", "7za", "7zr", "unar", "unrar")
	if bin == "" {
		return fmt.Errorf("no extractor available for %s", filepath.Base(path))
	}
	base := filepath.Base(bin)
	var cmd *exec.Cmd
	switch {
	case strings.Contains(base, "unar"):
		cmd = exec.CommandContext(ctx, bin, "-o", destDir, path)
	case strings.Contains(base, "unrar"):
		cmd = exec.CommandContext(ctx, bin, "x", "-y", path, destDir+"/")
	default:
		cmd = exec.CommandContext(ctx, bin, "x", "-y", "-o"+destDir, "--", path)
	}
	if progress != nil {
		progress(filepath.Base(path), 0, 0, 0, 0)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// safeJoin joins an archive member path to a destination, rejecting entries
// that would escape the destination directory (the "zip slip" attack).
//
// Absolute names and names containing a ".." segment are refused outright
// rather than silently rewritten, so a malicious archive is reported to the
// user instead of being quietly sanitised.
func safeJoin(destDir, name string) (string, error) {
	normalised := strings.ReplaceAll(name, "\\", "/")
	for _, segment := range strings.Split(normalised, "/") {
		if segment == ".." {
			return "", fmt.Errorf("archive entry %q escapes the destination", name)
		}
	}
	cleaned := filepath.Clean("/" + normalised)
	if cleaned == "/" {
		return destDir, nil
	}
	target := filepath.Join(destDir, cleaned)
	if !fsx.IsSameOrInside(destDir, target) {
		return "", fmt.Errorf("archive entry %q escapes the destination", name)
	}
	return target, nil
}
