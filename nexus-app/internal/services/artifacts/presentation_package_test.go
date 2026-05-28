package artifacts

import (
	"archive/zip"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestBuildPresentationPackageReportUsesSlideOutlineSection(t *testing.T) {
	report := BuildPresentationPackageReport(
		"",
		".nexusdesk/artifacts/presentations/slides.md",
		"Presentation Outline - Architecture Notes",
		"presentation-outline",
		"# Presentation Outline - Architecture Notes\n\n- **Generated:** today\n\n## Slide Outline\n\n### Slide 1: Goals\n\n- Keep shell native\n",
		[]string{".nexusdesk/artifacts/document-sets/report.md", "docs/a.md"},
	)
	if report.Title != "Presentation Package - Architecture Notes" || report.SlideCount != 1 {
		t.Fatalf("unexpected package report: %#v", report)
	}
	if strings.Contains(report.Outline, "Generated:**") || !strings.Contains(report.Outline, "### Slide 1: Goals") {
		t.Fatalf("expected only slide outline section, got:\n%s", report.Outline)
	}
}

func TestWritePresentationPackageReportCreatesZipAndMetadata(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.WritePresentationPackageReport(PresentationPackageReport{
		Title:       "Presentation Package - Architecture Notes",
		SourcePath:  ".nexusdesk/artifacts/presentations/slides.md",
		SourceTitle: "Presentation Outline - Architecture Notes",
		SourceKind:  "presentation-outline",
		SourcePaths: []string{".nexusdesk/artifacts/document-sets/report.md", "docs/a.md"},
		Outline:     "### Slide 1: Goals\n\n- Keep shell native\n",
		SlideCount:  1,
		GeneratedBy: "test",
	})
	if err != nil {
		t.Fatalf("WritePresentationPackageReport returned error: %v", err)
	}
	if artifact.Kind != "presentation-package" || artifact.MetadataPath == "" || artifact.Size == 0 {
		t.Fatalf("expected metadata-backed package artifact, got %#v", artifact)
	}
	if !strings.HasPrefix(artifact.RelPath, ".nexusdesk/artifacts/presentation-packages/") || !strings.HasSuffix(artifact.RelPath, ".zip") {
		t.Fatalf("unexpected package path: %q", artifact.RelPath)
	}
	metadata, err := store.ReadArtifactMetadata(artifact.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactMetadata returned error: %v", err)
	}
	if metadata.Kind != "presentation-package" || metadata.ExportFormat != "zip" || len(metadata.PackageFiles) != 5 {
		t.Fatalf("unexpected package metadata: %#v", metadata)
	}
	if len(metadata.SourcePaths) != 3 || metadata.SourcePaths[0] != ".nexusdesk/artifacts/presentations/slides.md" {
		t.Fatalf("expected source outline first in metadata paths, got %#v", metadata.SourcePaths)
	}
	reader, err := zip.OpenReader(artifact.AbsPath)
	if err != nil {
		t.Fatalf("OpenReader returned error: %v", err)
	}
	defer reader.Close()
	files := map[string]string{}
	for _, file := range reader.File {
		opened, err := file.Open()
		if err != nil {
			t.Fatalf("open zip member %s: %v", file.Name, err)
		}
		data, err := io.ReadAll(opened)
		if err != nil {
			_ = opened.Close()
			t.Fatalf("read zip member %s: %v", file.Name, err)
		}
		_ = opened.Close()
		files[file.Name] = string(data)
	}
	for _, expected := range []string{"manifest.json", "outline.md", "slides.json", "slides.md", "README.md"} {
		if _, ok := files[expected]; !ok {
			t.Fatalf("package missing %s; files=%v", expected, files)
		}
	}
	var manifest presentationPackageManifest
	if err := json.Unmarshal([]byte(files["manifest.json"]), &manifest); err != nil {
		t.Fatalf("manifest JSON invalid: %v\n%s", err, files["manifest.json"])
	}
	if manifest.Format != presentationPackageFormat || manifest.SlideCount != 1 || manifest.GeneratedBy != "test" {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	if !strings.Contains(files["slides.md"], "### Slide 1: Goals") || !strings.Contains(files["README.md"], "Source artifact") {
		t.Fatalf("package content lost slide/readme details: %#v", files)
	}
}

func TestWritePresentationPackageReportRequiresOutline(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.WritePresentationPackageReport(PresentationPackageReport{Title: "Empty"}); err == nil {
		t.Fatal("expected empty presentation package to be rejected")
	}
}
