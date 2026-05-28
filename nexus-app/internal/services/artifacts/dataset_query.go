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

func (s *Store) WriteDatasetQueryCSVArtifact(report DatasetQueryReport) (Artifact, error) {
	if strings.TrimSpace(report.SourcePath) == "" {
		return Artifact{}, errors.New("dataset query source path is required")
	}
	if len(report.Columns) == 0 {
		return Artifact{}, errors.New("dataset query CSV export requires result columns")
	}
	createdAt := time.Now().UTC()
	title := datasetQueryArtifactTitle(report, "CSV")
	relPath := s.relPath("dataset-queries", fmt.Sprintf("%s-%s.csv", createdAt.Format("20060102-150405-000000000"), safeName(title)))
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
	metadata := Metadata{
		Kind:        "dataset-query-csv",
		Title:       title,
		RelPath:     relPath,
		Source:      datasetQuerySourceSummary(report),
		SourcePaths: []string{report.SourcePath},
		GeneratedAt: createdAt,
	}
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	size := int64(0)
	if info != nil {
		size = info.Size()
	}
	return Artifact{
		Kind:         metadata.Kind,
		Title:        title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: relPath + ".json",
		Message:      "Dataset query CSV artifact created at " + relPath + ".",
		Size:         size,
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		Source:       metadata.Source,
		SourcePaths:  []string{report.SourcePath},
	}, nil
}

func (s *Store) WriteDatasetSQLMarkdownArtifact(report DatasetSQLReport) (Artifact, error) {
	if strings.TrimSpace(report.SourcePath) == "" {
		return Artifact{}, errors.New("dataset SQL source path is required")
	}
	if strings.TrimSpace(report.SQL) == "" {
		return Artifact{}, errors.New("dataset SQL report requires SQL text")
	}
	createdAt := time.Now().UTC()
	title := datasetSQLArtifactTitle(report)
	content := datasetSQLMarkdown(report, title, createdAt)
	relPath := s.relPath("dataset-sql", fmt.Sprintf("%s-%s.md", createdAt.Format("20060102-150405-000000000"), safeName(title)))
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
	metadata := Metadata{
		Kind:        "dataset-sql-report",
		Title:       title,
		RelPath:     relPath,
		Source:      datasetSQLSourceSummary(report),
		SourcePaths: []string{report.SourcePath},
		GeneratedAt: createdAt,
	}
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:         metadata.Kind,
		Title:        title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: relPath + ".json",
		Message:      "Dataset SQL report artifact created at " + relPath + ".",
		Size:         int64(len(content)),
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		Source:       metadata.Source,
		SourcePaths:  []string{report.SourcePath},
	}, nil
}

func (s *Store) WriteDatasetSummaryMarkdownArtifact(report DatasetSummaryReport) (Artifact, error) {
	if strings.TrimSpace(report.SourcePath) == "" {
		return Artifact{}, errors.New("dataset summary source path is required")
	}
	if len(report.Columns) == 0 {
		return Artifact{}, errors.New("dataset summary requires column profiles")
	}
	createdAt := time.Now().UTC()
	title := datasetSummaryArtifactTitle(report)
	content := datasetSummaryMarkdown(report, title, createdAt)
	relPath := s.relPath("dataset-summaries", fmt.Sprintf("%s-%s.md", createdAt.Format("20060102-150405-000000000"), safeName(title)))
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
	metadata := Metadata{
		Kind:           "dataset-summary",
		Title:          title,
		RelPath:        relPath,
		Source:         datasetSummarySourceSummary(report),
		ContextRelPath: strings.TrimSpace(report.SourcePath),
		SourcePaths:    []string{report.SourcePath},
		GeneratedAt:    createdAt,
	}
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:         metadata.Kind,
		Title:        title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: relPath + ".json",
		Message:      "Dataset summary artifact created at " + relPath + ".",
		Size:         int64(len(content)),
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		Source:       metadata.Source,
		SourcePaths:  []string{report.SourcePath},
	}, nil
}

func datasetQueryArtifactTitle(report DatasetQueryReport, suffix string) string {
	if strings.TrimSpace(report.Title) != "" {
		return strings.TrimSpace(report.Title)
	}
	name := filepath.Base(filepath.ToSlash(report.SourcePath))
	if name == "." || name == "/" || name == "" {
		name = "Dataset"
	}
	return "Dataset Query " + suffix + " - " + name
}

func datasetSQLArtifactTitle(report DatasetSQLReport) string {
	if strings.TrimSpace(report.Title) != "" {
		return strings.TrimSpace(report.Title)
	}
	name := filepath.Base(filepath.ToSlash(report.SourcePath))
	if name == "." || name == "/" || name == "" {
		name = "Dataset"
	}
	return "Dataset SQL Report - " + name
}

func datasetSummaryArtifactTitle(report DatasetSummaryReport) string {
	if strings.TrimSpace(report.Title) != "" {
		return strings.TrimSpace(report.Title)
	}
	name := filepath.Base(filepath.ToSlash(report.SourcePath))
	if name == "." || name == "/" || name == "" {
		name = "Dataset"
	}
	return "Dataset Summary - " + name
}

func datasetQuerySourceSummary(report DatasetQueryReport) string {
	parts := []string{report.SourcePath}
	if strings.TrimSpace(report.Query) != "" {
		parts = append(parts, "query: "+compactArtifactLine(report.Query, 240))
	}
	if report.Format != "" {
		parts = append(parts, "format: "+report.Format)
	}
	if report.Truncated {
		parts = append(parts, "bounded sample")
	}
	return strings.Join(parts, " | ")
}

func datasetSQLSourceSummary(report DatasetSQLReport) string {
	parts := []string{report.SourcePath}
	if strings.TrimSpace(report.Engine) != "" {
		parts = append(parts, "engine: "+strings.TrimSpace(report.Engine))
	}
	if strings.TrimSpace(report.SQL) != "" {
		parts = append(parts, "sql: "+compactArtifactLine(report.SQL, 240))
	}
	if report.Truncated {
		parts = append(parts, "bounded sample")
	}
	return strings.Join(parts, " | ")
}

func datasetSummarySourceSummary(report DatasetSummaryReport) string {
	parts := []string{report.SourcePath}
	if strings.TrimSpace(report.Format) != "" {
		parts = append(parts, "format: "+strings.TrimSpace(report.Format))
	}
	parts = append(parts, fmt.Sprintf("rows: %d", report.Rows))
	parts = append(parts, fmt.Sprintf("columns: %d", len(report.Columns)))
	if report.Truncated {
		parts = append(parts, "bounded profile")
	}
	return strings.Join(parts, " | ")
}

func datasetSQLMarkdown(report DatasetSQLReport, title string, createdAt time.Time) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	writeKV(&builder, "Source", report.SourcePath)
	writeKV(&builder, "Engine", report.Engine)
	writeKV(&builder, "Generated", formatArtifactTime(createdAt))
	writeKV(&builder, "Rows shown", fmt.Sprintf("%d", firstNonZeroInt(report.ShownRows, len(report.Rows))))
	writeKV(&builder, "Matched rows", fmt.Sprintf("%d", report.MatchedRows))
	writeKV(&builder, "Loaded rows", fmt.Sprintf("%d", report.TotalRows))
	writeKV(&builder, "Duration", fmt.Sprintf("%d ms", report.DurationMs))
	writeKV(&builder, "Message", report.Message)
	if report.Truncated {
		builder.WriteString("\nResult stopped at the configured native row cap.\n")
	}
	builder.WriteString("\n## SQL\n\n```sql\n")
	builder.WriteString(strings.TrimSpace(report.SQL))
	builder.WriteString("\n```\n")
	if len(report.Plan) > 0 {
		builder.WriteString("\n## Plan\n\n")
		for _, step := range report.Plan {
			builder.WriteString("- ")
			builder.WriteString(step)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\n## Rows\n\n")
	if len(report.Columns) == 0 {
		builder.WriteString("No columns were returned.\n")
		return builder.String()
	}
	writeMarkdownTable(&builder, report.Columns, report.Rows)
	return builder.String()
}

func datasetSummaryMarkdown(report DatasetSummaryReport, title string, createdAt time.Time) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	writeKV(&builder, "Source", report.SourcePath)
	writeKV(&builder, "Format", report.Format)
	writeKV(&builder, "Generated", formatArtifactTime(createdAt))
	writeKV(&builder, "Rows", fmt.Sprintf("%d", report.Rows))
	writeKV(&builder, "Columns", fmt.Sprintf("%d", len(report.Columns)))
	writeKV(&builder, "Size", fmt.Sprintf("%d bytes", report.Size))
	writeKV(&builder, "Media type", report.MediaType)
	writeKV(&builder, "Sheet", report.Sheet)
	if len(report.Sheets) > 0 {
		writeKV(&builder, "Sheets", strings.Join(report.Sheets, ", "))
	}
	writeKV(&builder, "Truncated", fmt.Sprintf("%t", report.Truncated))
	if len(report.Notes) > 0 {
		builder.WriteString("\n## Notes\n\n")
		for _, note := range report.Notes {
			if note = strings.TrimSpace(note); note != "" {
				builder.WriteString("- ")
				builder.WriteString(note)
				builder.WriteString("\n")
			}
		}
	}
	builder.WriteString("\n## Columns\n\n")
	writeMarkdownTable(&builder, []string{"Column", "Type", "Non-empty", "Empty", "Samples"}, datasetSummaryColumnRows(report.Columns))
	questions := datasetSummarySuggestedQuestions(report.Columns)
	if len(questions) > 0 {
		builder.WriteString("\n## Suggested Questions\n\n")
		for _, question := range questions {
			builder.WriteString("- ")
			builder.WriteString(question)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func datasetSummaryColumnRows(columns []DatasetSummaryColumnReport) [][]string {
	rows := make([][]string, 0, len(columns))
	for _, column := range columns {
		rows = append(rows, []string{
			column.Name,
			column.Type,
			fmt.Sprintf("%d", column.NonEmpty),
			fmt.Sprintf("%d", column.Empty),
			strings.Join(column.Samples, ", "),
		})
	}
	return rows
}

func datasetSummarySuggestedQuestions(columns []DatasetSummaryColumnReport) []string {
	questions := []string{}
	for _, column := range columns {
		name := strings.TrimSpace(column.Name)
		if name == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(column.Type)) {
		case "number", "integer", "float", "double", "decimal", "int64":
			questions = append(questions, "Which segments explain the largest values in `"+strings.ReplaceAll(name, "`", "'")+"`?")
		case "date", "datetime", "time":
			questions = append(questions, "What trend appears over `"+strings.ReplaceAll(name, "`", "'")+"`?")
		}
	}
	questions = append(questions,
		"Which rows are missing important values?",
		"What chart best communicates the top categories or trends?",
	)
	return questions
}

func firstNonZeroInt(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
