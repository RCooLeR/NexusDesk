package shell

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	artifactsSvc "nexusdesk/internal/services/artifacts"
	datasetsSvc "nexusdesk/internal/services/datasets"
	dbconnectorSvc "nexusdesk/internal/services/dbconnector"
	metadataSvc "nexusdesk/internal/services/metadata"
)

func profileStatus(profile datasetsSvc.Profile) string {
	truncated := ""
	if profile.Truncated {
		truncated = " sample"
	}
	return fmt.Sprintf("%s: %s%s, %d rows, %d columns", profile.RelPath, profile.Format, truncated, profile.Rows, len(profile.Columns))
}

func formatDatasetProfile(profile datasetsSvc.Profile) string {
	var builder strings.Builder
	builder.WriteString("# Dataset Profile\n\n")
	builder.WriteString("Path: ")
	builder.WriteString(profile.RelPath)
	builder.WriteString("\nFormat: ")
	builder.WriteString(profile.Format)
	builder.WriteString("\nMedia type: ")
	builder.WriteString(profile.MediaType)
	builder.WriteString(fmt.Sprintf("\nSize: %d bytes\nRows: %d\nColumns: %d\n", profile.Size, profile.Rows, len(profile.Columns)))
	if len(profile.Sheets) > 0 {
		builder.WriteString("Sheets: ")
		builder.WriteString(strings.Join(profile.Sheets, ", "))
		builder.WriteString("\n")
	}
	if profile.Sheet != "" {
		builder.WriteString("Profiled sheet: ")
		builder.WriteString(profile.Sheet)
		builder.WriteString("\n")
	}
	if profile.Truncated {
		builder.WriteString("Scope: preview sample is truncated by the safe preview cap\n")
	}
	if len(profile.Notes) > 0 {
		builder.WriteString("\nNotes\n")
		for _, note := range profile.Notes {
			builder.WriteString("- ")
			builder.WriteString(note)
			builder.WriteString("\n")
		}
	}
	if profile.JSONProfile != nil {
		builder.WriteString("\nJSON\n")
		builder.WriteString("- Top level: ")
		builder.WriteString(profile.JSONProfile.TopLevel)
		builder.WriteString(fmt.Sprintf("\n- Count: %d\n", profile.JSONProfile.Count))
		for _, note := range profile.JSONProfile.Notes {
			builder.WriteString("- ")
			builder.WriteString(note)
			builder.WriteString("\n")
		}
	}
	if profile.Parquet != nil {
		builder.WriteString("\nParquet\n")
		builder.WriteString(fmt.Sprintf("- Footer metadata: %d bytes\n", profile.Parquet.FooterLength))
		builder.WriteString(fmt.Sprintf("- Data bytes: %d\n", profile.Parquet.DataBytes))
		if profile.Parquet.MetadataDecoded {
			builder.WriteString(fmt.Sprintf("- Version: %d\n", profile.Parquet.Version))
			if strings.TrimSpace(profile.Parquet.CreatedBy) != "" {
				builder.WriteString("- Created by: ")
				builder.WriteString(profile.Parquet.CreatedBy)
				builder.WriteString("\n")
			}
			builder.WriteString(fmt.Sprintf("- Schema columns: %d\n", len(profile.Parquet.SchemaColumns)))
			builder.WriteString(fmt.Sprintf("- Row groups: %d\n", len(profile.Parquet.RowGroups)))
			for _, rowGroup := range profile.Parquet.RowGroups {
				builder.WriteString(fmt.Sprintf("  - row group %d: rows %d | columns %d | bytes %d compressed / %d uncompressed\n",
					rowGroup.Index,
					rowGroup.Rows,
					rowGroup.Columns,
					rowGroup.TotalCompressedSize,
					rowGroup.TotalUncompressedSize,
				))
			}
		}
		if profile.Parquet.Truncated {
			builder.WriteString("- Footer decode skipped by native metadata cap\n")
		}
	}
	if len(profile.Columns) == 0 {
		builder.WriteString("\nNo tabular fields were found.\n")
		return builder.String()
	}
	builder.WriteString("\nFields\n")
	for _, column := range profile.Columns {
		builder.WriteString("- ")
		builder.WriteString(column.Name)
		builder.WriteString(" | ")
		builder.WriteString(column.Type)
		builder.WriteString(fmt.Sprintf(" | non-empty %d | empty %d", column.NonEmpty, column.Empty))
		if len(column.Samples) > 0 {
			builder.WriteString(" | samples: ")
			builder.WriteString(strings.Join(column.Samples, ", "))
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

func queryStatus(result datasetsSvc.QueryResult) string {
	truncated := ""
	if result.Truncated {
		truncated = " bounded"
	}
	return fmt.Sprintf("%s: %s%s query, %d/%d rows shown", result.RelPath, result.Format, truncated, len(result.Rows), result.MatchedRows)
}

func sqlStatus(result datasetsSvc.SQLResult) string {
	return fmt.Sprintf("%s: %s SQL, %d/%d rows shown", result.RelPath, result.Engine, len(result.Rows), result.MatchedRows)
}

func sqliteQueryStatus(result dbconnectorSvc.SQLiteQueryResult) string {
	truncated := ""
	if result.Truncated {
		truncated = " capped"
	}
	return fmt.Sprintf("%s: SQLite%s query, %d row(s) shown, cap %d, timeout %ds", result.RelPath, truncated, len(result.Rows), result.ResultLimit, result.TimeoutSeconds)
}

func formatDatasetQueryResult(result datasetsSvc.QueryResult) string {
	var builder strings.Builder
	builder.WriteString("# Dataset Query\n\n")
	builder.WriteString("Path: ")
	builder.WriteString(result.RelPath)
	builder.WriteString("\nFormat: ")
	builder.WriteString(result.Format)
	builder.WriteString("\nQuery: ")
	if strings.TrimSpace(result.Query) == "" {
		builder.WriteString("(all rows)")
	} else {
		builder.WriteString(result.Query)
	}
	builder.WriteString(fmt.Sprintf("\nLoaded rows: %d\nMatched rows: %d\nShown rows: %d\n", result.TotalRows, result.MatchedRows, len(result.Rows)))
	if result.Truncated {
		builder.WriteString("Scope: result is bounded by the native query cap or source preview cap\n")
	}
	if result.Message != "" {
		builder.WriteString("\n")
		builder.WriteString(result.Message)
		builder.WriteString("\n")
	}
	if len(result.Columns) == 0 {
		builder.WriteString("\nNo columns were found.\n")
		return builder.String()
	}
	builder.WriteString("\n")
	builder.WriteString(strings.Join(result.Columns, "\t"))
	builder.WriteString("\n")
	for _, row := range result.Rows {
		values := make([]string, len(result.Columns))
		for index := range values {
			if index < len(row) {
				values[index] = row[index]
			}
		}
		builder.WriteString(strings.Join(values, "\t"))
		builder.WriteString("\n")
	}
	return builder.String()
}

func formatDatasetSQLResult(result datasetsSvc.SQLResult) string {
	var builder strings.Builder
	builder.WriteString("# Dataset SQL\n\n")
	builder.WriteString("Path: ")
	builder.WriteString(result.RelPath)
	builder.WriteString("\nEngine: ")
	builder.WriteString(result.Engine)
	builder.WriteString("\nSQL: ")
	builder.WriteString(result.SQL)
	builder.WriteString(fmt.Sprintf("\nLoaded rows: %d\nMatched rows: %d\nShown rows: %d\nDuration: %d ms\n", result.TotalRows, result.MatchedRows, len(result.Rows), result.DurationMs))
	if len(result.Plan) > 0 {
		builder.WriteString("\nPlan\n")
		for _, step := range result.Plan {
			builder.WriteString("- ")
			builder.WriteString(step)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\n")
	builder.WriteString(formatDatasetQueryResult(result.QueryResult))
	return builder.String()
}

func formatSQLiteQueryResult(result dbconnectorSvc.SQLiteQueryResult) string {
	var builder strings.Builder
	builder.WriteString("# SQLite Query Preview\n\n")
	builder.WriteString("Path: ")
	builder.WriteString(result.RelPath)
	builder.WriteString("\nEngine: ")
	builder.WriteString(result.Engine)
	builder.WriteString("\nSQL: ")
	builder.WriteString(result.SQL)
	builder.WriteString(fmt.Sprintf("\nShown rows: %d\nObserved rows: %d\nRow cap: %d\nTimeout: %d seconds\nDuration: %d ms\n", len(result.Rows), result.TotalRows, result.ResultLimit, result.TimeoutSeconds, result.DurationMs))
	if result.Truncated {
		builder.WriteString("Scope: result was stopped at the visible SQLite row cap\n")
	}
	if result.Message != "" {
		builder.WriteString("\n")
		builder.WriteString(result.Message)
		builder.WriteString("\n")
	}
	if len(result.Columns) == 0 {
		builder.WriteString("\nNo columns were returned.\n")
		return builder.String()
	}
	builder.WriteString("\n")
	builder.WriteString(strings.Join(result.Columns, "\t"))
	builder.WriteString("\n")
	for _, row := range result.Rows {
		values := make([]string, len(result.Columns))
		for index := range values {
			if index < len(row) {
				values[index] = row[index]
			}
		}
		builder.WriteString(strings.Join(values, "\t"))
		builder.WriteString("\n")
	}
	return builder.String()
}

func formatSavedQueries(title string, queries []datasetsSvc.SavedQuery) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	if len(queries) == 0 {
		builder.WriteString("No saved queries for the selected source.\n")
		return builder.String()
	}
	for index, query := range queries {
		builder.WriteString(fmt.Sprintf("%d. %s\n", index+1, firstNonEmptyString(query.Label, "Saved query")))
		builder.WriteString("   Path: ")
		builder.WriteString(query.RelPath)
		builder.WriteString("\n   Kind: ")
		builder.WriteString(query.Kind)
		if !query.UpdatedAt.IsZero() {
			builder.WriteString("\n   Updated: ")
			builder.WriteString(formatDataTime(query.UpdatedAt))
		}
		builder.WriteString("\n   SQL: ")
		builder.WriteString(compactDataLine(query.Query, 220))
		builder.WriteString("\n\n")
	}
	return builder.String()
}

func formatSQLiteMetadata(metadata dbconnectorSvc.SQLiteMetadata) string {
	var builder strings.Builder
	builder.WriteString("# SQLite Workspace Connector\n\n")
	builder.WriteString("Path: ")
	builder.WriteString(metadata.RelPath)
	builder.WriteString("\nEngine: ")
	builder.WriteString(metadata.Engine)
	builder.WriteString("\nMode: ")
	if metadata.ReadOnly {
		builder.WriteString("read-only")
	} else {
		builder.WriteString("read/write")
	}
	builder.WriteString(fmt.Sprintf("\nTables: %d\nViews: %d\nIndexes: %d\nRelationships: %d\n", len(metadata.Tables), len(metadata.Views), len(metadata.Indexes), len(metadata.Relationships)))
	if metadata.Message != "" {
		builder.WriteString("\n")
		builder.WriteString(metadata.Message)
		builder.WriteString("\n")
	}
	writeSQLiteObjects(&builder, "Tables", metadata.Tables)
	writeSQLiteObjects(&builder, "Views", metadata.Views)
	if len(metadata.Relationships) > 0 {
		builder.WriteString("\nRelationships\n")
		for _, relationship := range metadata.Relationships {
			builder.WriteString(fmt.Sprintf("- %s.%s -> %s.%s | %s | %s\n", relationship.FromTable, relationship.FromColumn, relationship.ToTable, relationship.ToColumn, relationship.Confidence, relationship.Kind))
			if strings.TrimSpace(relationship.Reason) != "" {
				builder.WriteString("  ")
				builder.WriteString(relationship.Reason)
				builder.WriteString("\n")
			}
		}
	}
	return builder.String()
}

func writeSQLiteObjects(builder *strings.Builder, title string, objects []dbconnectorSvc.SQLiteObject) {
	builder.WriteString("\n")
	builder.WriteString(title)
	builder.WriteString("\n")
	if len(objects) == 0 {
		builder.WriteString("- None.\n")
		return
	}
	for _, object := range objects {
		builder.WriteString(fmt.Sprintf("- %s | %s | %d row(s) | %d column(s)\n", object.Name, object.Type, object.RowCount, len(object.Columns)))
		if len(object.Columns) > 0 {
			builder.WriteString("  Columns: ")
			columnParts := make([]string, 0, len(object.Columns))
			for _, column := range object.Columns {
				part := column.Name
				if strings.TrimSpace(column.Type) != "" {
					part += " " + column.Type
				}
				if column.PrimaryKey {
					part += " pk"
				}
				if !column.Nullable {
					part += " not-null"
				}
				columnParts = append(columnParts, part)
			}
			builder.WriteString(strings.Join(columnParts, ", "))
			builder.WriteString("\n")
		}
		for _, index := range object.Indexes {
			unique := ""
			if index.Unique {
				unique = " unique"
			}
			builder.WriteString(fmt.Sprintf("  Index: %s%s", index.Name, unique))
			if len(index.Columns) > 0 {
				builder.WriteString(" on ")
				builder.WriteString(strings.Join(index.Columns, ", "))
			}
			builder.WriteString("\n")
		}
		if len(object.SampleRows) > 0 {
			headers := make([]string, 0, len(object.Columns))
			for _, column := range object.Columns {
				headers = append(headers, column.Name)
			}
			if len(headers) > 0 {
				builder.WriteString("  Sample: ")
				builder.WriteString(strings.Join(headers, "\t"))
				builder.WriteString("\n")
			}
			for _, row := range object.SampleRows {
				builder.WriteString("    ")
				builder.WriteString(strings.Join(row, "\t"))
				builder.WriteString("\n")
			}
		}
	}
}

func formatDatasetChart(chart datasetsSvc.ChartResult) string {
	var builder strings.Builder
	builder.WriteString("# Dataset Chart\n\n")
	builder.WriteString("Path: ")
	builder.WriteString(chart.RelPath)
	builder.WriteString("\nFormat: ")
	builder.WriteString(chart.Format)
	builder.WriteString("\nMode: ")
	builder.WriteString(chart.Mode)
	builder.WriteString("\nCategory column: ")
	builder.WriteString(chart.CategoryColumn)
	if chart.ValueColumn != "" {
		builder.WriteString("\nValue column: ")
		builder.WriteString(chart.ValueColumn)
	}
	if strings.TrimSpace(chart.Query) != "" {
		builder.WriteString("\nQuery: ")
		builder.WriteString(chart.Query)
	}
	builder.WriteString(fmt.Sprintf("\nPoints: %d\n", len(chart.Points)))
	if chart.Truncated {
		builder.WriteString("Scope: chart points are bounded by query and chart caps\n")
	}
	builder.WriteString("\n")
	builder.WriteString(chart.Message)
	builder.WriteString("\n\n")
	for _, point := range chart.Points {
		builder.WriteString(fmt.Sprintf("- %s: %.4g\n", point.Label, point.Value))
	}
	builder.WriteString("\nSVG\n\n")
	builder.WriteString(chart.SVG)
	return builder.String()
}

func formatDatasetDashboard(dashboard datasetsSvc.DashboardResult) string {
	var builder strings.Builder
	builder.WriteString("# Dataset Dashboard\n\n")
	builder.WriteString("Path: ")
	builder.WriteString(dashboard.RelPath)
	builder.WriteString("\nFormat: ")
	builder.WriteString(dashboard.Format)
	if strings.TrimSpace(dashboard.Query) != "" {
		builder.WriteString("\nQuery: ")
		builder.WriteString(dashboard.Query)
	}
	builder.WriteString("\n")
	if dashboard.Truncated {
		builder.WriteString("Scope: dashboard is bounded by query and chart caps\n")
	}
	builder.WriteString("\nMetrics\n")
	for _, metric := range dashboard.Metrics {
		builder.WriteString("- ")
		builder.WriteString(metric.Label)
		builder.WriteString(": ")
		builder.WriteString(metric.Value)
		if strings.TrimSpace(metric.Detail) != "" {
			builder.WriteString(" | ")
			builder.WriteString(metric.Detail)
		}
		builder.WriteString("\n")
	}
	builder.WriteString("\nChart\n")
	builder.WriteString("- Mode: ")
	builder.WriteString(dashboard.Chart.Mode)
	builder.WriteString("\n- Category: ")
	builder.WriteString(dashboard.Chart.CategoryColumn)
	if dashboard.Chart.ValueColumn != "" {
		builder.WriteString("\n- Value: ")
		builder.WriteString(dashboard.Chart.ValueColumn)
	}
	builder.WriteString(fmt.Sprintf("\n- Points: %d\n\n", len(dashboard.Chart.Points)))
	builder.WriteString(dashboard.Message)
	builder.WriteString("\n\nSVG\n\n")
	builder.WriteString(dashboard.SVG)
	return builder.String()
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

func dashboardArtifactInput(dashboard datasetsSvc.DashboardResult) artifactsSvc.ChartArtifactReport {
	return artifactsSvc.ChartArtifactReport{
		Title:          dashboardArtifactTitle(dashboard),
		SourcePath:     dashboard.RelPath,
		Query:          dashboard.Query,
		Format:         dashboard.Format,
		Mode:           "dashboard",
		CategoryColumn: dashboard.Chart.CategoryColumn,
		ValueColumn:    dashboard.Chart.ValueColumn,
		SVG:            dashboard.SVG,
		PointCount:     len(dashboard.Chart.Points),
		Truncated:      dashboard.Truncated,
	}
}

func notebookRunArtifactInput(result datasetsSvc.NotebookRunResult) artifactsSvc.NotebookRunReport {
	report := artifactsSvc.NotebookRunReport{
		Title:       "SQL Notebook Run - " + firstNonEmptyString(result.Label, result.NotebookID, result.RelPath),
		SourcePath:  result.RelPath,
		NotebookID:  result.NotebookID,
		Label:       result.Label,
		Message:     result.Message,
		StartedAt:   result.StartedAt,
		CompletedAt: result.CompletedAt,
		DurationMs:  result.DurationMs,
		Cells:       []artifactsSvc.NotebookRunCellReport{},
	}
	for _, cell := range result.Cells {
		status := "success"
		if cell.Error != "" {
			status = "failed"
		}
		sqlResult := cell.SQLResult
		chartResult := cell.ChartResult
		report.Cells = append(report.Cells, artifactsSvc.NotebookRunCellReport{
			CellID:       cell.CellID,
			Label:        cell.Label,
			Kind:         cell.Kind,
			SQL:          cell.SQL,
			Status:       status,
			Error:        cell.Error,
			Engine:       sqlResult.Engine,
			Columns:      append([]string{}, sqlResult.Columns...),
			Rows:         copyTableRows(sqlResult.Rows),
			MatchedRows:  sqlResult.MatchedRows,
			ShownRows:    len(sqlResult.Rows),
			Plan:         append([]string{}, sqlResult.Plan...),
			ChartMode:    chartResult.Mode,
			ChartMessage: chartResult.Message,
			ChartSVG:     chartResult.SVG,
			ChartPoints:  len(chartResult.Points),
			StartedAt:    cell.StartedAt,
			CompletedAt:  cell.CompletedAt,
			DurationMs:   cell.DurationMs,
		})
	}
	return report
}

func sqliteQueryArtifactInput(result dbconnectorSvc.SQLiteQueryResult) artifactsSvc.SQLiteQueryReport {
	return artifactsSvc.SQLiteQueryReport{
		Title:          "SQLite Query - " + result.RelPath,
		SourcePath:     result.RelPath,
		SQL:            result.SQL,
		Engine:         result.Engine,
		Columns:        append([]string{}, result.Columns...),
		Rows:           copyTableRows(result.Rows),
		TotalRows:      result.TotalRows,
		ResultLimit:    result.ResultLimit,
		TimeoutSeconds: result.TimeoutSeconds,
		DurationMs:     result.DurationMs,
		Truncated:      result.Truncated,
		Message:        result.Message,
	}
}

func copyTableRows(rows [][]string) [][]string {
	if len(rows) == 0 {
		return nil
	}
	copied := make([][]string, 0, len(rows))
	for _, row := range rows {
		copied = append(copied, append([]string{}, row...))
	}
	return copied
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

func dashboardArtifactTitle(dashboard datasetsSvc.DashboardResult) string {
	if dashboard.Chart.ValueColumn != "" {
		return fmt.Sprintf("Dashboard - %s by %s", dashboard.Chart.ValueColumn, dashboard.Chart.CategoryColumn)
	}
	return fmt.Sprintf("Dashboard - rows by %s", dashboard.Chart.CategoryColumn)
}

func datasetHistoryStatus(selected string, runs []metadataSvc.SQLRunRecord, dependencies []metadataSvc.DatasetDependencyRecord) string {
	if strings.TrimSpace(selected) != "" {
		count := 0
		for _, run := range runs {
			if run.RelPath == selected {
				count++
			}
		}
		return fmt.Sprintf("%s: %d SQL run(s), %d dependency record(s).", selected, count, len(dependencies))
	}
	return fmt.Sprintf("Dataset history: %d recent SQL run(s), %d dependency record(s).", len(runs), len(dependencies))
}

func formatDatasetHistory(selected string, runs []metadataSvc.SQLRunRecord, dependencies []metadataSvc.DatasetDependencyRecord) string {
	var builder strings.Builder
	builder.WriteString("# Dataset SQL History\n\n")
	if strings.TrimSpace(selected) != "" {
		builder.WriteString("Selected dataset: ")
		builder.WriteString(selected)
		builder.WriteString("\n\n")
	}
	builder.WriteString("Recent SQL runs\n")
	runCount := 0
	for _, run := range runs {
		if strings.TrimSpace(selected) != "" && run.RelPath != selected {
			continue
		}
		runCount++
		builder.WriteString(fmt.Sprintf("- %s | %s | %s | shown %d/%d | %d ms\n", formatDataTime(run.CompletedAt), run.RelPath, firstNonEmptyString(run.Status, "unknown"), run.ShownRows, run.MatchedRows, run.DurationMs))
		builder.WriteString("  SQL: ")
		builder.WriteString(compactDataLine(run.SQL, 180))
		builder.WriteString("\n")
		if strings.TrimSpace(firstNonEmptyString(run.Message, run.Error)) != "" {
			builder.WriteString("  Message: ")
			builder.WriteString(compactDataLine(firstNonEmptyString(run.Message, run.Error), 180))
			builder.WriteString("\n")
		}
	}
	if runCount == 0 {
		builder.WriteString("- No SQL runs found.\n")
	}
	builder.WriteString("\nDataset dependencies\n")
	if len(dependencies) == 0 {
		builder.WriteString("- No dependency records found.\n")
		return builder.String()
	}
	for _, dependency := range dependencies {
		builder.WriteString(fmt.Sprintf("- %s | %s %s %s:%s\n", formatDataTime(dependency.UpdatedAt), dependency.SourcePath, firstNonEmptyString(dependency.Relation, "links"), dependency.DependentKind, dependency.DependentRef))
		if len(dependency.Metadata) > 0 {
			parts := make([]string, 0, len(dependency.Metadata))
			for key, value := range dependency.Metadata {
				parts = append(parts, key+"="+value)
			}
			sort.Strings(parts)
			builder.WriteString("  Metadata: ")
			builder.WriteString(strings.Join(parts, ", "))
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func latestRebuildableDatasetDependency(dependencies []metadataSvc.DatasetDependencyRecord) (metadataSvc.DatasetDependencyRecord, bool) {
	for _, dependency := range dependencies {
		if canRebuildDatasetDependency(dependency) {
			return dependency, true
		}
	}
	return metadataSvc.DatasetDependencyRecord{}, false
}

func canRebuildDatasetDependency(dependency metadataSvc.DatasetDependencyRecord) bool {
	switch dependency.DependentKind {
	case "filter-export", "dataset-query-csv":
		return strings.TrimSpace(firstNonEmptyString(dependency.Metadata["query"], dependency.Metadata["filter"])) != ""
	case "sql-report", "dataset-sql-report":
		return strings.TrimSpace(firstNonEmptyString(dependency.Metadata["sql"], dependency.Metadata["query"])) != ""
	case "chart", "dashboard":
		return strings.TrimSpace(dependency.SourcePath) != ""
	case "sql-notebook", "sql-notebook-run":
		return strings.TrimSpace(dependency.SourcePath) != ""
	default:
		return false
	}
}

func notebookForDatasetDependency(notebooks []datasetsSvc.Notebook, dependency metadataSvc.DatasetDependencyRecord) (datasetsSvc.Notebook, bool) {
	targetID := strings.TrimSpace(firstNonEmptyString(dependency.Metadata["notebookID"], dependency.Metadata["notebook"], dependency.DependentRef))
	targetLabel := strings.TrimSpace(dependency.Metadata["label"])
	if targetID != "" && !strings.Contains(filepath.ToSlash(targetID), "/") {
		for _, notebook := range notebooks {
			if notebook.ID == targetID {
				return notebook, true
			}
		}
		return datasetsSvc.Notebook{}, false
	}
	if targetLabel != "" {
		for _, notebook := range notebooks {
			if strings.EqualFold(strings.TrimSpace(notebook.Label), targetLabel) {
				return notebook, true
			}
		}
	}
	if len(notebooks) > 0 {
		return notebooks[0], true
	}
	return datasetsSvc.Notebook{}, false
}

func latestReusableSQLRun(runs []metadataSvc.SQLRunRecord, selected string) (metadataSvc.SQLRunRecord, bool) {
	selected = strings.TrimSpace(selected)
	for _, run := range runs {
		if strings.TrimSpace(run.SQL) == "" {
			continue
		}
		if selected != "" && run.RelPath != selected {
			continue
		}
		return run, true
	}
	return metadataSvc.SQLRunRecord{}, false
}

func formatSQLRunReuse(title string, run metadataSvc.SQLRunRecord) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	builder.WriteString("Path: ")
	builder.WriteString(run.RelPath)
	builder.WriteString("\nEngine: ")
	builder.WriteString(firstNonEmptyString(run.Engine, "unknown"))
	builder.WriteString("\nStatus: ")
	builder.WriteString(firstNonEmptyString(run.Status, "unknown"))
	builder.WriteString(fmt.Sprintf("\nShown: %d/%d\nDuration: %d ms\n", run.ShownRows, run.MatchedRows, run.DurationMs))
	if strings.TrimSpace(firstNonEmptyString(run.Message, run.Error)) != "" {
		builder.WriteString("Message: ")
		builder.WriteString(firstNonEmptyString(run.Message, run.Error))
		builder.WriteString("\n")
	}
	builder.WriteString("\nSQL\n\n")
	builder.WriteString(strings.TrimSpace(run.SQL))
	builder.WriteString("\n")
	return builder.String()
}

func formatSQLRunReuseEmpty(selected string) string {
	if strings.TrimSpace(selected) == "" {
		return "# Dataset SQL History\n\nNo reusable SQL history entry found.\n"
	}
	return "# Dataset SQL History\n\nNo reusable SQL history entry found for " + selected + ".\n"
}

func connectorSourcePath(profileID string) string {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return ""
	}
	return "connector:" + profileID
}

func sqlHistorySources(selectedPath string, connectorProfileID string) []string {
	sources := []string{}
	selectedPath = strings.TrimSpace(selectedPath)
	if selectedPath != "" {
		sources = append(sources, selectedPath)
	}
	connectorPath := connectorSourcePath(connectorProfileID)
	if connectorPath != "" && connectorPath != selectedPath {
		sources = append(sources, connectorPath)
	}
	return sources
}

func primarySQLHistorySource(selectedPath string, connectorProfileID string) string {
	sources := sqlHistorySources(selectedPath, connectorProfileID)
	if len(sources) == 0 {
		return ""
	}
	return sources[0]
}

func isConnectorRun(run metadataSvc.SQLRunRecord) bool {
	return connectorProfileIDFromSourcePath(run.RelPath) != ""
}

func connectorProfileIDFromSourcePath(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return ""
	}
	lower := strings.ToLower(source)
	if !strings.HasPrefix(lower, "connector:") {
		return ""
	}
	return strings.TrimSpace(source[len("connector:"):])
}

func isSQLiteRun(run metadataSvc.SQLRunRecord) bool {
	engine := strings.ToLower(strings.TrimSpace(run.Engine))
	if strings.Contains(engine, "sqlite") {
		return true
	}
	lowerPath := strings.ToLower(strings.TrimSpace(run.RelPath))
	return strings.HasSuffix(lowerPath, ".sqlite") || strings.HasSuffix(lowerPath, ".sqlite3") || strings.HasSuffix(lowerPath, ".db")
}

func formatDatasetNotebooks(notebooks []datasetsSvc.Notebook) string {
	var builder strings.Builder
	builder.WriteString("# Dataset SQL Notebooks\n\n")
	if len(notebooks) == 0 {
		builder.WriteString("No saved notebooks for the selected dataset.\n")
		return builder.String()
	}
	for _, notebook := range notebooks {
		builder.WriteString(fmt.Sprintf("- %s | %s | %d cell(s) | updated %s\n", notebook.Label, notebook.ID, len(notebook.Cells), formatDataTime(notebook.UpdatedAt)))
		for _, cell := range notebook.Cells {
			builder.WriteString("  - ")
			builder.WriteString(firstNonEmptyString(cell.Label, cell.ID))
			builder.WriteString(" [")
			builder.WriteString(cell.Kind)
			builder.WriteString("]")
			if strings.TrimSpace(cell.SQL) != "" {
				builder.WriteString(": ")
				builder.WriteString(compactDataLine(cell.SQL, 180))
			}
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func formatNotebookRunResult(result datasetsSvc.NotebookRunResult) string {
	var builder strings.Builder
	builder.WriteString("# SQL Notebook Run\n\n")
	builder.WriteString("Notebook: ")
	builder.WriteString(result.Label)
	builder.WriteString("\nPath: ")
	builder.WriteString(result.RelPath)
	builder.WriteString(fmt.Sprintf("\nCells: %d\nDuration: %d ms\n", len(result.Cells), result.DurationMs))
	if result.Message != "" {
		builder.WriteString("\n")
		builder.WriteString(result.Message)
		builder.WriteString("\n")
	}
	for index, cell := range result.Cells {
		builder.WriteString(fmt.Sprintf("\n## Cell %d: %s [%s]\n\n", index+1, firstNonEmptyString(cell.Label, cell.CellID), cell.Kind))
		if strings.TrimSpace(cell.SQL) != "" {
			builder.WriteString("SQL: ")
			builder.WriteString(compactDataLine(cell.SQL, 220))
			builder.WriteString("\n")
		}
		if cell.Error != "" {
			builder.WriteString("Status: failed\nError: ")
			builder.WriteString(cell.Error)
			builder.WriteString("\n")
			continue
		}
		builder.WriteString(fmt.Sprintf("Status: success | shown %d/%d | %d ms\n", len(cell.SQLResult.Rows), cell.SQLResult.MatchedRows, cell.DurationMs))
		if len(cell.SQLResult.Plan) > 0 {
			builder.WriteString("Plan\n")
			for _, step := range cell.SQLResult.Plan {
				builder.WriteString("- ")
				builder.WriteString(step)
				builder.WriteString("\n")
			}
		}
		if len(cell.SQLResult.Columns) > 0 {
			builder.WriteString("\nRows\n")
			builder.WriteString(strings.Join(cell.SQLResult.Columns, "\t"))
			builder.WriteString("\n")
			for _, row := range cell.SQLResult.Rows {
				builder.WriteString(strings.Join(row, "\t"))
				builder.WriteString("\n")
			}
		}
		if cell.ChartResult.SVG != "" {
			builder.WriteString("\nChart\n")
			builder.WriteString(cell.ChartResult.Message)
			builder.WriteString(fmt.Sprintf("\nPoints: %d\n", len(cell.ChartResult.Points)))
		}
	}
	return builder.String()
}

func formatNotebookRowsTab(result datasetsSvc.NotebookRunResult) string {
	var builder strings.Builder
	builder.WriteString("# Notebook Rows\n\n")
	written := 0
	for index, cell := range result.Cells {
		if cell.Error != "" || len(cell.SQLResult.Columns) == 0 {
			continue
		}
		written++
		builder.WriteString(fmt.Sprintf("## Cell %d: %s\n\n", index+1, firstNonEmptyString(cell.Label, cell.CellID)))
		builder.WriteString(fmt.Sprintf("Shown %d/%d row(s)\n\n", len(cell.SQLResult.Rows), cell.SQLResult.MatchedRows))
		builder.WriteString(strings.Join(cell.SQLResult.Columns, "\t"))
		builder.WriteString("\n")
		for _, row := range cell.SQLResult.Rows {
			builder.WriteString(strings.Join(row, "\t"))
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}
	if written == 0 {
		builder.WriteString("No tabular rows were produced by the latest notebook run.\n")
	}
	return builder.String()
}

func formatNotebookPlanTab(result datasetsSvc.NotebookRunResult) string {
	var builder strings.Builder
	builder.WriteString("# Notebook Plan\n\n")
	written := 0
	for index, cell := range result.Cells {
		if cell.Error != "" {
			written++
			builder.WriteString(fmt.Sprintf("## Cell %d: %s\n\n", index+1, firstNonEmptyString(cell.Label, cell.CellID)))
			builder.WriteString("Status: failed\n")
			builder.WriteString(cell.Error)
			builder.WriteString("\n\n")
			continue
		}
		if len(cell.SQLResult.Plan) == 0 {
			continue
		}
		written++
		builder.WriteString(fmt.Sprintf("## Cell %d: %s\n\n", index+1, firstNonEmptyString(cell.Label, cell.CellID)))
		for _, step := range cell.SQLResult.Plan {
			builder.WriteString("- ")
			builder.WriteString(step)
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}
	if written == 0 {
		builder.WriteString("No execution plan was produced by the latest notebook run.\n")
	}
	return builder.String()
}

func formatNotebookChartsTab(result datasetsSvc.NotebookRunResult) string {
	var builder strings.Builder
	builder.WriteString("# Notebook Charts\n\n")
	written := 0
	for index, cell := range result.Cells {
		if cell.Error != "" || cell.ChartResult.SVG == "" {
			continue
		}
		written++
		builder.WriteString(fmt.Sprintf("## Cell %d: %s\n\n", index+1, firstNonEmptyString(cell.Label, cell.CellID)))
		builder.WriteString("Mode: ")
		builder.WriteString(cell.ChartResult.Mode)
		builder.WriteString("\nCategory: ")
		builder.WriteString(cell.ChartResult.CategoryColumn)
		if cell.ChartResult.ValueColumn != "" {
			builder.WriteString("\nValue: ")
			builder.WriteString(cell.ChartResult.ValueColumn)
		}
		builder.WriteString(fmt.Sprintf("\nPoints: %d\n\n", len(cell.ChartResult.Points)))
		if strings.TrimSpace(cell.ChartResult.Message) != "" {
			builder.WriteString(cell.ChartResult.Message)
			builder.WriteString("\n\n")
		}
		builder.WriteString(cell.ChartResult.SVG)
		builder.WriteString("\n\n")
	}
	if written == 0 {
		builder.WriteString("No charts were produced by the latest notebook run.\n")
	}
	return builder.String()
}

func formatDataTime(value time.Time) string {
	if value.IsZero() {
		return "unknown time"
	}
	return value.Local().Format("2006-01-02 15:04")
}

func compactDataLine(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func sqlRunRecord(result datasetsSvc.SQLResult, relPath string, sqlText string, runErr error) metadataSvc.SQLRunRecord {
	status := "success"
	message := result.Message
	errorText := ""
	completed := result.CompletedAt
	if completed.IsZero() {
		completed = result.StartedAt
	}
	if runErr != nil {
		if isDataJobCanceled(runErr) {
			status = "canceled"
		} else {
			status = "failed"
		}
		errorText = runErr.Error()
		message = errorText
	}
	return metadataSvc.SQLRunRecord{
		RelPath:     firstNonEmptyString(result.RelPath, relPath),
		SQL:         strings.TrimSpace(firstNonEmptyString(result.SQL, sqlText)),
		Engine:      firstNonEmptyString(result.Engine, "native-dataset-sql"),
		Status:      status,
		RowCount:    result.TotalRows,
		MatchedRows: result.MatchedRows,
		ShownRows:   len(result.Rows),
		Message:     message,
		Error:       errorText,
		StartedAt:   firstNonZeroTime(result.StartedAt, completed),
		CompletedAt: completed,
		DurationMs:  result.DurationMs,
	}
}

func sqliteSQLRunRecord(result dbconnectorSvc.SQLiteQueryResult, relPath string, sqlText string, started time.Time, runErr error) metadataSvc.SQLRunRecord {
	status := "success"
	message := result.Message
	errorText := ""
	completed := time.Now().UTC()
	if runErr != nil {
		if isDataJobCanceled(runErr) || isSQLiteQueryCanceled(runErr) {
			status = "canceled"
		} else {
			status = "failed"
		}
		errorText = runErr.Error()
		message = errorText
	}
	duration := result.DurationMs
	if duration == 0 && !started.IsZero() {
		duration = completed.Sub(started).Milliseconds()
	}
	return metadataSvc.SQLRunRecord{
		RelPath:     firstNonEmptyString(result.RelPath, relPath),
		SQL:         strings.TrimSpace(firstNonEmptyString(result.SQL, sqlText)),
		Engine:      firstNonEmptyString(result.Engine, "sqlite-readonly"),
		Status:      status,
		RowCount:    result.TotalRows,
		MatchedRows: result.TotalRows,
		ShownRows:   len(result.Rows),
		Message:     message,
		Error:       errorText,
		StartedAt:   firstNonZeroTime(started, completed),
		CompletedAt: completed,
		DurationMs:  duration,
	}
}

func datasetDependencyRecord(source string, sqlRun metadataSvc.SQLRunRecord) metadataSvc.DatasetDependencyRecord {
	return metadataSvc.DatasetDependencyRecord{
		SourcePath:    source,
		DependentKind: "sql-run",
		DependentRef:  sqlRun.ID,
		Relation:      "reads",
		Metadata: map[string]string{
			"engine": sqlRun.Engine,
			"sql":    sqlRun.SQL,
		},
		CreatedAt: sqlRun.StartedAt,
		UpdatedAt: sqlRun.CompletedAt,
	}
}

func sqliteDependencyRecord(source string, sqlRun metadataSvc.SQLRunRecord) metadataSvc.DatasetDependencyRecord {
	return metadataSvc.DatasetDependencyRecord{
		SourcePath:    source,
		DependentKind: "sqlite-query",
		DependentRef:  sqlRun.ID,
		Relation:      "reads",
		Metadata: map[string]string{
			"engine": sqlRun.Engine,
			"sql":    sqlRun.SQL,
		},
		CreatedAt: sqlRun.StartedAt,
		UpdatedAt: sqlRun.CompletedAt,
	}
}

func sqliteSavedQueryDependencyRecord(source string, query datasetsSvc.SavedQuery) metadataSvc.DatasetDependencyRecord {
	updated := query.UpdatedAt
	if updated.IsZero() {
		updated = time.Now().UTC()
	}
	return metadataSvc.DatasetDependencyRecord{
		SourcePath:    source,
		DependentKind: "sqlite-query-snippet",
		DependentRef:  query.Label,
		Relation:      "saves",
		Metadata: map[string]string{
			"kind":  query.Kind,
			"query": query.Query,
		},
		CreatedAt: updated,
		UpdatedAt: updated,
	}
}

func sqliteArtifactDependencyRecord(source string, sqlRun metadataSvc.SQLRunRecord, artifact artifactsSvc.Artifact) metadataSvc.DatasetDependencyRecord {
	return metadataSvc.DatasetDependencyRecord{
		SourcePath:    source,
		DependentKind: "sqlite-query-artifact",
		DependentRef:  artifact.RelPath,
		Relation:      "exports",
		Metadata: map[string]string{
			"engine":   sqlRun.Engine,
			"sql":      sqlRun.SQL,
			"artifact": artifact.RelPath,
		},
		CreatedAt: sqlRun.StartedAt,
		UpdatedAt: sqlRun.CompletedAt,
	}
}

func datasetQueryArtifactDependencyRecord(result datasetsSvc.QueryResult, artifact artifactsSvc.Artifact) metadataSvc.DatasetDependencyRecord {
	now := firstNonZeroTime(artifact.GeneratedAt, artifact.CreatedAt, time.Now().UTC())
	return metadataSvc.DatasetDependencyRecord{
		SourcePath:    result.RelPath,
		DependentKind: "filter-export",
		DependentRef:  artifact.RelPath,
		Relation:      "exports",
		Metadata: map[string]string{
			"artifact": artifact.RelPath,
			"format":   result.Format,
			"query":    result.Query,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func datasetSQLArtifactDependencyRecord(result datasetsSvc.SQLResult, sqlRun metadataSvc.SQLRunRecord, artifact artifactsSvc.Artifact) metadataSvc.DatasetDependencyRecord {
	now := firstNonZeroTime(result.CompletedAt, artifact.GeneratedAt, artifact.CreatedAt, time.Now().UTC())
	return metadataSvc.DatasetDependencyRecord{
		SourcePath:    result.RelPath,
		DependentKind: "sql-report",
		DependentRef:  artifact.RelPath,
		Relation:      "exports",
		Metadata: map[string]string{
			"artifact": artifact.RelPath,
			"engine":   result.Engine,
			"sql":      result.SQL,
			"sqlRunId": sqlRun.ID,
		},
		CreatedAt: firstNonZeroTime(result.StartedAt, now),
		UpdatedAt: now,
	}
}

func chartArtifactDependencyRecord(result datasetsSvc.QueryResult, chart datasetsSvc.ChartResult, artifact artifactsSvc.Artifact) metadataSvc.DatasetDependencyRecord {
	now := firstNonZeroTime(artifact.GeneratedAt, artifact.CreatedAt, time.Now().UTC())
	return metadataSvc.DatasetDependencyRecord{
		SourcePath:    chart.RelPath,
		DependentKind: "chart",
		DependentRef:  artifact.RelPath,
		Relation:      "exports",
		Metadata: map[string]string{
			"artifact": artifact.RelPath,
			"category": chart.CategoryColumn,
			"format":   result.Format,
			"mode":     chart.Mode,
			"query":    result.Query,
			"value":    chart.ValueColumn,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func dashboardArtifactDependencyRecord(result datasetsSvc.QueryResult, dashboard datasetsSvc.DashboardResult, artifact artifactsSvc.Artifact) metadataSvc.DatasetDependencyRecord {
	now := firstNonZeroTime(artifact.GeneratedAt, artifact.CreatedAt, time.Now().UTC())
	return metadataSvc.DatasetDependencyRecord{
		SourcePath:    dashboard.RelPath,
		DependentKind: "dashboard",
		DependentRef:  artifact.RelPath,
		Relation:      "exports",
		Metadata: map[string]string{
			"artifact": artifact.RelPath,
			"category": dashboard.Chart.CategoryColumn,
			"format":   result.Format,
			"mode":     dashboard.Chart.Mode,
			"query":    result.Query,
			"value":    dashboard.Chart.ValueColumn,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func datasetQueryArtifactInput(result datasetsSvc.QueryResult) artifactsSvc.DatasetQueryReport {
	return artifactsSvc.DatasetQueryReport{
		SourcePath:  result.RelPath,
		Query:       result.Query,
		Format:      result.Format,
		Columns:     append([]string{}, result.Columns...),
		Rows:        copyTableRows(result.Rows),
		TotalRows:   result.TotalRows,
		MatchedRows: result.MatchedRows,
		Truncated:   result.Truncated,
		Message:     result.Message,
	}
}

func datasetSQLArtifactInput(result datasetsSvc.SQLResult) artifactsSvc.DatasetSQLReport {
	return artifactsSvc.DatasetSQLReport{
		SourcePath:  result.RelPath,
		SQL:         result.SQL,
		Engine:      result.Engine,
		Columns:     append([]string{}, result.Columns...),
		Rows:        copyTableRows(result.Rows),
		TotalRows:   result.TotalRows,
		MatchedRows: result.MatchedRows,
		ShownRows:   len(result.Rows),
		DurationMs:  result.DurationMs,
		Truncated:   result.Truncated,
		Plan:        append([]string{}, result.Plan...),
		Message:     result.Message,
	}
}

func notebookDependencyRecord(source string, notebook datasetsSvc.Notebook) metadataSvc.DatasetDependencyRecord {
	return metadataSvc.DatasetDependencyRecord{
		SourcePath:    source,
		DependentKind: "sql-notebook",
		DependentRef:  notebook.ID,
		Relation:      "saves",
		Metadata: map[string]string{
			"notebookID": notebook.ID,
			"label":      notebook.Label,
			"cells":      fmt.Sprintf("%d", len(notebook.Cells)),
		},
		CreatedAt: notebook.CreatedAt,
		UpdatedAt: notebook.UpdatedAt,
	}
}

func firstNotebookSQL(notebook datasetsSvc.Notebook) string {
	for _, cell := range notebook.Cells {
		if cell.Kind == "sql" && strings.TrimSpace(cell.SQL) != "" {
			return strings.TrimSpace(cell.SQL)
		}
	}
	return ""
}

func notebookCellsFromEditor(value string) []datasetsSvc.NotebookCell {
	lines := strings.Split(value, "\n")
	cells := []datasetsSvc.NotebookCell{}
	currentKind := "sql"
	currentLabel := "Query"
	currentLines := []string{}
	flush := func() {
		sqlText := strings.TrimSpace(strings.Join(currentLines, "\n"))
		if currentKind == "sql" && sqlText == "" {
			currentLines = []string{}
			return
		}
		cells = append(cells, datasetsSvc.NotebookCell{
			ID:    fmt.Sprintf("cell-%d", len(cells)+1),
			Kind:  currentKind,
			Label: currentLabel,
			SQL:   sqlText,
		})
		currentLines = []string{}
	}
	for _, line := range lines {
		if label, ok := notebookDirective(line, "cell"); ok {
			flush()
			currentKind = "sql"
			currentLabel = label
			continue
		}
		if label, ok := notebookDirective(line, "chart"); ok {
			flush()
			currentKind = "chart"
			currentLabel = label
			continue
		}
		currentLines = append(currentLines, line)
	}
	flush()
	return cells
}

func appendNotebookCellTemplate(current string, kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	directive := "cell"
	labelPrefix := "Query"
	if kind == "chart" {
		directive = "chart"
		labelPrefix = "Chart"
	}
	index := len(notebookCellsFromEditor(current)) + 1
	block := fmt.Sprintf("-- %s: %s %d\nselect * from dataset limit 50", directive, labelPrefix, index)
	if strings.TrimSpace(current) == "" {
		return block
	}
	return strings.TrimRight(current, "\r\n") + "\n\n" + block
}

func notebookDirective(line string, kind string) (string, bool) {
	prefix := "-- " + kind + ":"
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(strings.ToLower(trimmed), prefix) {
		return "", false
	}
	label := strings.TrimSpace(trimmed[len(prefix):])
	if label == "" {
		label = "Cell"
		if kind == "chart" {
			label = "Chart"
		}
	}
	return label, true
}

func formatNotebookForEditor(notebook datasetsSvc.Notebook) string {
	parts := []string{}
	for _, cell := range notebook.Cells {
		kind := cell.Kind
		if kind != "chart" {
			kind = "cell"
		}
		parts = append(parts, fmt.Sprintf("-- %s: %s\n%s", kind, firstNonEmptyString(cell.Label, cell.ID), strings.TrimSpace(cell.SQL)))
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func (v *View) persistNotebookRunSQL(result datasetsSvc.NotebookRunResult) {
	if v.metadataStore == nil {
		return
	}
	for _, cell := range result.Cells {
		if cell.Error != "" || strings.TrimSpace(cell.SQLResult.SQL) == "" {
			continue
		}
		record := v.metadataStore.NormalizeSQLRunRecord(sqlRunRecord(cell.SQLResult, result.RelPath, cell.SQL, nil))
		if err := v.metadataStore.SaveSQLRun(record); err != nil {
			v.addActivity("Could not persist notebook SQL run metadata: " + err.Error())
			continue
		}
		v.persistDatasetDependency(datasetDependencyRecord(result.RelPath, record))
	}
}

func lastNotebookQueryResult(result datasetsSvc.NotebookRunResult) datasetsSvc.QueryResult {
	for index := len(result.Cells) - 1; index >= 0; index-- {
		if result.Cells[index].Error == "" && result.Cells[index].SQLResult.RelPath != "" {
			return result.Cells[index].SQLResult.QueryResult
		}
	}
	return datasetsSvc.QueryResult{}
}

func lastNotebookChartResult(result datasetsSvc.NotebookRunResult) datasetsSvc.ChartResult {
	for index := len(result.Cells) - 1; index >= 0; index-- {
		if result.Cells[index].Error == "" && result.Cells[index].ChartResult.SVG != "" {
			return result.Cells[index].ChartResult
		}
	}
	return datasetsSvc.ChartResult{}
}

func sqliteQueryAsDatasetResult(result dbconnectorSvc.SQLiteQueryResult) datasetsSvc.QueryResult {
	return datasetsSvc.QueryResult{
		RelPath:     result.RelPath,
		Format:      "SQLite",
		Query:       result.SQL,
		Columns:     result.Columns,
		Rows:        result.Rows,
		TotalRows:   result.TotalRows,
		MatchedRows: result.TotalRows,
		Truncated:   result.Truncated,
		Message:     result.Message,
	}
}

func notebookLabelForDataset(relPath string) string {
	name := strings.TrimSpace(relPath)
	if name == "" {
		return "SQL Notebook"
	}
	return "SQL Notebook - " + name
}

func (v *View) persistDatasetDependency(record metadataSvc.DatasetDependencyRecord) {
	if v.metadataStore == nil {
		return
	}
	if err := v.metadataStore.SaveDatasetDependency(record); err != nil {
		v.addActivity("Could not persist dataset dependency metadata: " + err.Error())
	}
}

func (v *View) persistSQLiteArtifactLineage(result dbconnectorSvc.SQLiteQueryResult, artifact artifactsSvc.Artifact) {
	if v.metadataStore == nil {
		return
	}
	record := v.metadataStore.NormalizeSQLRunRecord(sqliteSQLRunRecord(result, result.RelPath, result.SQL, time.Now().UTC(), nil))
	record.ArtifactPath = artifact.RelPath
	if err := v.metadataStore.SaveSQLRun(record); err != nil {
		v.addActivity("Could not persist SQLite artifact SQL metadata: " + err.Error())
		return
	}
	v.persistDatasetDependency(sqliteArtifactDependencyRecord(result.RelPath, record, artifact))
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Now().UTC()
}
