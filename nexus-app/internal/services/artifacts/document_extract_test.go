package artifacts

import (
	"strings"
	"testing"
)

func TestWriteDocumentExtractionReportCreatesArtifact(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	artifact, err := store.WriteDocumentExtractionReport(DocumentExtractionReport{
		Title:     "Guide",
		RelPath:   "docs/guide.html",
		Format:    "html",
		MediaType: "text/html",
		Encoding:  "utf-8",
		Content:   "Readable guide text",
		Size:      128,
		Lines:     1,
		Words:     3,
		Pages:     2,
	})
	if err != nil {
		t.Fatalf("WriteDocumentExtractionReport returned error: %v", err)
	}
	if artifact.Kind != "document-extract" || artifact.MetadataPath == "" || len(artifact.SourcePaths) != 1 {
		t.Fatalf("unexpected artifact: %#v", artifact)
	}
	text, err := store.ReadArtifactText(artifact.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText returned error: %v", err)
	}
	for _, expected := range []string{"# Document Extraction - Guide", "Source:** docs/guide.html", "Pages:** 2", "Readable guide text"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("artifact text missing %q:\n%s", expected, text)
		}
	}
	matches, err := store.ListArtifacts(ListOptions{Query: "kind:document-extract"})
	if err != nil {
		t.Fatalf("ListArtifacts returned error: %v", err)
	}
	if len(matches) != 1 || matches[0].RelPath != artifact.RelPath {
		t.Fatalf("expected searchable document extract, got %#v", matches)
	}
}

func TestWriteDocumentExtractionReportRequiresSourceAndContent(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.WriteDocumentExtractionReport(DocumentExtractionReport{RelPath: "docs/a.md"}); err == nil {
		t.Fatal("expected empty extraction content to be rejected")
	}
	if _, err := store.WriteDocumentExtractionReport(DocumentExtractionReport{Content: "text"}); err == nil {
		t.Fatal("expected missing source path to be rejected")
	}
}
