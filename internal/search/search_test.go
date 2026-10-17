package search

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

// run collects every batch produced by a search.
func run(t *testing.T, query model.SearchQuery) []model.SearchResults {
	t.Helper()
	engine := New()
	done := make(chan []model.SearchResults, 1)
	var batches []model.SearchResults
	engine.Start(query, func(batch model.SearchResults) {
		batches = append(batches, batch)
		if batch.Done {
			select {
			case done <- batches:
			default:
			}
		}
	})
	select {
	case result := <-done:
		return result
	case <-time.After(10 * time.Second):
		t.Fatal("search did not finish in time")
		return nil
	}
}

func collect(batches []model.SearchResults) ([]model.Entry, int) {
	var entries []model.Entry
	scanned := 0
	for _, batch := range batches {
		entries = append(entries, batch.Entries...)
		if batch.Scanned > scanned {
			scanned = batch.Scanned
		}
	}
	return entries, scanned
}

func TestGlobSearch(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "report-final.txt"), "x")
	write(t, filepath.Join(root, "notes.txt"), "x")
	write(t, filepath.Join(root, "nested", "report-draft.txt"), "x")
	write(t, filepath.Join(root, "nested", "image.png"), "x")

	batches := run(t, model.SearchQuery{
		Roots: []string{root}, Pattern: "report-*.txt", Mode: "glob",
		IncludeFiles: true, IncludeDirs: true,
	})
	entries, scanned := collect(batches)
	if len(entries) != 2 {
		t.Fatalf("expected 2 matches, got %d (%v)", len(entries), entries)
	}
	if scanned < 4 {
		t.Errorf("expected at least 4 items scanned, got %d", scanned)
	}
	if !batches[len(batches)-1].Done {
		t.Error("the final batch must be flagged as done")
	}
}

func TestContainsAndExactModes(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "changelog.md"), "x")
	write(t, filepath.Join(root, "log.txt"), "x")

	contains := run(t, model.SearchQuery{
		Roots: []string{root}, Pattern: "log", Mode: "contains", IncludeFiles: true,
	})
	entries, _ := collect(contains)
	if len(entries) != 2 {
		t.Fatalf("contains should match 2 names, got %d", len(entries))
	}

	exact := run(t, model.SearchQuery{
		Roots: []string{root}, Pattern: "log.txt", Mode: "exact", IncludeFiles: true,
	})
	entries, _ = collect(exact)
	if len(entries) != 1 {
		t.Fatalf("exact should match 1 name, got %d", len(entries))
	}
}

func TestRegexSearchAndCaseSensitivity(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "Photo-2024.jpg"), "x")
	write(t, filepath.Join(root, "photo-2023.jpg"), "x")
	write(t, filepath.Join(root, "scan.png"), "x")

	insensitive := run(t, model.SearchQuery{
		Roots: []string{root}, Pattern: "^photo-\\d{4}\\.jpg$", Mode: "regex", IncludeFiles: true,
	})
	entries, _ := collect(insensitive)
	if len(entries) != 2 {
		t.Fatalf("case-insensitive regex should match 2, got %d", len(entries))
	}

	sensitive := run(t, model.SearchQuery{
		Roots: []string{root}, Pattern: "^photo-\\d{4}\\.jpg$", Mode: "regex",
		IncludeFiles: true, MatchCase: true,
	})
	entries, _ = collect(sensitive)
	if len(entries) != 1 {
		t.Fatalf("case-sensitive regex should match 1, got %d", len(entries))
	}
}

func TestHiddenAndDirectoryFilters(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ".secret", "hidden.txt"), "x")
	write(t, filepath.Join(root, "visible", "shown.txt"), "x")

	withoutHidden := run(t, model.SearchQuery{
		Roots: []string{root}, Pattern: "*", Mode: "glob", IncludeFiles: true, IncludeDirs: true,
	})
	entries, _ := collect(withoutHidden)
	for _, entry := range entries {
		if entry.Name == ".secret" {
			t.Error("hidden folders must be skipped by default")
		}
	}

	withHidden := run(t, model.SearchQuery{
		Roots: []string{root}, Pattern: "*.txt", Mode: "glob",
		IncludeFiles: true, IncludeHidden: true,
	})
	entries, _ = collect(withHidden)
	if len(entries) != 2 {
		t.Fatalf("expected both files with hidden enabled, got %d", len(entries))
	}

	dirsOnly := run(t, model.SearchQuery{
		Roots: []string{root}, Pattern: "visible", Mode: "exact",
		IncludeFiles: false, IncludeDirs: true,
	})
	entries, _ = collect(dirsOnly)
	if len(entries) != 1 || !entries[0].IsDir {
		t.Fatalf("expected exactly one directory match, got %+v", entries)
	}
}

func TestSizeAndExtensionFilters(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "small.txt"), "1")
	write(t, filepath.Join(root, "large.txt"), string(make([]byte, 500)))
	write(t, filepath.Join(root, "medium.png"), string(make([]byte, 100)))

	bigOnly := run(t, model.SearchQuery{
		Roots: []string{root}, Pattern: "*", Mode: "glob",
		IncludeFiles: true, MinSize: 200,
	})
	entries, _ := collect(bigOnly)
	if len(entries) != 1 || entries[0].Name != "large.txt" {
		t.Fatalf("size filter returned %v", entries)
	}

	extOnly := run(t, model.SearchQuery{
		Roots: []string{root}, Pattern: "*", Mode: "glob",
		IncludeFiles: true, ExtFilter: []string{".png"},
	})
	entries, _ = collect(extOnly)
	if len(entries) != 1 || entries[0].Name != "medium.png" {
		t.Fatalf("extension filter returned %v", entries)
	}
}

func TestKindFilter(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "photo.png"), "x")
	write(t, filepath.Join(root, "song.mp3"), "x")
	write(t, filepath.Join(root, "notes.txt"), "x")

	images := run(t, model.SearchQuery{
		Roots: []string{root}, Pattern: "*", Mode: "glob",
		IncludeFiles: true, KindFilter: "image",
	})
	entries, _ := collect(images)
	if len(entries) != 1 || entries[0].Name != "photo.png" {
		t.Fatalf("kind filter returned %v", entries)
	}
}

func TestContentSearch(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "match.txt"), "hello\nworld\nNEEDLE here\n")
	write(t, filepath.Join(root, "miss.txt"), "nothing to see\n")
	write(t, filepath.Join(root, "binary.bin"), "\x00\x01NEEDLE\x02")

	batches := run(t, model.SearchQuery{
		Roots: []string{root}, Pattern: "needle", Mode: "contains",
		IncludeFiles: true, Content: true, MatchCase: false,
	})
	entries, _ := collect(batches)
	if len(entries) != 1 {
		t.Fatalf("expected only the text file to match, got %v", entries)
	}
	if entries[0].Name != "match.txt" {
		t.Errorf("unexpected match %q", entries[0].Name)
	}
}

func TestMaxResultsAndDepth(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 10; i++ {
		write(t, filepath.Join(root, "f", string(rune('a'+i))+".txt"), "x")
	}

	limited := run(t, model.SearchQuery{
		Roots: []string{root}, Pattern: "*.txt", Mode: "glob",
		IncludeFiles: true, MaxResults: 3,
	})
	entries, _ := collect(limited)
	if len(entries) != 3 {
		t.Fatalf("MaxResults should cap the hits at 3, got %d", len(entries))
	}
	if limited[len(limited)-1].Error == "" {
		t.Error("hitting the limit should be reported")
	}

	deep := t.TempDir()
	write(t, filepath.Join(deep, "one", "two", "three", "deep.txt"), "x")
	write(t, filepath.Join(deep, "shallow.txt"), "x")
	shallowOnly := run(t, model.SearchQuery{
		Roots: []string{deep}, Pattern: "*.txt", Mode: "glob",
		IncludeFiles: true, MaxDepth: 1,
	})
	entries, _ = collect(shallowOnly)
	if len(entries) != 1 || entries[0].Name != "shallow.txt" {
		t.Fatalf("MaxDepth filter returned %v", entries)
	}
}

func TestInvalidRegexReportsError(t *testing.T) {
	batches := run(t, model.SearchQuery{
		Roots: []string{t.TempDir()}, Pattern: "([unclosed", Mode: "regex", IncludeFiles: true,
	})
	if len(batches) == 0 || batches[len(batches)-1].Error == "" {
		t.Fatal("an invalid pattern should surface an error")
	}
}

func TestCancel(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 40; i++ {
		write(t, filepath.Join(root, string(rune('a'+i%26)), "f.txt"), "x")
	}
	engine := New()
	done := make(chan struct{})
	id := engine.Start(model.SearchQuery{
		Roots: []string{root}, Pattern: "*", Mode: "glob", IncludeFiles: true,
	}, func(batch model.SearchResults) {
		if batch.Done {
			close(done)
		}
	})
	if len(engine.Running()) != 1 {
		t.Errorf("expected 1 running search, got %d", len(engine.Running()))
	}
	if !engine.Cancel(id) {
		t.Error("Cancel should report success for a live search")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled search did not finish")
	}
}

func TestGlobToRegex(t *testing.T) {
	cases := map[string]string{
		"*.txt":     `^.*\.txt$`,
		"a?c":       `^a.c$`,
		"file.name": `^file\.name$`,
	}
	for glob, want := range cases {
		if got := globToRegex(glob); got != want {
			t.Errorf("globToRegex(%q) = %q, want %q", glob, got, want)
		}
	}
}
