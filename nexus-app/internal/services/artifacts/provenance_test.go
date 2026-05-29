package artifacts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInspectProvenanceAcceptsMetadataBackedArtifacts(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.WriteChartArtifact(ChartArtifactReport{
		SourcePath:     "data/sales.csv",
		Query:          "channel=search",
		Mode:           "sum",
		CategoryColumn: "channel",
		ValueColumn:    "spend",
		SVG:            "<svg></svg>",
		PointCount:     1,
	}); err != nil {
		t.Fatalf("WriteChartArtifact returned error: %v", err)
	}

	summary, err := store.InspectProvenance(ListOptions{})
	if err != nil {
		t.Fatalf("InspectProvenance returned error: %v", err)
	}
	if summary.Status() != ProvenanceStatusOK || summary.ArtifactCount != 1 || summary.WithLineage != 1 {
		t.Fatalf("expected healthy provenance summary, got %#v", summary)
	}
	text := FormatProvenanceSummary(summary, 8)
	if !strings.Contains(text, "1 artifact(s) have readable metadata") {
		t.Fatalf("unexpected provenance summary:\n%s", text)
	}
}

func TestInspectProvenanceReportsMissingMetadataSidecar(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	relPath := store.relPath("manual", "orphan.md")
	absPath := store.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte("# orphan\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	summary, err := store.InspectProvenance(ListOptions{})
	if err != nil {
		t.Fatalf("InspectProvenance returned error: %v", err)
	}
	if summary.Status() != ProvenanceStatusWarning || summary.MissingMetadata != 1 || len(summary.Issues) != 1 {
		t.Fatalf("expected missing metadata warning, got %#v", summary)
	}
	if !strings.Contains(FormatProvenanceSummary(summary, 8), "metadata sidecar is missing or unreadable") {
		t.Fatalf("expected missing sidecar text, got:\n%s", FormatProvenanceSummary(summary, 8))
	}
}

func TestInspectProvenanceReportsMissingLineageSignals(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	relPath := store.relPath("manual", "weak.md")
	absPath := store.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte("# weak\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := store.writeMetadata(Metadata{
		Kind:        "manual-report",
		Title:       "Weak report",
		RelPath:     relPath,
		GeneratedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("writeMetadata returned error: %v", err)
	}

	summary, err := store.InspectProvenance(ListOptions{})
	if err != nil {
		t.Fatalf("InspectProvenance returned error: %v", err)
	}
	if summary.Status() != ProvenanceStatusWarning || summary.MissingLineage != 1 || len(summary.Issues) != 1 {
		t.Fatalf("expected missing lineage warning, got %#v", summary)
	}
	joined := FormatProvenanceSummary(summary, 8)
	if !strings.Contains(joined, "missing source, job, prompt, query, package, or tool-run lineage") {
		t.Fatalf("expected missing lineage text, got:\n%s", joined)
	}
}
