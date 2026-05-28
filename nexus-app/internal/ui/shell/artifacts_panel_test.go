package shell

import (
	"context"
	"strings"
	"testing"
	"time"

	artifactsSvc "nexusdesk/internal/services/artifacts"
	documentsSvc "nexusdesk/internal/services/documents"
	workspaceSvc "nexusdesk/internal/services/workspace"
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
	if got := documentSetArtifactTitleForRoots([]string{"docs", "README.md"}); got != "Document Set Report - 2 sources" {
		t.Fatalf("unexpected multi-root document title: %q", got)
	}
}

func TestDocumentArtifactJobLabels(t *testing.T) {
	if got := documentSetArtifactJobLabel(""); got != "Document report (project)" {
		t.Fatalf("unexpected empty document report job label: %q", got)
	}
	if got := documentSetArtifactJobLabel("docs"); got != "Document report (docs)" {
		t.Fatalf("unexpected document report job label: %q", got)
	}
	if got := documentExtractionArtifactJobLabel(""); got != "Document extraction" {
		t.Fatalf("unexpected empty extraction job label: %q", got)
	}
	if got := documentExtractionArtifactJobLabel("docs/a.md"); got != "Document extraction (docs/a.md)" {
		t.Fatalf("unexpected extraction job label: %q", got)
	}
	if got := workspaceScanReportJobLabel(""); got != "Workspace scan report" {
		t.Fatalf("unexpected empty scan report job label: %q", got)
	}
	if got := workspaceScanReportJobLabel("repo"); got != "Workspace scan report (repo)" {
		t.Fatalf("unexpected scan report job label: %q", got)
	}
}

func TestWorkspaceScanArtifactInputMapsReportFields(t *testing.T) {
	input := workspaceScanArtifactInput(workspaceSvc.ScanReport{
		Name:           "repo",
		Included:       10,
		Ignored:        2,
		DepthSkipped:   1,
		EntrySkipped:   3,
		Unreadable:     1,
		MaxDepth:       12,
		MaxEntries:     5000,
		Truncated:      true,
		IgnoredSamples: []string{"ignored: node_modules"},
		SkippedSamples: []string{"entry cap: vendor"},
	})
	if input.WorkspaceName != "repo" || input.Included != 10 || input.EntrySkipped != 3 || !input.Truncated {
		t.Fatalf("unexpected scan artifact input: %#v", input)
	}
	if len(input.IgnoredSamples) != 1 || len(input.SkippedSamples) != 1 || !strings.Contains(input.Message, "Scanned 10") {
		t.Fatalf("expected scan samples and message, got %#v", input)
	}
}

func TestDocumentExtractionArtifactInputMapsDocumentFields(t *testing.T) {
	input := documentExtractionArtifactInput(documentsSvc.ExtractedDocument{
		Title:     "Guide",
		RelPath:   "docs/guide.md",
		Format:    "markdown",
		MediaType: "text/markdown",
		Encoding:  "utf-8",
		Text:      "content",
		Size:      42,
		Lines:     2,
		Words:     1,
		Pages:     3,
		Truncated: true,
	})
	if input.Title != "Guide" || input.RelPath != "docs/guide.md" || input.Content != "content" || input.Pages != 3 || !input.Truncated {
		t.Fatalf("unexpected document extraction artifact input: %#v", input)
	}
}

func TestArtifactMetadataRecordMapsArtifactFields(t *testing.T) {
	generated := time.Date(2026, 5, 27, 12, 30, 0, 0, time.UTC)
	record := artifactMetadataRecord(artifactsSvc.Artifact{
		Kind:         "document-report",
		Title:        "Report",
		RelPath:      ".nexusdesk/artifacts/document-sets/report.md",
		MetadataPath: ".nexusdesk/artifacts/document-sets/report.md.json",
		Size:         42,
		JobID:        "job-1",
		TaskID:       "task-1",
		Source:       "docs",
		SourcePaths:  []string{"docs/a.md"},
		Archived:     true,
		CreatedAt:    generated,
		GeneratedAt:  generated,
	})
	if record.Kind != "document-report" || record.RelPath == "" || !record.Archived || len(record.SourcePaths) != 1 {
		t.Fatalf("unexpected metadata record: %#v", record)
	}
}

func TestArtifactCanRegenerateSupportedKinds(t *testing.T) {
	cases := []struct {
		name     string
		artifact artifactsSvc.Artifact
		want     bool
	}{
		{
			name:     "scan report",
			artifact: artifactsSvc.Artifact{Kind: "scan-report"},
			want:     true,
		},
		{
			name:     "document report with sources",
			artifact: artifactsSvc.Artifact{Kind: "document-report", SourcePaths: []string{"docs/a.md"}},
			want:     true,
		},
		{
			name:     "document report with project root source",
			artifact: artifactsSvc.Artifact{Kind: "document-report", Source: "."},
			want:     true,
		},
		{
			name:     "document report without source",
			artifact: artifactsSvc.Artifact{Kind: "document-report"},
			want:     false,
		},
		{
			name:     "document extract with source",
			artifact: artifactsSvc.Artifact{Kind: "document-extract", SourcePaths: []string{"docs/a.md"}},
			want:     true,
		},
		{
			name:     "document extract without source",
			artifact: artifactsSvc.Artifact{Kind: "document-extract"},
			want:     false,
		},
		{
			name:     "operations runbook with source",
			artifact: artifactsSvc.Artifact{Kind: "operations-runbook", SourcePaths: []string{"compose.yml"}},
			want:     true,
		},
		{
			name:     "operations runbook without source",
			artifact: artifactsSvc.Artifact{Kind: "operations-runbook"},
			want:     false,
		},
		{
			name: "comparison with compared artifacts",
			artifact: artifactsSvc.Artifact{
				Kind:        "artifact-comparison",
				SourcePaths: []string{".nexusdesk/artifacts/document-sets/left.md", ".nexusdesk/artifacts/document-sets/right.md"},
			},
			want: true,
		},
		{
			name:     "comparison without compared artifacts",
			artifact: artifactsSvc.Artifact{Kind: "artifact-comparison", SourcePaths: []string{".nexusdesk/artifacts/document-sets/left.md"}},
			want:     false,
		},
		{
			name:     "archived scan report",
			artifact: artifactsSvc.Artifact{Kind: "scan-report", Archived: true},
			want:     false,
		},
		{
			name:     "dataset artifact remains data panel rebuild",
			artifact: artifactsSvc.Artifact{Kind: "dataset-summary", SourcePaths: []string{"data.csv"}},
			want:     false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := artifactCanRegenerate(tc.artifact); got != tc.want {
				t.Fatalf("artifactCanRegenerate() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestArtifactRegenerationSourceUsesSourcePathsBeforeSource(t *testing.T) {
	source, ok := artifactRegenerationSource(artifactsSvc.Artifact{
		Source:      "fallback.md",
		SourcePaths: []string{"docs/a.md"},
	})
	if !ok || source != "docs/a.md" {
		t.Fatalf("unexpected regeneration source: %q ok=%t", source, ok)
	}
	if _, ok := artifactRegenerationSource(artifactsSvc.Artifact{Source: "docs/a.md, docs/b.md"}); ok {
		t.Fatal("expected comma-separated source summary to be rejected")
	}
}

func TestArtifactRegenerationSourcesAllowDocumentReportRoots(t *testing.T) {
	sources, ok := artifactRegenerationSources(artifactsSvc.Artifact{
		SourcePaths: []string{"docs/a.md", "docs/a.md", "docs/b.md"},
		Source:      "docs",
	})
	if !ok || strings.Join(sources, ",") != "docs" {
		t.Fatalf("expected document report roots to take precedence: %#v ok=%t", sources, ok)
	}
	sources, ok = artifactRegenerationSources(artifactsSvc.Artifact{
		SourcePaths: []string{"docs/a.md", "docs/a.md", "docs/b.md"},
	})
	if !ok || strings.Join(sources, ",") != "docs/a.md,docs/b.md" {
		t.Fatalf("unexpected regeneration sources: %#v ok=%t", sources, ok)
	}
	sources, ok = artifactRegenerationSources(artifactsSvc.Artifact{Source: "docs, README.md"})
	if !ok || strings.Join(sources, ",") != "docs,README.md" {
		t.Fatalf("unexpected comma source fallback: %#v ok=%t", sources, ok)
	}
	sources, ok = artifactRegenerationSources(artifactsSvc.Artifact{Source: "."})
	if !ok || len(sources) != 1 || sources[0] != "." {
		t.Fatalf("expected project root source, got %#v ok=%t", sources, ok)
	}
}

func TestArtifactRegenerationPairUsesSourcePathsAndSourceFallback(t *testing.T) {
	left, right, ok := artifactRegenerationPair(artifactsSvc.Artifact{
		SourcePaths: []string{".nexusdesk/artifacts/a.md", ".nexusdesk/artifacts/b.md"},
	})
	if !ok || left != ".nexusdesk/artifacts/a.md" || right != ".nexusdesk/artifacts/b.md" {
		t.Fatalf("unexpected source path pair: left=%q right=%q ok=%t", left, right, ok)
	}
	left, right, ok = artifactRegenerationPair(artifactsSvc.Artifact{
		Source: ".nexusdesk/artifacts/a.md, .nexusdesk/artifacts/b.md",
	})
	if !ok || left != ".nexusdesk/artifacts/a.md" || right != ".nexusdesk/artifacts/b.md" {
		t.Fatalf("unexpected fallback pair: left=%q right=%q ok=%t", left, right, ok)
	}
	left, right, ok = artifactRegenerationPair(artifactsSvc.Artifact{
		SourcePaths: []string{".nexusdesk/artifacts/stale.md"},
		Source:      ".nexusdesk/artifacts/a.md, .nexusdesk/artifacts/b.md",
	})
	if !ok || left != ".nexusdesk/artifacts/a.md" || right != ".nexusdesk/artifacts/b.md" {
		t.Fatalf("unexpected incomplete-sourcepaths fallback pair: left=%q right=%q ok=%t", left, right, ok)
	}
	if _, _, ok := artifactRegenerationPair(artifactsSvc.Artifact{
		SourcePaths: []string{".nexusdesk/artifacts/a.md", ".nexusdesk/artifacts/a.md"},
	}); ok {
		t.Fatal("expected identical comparison paths to be rejected")
	}
}

func TestBuildArtifactComparisonReportRegeneratesFromSourceArtifacts(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	left, err := store.WriteDocumentSetReport(artifactsSvc.DocumentSetReport{
		Title:       "Left",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md"},
		Content:     "old",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport(left) error = %v", err)
	}
	right, err := store.WriteDocumentSetReport(artifactsSvc.DocumentSetReport{
		Title:       "Right",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md"},
		Content:     "new",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport(right) error = %v", err)
	}
	rebuilt, err := buildArtifactComparisonReport(context.Background(), root, left.RelPath, right.RelPath)
	if err != nil {
		t.Fatalf("buildArtifactComparisonReport() error = %v", err)
	}
	if rebuilt.Kind != "artifact-comparison" || len(rebuilt.SourcePaths) != 2 {
		t.Fatalf("unexpected rebuilt comparison artifact: %#v", rebuilt)
	}
	text, err := store.ReadArtifactText(rebuilt.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText() error = %v", err)
	}
	for _, expected := range []string{"# Artifact Comparison", left.RelPath, right.RelPath, "-old", "+new"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("comparison report missing %q:\n%s", expected, text)
		}
	}
}

func TestArtifactRegenerationJobLabelUsesTitle(t *testing.T) {
	if got := artifactRegenerationJobLabel(artifactsSvc.Artifact{Kind: "scan-report", Title: "Workspace Scan"}); got != "Regenerate artifact (Workspace Scan)" {
		t.Fatalf("unexpected regeneration label: %q", got)
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

func TestArtifactFreshnessTextIncludesChangedSources(t *testing.T) {
	text := artifactFreshnessText(artifactsSvc.SourceFreshness{
		Message: "Artifact may be stale: 1 changed source.",
		Sources: []artifactsSvc.SourceFreshnessStatus{
			{RelPath: "docs/a.md", Exists: true, Changed: true, Message: "Source changed after artifact generation."},
		},
	})
	for _, expected := range []string{"Source Freshness", "Artifact may be stale", "docs/a.md (changed"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("freshness text missing %q:\n%s", expected, text)
		}
	}
}

func TestArtifactSourceStatusTextSummarizesProblems(t *testing.T) {
	status := artifactSourceStatusText([]artifactsSvc.SourceFreshnessStatus{
		{RelPath: "docs/a.md", Exists: true, Changed: true},
		{RelPath: "docs/missing.md"},
	})
	if status != "Sources: 2 cited, 1 changed, 1 missing." {
		t.Fatalf("unexpected source status: %q", status)
	}
}

func TestArtifactSourceLabelPrioritizesUnknownOverMissing(t *testing.T) {
	label := artifactSourceLabel(artifactsSvc.SourceFreshnessStatus{
		RelPath: "docs/unsafe.md",
		Unknown: true,
		Message: "source path must stay inside the workspace",
	})
	if !strings.Contains(label, "(unchecked:") {
		t.Fatalf("expected unchecked source label, got %q", label)
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
