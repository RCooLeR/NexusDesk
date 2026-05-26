package workspace

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPDFPagesFindsLiteralTextPerPage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.pdf")
	content := "%PDF\n/Type /Pages\nBT (Pages container) Tj ET\n/Type /Page\nBT (First page) Tj ET\n/Type /Page\nBT (Second page) Tj ET\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	pages := extractPDFPages(path, 1024)
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %#v", pages)
	}
	if pages[0].Text != "First page" || pages[1].Text != "Second page" {
		t.Fatalf("unexpected pages: %#v", pages)
	}
}

func TestExtractDOCXTextReadsDocumentXML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.docx")
	createDOCXFixture(t, path, `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Hello DOCX</w:t></w:r></w:p></w:body></w:document>`)

	text, err := extractDOCXText(path, 1024)
	if err != nil {
		t.Fatalf("extractDOCXText returned error: %v", err)
	}
	if text != "Hello DOCX" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func createDOCXFixture(t *testing.T, path string, documentXML string) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	writer, err := archive.Create("word/document.xml")
	if err != nil {
		t.Fatalf("Create document.xml failed: %v", err)
	}
	if _, err := writer.Write([]byte(documentXML)); err != nil {
		t.Fatalf("Write document.xml failed: %v", err)
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("Close archive failed: %v", err)
	}
}
