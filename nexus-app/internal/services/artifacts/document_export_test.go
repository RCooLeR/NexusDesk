package artifacts

import (
	"archive/zip"
	"io"
	"strings"
	"testing"
)

func TestWriteDocumentExportReportCreatesDocxArtifact(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.WriteDocumentExportReport(DocumentExportReport{
		Title:       "Document Export - Architecture Notes",
		SourcePath:  ".nexusdesk/artifacts/document-briefs/brief.md",
		SourceTitle: "Document Brief - Architecture Notes",
		SourceKind:  "document-brief",
		SourcePaths: []string{"docs/a.md"},
		Content:     "### Executive Summary\n\n- Native parity is close.\n\n### Risks And Gaps\n\n- Packaging smoke remains a blocker.\n",
		GeneratedBy: "test",
	})
	if err != nil {
		t.Fatalf("WriteDocumentExportReport returned error: %v", err)
	}
	if artifact.Kind != "document-export" || artifact.MetadataPath == "" || artifact.Size == 0 {
		t.Fatalf("expected metadata-backed document export artifact, got %#v", artifact)
	}
	if !strings.HasPrefix(artifact.RelPath, ".nexusdesk/artifacts/document-exports/") || !strings.HasSuffix(artifact.RelPath, ".docx") {
		t.Fatalf("unexpected document export path: %q", artifact.RelPath)
	}
	metadata, err := store.ReadArtifactMetadata(artifact.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactMetadata returned error: %v", err)
	}
	if metadata.Kind != "document-export" || metadata.ExportFormat != "docx" || len(metadata.PackageFiles) == 0 || metadata.Source != ".nexusdesk/artifacts/document-briefs/brief.md" {
		t.Fatalf("unexpected document export metadata: %#v", metadata)
	}
	if metadata.PackageValidation == nil || !metadata.PackageValidation.Valid || metadata.PackageValidation.XMLFiles == 0 {
		t.Fatalf("expected valid document package validation metadata, got %#v", metadata.PackageValidation)
	}
	if metadata.ExportTemplate != officeExportTemplateName || metadata.ThemeName != officeExportThemeName {
		t.Fatalf("expected Office theme metadata, got template=%q theme=%q", metadata.ExportTemplate, metadata.ThemeName)
	}

	reader, err := zip.OpenReader(artifact.AbsPath)
	if err != nil {
		t.Fatalf("expected valid docx zip: %v", err)
	}
	defer reader.Close()
	documentXML := readZipText(t, reader.File, "word/document.xml")
	for _, expected := range []string{"Document Export - Architecture Notes", "Executive Summary", "- Native parity is close.", "Packaging smoke remains a blocker.", "Source artifact: .nexusdesk/artifacts/document-briefs/brief.md", officeExportThemeName} {
		if !strings.Contains(documentXML, expected) {
			t.Fatalf("document.xml missing %q:\n%s", expected, documentXML)
		}
	}
	stylesXML := readZipText(t, reader.File, "word/styles.xml")
	if readZipText(t, reader.File, "[Content_Types].xml") == "" || stylesXML == "" {
		t.Fatal("expected required docx package parts")
	}
	for _, expected := range []string{officeFontHeading, officeFontBody, officeColorAccent, officeColorAccentAlt} {
		if !strings.Contains(stylesXML, expected) {
			t.Fatalf("styles.xml missing themed value %q:\n%s", expected, stylesXML)
		}
	}
}

func TestBuildDocumentExportReportUsesBriefSection(t *testing.T) {
	report := BuildDocumentExportReport("", ".nexusdesk/artifacts/document-briefs/brief.md", "Document Brief - Architecture Notes", "document-brief", "# Wrapper\n\n## Brief\n\n### Executive Summary\n\n- Keep shell native.\n", []string{"docs/a.md"})
	if report.Title != "Document Export - Architecture Notes" || !strings.Contains(report.Content, "Keep shell native") {
		t.Fatalf("unexpected document export report: %#v", report)
	}
	if strings.Contains(report.Content, "# Wrapper") {
		t.Fatalf("document export should use brief body, got:\n%s", report.Content)
	}
}

func TestWriteDocumentExportReportRequiresContent(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.WriteDocumentExportReport(DocumentExportReport{Title: "Empty"}); err == nil {
		t.Fatal("expected empty document export to be rejected")
	}
}

func TestInferKindDetectsDocumentExports(t *testing.T) {
	if got := inferKind(".nexusdesk/artifacts/document-exports/export.docx"); got != "document-export" {
		t.Fatalf("unexpected document export kind: %q", got)
	}
}

func readZipText(t *testing.T, files []*zip.File, name string) string {
	t.Helper()
	for _, file := range files {
		if file.Name != name {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatalf("open zip part %s: %v", name, err)
		}
		defer reader.Close()
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("read zip part %s: %v", name, err)
		}
		return string(data)
	}
	t.Fatalf("zip part %s not found", name)
	return ""
}
