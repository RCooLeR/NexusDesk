package workspace

import (
	"path/filepath"
	"testing"
	"strings"
)

func TestSearchFindsPathAndContentMatches(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "main.go"), "package main\nfunc SearchNeedle() {}\n")
	writeFile(t, filepath.Join(root, "docs", "needle-guide.md"), "plain docs")

	results, err := New().Search(root, "needle", SearchOptions{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	assertSearchContains(t, results, "src/main.go", "content")
	assertSearchContains(t, results, "docs/needle-guide.md", "path")
}

func TestSearchSkipsIgnoredFoldersAndMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "node_modules", "pkg", "index.js"), "needle")
	writeFile(t, filepath.Join(root, ".nexusdesk", "config.json"), "needle")

	results, err := New().Search(root, "needle", SearchOptions{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected ignored folders to be skipped, got %#v", results)
	}
}

func TestSearchSupportsRegex(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "search_panel.tsx"), "const panel = 'SearchPanel';\n")
	writeFile(t, filepath.Join(root, "docs", "search-panel.md"), "plain docs")

	results, err := New().Search(root, "search[-_]?panel", SearchOptions{Regex: true})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	assertSearchContains(t, results, "src/search_panel.tsx", "path-regex")
	assertSearchContains(t, results, "src/search_panel.tsx", "content-regex")
	assertSearchContains(t, results, "docs/search-panel.md", "path-regex")
}

func TestSearchRejectsInvalidRegex(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "main.go"), "package main\n")

	if _, err := New().Search(root, "[", SearchOptions{Regex: true}); err == nil {
		t.Fatal("expected invalid regex to return an error")
	}
}

func TestSearchHonorsMaxResults(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.txt"), "needle\n")
	writeFile(t, filepath.Join(root, "b.txt"), "needle\n")
	writeFile(t, filepath.Join(root, "c.txt"), "needle\n")

	results, err := New().Search(root, "needle", SearchOptions{MaxResults: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected capped results, got %#v", results)
	}
}

func TestSearchReturnsMultipleContentMatchesPerFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "notes.txt"), "needle first\nneedle second\nno match\nneedle third\n")

	results, err := New().Search(root, "needle", SearchOptions{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	var contentMatches int
	lines := map[int]bool{}
	for _, result := range results {
		if result.RelPath == "notes.txt" && result.MatchType == "content" {
			contentMatches++
			lines[result.Line] = true
		}
	}
	if contentMatches != 3 {
		t.Fatalf("expected three content matches in one file, got %d (%#v)", contentMatches, results)
	}
	for _, line := range []int{1, 2, 4} {
		if !lines[line] {
			t.Fatalf("expected content match on line %d, got %#v", line, results)
		}
	}
}

func TestSearchCapsContentMatchesPerFile(t *testing.T) {
	root := t.TempDir()
	content := ""
	for index := 0; index < defaultSearchPerFileMax+5; index++ {
		content += "needle line\n"
	}
	writeFile(t, filepath.Join(root, "many.txt"), content)

	results, err := New().Search(root, "needle", SearchOptions{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	var contentMatches int
	for _, result := range results {
		if result.RelPath == "many.txt" && result.MatchType == "content" {
			contentMatches++
		}
	}
	if contentMatches != defaultSearchPerFileMax {
		t.Fatalf("expected per-file cap %d, got %d", defaultSearchPerFileMax, contentMatches)
	}
}

func TestTrimSearchSnippetCentersMatchOnLongLines(t *testing.T) {
	line := strings.Repeat("a", 90) + "target" + strings.Repeat("b", 90)
	matchStart := strings.Index(line, "target")
	got := trimSearchSnippet(line, matchStart)
	if !strings.Contains(got, "target") {
		t.Fatalf("expected snippet to include match, got %q", got)
	}
	if !strings.HasPrefix(got, "...") || !strings.HasSuffix(got, "...") {
		t.Fatalf("expected abbreviated boundaries for long line, got %q", got)
	}
}

func assertSearchContains(t *testing.T, results []SearchResult, relPath string, matchType string) {
	t.Helper()
	for _, result := range results {
		if result.RelPath == relPath && result.MatchType == matchType {
			return
		}
	}
	t.Fatalf("expected search result %s/%s in %#v", relPath, matchType, results)
}
