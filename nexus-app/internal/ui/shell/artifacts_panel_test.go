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
			name:     "chat answer with metadata sidecar",
			artifact: artifactsSvc.Artifact{Kind: "chat-answer", RelPath: ".nexusdesk/artifacts/chat-answers/answer.md", MetadataPath: ".nexusdesk/artifacts/chat-answers/answer.md.json"},
			want:     true,
		},
		{
			name:     "chat answer without metadata sidecar",
			artifact: artifactsSvc.Artifact{Kind: "chat-answer", RelPath: ".nexusdesk/artifacts/chat-answers/answer.md"},
			want:     false,
		},
		{
			name:     "document brief with source artifact",
			artifact: artifactsSvc.Artifact{Kind: "document-brief", SourcePaths: []string{".nexusdesk/artifacts/document-sets/report.md"}},
			want:     true,
		},
		{
			name:     "document brief without source artifact",
			artifact: artifactsSvc.Artifact{Kind: "document-brief", SourcePaths: []string{"docs/report.md"}},
			want:     false,
		},
		{
			name:     "document export with source brief",
			artifact: artifactsSvc.Artifact{Kind: "document-export", SourcePaths: []string{".nexusdesk/artifacts/document-briefs/brief.md"}},
			want:     true,
		},
		{
			name:     "document export without source brief",
			artifact: artifactsSvc.Artifact{Kind: "document-export", SourcePaths: []string{".nexusdesk/artifacts/document-sets/report.md"}},
			want:     false,
		},
		{
			name:     "presentation outline with source artifact",
			artifact: artifactsSvc.Artifact{Kind: "presentation-outline", SourcePaths: []string{".nexusdesk/artifacts/document-sets/report.md"}},
			want:     true,
		},
		{
			name:     "presentation package with source outline",
			artifact: artifactsSvc.Artifact{Kind: "presentation-package", SourcePaths: []string{".nexusdesk/artifacts/presentations/slides.md"}},
			want:     true,
		},
		{
			name:     "presentation deck with source outline",
			artifact: artifactsSvc.Artifact{Kind: "presentation-deck", SourcePaths: []string{".nexusdesk/artifacts/presentations/slides.md"}},
			want:     true,
		},
		{
			name:     "presentation outline without source artifact",
			artifact: artifactsSvc.Artifact{Kind: "presentation-outline", SourcePaths: []string{"docs/report.md"}},
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

func TestArtifactCanGenerateDocumentBriefForReportArtifacts(t *testing.T) {
	cases := []struct {
		name     string
		artifact artifactsSvc.Artifact
		want     bool
	}{
		{
			name:     "document report",
			artifact: artifactsSvc.Artifact{Kind: "document-report", RelPath: ".nexusdesk/artifacts/document-sets/report.md"},
			want:     true,
		},
		{
			name:     "chat answer",
			artifact: artifactsSvc.Artifact{Kind: "chat-answer", RelPath: ".nexusdesk/artifacts/chat-answers/answer.md"},
			want:     true,
		},
		{
			name:     "presentation outline",
			artifact: artifactsSvc.Artifact{Kind: "presentation-outline", RelPath: ".nexusdesk/artifacts/presentations/slides.md"},
			want:     true,
		},
		{
			name:     "document brief does not brief itself",
			artifact: artifactsSvc.Artifact{Kind: "document-brief", RelPath: ".nexusdesk/artifacts/document-briefs/brief.md"},
			want:     false,
		},
		{
			name:     "archived report",
			artifact: artifactsSvc.Artifact{Kind: "document-report", RelPath: ".nexusdesk/artifacts/document-sets/report.md", Archived: true},
			want:     false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := artifactCanGenerateDocumentBrief(tc.artifact); got != tc.want {
				t.Fatalf("artifactCanGenerateDocumentBrief() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestArtifactCanGenerateDocumentArtifactIncludesExports(t *testing.T) {
	if !artifactCanGenerateDocumentArtifact(artifactsSvc.Artifact{Kind: "document-report", RelPath: ".nexusdesk/artifacts/document-sets/report.md"}) {
		t.Fatal("document reports should generate document briefs")
	}
	if !artifactCanGenerateDocumentArtifact(artifactsSvc.Artifact{Kind: "document-brief", RelPath: ".nexusdesk/artifacts/document-briefs/brief.md"}) {
		t.Fatal("document briefs should generate DOCX document exports")
	}
	if artifactCanGenerateDocumentArtifact(artifactsSvc.Artifact{Kind: "document-export", RelPath: ".nexusdesk/artifacts/document-exports/export.docx"}) {
		t.Fatal("document exports should not recursively generate exports")
	}
}

func TestArtifactCanGeneratePresentationOutlineForReportArtifacts(t *testing.T) {
	cases := []struct {
		name     string
		artifact artifactsSvc.Artifact
		want     bool
	}{
		{
			name:     "document report",
			artifact: artifactsSvc.Artifact{Kind: "document-report", RelPath: ".nexusdesk/artifacts/document-sets/report.md"},
			want:     true,
		},
		{
			name:     "chat answer",
			artifact: artifactsSvc.Artifact{Kind: "chat-answer", RelPath: ".nexusdesk/artifacts/chat-answers/answer.md"},
			want:     true,
		},
		{
			name:     "presentation outline cannot outline itself",
			artifact: artifactsSvc.Artifact{Kind: "presentation-outline", RelPath: ".nexusdesk/artifacts/presentations/slides.md"},
			want:     false,
		},
		{
			name:     "dataset csv not report-like",
			artifact: artifactsSvc.Artifact{Kind: "dataset-query-csv", RelPath: ".nexusdesk/artifacts/dataset-queries/query.csv"},
			want:     false,
		},
		{
			name:     "archived report",
			artifact: artifactsSvc.Artifact{Kind: "document-report", RelPath: ".nexusdesk/artifacts/document-sets/report.md", Archived: true},
			want:     false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := artifactCanGeneratePresentationOutline(tc.artifact); got != tc.want {
				t.Fatalf("artifactCanGeneratePresentationOutline() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestArtifactCanGeneratePresentationArtifactIncludesPackages(t *testing.T) {
	if !artifactCanGeneratePresentationArtifact(artifactsSvc.Artifact{Kind: "document-report", RelPath: ".nexusdesk/artifacts/document-sets/report.md"}) {
		t.Fatal("document reports should generate presentation outlines")
	}
	if !artifactCanGeneratePresentationArtifact(artifactsSvc.Artifact{Kind: "presentation-outline", RelPath: ".nexusdesk/artifacts/presentations/slides.md"}) {
		t.Fatal("presentation outlines should generate presentation packages")
	}
	if !artifactCanGeneratePresentationArtifact(artifactsSvc.Artifact{Kind: "presentation-package", RelPath: ".nexusdesk/artifacts/presentation-packages/deck.zip"}) {
		t.Fatal("presentation packages should generate PPTX decks")
	}
	if artifactCanGeneratePresentationArtifact(artifactsSvc.Artifact{Kind: "presentation-deck", RelPath: ".nexusdesk/artifacts/presentation-decks/deck.pptx"}) {
		t.Fatal("presentation decks should not recursively generate decks")
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

func TestBuildDocumentBriefArtifactUsesSourceArtifactMetadata(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	source, err := store.WriteDocumentSetReport(artifactsSvc.DocumentSetReport{
		Title:       "Architecture Notes",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md"},
		Content:     "## Goals\n\n- Keep shell native\n- Missing packaging smoke remains a blocker\n- Next action: verify release diagnostics\n",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport() error = %v", err)
	}
	created, err := buildDocumentBriefArtifact(context.Background(), root, source)
	if err != nil {
		t.Fatalf("buildDocumentBriefArtifact() error = %v", err)
	}
	if created.Kind != "document-brief" || len(created.SourcePaths) != 2 || created.SourcePaths[0] != source.RelPath {
		t.Fatalf("unexpected document brief artifact: %#v", created)
	}
	text, err := store.ReadArtifactText(created.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText() error = %v", err)
	}
	for _, expected := range []string{"# Document Brief - Architecture Notes", "Source artifact:** " + source.RelPath, "### Executive Summary", "Keep shell native", "### Risks And Gaps", "blocker"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("document brief missing %q:\n%s", expected, text)
		}
	}
}

func TestBuildDocumentBriefRefreshArtifactRegeneratesFromSource(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	source, err := store.WriteDocumentSetReport(artifactsSvc.DocumentSetReport{
		Title:       "Architecture Notes",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md"},
		Content:     "## Goals\n\n- Keep shell native\n",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport() error = %v", err)
	}
	brief, err := buildDocumentBriefArtifact(context.Background(), root, source)
	if err != nil {
		t.Fatalf("buildDocumentBriefArtifact() error = %v", err)
	}
	rebuilt, err := buildDocumentBriefRefreshArtifact(context.Background(), root, brief)
	if err != nil {
		t.Fatalf("buildDocumentBriefRefreshArtifact() error = %v", err)
	}
	if rebuilt.Kind != "document-brief" || rebuilt.RelPath == brief.RelPath {
		t.Fatalf("unexpected rebuilt document brief: %#v", rebuilt)
	}
	text, err := store.ReadArtifactText(rebuilt.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText() error = %v", err)
	}
	if !strings.Contains(text, source.RelPath) || !strings.Contains(text, "Keep shell native") {
		t.Fatalf("rebuilt brief lost source linkage/content:\n%s", text)
	}
}

func TestBuildDocumentExportArtifactUsesBriefMetadata(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	brief, err := store.WriteDocumentBriefReport(artifactsSvc.DocumentBriefReport{
		Title:       "Document Brief - Architecture Notes",
		SourcePath:  ".nexusdesk/artifacts/document-sets/report.md",
		SourceKind:  "document-report",
		SourcePaths: []string{"docs/a.md"},
		Content:     "### Executive Summary\n\n- Keep shell native.\n\n### Risks And Gaps\n\n- Packaging smoke remains a blocker.\n",
	})
	if err != nil {
		t.Fatalf("WriteDocumentBriefReport() error = %v", err)
	}
	created, err := buildDocumentExportArtifact(context.Background(), root, brief)
	if err != nil {
		t.Fatalf("buildDocumentExportArtifact() error = %v", err)
	}
	if created.Kind != "document-export" || len(created.SourcePaths) != 3 || created.SourcePaths[0] != brief.RelPath {
		t.Fatalf("unexpected document export artifact: %#v", created)
	}
	metadata, err := store.ReadArtifactMetadata(created.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactMetadata() error = %v", err)
	}
	if metadata.ExportFormat != "docx" || len(metadata.PackageFiles) == 0 || metadata.Source != brief.RelPath {
		t.Fatalf("unexpected export metadata: %#v", metadata)
	}
	preview, err := artifactPreviewText(store, created)
	if err != nil {
		t.Fatalf("artifactPreviewText() error = %v", err)
	}
	if !strings.Contains(preview, "DOCX document export") || !strings.Contains(preview, "word/document.xml") {
		t.Fatalf("document export preview missing details:\n%s", preview)
	}
}

func TestBuildDocumentExportRefreshArtifactRegeneratesFromBrief(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	brief, err := store.WriteDocumentBriefReport(artifactsSvc.DocumentBriefReport{
		Title:       "Document Brief - Architecture Notes",
		SourcePath:  ".nexusdesk/artifacts/document-sets/report.md",
		SourceKind:  "document-report",
		SourcePaths: []string{"docs/a.md"},
		Content:     "### Executive Summary\n\n- Keep shell native.\n",
	})
	if err != nil {
		t.Fatalf("WriteDocumentBriefReport() error = %v", err)
	}
	exported, err := buildDocumentExportArtifact(context.Background(), root, brief)
	if err != nil {
		t.Fatalf("buildDocumentExportArtifact() error = %v", err)
	}
	rebuilt, err := buildDocumentExportRefreshArtifact(context.Background(), root, exported)
	if err != nil {
		t.Fatalf("buildDocumentExportRefreshArtifact() error = %v", err)
	}
	if rebuilt.Kind != "document-export" || rebuilt.RelPath == exported.RelPath {
		t.Fatalf("unexpected rebuilt document export: %#v", rebuilt)
	}
	if len(rebuilt.SourcePaths) == 0 || rebuilt.SourcePaths[0] != brief.RelPath {
		t.Fatalf("rebuilt export lost brief source: %#v", rebuilt.SourcePaths)
	}
}

func TestBuildPresentationOutlineArtifactUsesSourceArtifactMetadata(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	source, err := store.WriteDocumentSetReport(artifactsSvc.DocumentSetReport{
		Title:       "Architecture Notes",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md"},
		Content:     "## Goals\n\n- Keep shell native\n- Preserve lineage\n",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport() error = %v", err)
	}
	created, err := buildPresentationOutlineArtifact(context.Background(), root, source)
	if err != nil {
		t.Fatalf("buildPresentationOutlineArtifact() error = %v", err)
	}
	if created.Kind != "presentation-outline" || len(created.SourcePaths) != 2 || created.SourcePaths[0] != source.RelPath {
		t.Fatalf("unexpected presentation artifact: %#v", created)
	}
	text, err := store.ReadArtifactText(created.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText() error = %v", err)
	}
	for _, expected := range []string{"# Presentation Outline - Architecture Notes", "Source artifact:** " + source.RelPath, "### Slide 1: Architecture Notes", "Keep shell native"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("presentation outline missing %q:\n%s", expected, text)
		}
	}
}

func TestBuildPresentationOutlineRefreshArtifactRegeneratesFromSource(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	source, err := store.WriteDocumentSetReport(artifactsSvc.DocumentSetReport{
		Title:       "Architecture Notes",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md"},
		Content:     "## Goals\n\n- Keep shell native\n",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport() error = %v", err)
	}
	outline, err := buildPresentationOutlineArtifact(context.Background(), root, source)
	if err != nil {
		t.Fatalf("buildPresentationOutlineArtifact() error = %v", err)
	}
	rebuilt, err := buildPresentationOutlineRefreshArtifact(context.Background(), root, outline)
	if err != nil {
		t.Fatalf("buildPresentationOutlineRefreshArtifact() error = %v", err)
	}
	if rebuilt.Kind != "presentation-outline" || rebuilt.RelPath == outline.RelPath {
		t.Fatalf("unexpected rebuilt presentation outline: %#v", rebuilt)
	}
	text, err := store.ReadArtifactText(rebuilt.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText() error = %v", err)
	}
	if !strings.Contains(text, source.RelPath) || !strings.Contains(text, "Keep shell native") {
		t.Fatalf("rebuilt outline lost source linkage/content:\n%s", text)
	}
}

func TestBuildPresentationPackageArtifactUsesOutlineMetadata(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	source, err := store.WriteDocumentSetReport(artifactsSvc.DocumentSetReport{
		Title:       "Architecture Notes",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md"},
		Content:     "## Goals\n\n- Keep shell native\n",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport() error = %v", err)
	}
	outline, err := buildPresentationOutlineArtifact(context.Background(), root, source)
	if err != nil {
		t.Fatalf("buildPresentationOutlineArtifact() error = %v", err)
	}
	created, err := buildPresentationPackageArtifact(context.Background(), root, outline)
	if err != nil {
		t.Fatalf("buildPresentationPackageArtifact() error = %v", err)
	}
	if created.Kind != "presentation-package" || len(created.SourcePaths) != 3 || created.SourcePaths[0] != outline.RelPath {
		t.Fatalf("unexpected package artifact: %#v", created)
	}
	metadata, err := store.ReadArtifactMetadata(created.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactMetadata() error = %v", err)
	}
	if metadata.ExportFormat != "zip" || len(metadata.PackageFiles) == 0 || metadata.Source != outline.RelPath {
		t.Fatalf("unexpected package metadata: %#v", metadata)
	}
	preview, err := artifactPreviewText(store, created)
	if err != nil {
		t.Fatalf("artifactPreviewText() error = %v", err)
	}
	if !strings.Contains(preview, "Packaged presentation export") || !strings.Contains(preview, "slides.json") {
		t.Fatalf("package preview missing details:\n%s", preview)
	}
}

func TestBuildPresentationPackageRefreshArtifactRegeneratesFromOutline(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	outline, err := store.WritePresentationOutlineReport(artifactsSvc.PresentationOutlineReport{
		Title:       "Presentation Outline - Architecture Notes",
		SourcePath:  ".nexusdesk/artifacts/document-sets/report.md",
		SourceTitle: "Architecture Notes",
		SourceKind:  "document-report",
		SourcePaths: []string{"docs/a.md"},
		Content:     "### Slide 1: Goals\n\n- Keep shell native\n",
		SlideCount:  1,
	})
	if err != nil {
		t.Fatalf("WritePresentationOutlineReport() error = %v", err)
	}
	pkg, err := buildPresentationPackageArtifact(context.Background(), root, outline)
	if err != nil {
		t.Fatalf("buildPresentationPackageArtifact() error = %v", err)
	}
	rebuilt, err := buildPresentationPackageRefreshArtifact(context.Background(), root, pkg)
	if err != nil {
		t.Fatalf("buildPresentationPackageRefreshArtifact() error = %v", err)
	}
	if rebuilt.Kind != "presentation-package" || rebuilt.RelPath == pkg.RelPath {
		t.Fatalf("unexpected rebuilt package: %#v", rebuilt)
	}
	if len(rebuilt.SourcePaths) == 0 || rebuilt.SourcePaths[0] != outline.RelPath {
		t.Fatalf("rebuilt package lost outline source: %#v", rebuilt.SourcePaths)
	}
}

func TestBuildPresentationDeckFromPackageArtifactUsesOutlineMetadata(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	outline, err := store.WritePresentationOutlineReport(artifactsSvc.PresentationOutlineReport{
		Title:       "Presentation Outline - Architecture Notes",
		SourcePath:  ".nexusdesk/artifacts/document-sets/report.md",
		SourceTitle: "Architecture Notes",
		SourceKind:  "document-report",
		SourcePaths: []string{"docs/a.md"},
		Content:     "### Slide 1: Goals\n\n- Keep shell native\n",
		SlideCount:  1,
	})
	if err != nil {
		t.Fatalf("WritePresentationOutlineReport() error = %v", err)
	}
	pkg, err := buildPresentationPackageArtifact(context.Background(), root, outline)
	if err != nil {
		t.Fatalf("buildPresentationPackageArtifact() error = %v", err)
	}
	deck, err := buildPresentationDeckFromPackageArtifact(context.Background(), root, pkg)
	if err != nil {
		t.Fatalf("buildPresentationDeckFromPackageArtifact() error = %v", err)
	}
	if deck.Kind != "presentation-deck" || len(deck.SourcePaths) != 3 || deck.SourcePaths[0] != outline.RelPath {
		t.Fatalf("unexpected deck artifact: %#v", deck)
	}
	metadata, err := store.ReadArtifactMetadata(deck.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactMetadata() error = %v", err)
	}
	if metadata.ExportFormat != "pptx" || len(metadata.PackageFiles) == 0 || metadata.Source != outline.RelPath {
		t.Fatalf("unexpected deck metadata: %#v", metadata)
	}
	preview, err := artifactPreviewText(store, deck)
	if err != nil {
		t.Fatalf("artifactPreviewText() error = %v", err)
	}
	if !strings.Contains(preview, "PPTX presentation deck export") || !strings.Contains(preview, "ppt/slides/slide1.xml") {
		t.Fatalf("deck preview missing details:\n%s", preview)
	}
}

func TestBuildPresentationDeckRefreshArtifactRegeneratesFromOutline(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	outline, err := store.WritePresentationOutlineReport(artifactsSvc.PresentationOutlineReport{
		Title:       "Presentation Outline - Architecture Notes",
		SourcePath:  ".nexusdesk/artifacts/document-sets/report.md",
		SourceTitle: "Architecture Notes",
		SourceKind:  "document-report",
		SourcePaths: []string{"docs/a.md"},
		Content:     "### Slide 1: Goals\n\n- Keep shell native\n",
		SlideCount:  1,
	})
	if err != nil {
		t.Fatalf("WritePresentationOutlineReport() error = %v", err)
	}
	deck, err := buildPresentationDeckArtifact(context.Background(), root, outline)
	if err != nil {
		t.Fatalf("buildPresentationDeckArtifact() error = %v", err)
	}
	rebuilt, err := buildPresentationDeckRefreshArtifact(context.Background(), root, deck)
	if err != nil {
		t.Fatalf("buildPresentationDeckRefreshArtifact() error = %v", err)
	}
	if rebuilt.Kind != "presentation-deck" || rebuilt.RelPath == deck.RelPath {
		t.Fatalf("unexpected rebuilt deck: %#v", rebuilt)
	}
	if len(rebuilt.SourcePaths) == 0 || rebuilt.SourcePaths[0] != outline.RelPath {
		t.Fatalf("rebuilt deck lost outline source: %#v", rebuilt.SourcePaths)
	}
}

func TestBuildChatAnswerRefreshArtifactPreservesMetadata(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	original, err := store.WriteChatAnswer(artifactsSvc.ChatAnswerReport{
		Title:                  "Saved Assistant Answer",
		Prompt:                 "Summarize README",
		Content:                "Use the setup guide.\n\nIt has three steps.",
		Source:                 "Nexus assistant",
		ContextRelPath:         "README.md",
		Model:                  "model-a",
		SourcePaths:            []string{"README.md"},
		CitationRefs:           []string{"README.md:L12"},
		UnverifiedCitationRefs: []string{"outside.md:L3"},
		CitationSnippets:       []string{"README.md:L12 Third setup step."},
		CitedSourcePaths:       []string{"README.md"},
		UncitedSourcePaths:     []string{"docs/guide.md"},
		EvidenceQuality:        "line-cited",
		EvidenceSummary:        "line-cited (1 source(s), 1 line ref(s); 1 citation outside selected sources).",
	})
	if err != nil {
		t.Fatalf("WriteChatAnswer() error = %v", err)
	}

	rebuilt, err := buildChatAnswerRefreshArtifact(context.Background(), root, original)
	if err != nil {
		t.Fatalf("buildChatAnswerRefreshArtifact() error = %v", err)
	}
	if rebuilt.Kind != "chat-answer" || rebuilt.RelPath == "" {
		t.Fatalf("unexpected rebuilt artifact: %#v", rebuilt)
	}
	text, err := store.ReadArtifactText(rebuilt.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText() error = %v", err)
	}
	for _, expected := range []string{"# Saved Assistant Answer", "## Citations", "README.md:L12", "## Unverified Citations", "outside.md:L3", "## Citation Snippets", "Third setup step", "## Evidence", "## Source Coverage", "docs/guide.md", "## Prompt", "Summarize README", "## Answer", "Use the setup guide."} {
		if !strings.Contains(text, expected) {
			t.Fatalf("rebuilt chat answer missing %q:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "It has three steps.\n\n## Citations") {
		t.Fatalf("rebuilt answer folded old metadata sections into answer content:\n%s", text)
	}
	metadata, err := store.ReadArtifactMetadata(rebuilt.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactMetadata() error = %v", err)
	}
	if metadata.Prompt != "Summarize README" || metadata.Model != "model-a" || metadata.ContextRelPath != "README.md" || metadata.EvidenceQuality != "line-cited" {
		t.Fatalf("rebuilt metadata lost prompt/model/context/evidence fields: %#v", metadata)
	}
	if len(metadata.SourcePaths) != 1 || metadata.SourcePaths[0] != "README.md" {
		t.Fatalf("rebuilt metadata lost source paths: %#v", metadata.SourcePaths)
	}
	if len(metadata.CitationRefs) != 1 || metadata.CitationRefs[0] != "README.md:L12" {
		t.Fatalf("rebuilt metadata lost citation refs: %#v", metadata.CitationRefs)
	}
	if len(metadata.UnverifiedCitationRefs) != 1 || metadata.UnverifiedCitationRefs[0] != "outside.md:L3" {
		t.Fatalf("rebuilt metadata lost unverified citation refs: %#v", metadata.UnverifiedCitationRefs)
	}
	if len(metadata.CitationSnippets) != 1 || !strings.Contains(metadata.CitationSnippets[0], "Third setup step") {
		t.Fatalf("rebuilt metadata lost citation snippets: %#v", metadata.CitationSnippets)
	}
	if len(metadata.CitedSourcePaths) != 1 || metadata.CitedSourcePaths[0] != "README.md" {
		t.Fatalf("rebuilt metadata lost cited source coverage: %#v", metadata.CitedSourcePaths)
	}
	if len(metadata.UncitedSourcePaths) != 1 || metadata.UncitedSourcePaths[0] != "docs/guide.md" {
		t.Fatalf("rebuilt metadata lost uncited source coverage: %#v", metadata.UncitedSourcePaths)
	}
}

func TestBuildChatAnswerRefreshArtifactHonorsCancelledContext(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	artifact, err := store.WriteChatAnswer(artifactsSvc.ChatAnswerReport{Prompt: "Q", Content: "A"})
	if err != nil {
		t.Fatalf("WriteChatAnswer() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := buildChatAnswerRefreshArtifact(ctx, root, artifact); err == nil {
		t.Fatal("expected cancelled context to stop chat-answer regeneration")
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
