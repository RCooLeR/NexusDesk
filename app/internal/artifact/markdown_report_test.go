package artifact

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"NexusDesk/internal/workspace"
)

func TestCreateMarkdownReportWritesInsideArtifactDirectory(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC)

	report, err := CreateMarkdownReport(root, workspace.FilePreview{
		RelPath:  "docs/notes.md",
		Name:     "notes.md",
		Kind:     "file",
		FileType: "code",
		Encoding: "utf-8",
		Content:  "hello report",
		Size:     12,
	}, now)
	if err != nil {
		t.Fatalf("CreateMarkdownReport returned error: %v", err)
	}

	if report.RelPath != ".nexusdesk/artifacts/notes-20260513-123000.md" {
		t.Fatalf("unexpected report rel path: %s", report.RelPath)
	}

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(report.RelPath)))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "# Report: notes.md") {
		t.Fatalf("expected report title, got %q", text)
	}
	if !strings.Contains(text, "hello report") {
		t.Fatalf("expected source excerpt, got %q", text)
	}

	metadata := readTestMetadata(t, root, report.RelPath)
	if metadata.Kind != "markdown-report" {
		t.Fatalf("unexpected metadata kind: %s", metadata.Kind)
	}
	if len(metadata.SourcePaths) != 1 || metadata.SourcePaths[0] != "docs/notes.md" {
		t.Fatalf("unexpected metadata source paths: %#v", metadata.SourcePaths)
	}
}

func TestCreateMarkdownReportDoesNotOverwriteExistingReport(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC)
	source := workspace.FilePreview{Name: "notes.md", Content: "first"}

	if _, err := CreateMarkdownReport(root, source, now); err != nil {
		t.Fatalf("CreateMarkdownReport returned error: %v", err)
	}
	if _, err := CreateMarkdownReport(root, source, now); err == nil {
		t.Fatal("expected duplicate report write to fail")
	}
}

func TestCreateMarkdownReportTruncatesLargeExcerpt(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC)

	report, err := CreateMarkdownReport(root, workspace.FilePreview{
		Name:      "large.txt",
		Kind:      "file",
		Content:   strings.Repeat("a", reportContentLimit+10),
		Truncated: true,
	}, now)
	if err != nil {
		t.Fatalf("CreateMarkdownReport returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(report.RelPath)))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(content), "Source excerpt was truncated") {
		t.Fatalf("expected truncation note, got %q", string(content))
	}
}

func TestCreateMarkdownReportSkipsNonTextExcerpt(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC)

	report, err := CreateMarkdownReport(root, workspace.FilePreview{
		Name:    "brief.pdf",
		Kind:    "pdf",
		Content: "data:application/pdf;base64,abc",
	}, now)
	if err != nil {
		t.Fatalf("CreateMarkdownReport returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(report.RelPath)))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if strings.Contains(string(content), "data:application/pdf") {
		t.Fatalf("did not expect PDF data URL in report, got %q", string(content))
	}
}

func TestCreateGeneratedMarkdownWritesAssistantResponse(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC)

	report, err := CreateGeneratedMarkdown(root, MarkdownArtifactRequest{
		Title:          "Architecture Notes",
		Content:        "## Summary\n\n- Keep the shell small.",
		ContextRelPath: "docs/08_DELIVERY_PLAN.md",
		Prompt:         "Summarize the architecture",
		Model:          "qwen3:8b",
		Source:         "NexusDesk chat",
		SourcePaths:    []string{"docs/08_DELIVERY_PLAN.md"},
	}, now)
	if err != nil {
		t.Fatalf("CreateGeneratedMarkdown returned error: %v", err)
	}

	if report.RelPath != ".nexusdesk/artifacts/architecture-notes-20260513-123000.md" {
		t.Fatalf("unexpected generated artifact rel path: %s", report.RelPath)
	}

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(report.RelPath)))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "# Architecture Notes") {
		t.Fatalf("expected generated artifact title, got %q", text)
	}
	if !strings.Contains(text, "- Context: `docs/08_DELIVERY_PLAN.md`") {
		t.Fatalf("expected context metadata, got %q", text)
	}
	if !strings.Contains(text, "## Summary\n\n- Keep the shell small.") {
		t.Fatalf("expected assistant markdown content, got %q", text)
	}

	metadata := readTestMetadata(t, root, report.RelPath)
	if metadata.Kind != "chat-answer" {
		t.Fatalf("unexpected metadata kind: %s", metadata.Kind)
	}
	if metadata.Prompt != "Summarize the architecture" {
		t.Fatalf("unexpected metadata prompt: %s", metadata.Prompt)
	}
	if metadata.Model != "qwen3:8b" {
		t.Fatalf("unexpected metadata model: %s", metadata.Model)
	}
	if len(metadata.SourcePaths) != 1 || metadata.SourcePaths[0] != "docs/08_DELIVERY_PLAN.md" {
		t.Fatalf("unexpected metadata source paths: %#v", metadata.SourcePaths)
	}
}

func TestCreateGeneratedMarkdownRejectsEmptyResponse(t *testing.T) {
	_, err := CreateGeneratedMarkdown(t.TempDir(), MarkdownArtifactRequest{
		Title:   "Empty",
		Content: "  \n\t",
	}, time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected empty generated artifact to fail")
	}
}

func TestCreateGeneratedMarkdownTruncatesLargeResponse(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC)

	report, err := CreateGeneratedMarkdown(root, MarkdownArtifactRequest{
		Title:   "Large Answer",
		Content: strings.Repeat("a", generatedArtifactContentLimit+10),
	}, now)
	if err != nil {
		t.Fatalf("CreateGeneratedMarkdown returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(report.RelPath)))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(content), "Response content was truncated") {
		t.Fatalf("expected truncation note, got %q", string(content))
	}
}

func TestCreateDatasetChartSVGWritesArtifact(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC)

	report, err := CreateDatasetChartSVG(root, workspace.DatasetChartResult{
		RelPath:        "data/leads.csv",
		ChartType:      "bar",
		CategoryColumn: "channel",
		ValueColumn:    "revenue",
		Mode:           "sum",
		TotalRows:      3,
		UsedRows:       3,
		Message:        "Charted top 2 revenue totals by channel from data/leads.csv.",
		Points: []workspace.DatasetChartPoint{
			{Label: "search", Value: 16, Count: 2},
			{Label: "social", Value: 4, Count: 1},
		},
	}, now)
	if err != nil {
		t.Fatalf("CreateDatasetChartSVG returned error: %v", err)
	}

	if report.RelPath != ".nexusdesk/artifacts/revenue-by-channel-20260513-123000.svg" {
		t.Fatalf("unexpected chart rel path: %s", report.RelPath)
	}

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(report.RelPath)))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "<svg") || !strings.Contains(text, "Revenue by channel") && !strings.Contains(text, "revenue by channel") {
		t.Fatalf("expected SVG chart content, got %q", text)
	}
	if !strings.Contains(text, "search") || !strings.Contains(text, "16") {
		t.Fatalf("expected chart point content, got %q", text)
	}

	metadata := readTestMetadata(t, root, report.RelPath)
	if metadata.Kind != "chart-svg" {
		t.Fatalf("unexpected metadata kind: %s", metadata.Kind)
	}
	if len(metadata.SourcePaths) != 1 || metadata.SourcePaths[0] != "data/leads.csv" {
		t.Fatalf("unexpected metadata source paths: %#v", metadata.SourcePaths)
	}
}

func TestCreateDatasetQueryCSVWritesArtifact(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC)

	report, err := CreateDatasetQueryCSV(root, workspace.DatasetQueryResult{
		RelPath:     "data/leads.csv",
		Query:       "channel=search",
		Columns:     []string{"channel", "revenue"},
		Rows:        [][]string{{"search", "10"}, {"search", "6"}},
		TotalRows:   3,
		MatchedRows: 2,
		Message:     "2 matching rows from data/leads.csv.",
	}, now)
	if err != nil {
		t.Fatalf("CreateDatasetQueryCSV returned error: %v", err)
	}

	if report.RelPath != ".nexusdesk/artifacts/leads-query-20260513-123000.csv" {
		t.Fatalf("unexpected query rel path: %s", report.RelPath)
	}

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(report.RelPath)))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "channel,revenue") || !strings.Contains(text, "search,10") {
		t.Fatalf("expected CSV query content, got %q", text)
	}

	metadata := readTestMetadata(t, root, report.RelPath)
	if metadata.Kind != "dataset-query-csv" {
		t.Fatalf("unexpected metadata kind: %s", metadata.Kind)
	}
	if metadata.Prompt != "Export query \"channel=search\" from data/leads.csv" {
		t.Fatalf("unexpected metadata prompt: %s", metadata.Prompt)
	}
}

func TestCreateDatasetSummaryMarkdownWritesArtifact(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC)

	report, err := CreateDatasetSummaryMarkdown(root, workspace.FilePreview{
		RelPath: "data/leads.csv",
		Name:    "leads.csv",
		Table: &workspace.TablePreview{
			Columns:   []string{"channel", "revenue"},
			Rows:      [][]string{{"search", "10"}},
			TotalRows: 1,
			Profiles: []workspace.ColumnProfile{
				{Name: "channel", Type: "text", Distinct: 1},
				{Name: "revenue", Type: "integer", Distinct: 1, Min: "10", Max: "10"},
			},
		},
	}, now)
	if err != nil {
		t.Fatalf("CreateDatasetSummaryMarkdown returned error: %v", err)
	}
	if report.RelPath != ".nexusdesk/artifacts/leads-summary-20260513-123000.md" {
		t.Fatalf("unexpected summary rel path: %s", report.RelPath)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(report.RelPath)))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "# Dataset Summary: leads.csv") || !strings.Contains(text, "revenue") {
		t.Fatalf("expected dataset summary content, got %q", text)
	}
	metadata := readTestMetadata(t, root, report.RelPath)
	if metadata.Kind != "dataset-summary" {
		t.Fatalf("unexpected metadata kind: %s", metadata.Kind)
	}
}

func TestCreateScanReportMarkdownWritesSnapshotSummary(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC)

	report, err := CreateScanReportMarkdown(root, workspace.WorkspaceSnapshot{
		Name:      "sample-workspace",
		Truncated: true,
		Scan: workspace.ScanStatus{
			Included:       5,
			Ignored:        2,
			DepthSkipped:   1,
			MaxDepth:       10,
			MaxEntries:     800,
			IgnoredSamples: []string{"ignored: node_modules"},
			SkippedSamples: []string{"depth: app/deep/file.txt"},
		},
	}, now)
	if err != nil {
		t.Fatalf("CreateScanReportMarkdown returned error: %v", err)
	}

	if report.RelPath != ".nexusdesk/artifacts/sample-workspace-scan-report-20260513-123000.md" {
		t.Fatalf("unexpected scan report rel path: %s", report.RelPath)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(report.RelPath)))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "# Workspace Scan Report: sample-workspace") || !strings.Contains(text, "| Included | 5 |") {
		t.Fatalf("expected scan report counters, got %q", text)
	}

	metadata := readTestMetadata(t, root, report.RelPath)
	if metadata.Kind != "scan-report" || metadata.ContextRelPath != "." {
		t.Fatalf("unexpected scan metadata: %#v", metadata)
	}
}

func TestArchiveMovesArtifactAndMetadata(t *testing.T) {
	root := t.TempDir()
	report, err := CreateMarkdownReport(root, workspace.FilePreview{Name: "notes.md", RelPath: "docs/notes.md"}, time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CreateMarkdownReport returned error: %v", err)
	}

	archived, err := Archive(root, report.RelPath)
	if err != nil {
		t.Fatalf("Archive returned error: %v", err)
	}
	if archived.RelPath != ".nexusdesk/artifacts/archive/notes-20260513-123000.md" {
		t.Fatalf("unexpected archive path: %s", archived.RelPath)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(report.RelPath))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected source artifact to be moved, got %v", err)
	}
	metadata := readTestMetadata(t, root, archived.RelPath)
	if metadata.Kind != "markdown-report" {
		t.Fatalf("expected metadata to move with artifact, got %#v", metadata)
	}
}

func TestDeleteRemovesArtifactAndMetadata(t *testing.T) {
	root := t.TempDir()
	report, err := CreateMarkdownReport(root, workspace.FilePreview{Name: "notes.md", RelPath: "docs/notes.md"}, time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CreateMarkdownReport returned error: %v", err)
	}

	deleted, err := Delete(root, report.RelPath)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted.RelPath != report.RelPath {
		t.Fatalf("unexpected delete path: %s", deleted.RelPath)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(report.RelPath))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected artifact to be removed, got %v", err)
	}
	metadataPath := filepath.Join(root, ".nexusdesk", "artifacts", "notes-20260513-123000.meta.json")
	if _, err := os.Stat(metadataPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected metadata to be removed, got %v", err)
	}
}

func TestCompareArtifactsReportsAddedAndRemovedLines(t *testing.T) {
	root := t.TempDir()
	first, err := CreateGeneratedMarkdown(root, MarkdownArtifactRequest{
		Title:   "First",
		Content: "Shared\nOld line\n",
	}, time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CreateGeneratedMarkdown first returned error: %v", err)
	}
	second, err := CreateGeneratedMarkdown(root, MarkdownArtifactRequest{
		Title:   "Second",
		Content: "Shared\nNew line\n",
	}, time.Date(2026, 5, 13, 12, 31, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CreateGeneratedMarkdown second returned error: %v", err)
	}

	comparison, err := Compare(root, first.RelPath, second.RelPath)
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}
	if len(comparison.AddedLines) == 0 || len(comparison.RemovedLines) == 0 {
		t.Fatalf("expected added and removed lines, got %#v", comparison)
	}
	if comparison.LeftTitle != "First" || comparison.RightTitle != "Second" {
		t.Fatalf("unexpected comparison titles: %#v", comparison)
	}
}

func TestListReturnsMarkdownArtifactsNewestFirst(t *testing.T) {
	root := t.TempDir()
	firstTime := time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC)
	secondTime := firstTime.Add(time.Minute)

	firstReport, err := CreateMarkdownReport(root, workspace.FilePreview{Name: "first.md", RelPath: "first.md"}, firstTime)
	if err != nil {
		t.Fatalf("CreateMarkdownReport first failed: %v", err)
	}
	secondReport, err := CreateMarkdownReport(root, workspace.FilePreview{Name: "second.md", RelPath: "second.md"}, secondTime)
	if err != nil {
		t.Fatalf("CreateMarkdownReport second failed: %v", err)
	}
	if err := os.Chtimes(firstReport.Path, firstTime, firstTime); err != nil {
		t.Fatalf("Chtimes first failed: %v", err)
	}
	if err := os.Chtimes(secondReport.Path, secondTime, secondTime); err != nil {
		t.Fatalf("Chtimes second failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".nexusdesk", "artifacts", "ignored.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	artifacts, err := List(root)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(artifacts) != 2 {
		t.Fatalf("expected 2 markdown artifacts, got %d", len(artifacts))
	}
	if artifacts[0].Name != "second-20260513-123100.md" {
		t.Fatalf("expected newest artifact first, got %s", artifacts[0].Name)
	}
	if artifacts[0].RelPath != ".nexusdesk/artifacts/second-20260513-123100.md" {
		t.Fatalf("unexpected artifact rel path: %s", artifacts[0].RelPath)
	}
	if artifacts[0].Summary != "second.md" {
		t.Fatalf("expected artifact summary from metadata, got %q", artifacts[0].Summary)
	}
	if artifacts[0].Source != "selected preview" {
		t.Fatalf("expected artifact source from metadata, got %q", artifacts[0].Source)
	}
}

func TestMetadataReturnsArtifactMetadata(t *testing.T) {
	root := t.TempDir()
	report, err := CreateDatasetQueryCSV(root, workspace.DatasetQueryResult{
		RelPath: "data/leads.csv",
		Columns: []string{"channel"},
		Rows:    [][]string{{"search"}},
	}, time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CreateDatasetQueryCSV returned error: %v", err)
	}

	metadata, err := Metadata(root, report.RelPath)
	if err != nil {
		t.Fatalf("Metadata returned error: %v", err)
	}
	if metadata.Kind != "dataset-query-csv" {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
}

func TestSearchFindsArtifactMetadata(t *testing.T) {
	root := t.TempDir()
	if _, err := CreateDatasetQueryCSV(root, workspace.DatasetQueryResult{
		RelPath: "data/leads.csv",
		Query:   "channel=search",
		Columns: []string{"channel"},
		Rows:    [][]string{{"search"}},
	}, time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC)); err != nil {
		t.Fatalf("CreateDatasetQueryCSV returned error: %v", err)
	}

	results, err := Search(root, "channel=search")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 || results[0].MatchType != "artifact" {
		t.Fatalf("unexpected search results: %#v", results)
	}
}

func TestListReturnsDatasetQueryCSVArtifacts(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC)

	queryExport, err := CreateDatasetQueryCSV(root, workspace.DatasetQueryResult{
		RelPath: "data/leads.csv",
		Columns: []string{"channel", "revenue"},
		Rows:    [][]string{{"search", "10"}},
		Message: "1 matching row from data/leads.csv.",
	}, now)
	if err != nil {
		t.Fatalf("CreateDatasetQueryCSV returned error: %v", err)
	}

	artifacts, err := List(root)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 query artifact, got %d", len(artifacts))
	}
	if artifacts[0].RelPath != queryExport.RelPath {
		t.Fatalf("unexpected query artifact rel path: %s", artifacts[0].RelPath)
	}
	if artifacts[0].Kind != "dataset-query-csv" {
		t.Fatalf("unexpected query artifact kind: %s", artifacts[0].Kind)
	}
}

func TestListReturnsSVGChartArtifacts(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 13, 12, 30, 0, 0, time.UTC)

	chart, err := CreateDatasetChartSVG(root, workspace.DatasetChartResult{
		RelPath:        "data/leads.csv",
		CategoryColumn: "channel",
		Points:         []workspace.DatasetChartPoint{{Label: "search", Value: 2, Count: 2}},
		Message:        "Charted top 1 categories from data/leads.csv.",
	}, now)
	if err != nil {
		t.Fatalf("CreateDatasetChartSVG returned error: %v", err)
	}

	artifacts, err := List(root)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 chart artifact, got %d", len(artifacts))
	}
	if artifacts[0].RelPath != chart.RelPath {
		t.Fatalf("unexpected chart artifact rel path: %s", artifacts[0].RelPath)
	}
	if artifacts[0].Kind != "chart-svg" {
		t.Fatalf("unexpected chart artifact kind: %s", artifacts[0].Kind)
	}
}

func TestListReturnsEmptyWhenArtifactDirectoryIsMissing(t *testing.T) {
	artifacts, err := List(t.TempDir())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(artifacts) != 0 {
		t.Fatalf("expected no artifacts, got %d", len(artifacts))
	}
}

func readTestMetadata(t *testing.T, root string, relPath string) ArtifactMetadata {
	t.Helper()

	artifactPath := filepath.Join(root, filepath.FromSlash(relPath))
	content, err := os.ReadFile(artifactMetadataPath(artifactPath))
	if err != nil {
		t.Fatalf("ReadFile metadata failed: %v", err)
	}

	var metadata ArtifactMetadata
	if err := json.Unmarshal(content, &metadata); err != nil {
		t.Fatalf("Unmarshal metadata failed: %v", err)
	}
	return metadata
}
