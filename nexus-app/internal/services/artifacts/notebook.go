package artifacts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) WriteNotebookRunReport(report NotebookRunReport) (Artifact, error) {
	source := strings.TrimSpace(report.SourcePath)
	if source == "" {
		return Artifact{}, errors.New("notebook run source path is required")
	}
	if len(report.Cells) == 0 {
		return Artifact{}, errors.New("notebook run cells are required")
	}
	createdAt := time.Now().UTC()
	title := strings.TrimSpace(report.Title)
	if title == "" {
		title = notebookRunTitle(report)
	}
	relPath := s.relPath("notebooks", fmt.Sprintf("%s-%s.md", createdAt.Format("20060102-150405-000000000"), safeName(title)))
	absPath := s.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Artifact{}, err
	}
	markdown := notebookRunMarkdown(report, title, createdAt)
	file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()
	if _, err := file.WriteString(markdown); err != nil {
		return Artifact{}, err
	}
	metadata := Metadata{
		Kind:        "sql-notebook-run",
		Title:       title,
		RelPath:     relPath,
		Source:      notebookRunSourceSummary(report),
		SourcePaths: []string{source},
		GeneratedAt: createdAt,
	}
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:         "sql-notebook-run",
		Title:        title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: relPath + ".json",
		Message:      "SQL notebook run artifact created at " + relPath + ".",
		Size:         int64(len(markdown)),
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		Source:       metadata.Source,
		SourcePaths:  []string{source},
	}, nil
}

func notebookRunMarkdown(report NotebookRunReport, title string, createdAt time.Time) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	writeKV(&builder, "Generated", formatArtifactTime(createdAt))
	writeKV(&builder, "Source", report.SourcePath)
	writeKV(&builder, "Notebook", firstNonEmptyArtifact(report.Label, report.NotebookID))
	writeKV(&builder, "Started", formatArtifactTime(report.StartedAt))
	writeKV(&builder, "Completed", formatArtifactTime(report.CompletedAt))
	writeKV(&builder, "Duration", fmt.Sprintf("%d ms", report.DurationMs))
	writeKV(&builder, "Cells", fmt.Sprintf("%d", len(report.Cells)))
	failed := notebookFailedCells(report.Cells)
	writeKV(&builder, "Failures", fmt.Sprintf("%d", failed))
	if strings.TrimSpace(report.Message) != "" {
		builder.WriteString("\n## Summary\n\n")
		builder.WriteString(strings.TrimSpace(report.Message))
		builder.WriteString("\n")
	}
	for index, cell := range report.Cells {
		builder.WriteString(fmt.Sprintf("\n## Cell %d: %s [%s]\n\n", index+1, firstNonEmptyArtifact(cell.Label, cell.CellID), firstNonEmptyArtifact(cell.Kind, "sql")))
		writeKV(&builder, "Status", notebookCellStatus(cell))
		writeKV(&builder, "Engine", cell.Engine)
		writeKV(&builder, "Duration", fmt.Sprintf("%d ms", cell.DurationMs))
		if strings.TrimSpace(cell.SQL) != "" {
			builder.WriteString("\n### SQL\n\n")
			writeFence(&builder, "sql", cell.SQL)
		}
		if strings.TrimSpace(cell.Error) != "" {
			builder.WriteString("\n### Error\n\n")
			writeFence(&builder, "", cell.Error)
			continue
		}
		if len(cell.Plan) > 0 {
			builder.WriteString("\n### Plan\n\n")
			for _, step := range cell.Plan {
				builder.WriteString("- ")
				builder.WriteString(step)
				builder.WriteString("\n")
			}
		}
		if len(cell.Columns) > 0 {
			builder.WriteString("\n### Rows\n\n")
			writeKV(&builder, "Matched rows", fmt.Sprintf("%d", cell.MatchedRows))
			writeKV(&builder, "Shown rows", fmt.Sprintf("%d", cell.ShownRows))
			writeMarkdownTable(&builder, cell.Columns, cell.Rows)
		}
		if strings.TrimSpace(cell.ChartSVG) != "" {
			builder.WriteString("\n### Chart\n\n")
			writeKV(&builder, "Mode", cell.ChartMode)
			writeKV(&builder, "Points", fmt.Sprintf("%d", cell.ChartPoints))
			if strings.TrimSpace(cell.ChartMessage) != "" {
				builder.WriteString(strings.TrimSpace(cell.ChartMessage))
				builder.WriteString("\n\n")
			}
			writeFence(&builder, "svg", cell.ChartSVG)
		}
	}
	return builder.String()
}

func notebookRunTitle(report NotebookRunReport) string {
	if strings.TrimSpace(report.Label) != "" {
		return "SQL Notebook Run - " + strings.TrimSpace(report.Label)
	}
	if strings.TrimSpace(report.SourcePath) != "" {
		return "SQL Notebook Run - " + filepath.Base(report.SourcePath)
	}
	return "SQL Notebook Run"
}

func notebookRunSourceSummary(report NotebookRunReport) string {
	parts := []string{strings.TrimSpace(report.SourcePath)}
	if strings.TrimSpace(report.Label) != "" {
		parts = append(parts, "notebook: "+strings.TrimSpace(report.Label))
	}
	if strings.TrimSpace(report.NotebookID) != "" {
		parts = append(parts, "id: "+strings.TrimSpace(report.NotebookID))
	}
	if len(report.Cells) > 0 {
		parts = append(parts, fmt.Sprintf("cells: %d", len(report.Cells)))
	}
	failed := notebookFailedCells(report.Cells)
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("failures: %d", failed))
	}
	return strings.Join(parts, " | ")
}

func notebookCellStatus(cell NotebookRunCellReport) string {
	if strings.TrimSpace(cell.Status) != "" {
		return cell.Status
	}
	if strings.TrimSpace(cell.Error) != "" {
		return "failed"
	}
	return "success"
}

func notebookFailedCells(cells []NotebookRunCellReport) int {
	failed := 0
	for _, cell := range cells {
		status := strings.ToLower(strings.TrimSpace(notebookCellStatus(cell)))
		if status == "failed" || strings.TrimSpace(cell.Error) != "" {
			failed++
		}
	}
	return failed
}

func writeFence(builder *strings.Builder, language string, value string) {
	builder.WriteString("```")
	builder.WriteString(strings.TrimSpace(language))
	builder.WriteString("\n")
	builder.WriteString(strings.TrimRight(value, "\r\n"))
	builder.WriteString("\n```\n")
}

func writeMarkdownTable(builder *strings.Builder, columns []string, rows [][]string) {
	builder.WriteString("| ")
	for _, column := range columns {
		builder.WriteString(escapeMarkdownCell(column))
		builder.WriteString(" | ")
	}
	builder.WriteString("\n| ")
	for range columns {
		builder.WriteString("--- | ")
	}
	builder.WriteString("\n")
	for _, row := range rows {
		builder.WriteString("| ")
		for index := range columns {
			value := ""
			if index < len(row) {
				value = row[index]
			}
			builder.WriteString(escapeMarkdownCell(value))
			builder.WriteString(" | ")
		}
		builder.WriteString("\n")
	}
}

func escapeMarkdownCell(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.TrimSpace(value)
}
