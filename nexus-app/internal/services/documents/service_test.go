package documents

import (
	"errors"
	"strings"
	"testing"

	"nexusdesk/internal/domain"
)

func TestExtractMarkdownDocument(t *testing.T) {
	service := New(fakePreviewer{preview: domain.FilePreview{
		RelPath:   "docs/readme.md",
		Kind:      domain.PreviewText,
		Text:      "# Project Brief\n\nSome **body** text.",
		Size:      32,
		MediaType: "text/markdown",
		Encoding:  "utf-8",
	}})

	document, err := service.Extract("C:/repo", "docs/readme.md")
	if err != nil {
		t.Fatal(err)
	}
	if document.Title != "Project Brief" || document.Format != "markdown" || document.Words == 0 {
		t.Fatalf("unexpected document extraction: %#v", document)
	}
}

func TestExtractHTMLStripsTagsAndScripts(t *testing.T) {
	service := New(fakePreviewer{preview: domain.FilePreview{
		RelPath: "site/index.html",
		Kind:    domain.PreviewText,
		Text:    "<html><head><title>Hello &amp; welcome</title><script>bad()</script></head><body><h1>Hello</h1><p>Visible<br>Text</p></body></html>",
	}})

	document, err := service.Extract("C:/repo", "site/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if document.Title != "Hello & welcome" {
		t.Fatalf("unexpected title: %q", document.Title)
	}
	if strings.Contains(document.Text, "bad") || strings.Contains(document.Text, "<p>") || !strings.Contains(document.Text, "Visible\nText") {
		t.Fatalf("unexpected HTML extraction: %q", document.Text)
	}
}

func TestExtractXMLStripsTags(t *testing.T) {
	service := New(fakePreviewer{preview: domain.FilePreview{
		RelPath: "feed.xml",
		Kind:    domain.PreviewText,
		Text:    `<?xml version="1.0"?><root><title>Feed</title><item>One</item></root>`,
	}})

	document, err := service.Extract("C:/repo", "feed.xml")
	if err != nil {
		t.Fatal(err)
	}
	if document.Format != "xml" || !strings.Contains(document.Text, "Feed One") {
		t.Fatalf("unexpected XML extraction: %#v", document)
	}
}

func TestExtractDOCXPreviewText(t *testing.T) {
	service := New(fakePreviewer{preview: domain.FilePreview{
		RelPath:   "docs/brief.docx",
		Kind:      domain.PreviewDoc,
		Text:      "Executive brief\n\nUse local-first workflows.",
		MediaType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Size:      2048,
		Document:  &domain.DocumentPreview{Text: "Executive brief\n\nUse local-first workflows.", Truncated: true},
	}})

	document, err := service.Extract("C:/repo", "docs/brief.docx")
	if err != nil {
		t.Fatal(err)
	}
	if document.Format != "docx" || document.Title != "Executive brief" || !document.Truncated {
		t.Fatalf("unexpected DOCX extraction: %#v", document)
	}
}

func TestExtractPDFPreviewTextAndPages(t *testing.T) {
	service := New(fakePreviewer{preview: domain.FilePreview{
		RelPath:   "reports/brief.pdf",
		Kind:      domain.PreviewPDF,
		Text:      "Page one\n\nPage two",
		MediaType: "application/pdf",
		Size:      4096,
		PDF: &domain.PDFPreview{
			Text:  "Page one\n\nPage two",
			Pages: []domain.TextPage{{Page: 1, Text: "Page one"}, {Page: 2, Text: "Page two"}},
		},
	}})

	document, err := service.Extract("C:/repo", "reports/brief.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if document.Format != "pdf" || document.Pages != 2 || document.Title != "Page one" {
		t.Fatalf("unexpected PDF extraction: %#v", document)
	}
}

func TestExtractXLSXPreviewText(t *testing.T) {
	service := New(fakePreviewer{preview: domain.FilePreview{
		RelPath:   "data/campaigns.xlsx",
		Kind:      domain.PreviewTable,
		Text:      "Workbook sheets: Campaigns\n\n## Campaigns\nchannel\tspend\nsearch\t12.5",
		MediaType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Size:      2048,
		Table:     &domain.TablePreview{Sheet: "Campaigns", Headers: []string{"channel", "spend"}, Rows: [][]string{{"search", "12.5"}}},
	}})

	document, err := service.Extract("C:/repo", "data/campaigns.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if document.Format != "xlsx" || document.Title != "Workbook sheets: Campaigns" || !strings.Contains(document.Text, "search") {
		t.Fatalf("unexpected XLSX extraction: %#v", document)
	}
}

func TestExtractRejectsPDFWithoutText(t *testing.T) {
	service := New(fakePreviewer{preview: domain.FilePreview{
		RelPath: "scanned.pdf",
		Kind:    domain.PreviewPDF,
		PDF:     &domain.PDFPreview{},
	}})
	if _, err := service.Extract("C:/repo", "scanned.pdf"); err == nil || !strings.Contains(err.Error(), "extractable text") {
		t.Fatalf("expected scanned PDF extraction to be rejected, got %v", err)
	}
}

func TestExtractRejectsUnsupportedAndNonText(t *testing.T) {
	service := New(fakePreviewer{preview: domain.FilePreview{RelPath: "image.png", Kind: domain.PreviewImage}})
	if _, err := service.Extract("C:/repo", "image.png"); err == nil {
		t.Fatal("expected unsupported image extraction to be rejected")
	}
}

func TestExtractPropagatesPreviewError(t *testing.T) {
	service := New(fakePreviewer{err: errors.New("preview failed")})
	if _, err := service.Extract("C:/repo", "missing.txt"); err == nil || !strings.Contains(err.Error(), "preview failed") {
		t.Fatalf("expected preview error, got %v", err)
	}
}

type fakePreviewer struct {
	preview domain.FilePreview
	err     error
}

func (p fakePreviewer) PreviewFile(_ string, _ string) (domain.FilePreview, error) {
	return p.preview, p.err
}
