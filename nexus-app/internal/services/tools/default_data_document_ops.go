package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"nexusdesk/internal/services/agent"
	datasetsSvc "nexusdesk/internal/services/datasets"
	documentsSvc "nexusdesk/internal/services/documents"
	operationsSvc "nexusdesk/internal/services/operations"
)

func (h defaultHandlers) profileDataset(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, relPath, err := requiredWorkspacePath(call, request, "relPath", "path")
	if err != nil {
		return toolError(call, "low", err), err
	}
	profile, err := datasetsSvc.New(h.deps.Workspace).ProfileContext(ctx, root, relPath)
	if err != nil {
		return toolError(call, "low", err), err
	}
	return toolOK(call, "low", formatDatasetProfile(profile)), nil
}

func (h defaultHandlers) queryDataset(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, relPath, err := requiredWorkspacePath(call, request, "relPath", "path")
	if err != nil {
		return toolError(call, "low", err), err
	}
	result, err := datasetsSvc.New(h.deps.Workspace).QueryContext(ctx, root, relPath, firstArg(call, "query", "filter"))
	if err != nil {
		return toolError(call, "low", err), err
	}
	return toolOK(call, "low", formatDatasetQuery(result)), nil
}

func (h defaultHandlers) queryDatasetSQL(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, relPath, err := requiredWorkspacePath(call, request, "relPath", "path")
	if err != nil {
		return toolError(call, "medium", err), err
	}
	sql := firstArg(call, "sql", "query")
	if strings.TrimSpace(sql) == "" {
		err := errors.New("sql is required")
		return toolError(call, "medium", err), err
	}
	result, err := datasetsSvc.New(h.deps.Workspace).QuerySQLContext(ctx, root, relPath, sql)
	if err != nil {
		return toolError(call, "medium", err), err
	}
	return toolOK(call, "medium", formatDatasetSQL(result)), nil
}

func (h defaultHandlers) extractDocument(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, relPath, err := requiredWorkspacePath(call, request, "relPath", "path")
	if err != nil {
		return toolError(call, "low", err), err
	}
	document, err := documentsSvc.New(h.deps.Workspace).Extract(root, relPath)
	if err != nil {
		return toolError(call, "low", err), err
	}
	return toolOK(call, "low", formatExtractedDocument(document)), nil
}

func (h defaultHandlers) inspectOperationsFiles(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	service := operationsSvc.New()
	relPath := firstArg(call, "relPath", "path")
	if strings.TrimSpace(relPath) == "" {
		result, err := service.ScanContext(ctx, root)
		if err != nil {
			return toolError(call, "low", err), err
		}
		return toolOK(call, "low", formatOperationsScan(result)), nil
	}
	inspection, err := service.InspectContext(ctx, root, relPath)
	if err != nil {
		return toolError(call, "low", err), err
	}
	return toolOK(call, "low", formatOperationsInspection(inspection)), nil
}

func requiredWorkspacePath(call agent.ToolCall, request agent.Request, keys ...string) (string, string, error) {
	root, err := workspaceRoot(request)
	if err != nil {
		return "", "", err
	}
	relPath := firstArg(call, keys...)
	if strings.TrimSpace(relPath) == "" {
		return "", "", errors.New("relPath is required")
	}
	return root, relPath, nil
}

func formatDatasetProfile(profile datasetsSvc.Profile) string {
	lines := []string{
		fmt.Sprintf("Dataset profile: %s", profile.RelPath),
		fmt.Sprintf("Format: %s media=%s size=%d rows=%d columns=%d truncated=%t", profile.Format, profile.MediaType, profile.Size, profile.Rows, len(profile.Columns), profile.Truncated),
	}
	if profile.Sheet != "" {
		lines = append(lines, "Sheet: "+profile.Sheet)
	}
	if len(profile.Sheets) > 0 {
		lines = append(lines, "Sheets: "+strings.Join(profile.Sheets, ", "))
	}
	if profile.JSONProfile != nil {
		lines = append(lines, fmt.Sprintf("JSON: topLevel=%s count=%d", profile.JSONProfile.TopLevel, profile.JSONProfile.Count))
		lines = append(lines, profile.JSONProfile.Notes...)
	}
	if profile.Parquet != nil {
		lines = append(lines, fmt.Sprintf("Parquet: version=%d createdBy=%q rowGroups=%d schemaColumns=%d metadataDecoded=%t truncated=%t", profile.Parquet.Version, profile.Parquet.CreatedBy, len(profile.Parquet.RowGroups), len(profile.Parquet.SchemaColumns), profile.Parquet.MetadataDecoded, profile.Parquet.Truncated))
		for index, column := range profile.Parquet.SchemaColumns {
			if index >= 12 {
				lines = append(lines, "[parquet schema truncated]")
				break
			}
			lines = append(lines, fmt.Sprintf("- %s type=%s repetition=%s converted=%s", column.Path, column.Type, column.RepetitionType, column.ConvertedType))
		}
	}
	if len(profile.Columns) > 0 {
		lines = append(lines, "Columns:")
		for index, column := range profile.Columns {
			if index >= 24 {
				lines = append(lines, "[columns truncated]")
				break
			}
			samples := strings.Join(column.Samples, ", ")
			lines = append(lines, fmt.Sprintf("- %s type=%s nonEmpty=%d empty=%d samples=[%s]", column.Name, column.Type, column.NonEmpty, column.Empty, samples))
		}
	}
	if len(profile.Notes) > 0 {
		lines = append(lines, "Notes:")
		lines = append(lines, profile.Notes...)
	}
	return strings.Join(lines, "\n")
}

func formatDatasetQuery(result datasetsSvc.QueryResult) string {
	lines := []string{
		result.Message,
		fmt.Sprintf("Dataset: %s format=%s query=%q totalRows=%d matchedRows=%d shownRows=%d truncated=%t", result.RelPath, result.Format, result.Query, result.TotalRows, result.MatchedRows, len(result.Rows), result.Truncated),
		formatMarkdownTable(result.Columns, result.Rows, 20),
	}
	return strings.Join(lines, "\n")
}

func formatDatasetSQL(result datasetsSvc.SQLResult) string {
	lines := []string{
		fmt.Sprintf("Dataset SQL result: %s", result.RelPath),
		fmt.Sprintf("Engine: %s durationMs=%d truncated=%t", result.Engine, result.DurationMs, result.Truncated),
		"SQL: " + result.SQL,
	}
	if len(result.Plan) > 0 {
		lines = append(lines, "Plan:")
		for _, step := range result.Plan {
			lines = append(lines, "- "+step)
		}
	}
	lines = append(lines, formatDatasetQuery(result.QueryResult))
	return strings.Join(lines, "\n")
}

func formatMarkdownTable(columns []string, rows [][]string, maxRows int) string {
	if len(columns) == 0 {
		return "No tabular rows available."
	}
	cleanColumns := make([]string, len(columns))
	for index, column := range columns {
		cleanColumns[index] = cleanTableCell(column)
	}
	lines := []string{
		"| " + strings.Join(cleanColumns, " | ") + " |",
		"| " + strings.Join(repeatString("---", len(columns)), " | ") + " |",
	}
	for index, row := range rows {
		if index >= maxRows {
			lines = append(lines, "[rows truncated]")
			break
		}
		cells := make([]string, len(columns))
		for column := range columns {
			if column < len(row) {
				cells[column] = cleanTableCell(row[column])
			}
		}
		lines = append(lines, "| "+strings.Join(cells, " | ")+" |")
	}
	return strings.Join(lines, "\n")
}

func repeatString(value string, count int) []string {
	values := make([]string, count)
	for index := range values {
		values[index] = value
	}
	return values
}

func cleanTableCell(value string) string {
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.TrimSpace(value)
}

func formatExtractedDocument(document documentsSvc.ExtractedDocument) string {
	lines := []string{
		fmt.Sprintf("Document extract: %s", document.RelPath),
		fmt.Sprintf("Title: %s", document.Title),
		fmt.Sprintf("Format: %s media=%s encoding=%s size=%d lines=%d words=%d pages=%d truncated=%t", document.Format, document.MediaType, document.Encoding, document.Size, document.Lines, document.Words, document.Pages, document.Truncated),
		"",
		document.Text,
	}
	return strings.Join(lines, "\n")
}

func formatOperationsScan(result operationsSvc.ScanResult) string {
	lines := []string{
		result.Message,
		fmt.Sprintf("Summary: files=%d compose=%d dockerfiles=%d env=%d config=%d logs=%d scripts=%d skippedDirs=%d unreadable=%d entryCap=%d", result.Summary.Files, result.Summary.Compose, result.Summary.Dockerfiles, result.Summary.Env, result.Summary.Config, result.Summary.Logs, result.Summary.Scripts, result.Summary.SkippedDirs, result.Summary.Unreadable, result.Summary.EntryCap),
	}
	for index, file := range result.Files {
		if index >= 40 {
			lines = append(lines, "[operations files truncated]")
			break
		}
		lines = append(lines, fmt.Sprintf("- %s [%s] size=%d", file.RelPath, file.Kind, file.Size))
	}
	return strings.Join(lines, "\n")
}
