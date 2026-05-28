package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanReportSkipsIgnoredAndCapsEntries(t *testing.T) {
	root := t.TempDir()
	mustWriteScanFile(t, root, "README.md", "hello")
	mustWriteScanFile(t, root, "src/main.go", "package main")
	mustWriteScanFile(t, root, "node_modules/pkg/index.js", "ignored")
	mustWriteScanFile(t, root, ".nexusdesk/artifacts/report.md", "metadata")
	mustWriteScanFile(t, root, "deep/a/b/c.txt", "too deep")

	report, err := New().ScanReport(context.Background(), root, ScanReportOptions{
		MaxDepth:   2,
		MaxEntries: 3,
		MaxSamples: 4,
	})
	if err != nil {
		t.Fatalf("ScanReport returned error: %v", err)
	}
	if report.Name == "" || report.Root == "" {
		t.Fatalf("expected report identity, got %#v", report)
	}
	if report.Included != 3 {
		t.Fatalf("expected entry cap to stop after 3 included entries, got %#v", report)
	}
	if report.Ignored == 0 || len(report.IgnoredSamples) == 0 || !strings.Contains(strings.Join(report.IgnoredSamples, "|"), "node_modules") {
		t.Fatalf("expected ignored samples, got %#v", report)
	}
	if report.DepthSkipped == 0 || report.EntrySkipped == 0 || !report.Truncated {
		t.Fatalf("expected depth and entry-cap truncation, got %#v", report)
	}
}

func TestScanReportHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New().ScanReport(ctx, t.TempDir(), ScanReportOptions{}); err == nil {
		t.Fatal("expected canceled scan report to fail")
	}
}

func mustWriteScanFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()
	absPath := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}
