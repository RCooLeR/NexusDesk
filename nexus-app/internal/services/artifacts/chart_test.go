package artifacts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteChartArtifactCreatesSVGAndMetadata(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "data.csv"), []byte("channel,spend\nsearch,12\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	artifact, err := store.WriteChartArtifact(ChartArtifactReport{
		SourcePath:     "data.csv",
		Query:          "channel=search",
		Format:         "CSV",
		Mode:           "sum",
		CategoryColumn: "channel",
		ValueColumn:    "spend",
		PointCount:     1,
		SVG:            `<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
	})
	if err != nil {
		t.Fatalf("WriteChartArtifact() error = %v", err)
	}
	if artifact.Kind != "chart" || !strings.HasSuffix(artifact.RelPath, ".svg") || artifact.MetadataPath == "" {
		t.Fatalf("unexpected chart artifact: %#v", artifact)
	}
	text, err := store.ReadArtifactText(artifact.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText() error = %v", err)
	}
	if !strings.Contains(text, "<svg") {
		t.Fatalf("unexpected SVG text: %s", text)
	}
	matches, err := store.ListArtifacts(ListOptions{Query: "kind:chart"})
	if err != nil {
		t.Fatalf("ListArtifacts() error = %v", err)
	}
	if len(matches) != 1 || matches[0].RelPath != artifact.RelPath {
		t.Fatalf("expected searchable chart artifact, got %#v", matches)
	}
}

func TestWriteChartArtifactCreatesDashboardKindForDashboardMode(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "data.csv"), []byte("channel,spend\nsearch,12\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	artifact, err := store.WriteChartArtifact(ChartArtifactReport{
		SourcePath:     "data.csv",
		Format:         "CSV",
		Mode:           "dashboard",
		CategoryColumn: "channel",
		ValueColumn:    "spend",
		PointCount:     1,
		SVG:            `<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
	})
	if err != nil {
		t.Fatalf("WriteChartArtifact() error = %v", err)
	}
	if artifact.Kind != "dashboard" || !strings.Contains(artifact.RelPath, "/dashboards/") {
		t.Fatalf("unexpected dashboard artifact: %#v", artifact)
	}
	matches, err := store.ListArtifacts(ListOptions{Query: "kind:dashboard"})
	if err != nil {
		t.Fatalf("ListArtifacts() error = %v", err)
	}
	if len(matches) != 1 || matches[0].RelPath != artifact.RelPath {
		t.Fatalf("expected searchable dashboard artifact, got %#v", matches)
	}
}
