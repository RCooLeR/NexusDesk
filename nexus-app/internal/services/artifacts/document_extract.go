package artifacts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) WriteDocumentExtractionReport(report DocumentExtractionReport) (Artifact, error) {
	content := strings.TrimSpace(report.Content)
	if content == "" {
		return Artifact{}, errors.New("document extraction content is required")
	}
	source := strings.TrimSpace(report.RelPath)
	if source == "" {
		return Artifact{}, errors.New("document extraction source path is required")
	}
	createdAt := time.Now().UTC()
	title := strings.TrimSpace(report.Title)
	if title == "" {
		title = filepath.Base(source)
	}
	relPath := s.relPath("document-extracts", fmt.Sprintf("%s-%s.md", artifactTimestamp(createdAt), safeName(title)))
	absPath := s.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Artifact{}, err
	}
	markdown := documentExtractionMarkdown(report, title, createdAt)
	file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()
	if _, err := file.WriteString(markdown); err != nil {
		return Artifact{}, err
	}
	metadata := Metadata{
		Kind:        "document-extract",
		Title:       title,
		RelPath:     relPath,
		Source:      source,
		SourcePaths: []string{source},
		GeneratedAt: createdAt,
	}
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:         "document-extract",
		Title:        title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: relPath + ".json",
		Message:      "Document extraction artifact created at " + relPath + ".",
		Size:         int64(len(markdown)),
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		Source:       source,
		SourcePaths:  []string{source},
	}, nil
}

func documentExtractionMarkdown(report DocumentExtractionReport, title string, createdAt time.Time) string {
	var builder strings.Builder
	builder.WriteString("# Document Extraction - ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	writeKV(&builder, "Generated", formatArtifactTime(createdAt))
	writeKV(&builder, "Source", report.RelPath)
	writeKV(&builder, "Format", report.Format)
	writeKV(&builder, "Media type", report.MediaType)
	writeKV(&builder, "Encoding", report.Encoding)
	writeKV(&builder, "Size", fmt.Sprintf("%d bytes", report.Size))
	writeKV(&builder, "Lines", fmt.Sprintf("%d", report.Lines))
	writeKV(&builder, "Words", fmt.Sprintf("%d", report.Words))
	if report.Pages > 0 {
		writeKV(&builder, "Pages", fmt.Sprintf("%d", report.Pages))
	}
	writeKV(&builder, "Truncated", fmt.Sprintf("%t", report.Truncated))
	builder.WriteString("\n## Extracted Text\n\n")
	builder.WriteString(strings.TrimSpace(report.Content))
	builder.WriteString("\n")
	return builder.String()
}
