package workspace

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestSearchUsesFastTextPathForCSV(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "data", "people.csv"), "id,name\n1,needle\n")

	results, err := New().Search(root, "needle", SearchOptions{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	assertSearchContains(t, results, "data/people.csv", "content")
}

func TestSearchSkipsBinaryContent(t *testing.T) {
	root := t.TempDir()
	writeBytes(t, filepath.Join(root, "blob.bin"), []byte{'n', 'e', 'e', 'd', 'l', 'e', 0x00, 'x'})

	results, err := New().Search(root, "needle", SearchOptions{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected binary content to be skipped, got %#v", results)
	}
}

func TestSearchSkipsStructuredPreviewFormats(t *testing.T) {
	root := t.TempDir()
	writeBytes(t, filepath.Join(root, "book.xlsx"), []byte("needle in fake structured package"))
	writeBytes(t, filepath.Join(root, "brief.docx"), []byte("needle in fake document"))
	writeBytes(t, filepath.Join(root, "brief.pdf"), []byte("%PDF needle"))

	results, err := New().Search(root, "needle", SearchOptions{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected structured preview formats to be skipped, got %#v", results)
	}
}

func TestSearchSkipsKnownBinaryExtensionsBeforeContentMatch(t *testing.T) {
	root := t.TempDir()
	writeBytes(t, filepath.Join(root, "asset.png"), []byte("needle in fake image"))
	writeBytes(t, filepath.Join(root, "bundle.zip"), []byte("needle in fake archive"))
	writeBytes(t, filepath.Join(root, "module.wasm"), []byte("needle in fake wasm"))

	results, err := New().Search(root, "needle", SearchOptions{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected known binary extensions to be skipped before content matching, got %#v", results)
	}
}

func TestSearchScansFullSafeTextFile(t *testing.T) {
	root := t.TempDir()
	content := strings.Repeat("a", 80*1024) + "\nneedle after old prefix cap\n"
	writeFile(t, filepath.Join(root, "large.txt"), content)

	results, err := New().Search(root, "needle", SearchOptions{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 || results[0].RelPath != "large.txt" {
		t.Fatalf("expected full safe text search to find late content, got %#v", results)
	}
}

func TestSearchStreamsLiteralMatchBeyondWritePreviewCap(t *testing.T) {
	root := t.TempDir()
	var builder strings.Builder
	for builder.Len() <= writeContentMaxBytes+64*1024 {
		builder.WriteString("ordinary filler line with no marker\n")
	}
	builder.WriteString("needle after safe write cap\n")
	writeFile(t, filepath.Join(root, "large.txt"), builder.String())

	results, err := New().Search(root, "needle", SearchOptions{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 || results[0].RelPath != "large.txt" || !strings.Contains(results[0].Snippet, "needle") {
		t.Fatalf("expected streaming literal search to find late content, got %#v", results)
	}
}

func TestSearchStreamsRegexMatchWithLineBound(t *testing.T) {
	root := t.TempDir()
	var builder strings.Builder
	for builder.Len() <= writeContentMaxBytes+64*1024 {
		builder.WriteString("ordinary filler line with no marker\n")
	}
	builder.WriteString("const SearchPanel = true\n")
	writeFile(t, filepath.Join(root, "large.ts"), builder.String())

	results, err := New().Search(root, "search[-_]?panel", SearchOptions{Regex: true})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 || results[0].RelPath != "large.ts" || results[0].MatchType != "content-regex" {
		t.Fatalf("expected streaming regex search to find late content, got %#v", results)
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

func TestSearchStopsAtWallClockLimit(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.txt"), "needle\n")
	writeFile(t, filepath.Join(root, "b.txt"), "needle\n")

	base := time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC)
	calls := 0
	oldNow := nowUTC
	nowUTC = func() time.Time {
		calls++
		return base.Add(time.Duration(calls) * time.Millisecond)
	}
	t.Cleanup(func() {
		nowUTC = oldNow
	})

	_, metadata, err := New().SearchWithMetadata(root, "needle", SearchOptions{MaxDuration: time.Nanosecond})
	if err != nil {
		t.Fatalf("SearchWithMetadata returned error: %v", err)
	}
	if !metadata.TimedOut || !metadata.Truncated {
		t.Fatalf("expected timed out truncated metadata, got %#v", metadata)
	}
	if metadata.DurationMs <= 0 {
		t.Fatalf("expected duration metadata, got %#v", metadata)
	}
}

func TestSearchWithMetadataContextHonorsCancellation(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.txt"), "needle\n")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := New().SearchWithMetadataContext(ctx, root, "needle", SearchOptions{})
	if err == nil || err != context.Canceled {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}

func TestSearchStreamsPartialResultsCallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.txt"), "needle one\n")
	writeFile(t, filepath.Join(root, "b.txt"), "needle two\n")
	var snapshots [][]SearchResult

	results, _, err := New().SearchWithMetadata(root, "needle", SearchOptions{
		ResultCallback: func(partial []SearchResult) {
			snapshots = append(snapshots, append([]SearchResult(nil), partial...))
		},
	})
	if err != nil {
		t.Fatalf("SearchWithMetadata returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected final results, got %#v", results)
	}
	if len(snapshots) == 0 {
		t.Fatal("expected partial result callback")
	}
	if len(snapshots[len(snapshots)-1]) != len(results) {
		t.Fatalf("expected final callback snapshot to match final results, got %#v vs %#v", snapshots[len(snapshots)-1], results)
	}
}

func TestSearchSyntheticLargeWorkspaceStaysBounded(t *testing.T) {
	root := t.TempDir()
	for dirIndex := 0; dirIndex < 12; dirIndex++ {
		for fileIndex := 0; fileIndex < 20; fileIndex++ {
			content := "ordinary line\n"
			if fileIndex%9 == 0 {
				content += "needle in larger workspace\n"
			}
			writeFile(t, filepath.Join(root, "pkg", fmt.Sprintf("dir-%02d", dirIndex), fmt.Sprintf("file-%02d.txt", fileIndex)), content)
		}
	}
	for fileIndex := 0; fileIndex < 50; fileIndex++ {
		writeFile(t, filepath.Join(root, "node_modules", "pkg", fmt.Sprintf("ignored-%02d.js", fileIndex)), "needle ignored\n")
	}

	results, metadata, err := New().SearchWithMetadata(root, "needle", SearchOptions{MaxResults: 25})
	if err != nil {
		t.Fatalf("SearchWithMetadata returned error: %v", err)
	}
	if len(results) != 25 {
		t.Fatalf("expected result cap to apply, got %d result(s)", len(results))
	}
	if !metadata.Truncated || metadata.ResultCount != 25 {
		t.Fatalf("expected truncated metadata at cap, got %#v", metadata)
	}
	if metadata.DirectoriesSkipped == 0 {
		t.Fatalf("expected ignored directories to be skipped, got %#v", metadata)
	}
	for _, result := range results {
		if strings.Contains(result.RelPath, "node_modules") {
			t.Fatalf("ignored dependency folder leaked into results: %#v", results)
		}
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
