package artifact

import (
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
