package workspace

import "testing"

func TestSearchFindsPathAndContentMatches(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/main.go", "package main\nfunc SearchNeedle() {}\n")
	writeFile(t, root, "docs/needle-guide.md", "plain docs")

	results, err := Search(root, "needle", SearchOptions{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	assertSearchContains(t, results, "src/main.go", "content")
	assertSearchContains(t, results, "docs/needle-guide.md", "path")
}

func TestSearchSkipsIgnoredFolders(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "node_modules/pkg/index.js", "needle")

	results, err := Search(root, "needle", SearchOptions{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected ignored folder to be skipped, got %#v", results)
	}
}

func TestSearchSupportsRegex(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/search_panel.tsx", "const panel = 'SearchPanel';\n")
	writeFile(t, root, "docs/search-panel.md", "plain docs")

	results, err := Search(root, "search[-_]?panel", SearchOptions{Regex: true})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	assertSearchContains(t, results, "src/search_panel.tsx", "path-regex")
	assertSearchContains(t, results, "src/search_panel.tsx", "content-regex")
	assertSearchContains(t, results, "docs/search-panel.md", "path-regex")
}

func TestSearchRejectsInvalidRegex(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/main.go", "package main\n")

	if _, err := Search(root, "[", SearchOptions{Regex: true}); err == nil {
		t.Fatal("expected invalid regex to return an error")
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
