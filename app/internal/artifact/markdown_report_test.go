package artifact

import (
	"encoding/json"
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
