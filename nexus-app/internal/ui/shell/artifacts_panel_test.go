package shell

import (
	"strings"
	"testing"
	"time"

	artifactsSvc "nexusdesk/internal/services/artifacts"
)

func TestArtifactMetaFormatsTaskReport(t *testing.T) {
	meta := artifactMeta(artifactsSvc.Artifact{
		Kind:        "task-report",
		Size:        1234,
		GeneratedAt: time.Date(2026, 5, 27, 12, 30, 0, 0, time.UTC),
		JobID:       "job-1",
	})
	for _, expected := range []string{"task-report", "2026-05-27 12:30:00", "1234 bytes", "job job-1"} {
		if !strings.Contains(meta, expected) {
			t.Fatalf("artifact meta %q missing %q", meta, expected)
		}
	}
}

func TestArtifactTitleFallsBackToFilename(t *testing.T) {
	if got := artifactTitle(artifactsSvc.Artifact{RelPath: ".nexusdesk/artifacts/task-runs/report.md"}); got != "report.md" {
		t.Fatalf("unexpected fallback title: %q", got)
	}
	if got := artifactTitle(artifactsSvc.Artifact{Title: "Task report", RelPath: "ignored.md"}); got != "Task report" {
		t.Fatalf("unexpected explicit title: %q", got)
	}
}

func TestDocumentSetArtifactTitle(t *testing.T) {
	if got := documentSetArtifactTitle(""); got != "Project Document Set Report" {
		t.Fatalf("unexpected empty document title: %q", got)
	}
	if got := documentSetArtifactTitle("docs"); got != "Document Set Report - docs" {
		t.Fatalf("unexpected selected document title: %q", got)
	}
}

func TestArtifactLineageTextIncludesNodesAndEdges(t *testing.T) {
	text := artifactLineageText(artifactsSvc.Lineage{
		Nodes: []artifactsSvc.LineageNode{{Kind: "artifact", Label: "report.md"}},
		Edges: []artifactsSvc.LineageEdge{{From: "job:1", To: "report.md", Label: "generated"}},
	})
	for _, expected := range []string{"Lineage", "artifact: report.md", "job:1 --generated--> report.md"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("lineage text %q missing %q", text, expected)
		}
	}
}

func TestArtifactSourcePathsPreferMetadataSources(t *testing.T) {
	sources := artifactSourcePaths(
		artifactsSvc.Artifact{SourcePaths: []string{"docs/a.md", "docs/a.md", "docs/b.md"}},
		artifactsSvc.Lineage{Nodes: []artifactsSvc.LineageNode{{Kind: "source", Label: "fallback.md"}}},
	)
	if len(sources) != 2 || sources[0] != "docs/a.md" || sources[1] != "docs/b.md" {
		t.Fatalf("unexpected source paths: %#v", sources)
	}
}

func TestArtifactSourcePathsFallsBackToLineage(t *testing.T) {
	sources := artifactSourcePaths(
		artifactsSvc.Artifact{},
		artifactsSvc.Lineage{Nodes: []artifactsSvc.LineageNode{
			{Kind: "artifact", Label: "report.md"},
			{Kind: "source", Label: "docs/a.md"},
			{Kind: "source", Label: "docs/a.md"},
		}},
	)
	if len(sources) != 1 || sources[0] != "docs/a.md" {
		t.Fatalf("unexpected lineage source paths: %#v", sources)
	}
}

func TestFormatArtifactComparison(t *testing.T) {
	text := formatArtifactComparison(artifactsSvc.ArtifactComparison{
		Kind:      "document-report",
		LeftPath:  ".nexusdesk/artifacts/document-sets/a.md",
		RightPath: ".nexusdesk/artifacts/document-sets/b.md",
		Diff:      "--- a\n+++ b\n-old\n+new\n",
		Message:   "Compared a with b.",
	})
	for _, expected := range []string{"Artifact Comparison", "Kind: document-report", "Left: .nexusdesk", "Compared a with b.", "-old", "+new"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("comparison text missing %q:\n%s", expected, text)
		}
	}
}

func TestArtifactComparisonReadyRequiresPathsAndDiff(t *testing.T) {
	if artifactComparisonReady(artifactsSvc.ArtifactComparison{LeftPath: "a", RightPath: "b"}) {
		t.Fatal("comparison without diff should not be exportable")
	}
	if !artifactComparisonReady(artifactsSvc.ArtifactComparison{LeftPath: "a", RightPath: "b", Diff: "--- a\n+++ b\n"}) {
		t.Fatal("comparison with paths and diff should be exportable")
	}
}
