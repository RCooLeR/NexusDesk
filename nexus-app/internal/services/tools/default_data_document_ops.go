package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"nexusdesk/internal/services/agent"
	artifactsSvc "nexusdesk/internal/services/artifacts"
	datasetsSvc "nexusdesk/internal/services/datasets"
	dbconnectorSvc "nexusdesk/internal/services/dbconnector"
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

func (h defaultHandlers) createDatasetChart(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, relPath, err := requiredWorkspacePath(call, request, "relPath", "path")
	if err != nil {
		return toolError(call, "high", err), err
	}
	query := firstArg(call, "query", "filter")
	result, err := datasetsSvc.New(h.deps.Workspace).QueryContext(ctx, root, relPath, query)
	if err != nil {
		return toolError(call, "high", err), err
	}
	chart, err := datasetsSvc.BuildChart(result)
	if err != nil {
		return toolError(call, "high", err), err
	}
	if err := ctx.Err(); err != nil {
		return toolError(call, "high", err), err
	}
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		return toolError(call, "high", err), err
	}
	artifact, err := store.WriteChartArtifact(chartArtifactInput(chart))
	if err != nil {
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{
		Name:    call.Name,
		Args:    call.Args,
		Risk:    "high",
		Mutated: true,
		Observation: fmt.Sprintf(
			"Generated dataset chart artifact.\nSource: %s\nArtifact: %s\nMode: %s\nCategory: %s\nValue: %s\nPoints: %d\nTruncated: %t",
			chart.RelPath,
			artifact.RelPath,
			chart.Mode,
			chart.CategoryColumn,
			firstNonEmptyTool(chart.ValueColumn, "row count"),
			len(chart.Points),
			chart.Truncated,
		),
	}, nil
}

func (h defaultHandlers) inspectSQLite(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, relPath, err := requiredWorkspacePath(call, request, "relPath", "path")
	if err != nil {
		return toolError(call, "medium", err), err
	}
	metadata, err := dbconnectorSvc.New().InspectWorkspaceSQLite(root, relPath)
	if err != nil {
		return toolError(call, "medium", err), err
	}
	return toolOK(call, "medium", formatSQLiteMetadata(metadata)), nil
}

func (h defaultHandlers) querySQLite(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, relPath, err := requiredWorkspacePath(call, request, "relPath", "path")
	if err != nil {
		return toolError(call, "medium", err), err
	}
	sql := firstArg(call, "sql", "query")
	if strings.TrimSpace(sql) == "" {
		err := errors.New("sql is required")
		return toolError(call, "medium", err), err
	}
	result, err := dbconnectorSvc.New().QueryWorkspaceSQLiteContext(ctx, root, dbconnectorSvc.SQLiteQueryRequest{
		RelPath:        relPath,
		SQL:            sql,
		ResultLimit:    intArg(call, "limit", dbconnectorSvc.DefaultSQLiteRows),
		TimeoutSeconds: intArg(call, "timeoutSeconds", dbconnectorSvc.DefaultSQLiteTimeoutSeconds),
	})
	if err != nil {
		return toolError(call, "medium", err), err
	}
	return toolOK(call, "medium", formatSQLiteQuery(result)), nil
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

func (h defaultHandlers) generateRunbook(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, relPath, err := requiredWorkspacePath(call, request, "relPath", "path")
	if err != nil {
		return toolError(call, "high", err), err
	}
	inspection, err := operationsSvc.New().InspectContext(ctx, root, relPath)
	if err != nil {
		return toolError(call, "high", err), err
	}
	if err := ctx.Err(); err != nil {
		return toolError(call, "high", err), err
	}
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		return toolError(call, "high", err), err
	}
	artifact, err := store.WriteOperationsRunbook(operationsRunbookArtifactInput(inspection))
	if err != nil {
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{
		Name:    call.Name,
		Args:    call.Args,
		Risk:    "high",
		Mutated: true,
		Observation: fmt.Sprintf(
			"Generated operations runbook artifact.\nSource: %s\nArtifact: %s\nKind: %s\nServices: %d\nWarnings: %d",
			inspection.File.RelPath,
			artifact.RelPath,
			inspection.File.Kind,
			len(inspection.Services),
			len(inspection.Warnings)+len(inspection.Topology.Warnings),
		),
	}, nil
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

func chartArtifactInput(chart datasetsSvc.ChartResult) artifactsSvc.ChartArtifactReport {
	return artifactsSvc.ChartArtifactReport{
		Title:          chartArtifactTitle(chart),
		SourcePath:     chart.RelPath,
		Query:          chart.Query,
		Format:         chart.Format,
		Mode:           chart.Mode,
		CategoryColumn: chart.CategoryColumn,
		ValueColumn:    chart.ValueColumn,
		SVG:            chart.SVG,
		PointCount:     len(chart.Points),
		Truncated:      chart.Truncated,
	}
}

func chartArtifactTitle(chart datasetsSvc.ChartResult) string {
	if chart.Mode == "line" && chart.ValueColumn != "" {
		return fmt.Sprintf("Chart - %s over %s", chart.ValueColumn, chart.CategoryColumn)
	}
	if chart.Mode == "sum" && chart.ValueColumn != "" {
		return fmt.Sprintf("Chart - %s by %s", chart.ValueColumn, chart.CategoryColumn)
	}
	return fmt.Sprintf("Chart - rows by %s", chart.CategoryColumn)
}

func formatSQLiteMetadata(metadata dbconnectorSvc.SQLiteMetadata) string {
	lines := []string{
		metadata.Message,
		fmt.Sprintf("SQLite metadata: %s", metadata.RelPath),
		fmt.Sprintf("Engine: %s readOnly=%t tables=%d views=%d indexes=%d relationships=%d", metadata.Engine, metadata.ReadOnly, len(metadata.Tables), len(metadata.Views), len(metadata.Indexes), len(metadata.Relationships)),
	}
	lines = append(lines, formatSQLiteObjects("Tables", metadata.Tables, 16)...)
	lines = append(lines, formatSQLiteObjects("Views", metadata.Views, 8)...)
	if len(metadata.Relationships) > 0 {
		lines = append(lines, "Relationships:")
		for index, relationship := range metadata.Relationships {
			if index >= 24 {
				lines = append(lines, "[relationships truncated]")
				break
			}
			lines = append(lines, fmt.Sprintf("- %s %s.%s -> %s.%s confidence=%s reason=%s", relationship.Kind, relationship.FromTable, relationship.FromColumn, relationship.ToTable, relationship.ToColumn, relationship.Confidence, relationship.Reason))
		}
	}
	return strings.Join(lines, "\n")
}

func formatSQLiteObjects(label string, objects []dbconnectorSvc.SQLiteObject, maxObjects int) []string {
	if len(objects) == 0 {
		return nil
	}
	lines := []string{label + ":"}
	for index, object := range objects {
		if index >= maxObjects {
			lines = append(lines, "["+strings.ToLower(label)+" truncated]")
			break
		}
		lines = append(lines, fmt.Sprintf("- %s type=%s rows=%d columns=%d indexes=%d", object.Name, object.Type, object.RowCount, len(object.Columns), len(object.Indexes)))
		if len(object.Columns) > 0 {
			columns := make([]string, 0, minInt(len(object.Columns), 24))
			for columnIndex, column := range object.Columns {
				if columnIndex >= 24 {
					columns = append(columns, "[columns truncated]")
					break
				}
				flags := []string{}
				if column.PrimaryKey {
					flags = append(flags, "pk")
				}
				if column.Nullable {
					flags = append(flags, "nullable")
				}
				suffix := ""
				if len(flags) > 0 {
					suffix = " " + strings.Join(flags, ",")
				}
				columns = append(columns, fmt.Sprintf("%s %s%s", column.Name, column.Type, suffix))
			}
			lines = append(lines, "  columns: "+strings.Join(columns, "; "))
		}
		if len(object.Indexes) > 0 {
			for indexIndex, sqliteIndex := range object.Indexes {
				if indexIndex >= 12 {
					lines = append(lines, "  [indexes truncated]")
					break
				}
				lines = append(lines, fmt.Sprintf("  index: %s unique=%t columns=%s", sqliteIndex.Name, sqliteIndex.Unique, strings.Join(sqliteIndex.Columns, ", ")))
			}
		}
		if len(object.SampleRows) > 0 {
			lines = append(lines, "  sample:")
			lines = append(lines, indentLines(formatMarkdownTable(sqliteObjectColumns(object), object.SampleRows, 5), "  "))
		}
	}
	return lines
}

func sqliteObjectColumns(object dbconnectorSvc.SQLiteObject) []string {
	columns := make([]string, 0, len(object.Columns))
	for _, column := range object.Columns {
		columns = append(columns, column.Name)
	}
	return columns
}

func formatSQLiteQuery(result dbconnectorSvc.SQLiteQueryResult) string {
	lines := []string{
		result.Message,
		fmt.Sprintf("SQLite query result: %s", result.RelPath),
		fmt.Sprintf("Engine: %s durationMs=%d rows=%d limit=%d timeoutSeconds=%d truncated=%t", result.Engine, result.DurationMs, result.TotalRows, result.ResultLimit, result.TimeoutSeconds, result.Truncated),
		"SQL: " + result.SQL,
		formatMarkdownTable(result.Columns, result.Rows, 50),
	}
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

func indentLines(value string, prefix string) string {
	if value == "" {
		return prefix
	}
	lines := strings.Split(value, "\n")
	for index := range lines {
		lines[index] = prefix + lines[index]
	}
	return strings.Join(lines, "\n")
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func firstNonEmptyTool(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
