package fsx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/R0N1n-dev/mura-files/internal/model"
)

func TestExpandAndAbs(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available")
	}
	if got := Expand("~"); got != home {
		t.Fatalf("Expand(~) = %q, want %q", got, home)
	}
	if got := Expand("~/Documents"); got != filepath.Join(home, "Documents") {
		t.Fatalf("Expand(~/Documents) = %q", got)
	}
	if got := Abs("relative"); !filepath.IsAbs(got) {
		t.Fatalf("Abs(relative) = %q, want an absolute path", got)
	}
}

func TestExtAndBaseExt(t *testing.T) {
	cases := []struct{ name, ext, base string }{
		{"report.pdf", ".pdf", ".pdf"},
		{"backup.tar.gz", ".gz", ".tar.gz"},
		{"archive.tar.bz2", ".bz2", ".tar.bz2"},
		{"noextension", "", ""},
		{"UPPER.PDF", ".pdf", ".pdf"},
		{"weird.tar.zst", ".zst", ".tar.zst"},
	}
	for _, tc := range cases {
		if got := Ext(tc.name); got != tc.ext {
			t.Errorf("Ext(%q) = %q, want %q", tc.name, got, tc.ext)
		}
		if got := BaseExt(tc.name); got != tc.base {
			t.Errorf("BaseExt(%q) = %q, want %q", tc.name, got, tc.base)
		}
	}
}

func TestKindClassification(t *testing.T) {
	cases := []struct {
		name string
		dir  bool
		want string
	}{
		{"src", true, "directory"},
		{"photo.png", false, "image"},
		{"clip.mkv", false, "video"},
		{"song.flac", false, "audio"},
		{"bundle.tar.gz", false, "archive"},
		{"paper.pdf", false, "document"},
		{"main.go", false, "code"},
		{"notes.txt", false, "text"},
		{"inter.ttf", false, "font"},
		{"mystery", false, "text"},
	}
	for _, tc := range cases {
		if got := Kind(tc.name, tc.dir); got != tc.want {
			t.Errorf("Kind(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestMimeOverrides(t *testing.T) {
	cases := map[string]string{
		"app.ts":     "text/typescript",
		"main.go":    "text/x-go",
		"data.json":  "application/json",
		"README.md":  "text/markdown",
		"Dockerfile": "text/x-dockerfile",
		"script.sh":  "text/x-shellscript",
		"vector.svg": "image/svg+xml",
		"bundle.7z":  "application/x-7z-compressed",
	}
	for name, want := range cases {
		if got := MimeOf(name); got != want {
			t.Errorf("MimeOf(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestIconKey(t *testing.T) {
	if got := IconKey("/tmp/x", true, "directory"); got != "folder" {
		t.Errorf("directory icon = %q", got)
	}
	if got := IconKey("/tmp/a.pdf", false, "document"); got != "file-pdf" {
		t.Errorf("pdf icon = %q", got)
	}
	if got := IconKey("/tmp/a.unknown", false, "other"); got != "file" {
		t.Errorf("unknown icon = %q", got)
	}
}

func TestNaturalLess(t *testing.T) {
	ordered := []string{"file1", "file2", "file10", "file20", "File100"}
	for i := 0; i < len(ordered)-1; i++ {
		if !naturalLess(ordered[i], ordered[i+1]) {
			t.Errorf("expected %q to sort before %q", ordered[i], ordered[i+1])
		}
	}
}

func TestListSortsAndFilters(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "beta.txt"), "bbb")
	mustWrite(t, filepath.Join(dir, "alpha.txt"), "a")
	mustWrite(t, filepath.Join(dir, "gamma.txt"), "cc")
	if err := os.Mkdir(filepath.Join(dir, "zeta"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, ".hidden"), "x")

	listing := List(dir, model.ListOptions{SortBy: "name", DirsFirst: true})
	if listing.Error != "" {
		t.Fatalf("unexpected listing error: %s", listing.Error)
	}
	if listing.Total != 4 {
		t.Fatalf("expected 4 visible entries, got %d", listing.Total)
	}
	if listing.Hidden != 1 {
		t.Fatalf("expected 1 hidden entry, got %d", listing.Hidden)
	}
	if !listing.Entries[0].IsDir || listing.Entries[0].Name != "zeta" {
		t.Fatalf("expected the folder first, got %q", listing.Entries[0].Name)
	}
	if listing.Entries[1].Name != "alpha.txt" {
		t.Fatalf("expected alpha.txt second, got %q", listing.Entries[1].Name)
	}

	withHidden := List(dir, model.ListOptions{ShowHidden: true, SortBy: "name"})
	if withHidden.Total != 5 {
		t.Fatalf("expected 5 entries with hidden files, got %d", withHidden.Total)
	}

	filtered := List(dir, model.ListOptions{Filter: "alpha", SortBy: "name"})
	if filtered.Total != 4 || filtered.Filtered != 1 {
		t.Fatalf("filter should keep 1 of 4, got %d of %d", filtered.Filtered, filtered.Total)
	}
}

func TestListSortBySizeDescending(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "small"), "1")
	mustWrite(t, filepath.Join(dir, "medium"), "12345")
	mustWrite(t, filepath.Join(dir, "large"), "1234567890")

	listing := List(dir, model.ListOptions{SortBy: "size", SortDesc: true, DirsFirst: false})
	if listing.Entries[0].Name != "large" || listing.Entries[2].Name != "small" {
		t.Fatalf("unexpected size order: %v", names(listing.Entries))
	}
}

func TestRecursiveListingRespectsDepth(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "one", "two", "three")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "one", "a.txt"), "a")
	mustWrite(t, filepath.Join(deep, "b.txt"), "b")

	listing := List(root, model.ListOptions{Recursive: true, Depth: 1, SortBy: "name"})
	if listing.Filtered != 1 {
		t.Fatalf("depth 1 should only see the first level, saw %v", names(listing.Entries))
	}
	full := List(root, model.ListOptions{Recursive: true, SortBy: "name"})
	if full.Filtered < 3 {
		t.Fatalf("recursive listing should include nested entries, saw %v", names(full.Entries))
	}
}

func TestBuildEntryFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "script.sh")
	mustWrite(t, path, "#!/bin/sh\necho hi\n")
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}

	entry, err := Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Ext != ".sh" {
		t.Errorf("Ext = %q", entry.Ext)
	}
	if entry.Kind != "code" {
		t.Errorf("Kind = %q", entry.Kind)
	}
	if !entry.IsExec {
		t.Error("expected the entry to be marked executable")
	}
	if entry.Perm != "-rwxr-xr-x" {
		t.Errorf("Perm = %q", entry.Perm)
	}
	if entry.Mode != "0755" {
		t.Errorf("Mode = %q", entry.Mode)
	}
	if entry.Owner == "" || entry.Group == "" {
		t.Errorf("owner/group should be resolved, got %q/%q", entry.Owner, entry.Group)
	}
	if entry.ModTime == 0 {
		t.Error("ModTime should be populated")
	}
}

func TestSymlinkDetection(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.txt")
	mustWrite(t, target, "hello")
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks are not supported here")
	}
	broken := filepath.Join(dir, "broken.txt")
	if err := os.Symlink(filepath.Join(dir, "nope"), broken); err != nil {
		t.Fatal(err)
	}

	entry, err := Stat(link)
	if err != nil {
		t.Fatal(err)
	}
	if !entry.IsSymlink || entry.LinkBroken {
		t.Errorf("valid link misdetected: %+v", entry)
	}
	if entry.Target != target {
		t.Errorf("Target = %q, want %q", entry.Target, target)
	}

	brokenEntry, err := Stat(broken)
	if err != nil {
		t.Fatal(err)
	}
	if !brokenEntry.LinkBroken {
		t.Error("expected the dangling link to be reported as broken")
	}
}

func TestUniqueName(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "report.txt"), "1")
	if got := UniqueName(dir, "report.txt"); got != "report (2).txt" {
		t.Fatalf("UniqueName = %q", got)
	}
	mustWrite(t, filepath.Join(dir, "report (2).txt"), "2")
	if got := UniqueName(dir, "report.txt"); got != "report (3).txt" {
		t.Fatalf("UniqueName = %q", got)
	}
	if got := UniqueName(dir, "fresh.txt"); got != "fresh.txt" {
		t.Fatalf("UniqueName for a new file = %q", got)
	}
}

func TestAncestorsAndContainment(t *testing.T) {
	ancestors := Ancestors("/a/b/c")
	want := []string{"/", "/a", "/a/b", "/a/b/c"}
	if len(ancestors) != len(want) {
		t.Fatalf("Ancestors = %v, want %v", ancestors, want)
	}
	for i := range want {
		if ancestors[i] != want[i] {
			t.Fatalf("Ancestors = %v, want %v", ancestors, want)
		}
	}
	if !IsSameOrInside("/a", "/a/b") {
		t.Error("expected /a/b to be inside /a")
	}
	if IsSameOrInside("/a", "/ab") {
		t.Error("/ab must not be treated as inside /a")
	}
	if !IsSameOrInside("/a", "/a") {
		t.Error("a path must be inside itself")
	}
}

func TestComplete(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "alpha.txt"), "a")
	mustWrite(t, filepath.Join(dir, "alpine.txt"), "a")
	mustWrite(t, filepath.Join(dir, "beta.txt"), "b")
	if err := os.Mkdir(filepath.Join(dir, "alps"), 0o755); err != nil {
		t.Fatal(err)
	}

	results := Complete(filepath.Join(dir, "alp"), 10)
	if len(results) != 3 {
		t.Fatalf("expected 3 completions, got %d (%v)", len(results), results)
	}
	if !results[0].IsDir {
		t.Errorf("folders should be listed first, got %+v", results[0])
	}
}

func TestFriendlyError(t *testing.T) {
	_, err := os.Stat(filepath.Join(t.TempDir(), "missing"))
	if got := FriendlyError(err); got != "Location does not exist" {
		t.Fatalf("FriendlyError = %q", got)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func names(entries []model.Entry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Name)
	}
	return out
}
