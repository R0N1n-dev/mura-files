package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/R0N1n-dev/mura-files/internal/model"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestDetect(t *testing.T) {
	cases := map[string]Format{
		"a.zip": FormatZip, "a.jar": FormatZip, "a.tar": FormatTar,
		"a.tar.gz": FormatTarGz, "a.tgz": FormatTarGz,
		"a.tar.bz2": FormatTarBz, "a.tar.xz": FormatTarXz,
		"a.tar.zst": FormatTarZst, "a.gz": FormatGz,
		"a.7z": Format7z, "a.rar": FormatRar, "a.deb": FormatDeb,
		"a.txt": FormatUnknown,
	}
	for name, want := range cases {
		if got := Detect(name); got != want {
			t.Errorf("Detect(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestFormatForExtension(t *testing.T) {
	cases := map[string]Format{
		"zip": FormatZip, ".zip": FormatZip, "tar": FormatTar,
		"tar.gz": FormatTarGz, "tgz": FormatTarGz, "7z": Format7z,
		"nonsense": FormatZip,
	}
	for ext, want := range cases {
		if got := FormatForExtension(ext); got != want {
			t.Errorf("FormatForExtension(%q) = %q, want %q", ext, got, want)
		}
	}
}

func TestZipRoundTrip(t *testing.T) {
	source := t.TempDir()
	write(t, filepath.Join(source, "a.txt"), "alpha")
	write(t, filepath.Join(source, "folder", "b.txt"), "beta")

	dest := filepath.Join(t.TempDir(), "bundle.zip")
	if err := Create(context.Background(), dest, []string{source}, FormatZip, nil); err != nil {
		t.Fatalf("Create: %v", err)
	}

	info, err := List(dest)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if info.Total != 4 {
		t.Fatalf("expected 4 entries (2 dirs, 2 files), got %d", info.Total)
	}
	if info.Uncompressed != 9 {
		t.Errorf("uncompressed size = %d, want 9", info.Uncompressed)
	}

	out := t.TempDir()
	if err := Extract(context.Background(), dest, out, nil); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got := read(t, filepath.Join(out, filepath.Base(source), "a.txt")); got != "alpha" {
		t.Errorf("a.txt = %q", got)
	}
	if got := read(t, filepath.Join(out, filepath.Base(source), "folder", "b.txt")); got != "beta" {
		t.Errorf("b.txt = %q", got)
	}
}

func TestTarGzRoundTrip(t *testing.T) {
	source := t.TempDir()
	write(t, filepath.Join(source, "one.txt"), "111")
	write(t, filepath.Join(source, "two.txt"), "2222")

	dest := filepath.Join(t.TempDir(), "bundle.tar.gz")
	if err := Create(context.Background(), dest, []string{source}, FormatTarGz, nil); err != nil {
		t.Fatalf("Create: %v", err)
	}

	info, err := List(dest)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if info.Uncompressed != 7 {
		t.Errorf("uncompressed size = %d, want 7", info.Uncompressed)
	}

	out := t.TempDir()
	if err := Extract(context.Background(), dest, out, nil); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got := read(t, filepath.Join(out, filepath.Base(source), "two.txt")); got != "2222" {
		t.Errorf("two.txt = %q", got)
	}
}

func TestCreateProducesValidZip(t *testing.T) {
	source := t.TempDir()
	write(t, filepath.Join(source, "payload.bin"), "data")
	dest := filepath.Join(t.TempDir(), "x.zip")
	if err := Create(context.Background(), dest, []string{source}, FormatZip, nil); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(dest)
	if err != nil {
		t.Fatalf("the produced zip is not readable: %v", err)
	}
	defer reader.Close()
	if len(reader.File) == 0 {
		t.Error("zip should contain entries")
	}
}

func TestCreateProducesValidTar(t *testing.T) {
	source := t.TempDir()
	write(t, filepath.Join(source, "payload.bin"), "data")
	dest := filepath.Join(t.TempDir(), "x.tar.gz")
	if err := Create(context.Background(), dest, []string{source}, FormatTarGz, nil); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("not a valid gzip stream: %v", err)
	}
	tr := tar.NewReader(gz)
	count := 0
	for {
		if _, err := tr.Next(); err != nil {
			break
		}
		count++
	}
	if count == 0 {
		t.Error("tar should contain entries")
	}
}

func TestExtractRejectsPathTraversal(t *testing.T) {
	dest := t.TempDir()
	for _, name := range []string{"../escape.txt", "../../escape.txt", "a/../../escape.txt"} {
		if target, err := safeJoin(dest, name); err == nil {
			t.Errorf("safeJoin(%q) unexpectedly succeeded with %q", name, target)
		}
	}
	if target, err := safeJoin(dest, "nested/file.txt"); err != nil {
		t.Errorf("safeJoin rejected a normal path: %v", err)
	} else if !filepath.IsAbs(target) {
		t.Errorf("safeJoin returned a relative path: %q", target)
	}
}

func TestArchiveName(t *testing.T) {
	if got := ArchiveName([]string{"/tmp/photos"}, FormatZip); got != "photos.zip" {
		t.Errorf("ArchiveName = %q", got)
	}
	if got := ArchiveName([]string{"/tmp/a.txt"}, FormatTarGz); got != "a.tar.gz" {
		t.Errorf("ArchiveName = %q", got)
	}
	if got := ArchiveName([]string{"/tmp/a", "/tmp/b"}, FormatZip); got != "archive.zip" {
		t.Errorf("ArchiveName = %q", got)
	}
}

func TestListTruncatesHugeArchives(t *testing.T) {
	// Build a zip with more members than the display cap.
	dest := filepath.Join(t.TempDir(), "many.zip")
	file, err := os.Create(dest)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(file)
	for i := 0; i < MaxEntries+25; i++ {
		w, err := zw.Create(filepath.Join("f", string(rune('a'+i%26))+string(rune('0'+i%10))+"x"+string(rune('0'+(i/10)%10))+".txt"))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte("x"))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	info, err := List(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Truncated {
		t.Error("expected the listing to be flagged as truncated")
	}
	if len(info.Entries) > MaxEntries {
		t.Errorf("entries should be capped at %d, got %d", MaxEntries, len(info.Entries))
	}
	if info.Total <= MaxEntries {
		t.Errorf("Total should count every member, got %d", info.Total)
	}
	var _ = model.ArchiveInfo{}
}
