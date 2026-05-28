package artifacts

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const documentExportFormat = "docx"

type documentExportPart struct {
	Style string
	Text  string
}

func BuildDocumentExportReport(title string, sourcePath string, sourceTitle string, sourceKind string, sourceText string, sourcePaths []string) DocumentExportReport {
	sourceTitle = strings.TrimSpace(sourceTitle)
	title = strings.TrimSpace(title)
	if title == "" {
		title = documentExportTitle(sourceTitle, sourcePath)
	}
	return DocumentExportReport{
		Title:       title,
		SourcePath:  filepath.ToSlash(strings.TrimSpace(sourcePath)),
		SourceTitle: sourceTitle,
		SourceKind:  strings.TrimSpace(sourceKind),
		SourcePaths: append([]string{}, sourcePaths...),
		Content:     documentExportContent(sourceText),
		GeneratedBy: "Nexus native DOCX writer",
	}
}

func (s *Store) WriteDocumentExportReport(report DocumentExportReport) (Artifact, error) {
	content := strings.TrimSpace(report.Content)
	if content == "" {
		return Artifact{}, errors.New("document export content is required")
	}
	createdAt := time.Now().UTC()
	title := strings.TrimSpace(report.Title)
	if title == "" {
		title = documentExportTitle(report.SourceTitle, report.SourcePath)
	}
	relPath := s.relPath("document-exports", fmt.Sprintf("%s-%s.docx", artifactTimestamp(createdAt), safeName(title)))
	absPath := s.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Artifact{}, err
	}
	sourcePaths := documentBriefSourcePaths(report.SourcePath, report.SourcePaths)
	files := []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"docProps/core.xml",
		"docProps/app.xml",
		"word/document.xml",
		"word/styles.xml",
	}
	file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return Artifact{}, err
	}
	zipWriter := zip.NewWriter(file)
	writeErr := writeDocumentExportDocx(zipWriter, report, title, content, sourcePaths, files, createdAt)
	closeZipErr := zipWriter.Close()
	closeFileErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(absPath)
		return Artifact{}, writeErr
	}
	if closeZipErr != nil {
		_ = os.Remove(absPath)
		return Artifact{}, closeZipErr
	}
	if closeFileErr != nil {
		_ = os.Remove(absPath)
		return Artifact{}, closeFileErr
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return Artifact{}, err
	}
	validation, err := ValidateOfficePackage(absPath, documentExportFormat, files)
	if err != nil {
		_ = os.Remove(absPath)
		return Artifact{}, err
	}
	metadata := Metadata{
		Kind:              "document-export",
		Title:             title,
		RelPath:           relPath,
		Source:            filepath.ToSlash(strings.TrimSpace(report.SourcePath)),
		SourcePaths:       sourcePaths,
		GeneratedAt:       createdAt,
		ExportFormat:      documentExportFormat,
		ExportTemplate:    officeExportTemplateName,
		ThemeName:         officeExportThemeName,
		PackageFiles:      append([]string{}, files...),
		PackageValidation: &validation,
	}
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:         "document-export",
		Title:        title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: relPath + ".json",
		Message:      "Document DOCX export artifact created at " + relPath + ".",
		Size:         info.Size(),
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		Source:       metadata.Source,
		SourcePaths:  append([]string{}, sourcePaths...),
	}, nil
}

func writeDocumentExportDocx(zipWriter *zip.Writer, report DocumentExportReport, title string, content string, sourcePaths []string, files []string, createdAt time.Time) error {
	if err := addZipText(zipWriter, "[Content_Types].xml", documentExportContentTypes()); err != nil {
		return err
	}
	if err := addZipText(zipWriter, "_rels/.rels", documentExportRootRels()); err != nil {
		return err
	}
	if err := addZipText(zipWriter, "docProps/core.xml", documentExportCoreProperties(report, title, createdAt)); err != nil {
		return err
	}
	if err := addZipText(zipWriter, "docProps/app.xml", documentExportAppProperties()); err != nil {
		return err
	}
	if err := addZipText(zipWriter, "word/styles.xml", documentExportStyles()); err != nil {
		return err
	}
	return addZipText(zipWriter, "word/document.xml", documentExportDocumentXML(report, title, content, sourcePaths, files, createdAt))
}

func documentExportContent(markdown string) string {
	markdown = strings.TrimSpace(strings.ReplaceAll(markdown, "\r\n", "\n"))
	if markdown == "" {
		return ""
	}
	marker := "\n## Brief"
	index := strings.Index(markdown, marker)
	if index != -1 {
		section := strings.TrimSpace(markdown[index+len(marker):])
		if section != "" {
			return section
		}
	}
	return markdown
}

func documentExportParts(markdown string) []documentExportPart {
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	parts := make([]documentExportPart, 0, len(lines))
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || line == "---" || strings.HasPrefix(line, "```") || strings.HasPrefix(line, "|") {
			continue
		}
		if strings.HasPrefix(line, "- **") && strings.Contains(line, ":**") {
			continue
		}
		if title, ok := markdownHeadingTitle(line); ok {
			style := "Heading1"
			if strings.HasPrefix(line, "### ") {
				style = "Heading3"
			} else if strings.HasPrefix(line, "## ") {
				style = "Heading2"
			}
			parts = append(parts, documentExportPart{Style: style, Text: title})
			continue
		}
		if bullet := presentationBulletText(line); bullet != "" {
			if bullet != line {
				bullet = "- " + bullet
			}
			parts = append(parts, documentExportPart{Text: bullet})
		}
	}
	return parts
}

func documentExportDocumentXML(report DocumentExportReport, title string, content string, sourcePaths []string, files []string, createdAt time.Time) string {
	parts := []documentExportPart{{Style: "Title", Text: title}}
	parts = append(parts, documentExportPart{Text: "Generated: " + formatArtifactTime(createdAt)})
	parts = append(parts, documentExportPart{Text: "Generated by: " + firstNonEmptyArtifact(report.GeneratedBy, "Nexus native DOCX writer")})
	parts = append(parts, documentExportPart{Text: "Template: " + officeExportTemplateName})
	parts = append(parts, documentExportPart{Text: "Theme: " + officeExportThemeName})
	if strings.TrimSpace(report.SourcePath) != "" {
		parts = append(parts, documentExportPart{Text: "Source artifact: " + filepath.ToSlash(strings.TrimSpace(report.SourcePath))})
	}
	if strings.TrimSpace(report.SourceKind) != "" {
		parts = append(parts, documentExportPart{Text: "Source kind: " + strings.TrimSpace(report.SourceKind)})
	}
	parts = append(parts, documentExportParts(content)...)
	if len(sourcePaths) > 0 {
		parts = append(parts, documentExportPart{Style: "Heading2", Text: "Sources"})
		for _, source := range sourcePaths {
			parts = append(parts, documentExportPart{Text: "- " + source})
		}
	}
	parts = append(parts, documentExportPart{Style: "Heading2", Text: "Package Metadata"})
	parts = append(parts, documentExportPart{Text: "Format: " + documentExportFormat})
	parts = append(parts, documentExportPart{Text: "Template: " + officeExportTemplateName})
	parts = append(parts, documentExportPart{Text: "Theme: " + officeExportThemeName})
	parts = append(parts, documentExportPart{Text: "Package files: " + strings.Join(files, ", ")})

	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	builder.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	for _, part := range parts {
		builder.WriteString(documentExportParagraph(part))
	}
	builder.WriteString(`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="720" w:footer="720" w:gutter="0"/></w:sectPr>`)
	builder.WriteString(`</w:body></w:document>`)
	return builder.String()
}

func documentExportParagraph(part documentExportPart) string {
	text := strings.TrimSpace(part.Text)
	if text == "" {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("<w:p>")
	if part.Style != "" {
		builder.WriteString(`<w:pPr><w:pStyle w:val="`)
		builder.WriteString(xmlEscape(part.Style))
		builder.WriteString(`"/></w:pPr>`)
	}
	builder.WriteString("<w:r><w:t>")
	builder.WriteString(xmlEscape(text))
	builder.WriteString("</w:t></w:r></w:p>")
	return builder.String()
}

func documentExportContentTypes() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>
`
}

func documentExportRootRels() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>
`
}

func documentExportCoreProperties(report DocumentExportReport, title string, createdAt time.Time) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <dc:title>` + xmlEscape(title) + `</dc:title>
  <dc:creator>` + xmlEscape(firstNonEmptyArtifact(report.GeneratedBy, "Nexus native DOCX writer")) + `</dc:creator>
  <cp:lastModifiedBy>NexusDesk</cp:lastModifiedBy>
  <dcterms:created xsi:type="dcterms:W3CDTF">` + createdAt.Format(time.RFC3339) + `</dcterms:created>
  <dcterms:modified xsi:type="dcterms:W3CDTF">` + createdAt.Format(time.RFC3339) + `</dcterms:modified>
</cp:coreProperties>
`
}

func documentExportAppProperties() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes">
  <Application>NexusDesk</Application>
</Properties>
`
}

func documentExportStyles() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="` + officeFontBody + `" w:hAnsi="` + officeFontBody + `"/><w:color w:val="` + officeColorInk + `"/><w:sz w:val="22"/></w:rPr></w:rPrDefault></w:docDefaults>
  <w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:rPr><w:rFonts w:ascii="` + officeFontBody + `" w:hAnsi="` + officeFontBody + `"/><w:color w:val="` + officeColorInk + `"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:after="260"/></w:pPr><w:rPr><w:rFonts w:ascii="` + officeFontHeading + `" w:hAnsi="` + officeFontHeading + `"/><w:b/><w:color w:val="` + officeColorAccent + `"/><w:sz w:val="40"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="260" w:after="130"/></w:pPr><w:rPr><w:rFonts w:ascii="` + officeFontHeading + `" w:hAnsi="` + officeFontHeading + `"/><w:b/><w:color w:val="` + officeColorAccent + `"/><w:sz w:val="30"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="220" w:after="110"/></w:pPr><w:rPr><w:rFonts w:ascii="` + officeFontHeading + `" w:hAnsi="` + officeFontHeading + `"/><w:b/><w:color w:val="` + officeColorAccentAlt + `"/><w:sz w:val="25"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="Heading3"><w:name w:val="heading 3"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="170" w:after="90"/></w:pPr><w:rPr><w:rFonts w:ascii="` + officeFontHeading + `" w:hAnsi="` + officeFontHeading + `"/><w:b/><w:color w:val="` + officeColorMuted + `"/><w:sz w:val="22"/></w:rPr></w:style>
</w:styles>
`
}

func documentExportTitle(sourceTitle string, sourcePath string) string {
	sourceTitle = strings.TrimSpace(sourceTitle)
	for _, prefix := range []string{"Document Export - ", "Document Brief - ", "Document Set Report - ", "Assistant Answer - "} {
		sourceTitle = strings.TrimPrefix(sourceTitle, prefix)
	}
	if sourceTitle != "" {
		return "Document Export - " + sourceTitle
	}
	sourcePath = filepath.ToSlash(strings.TrimSpace(sourcePath))
	if sourcePath != "" {
		name := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
		if strings.TrimSpace(name) != "" {
			return "Document Export - " + name
		}
	}
	return "Document Export"
}

func xmlEscape(value string) string {
	var buffer bytes.Buffer
	_ = xml.EscapeText(&buffer, []byte(value))
	return buffer.String()
}
