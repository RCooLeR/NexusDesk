package artifacts

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) WriteSQLiteQueryCSVArtifact(report SQLiteQueryReport) (Artifact, error) {
	if strings.TrimSpace(report.SourcePath) == "" {
		return Artifact{}, errors.New("SQLite query source path is required")
	}
	if len(report.Columns) == 0 {
		return Artifact{}, errors.New("SQLite query CSV export requires result columns")
	}
	createdAt := time.Now().UTC()
	title := sqliteQueryArtifactTitle(report, "CSV")
	relPath := s.relPath("sqlite-queries", fmt.Sprintf("%s-%s.csv", artifactTimestamp(createdAt), safeName(title)))
	absPath := s.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Artifact{}, err
	}
	file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return Artifact{}, err
	}
	writer := csv.NewWriter(file)
	if err := writer.Write(report.Columns); err != nil {
		_ = file.Close()
		return Artifact{}, err
	}
	for _, row := range report.Rows {
		if err := writer.Write(normalizeArtifactRow(row, len(report.Columns))); err != nil {
			_ = file.Close()
			return Artifact{}, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		_ = file.Close()
		return Artifact{}, err
	}
	if err := file.Close(); err != nil {
		return Artifact{}, err
	}
	info, _ := os.Stat(absPath)
	metadata := sqliteQueryMetadata("sqlite-query-csv", title, relPath, report, createdAt)
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	size := int64(0)
	if info != nil {
		size = info.Size()
	}
	return sqliteQueryArtifact("sqlite-query-csv", title, relPath, absPath, size, report, metadata, createdAt), nil
}

func (s *Store) WriteSQLiteQueryMarkdownArtifact(report SQLiteQueryReport) (Artifact, error) {
	if strings.TrimSpace(report.SourcePath) == "" {
		return Artifact{}, errors.New("SQLite query source path is required")
	}
	createdAt := time.Now().UTC()
	title := sqliteQueryArtifactTitle(report, "Report")
	content := sqliteQueryMarkdown(report, title, createdAt)
	relPath := s.relPath("sqlite-queries", fmt.Sprintf("%s-%s.md", artifactTimestamp(createdAt), safeName(title)))
	absPath := s.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Artifact{}, err
	}
	file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		return Artifact{}, err
	}
	metadata := sqliteQueryMetadata("sqlite-query-report", title, relPath, report, createdAt)
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	return sqliteQueryArtifact("sqlite-query-report", title, relPath, absPath, int64(len(content)), report, metadata, createdAt), nil
}

func sqliteQueryMetadata(kind string, title string, relPath string, report SQLiteQueryReport, createdAt time.Time) Metadata {
	return Metadata{
		Kind:        kind,
		Title:       title,
		RelPath:     relPath,
		Source:      sqliteQuerySourceSummary(report),
		SourcePaths: []string{report.SourcePath},
		GeneratedAt: createdAt,
	}
}

func sqliteQueryArtifact(kind string, title string, relPath string, absPath string, size int64, report SQLiteQueryReport, metadata Metadata, createdAt time.Time) Artifact {
	return Artifact{
		Kind:         kind,
		Title:        title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: relPath + ".json",
		Message:      title + " artifact created at " + relPath + ".",
		Size:         size,
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		Source:       metadata.Source,
		SourcePaths:  []string{report.SourcePath},
	}
}

func sqliteQueryArtifactTitle(report SQLiteQueryReport, suffix string) string {
	if strings.TrimSpace(report.Title) != "" {
		return strings.TrimSpace(report.Title)
	}
	name := filepath.Base(filepath.ToSlash(report.SourcePath))
	if name == "." || name == "/" || name == "" {
		name = "SQLite"
	}
	return "SQLite Query " + suffix + " - " + name
}

func sqliteQuerySourceSummary(report SQLiteQueryReport) string {
	parts := []string{report.SourcePath}
	if strings.TrimSpace(report.Engine) != "" {
		parts = append(parts, "engine: "+strings.TrimSpace(report.Engine))
	}
	if strings.TrimSpace(report.SQL) != "" {
		parts = append(parts, "sql: "+compactArtifactLine(report.SQL, 240))
	}
	if report.ResultLimit > 0 {
		parts = append(parts, fmt.Sprintf("cap: %d", report.ResultLimit))
	}
	if report.TimeoutSeconds > 0 {
		parts = append(parts, fmt.Sprintf("timeout: %ds", report.TimeoutSeconds))
	}
	return strings.Join(parts, " | ")
}

func sqliteQueryMarkdown(report SQLiteQueryReport, title string, createdAt time.Time) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	writeKV(&builder, "Source", report.SourcePath)
	writeKV(&builder, "Engine", report.Engine)
	writeKV(&builder, "Generated", formatArtifactTime(createdAt))
	writeKV(&builder, "Rows shown", fmt.Sprintf("%d", len(report.Rows)))
	writeKV(&builder, "Observed rows", fmt.Sprintf("%d", report.TotalRows))
	writeKV(&builder, "Row cap", fmt.Sprintf("%d", report.ResultLimit))
	writeKV(&builder, "Timeout", fmt.Sprintf("%d seconds", report.TimeoutSeconds))
	writeKV(&builder, "Duration", fmt.Sprintf("%d ms", report.DurationMs))
	writeKV(&builder, "Message", report.Message)
	if report.Truncated {
		builder.WriteString("\nResult stopped at the configured row cap.\n")
	}
	builder.WriteString("\n## SQL\n\n```sql\n")
	builder.WriteString(strings.TrimSpace(report.SQL))
	builder.WriteString("\n```\n\n## Rows\n\n")
	if len(report.Columns) == 0 {
		builder.WriteString("No columns were returned.\n")
		return builder.String()
	}
	writeMarkdownTable(&builder, report.Columns, report.Rows)
	return builder.String()
}

func normalizeArtifactRow(row []string, width int) []string {
	values := make([]string, width)
	for index := range values {
		if index < len(row) {
			values[index] = row[index]
		}
	}
	return values
}

func compactArtifactLine(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}
