package artifacts

import (
	"os"
	"strings"
	"testing"
)

func TestWriteWorkspaceScanReportCreatesMarkdownAndMetadata(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.WriteWorkspaceScanReport(WorkspaceScanReport{
		WorkspaceName:  "sample-workspace",
		Included:       5,
		Ignored:        2,
		DepthSkipped:   1,
		EntrySkipped:   3,
		Unreadable:     1,
		MaxDepth:       12,
		MaxEntries:     5000,
		Truncated:      true,
		IgnoredSamples: []string{"ignored: node_modules"},
		SkippedSamples: []string{"entry cap: vendor/pkg"},
		Message:        "Scanned 5 workspace entries, skipped 7.",
	})
	if err != nil {
		t.Fatalf("WriteWorkspaceScanReport returned error: %v", err)
	}
	if artifact.Kind != "scan-report" || !strings.Contains(artifact.RelPath, "/scan-reports/") || len(artifact.SourcePaths) != 1 {
		t.Fatalf("unexpected scan artifact: %#v", artifact)
	}
	data, err := os.ReadFile(artifact.AbsPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, expected := range []string{"# Workspace Scan Report - sample-workspace", "Indexed entries:** 5", "| Entry cap skipped | 3 |", "ignored: node_modules", "entry cap: vendor/pkg", "Next Actions"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected scan report to contain %q, got:\n%s", expected, text)
		}
	}
	metadata, _ := store.readMetadata(artifact.RelPath)
	if metadata.Kind != "scan-report" || metadata.ContextRelPath != "." || len(metadata.SourcePaths) != 1 || metadata.SourcePaths[0] != "." {
		t.Fatalf("unexpected scan report metadata: %#v", metadata)
	}
}

func TestWriteWorkspaceScanReportRequiresWorkspaceName(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.WriteWorkspaceScanReport(WorkspaceScanReport{}); err == nil {
		t.Fatal("expected workspace scan report without name to fail")
	}
}
