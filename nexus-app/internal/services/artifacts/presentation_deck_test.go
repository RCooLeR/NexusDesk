package artifacts

import (
	"archive/zip"
	"strings"
	"testing"
)

func TestWritePresentationDeckReportCreatesPPTXAndMetadata(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.WritePresentationDeckReport(PresentationDeckReport{
		Title:       "Presentation Deck - Architecture Notes",
		SourcePath:  ".nexusdesk/artifacts/presentations/slides.md",
		SourceTitle: "Presentation Outline - Architecture Notes",
		SourceKind:  "presentation-outline",
		SourcePaths: []string{".nexusdesk/artifacts/document-sets/report.md", "docs/a.md"},
		Outline:     "### Slide 1: Goals\n\n- Keep shell native\n\n### Slide 2: Risks\n\n- Packaging smoke remains\n",
		SlideCount:  2,
		GeneratedBy: "test",
	})
	if err != nil {
		t.Fatalf("WritePresentationDeckReport returned error: %v", err)
	}
	if artifact.Kind != "presentation-deck" || artifact.MetadataPath == "" || artifact.Size == 0 {
		t.Fatalf("expected metadata-backed deck artifact, got %#v", artifact)
	}
	if !strings.HasPrefix(artifact.RelPath, ".nexusdesk/artifacts/presentation-decks/") || !strings.HasSuffix(artifact.RelPath, ".pptx") {
		t.Fatalf("unexpected deck path: %q", artifact.RelPath)
	}
	metadata, err := store.ReadArtifactMetadata(artifact.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactMetadata returned error: %v", err)
	}
	if metadata.Kind != "presentation-deck" || metadata.ExportFormat != "pptx" || len(metadata.PackageFiles) == 0 {
		t.Fatalf("unexpected deck metadata: %#v", metadata)
	}
	if len(metadata.SourcePaths) != 3 || metadata.SourcePaths[0] != ".nexusdesk/artifacts/presentations/slides.md" {
		t.Fatalf("expected source outline first in metadata paths, got %#v", metadata.SourcePaths)
	}
	reader, err := zip.OpenReader(artifact.AbsPath)
	if err != nil {
		t.Fatalf("expected valid pptx zip: %v", err)
	}
	defer reader.Close()
	parts := map[string]string{}
	for _, file := range reader.File {
		parts[file.Name] = readZipText(t, reader.File, file.Name)
	}
	for _, expected := range []string{"[Content_Types].xml", "_rels/.rels", "ppt/presentation.xml", "ppt/slides/slide1.xml", "ppt/slides/slide2.xml"} {
		if _, ok := parts[expected]; !ok {
			t.Fatalf("deck missing %s; parts=%v", expected, parts)
		}
	}
	if !strings.Contains(parts["ppt/slides/slide1.xml"], "Keep shell native") || !strings.Contains(parts["ppt/slides/slide2.xml"], "Packaging smoke remains") {
		t.Fatalf("deck slides lost content: %#v", parts)
	}
}

func TestBuildPresentationDeckReportUsesSlideOutlineSection(t *testing.T) {
	report := BuildPresentationDeckReport("", ".nexusdesk/artifacts/presentations/slides.md", "Presentation Outline - Architecture Notes", "presentation-outline", "# Presentation Outline\n\n## Slide Outline\n\n### Slide 1: Goals\n\n- Keep shell native\n", []string{"docs/a.md"})
	if report.Title != "Presentation Deck - Architecture Notes" || report.SlideCount != 1 {
		t.Fatalf("unexpected deck report: %#v", report)
	}
	if strings.Contains(report.Outline, "Presentation Outline") || !strings.Contains(report.Outline, "### Slide 1: Goals") {
		t.Fatalf("expected only slide outline section, got:\n%s", report.Outline)
	}
}

func TestWritePresentationDeckReportRequiresOutline(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.WritePresentationDeckReport(PresentationDeckReport{Title: "Empty"}); err == nil {
		t.Fatal("expected empty presentation deck to be rejected")
	}
}
