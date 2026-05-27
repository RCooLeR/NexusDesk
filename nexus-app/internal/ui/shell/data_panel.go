package shell

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	artifactsSvc "nexusdesk/internal/services/artifacts"
	datasetsSvc "nexusdesk/internal/services/datasets"
	metadataSvc "nexusdesk/internal/services/metadata"
)

func (v *View) newDataPanel() fyne.CanvasObject {
	profileButton := widget.NewButtonWithIcon("Profile selected", theme.SearchIcon(), v.profileSelectedDataset)
	queryButton := widget.NewButtonWithIcon("Run query", theme.MediaPlayIcon(), func() {
		v.querySelectedDataset(v.dataQueryEntry.Text)
	})
	sqlButton := widget.NewButtonWithIcon("Run SQL", theme.ComputerIcon(), func() {
		v.runSelectedDatasetSQL(v.dataQueryEntry.Text)
	})
	v.dataQueryEntry.OnSubmitted = func(query string) {
		v.querySelectedDataset(query)
	}
	chartButton := widget.NewButtonWithIcon("Preview chart", theme.ViewFullScreenIcon(), v.previewDatasetChart)
	exportChartButton := widget.NewButtonWithIcon("Export chart", theme.DocumentSaveIcon(), v.exportDatasetChartArtifact)
	actions := container.NewHBox(profileButton, queryButton, sqlButton, chartButton, exportChartButton)
	queryBar := container.NewBorder(nil, nil, nil, actions, v.dataQueryEntry)
	header := container.NewVBox(v.dataProfileStatus, queryBar)
	detail := container.NewScroll(v.dataProfileDetail)
	detail.SetMinSize(fyne.NewSize(320, 130))
	return container.NewBorder(header, nil, nil, nil, detail)
}

func (v *View) runSelectedDatasetSQL(sqlText string) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before running dataset SQL.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.dataProfileStatus.SetText("Select a CSV, TSV, JSON, NDJSON, XLSX, or log file first.")
		return
	}
	result, err := v.datasetService.QuerySQL(workspace.Root, selected, sqlText)
	record := sqlRunRecord(result, selected, sqlText, err)
	if v.metadataStore != nil {
		record = v.metadataStore.NormalizeSQLRunRecord(record)
		if saveErr := v.metadataStore.SaveSQLRun(record); saveErr != nil {
			v.addActivity("Could not persist SQL run metadata: " + saveErr.Error())
		} else if err == nil {
			v.persistDatasetDependency(datasetDependencyRecord(selected, record))
		}
	}
	if err != nil {
		v.dataProfileStatus.SetText("SQL failed for " + selected)
		dialog.ShowError(err, v.window)
		return
	}
	v.dataProfileStatus.SetText(sqlStatus(result))
	v.dataProfileDetail.SetText(formatDatasetSQLResult(result))
	v.dataLastQuery = result.QueryResult
	v.dataLastChart = datasetsSvc.ChartResult{}
	v.addActivity("Ran native dataset SQL for " + result.RelPath + ".")
}

func (v *View) profileSelectedDataset() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before profiling data.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.dataProfileStatus.SetText("Select a CSV, TSV, JSON, NDJSON, XLSX, Parquet, or log file first.")
		return
	}
	profile, err := v.datasetService.Profile(workspace.Root, selected)
	if err != nil {
		v.dataProfileStatus.SetText("Profile failed for " + selected)
		dialog.ShowError(err, v.window)
		return
	}
	v.dataProfileStatus.SetText(profileStatus(profile))
	v.dataProfileDetail.SetText(formatDatasetProfile(profile))
	v.dataLastQuery = datasetsSvc.QueryResult{}
	v.dataLastChart = datasetsSvc.ChartResult{}
	v.addActivity("Profiled dataset " + profile.RelPath + ".")
}

func (v *View) querySelectedDataset(query string) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before querying data.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.dataProfileStatus.SetText("Select a CSV, TSV, JSON, NDJSON, XLSX, or log file first.")
		return
	}
	result, err := v.datasetService.Query(workspace.Root, selected, query)
	if err != nil {
		v.dataProfileStatus.SetText("Query failed for " + selected)
		dialog.ShowError(err, v.window)
		return
	}
	v.dataProfileStatus.SetText(queryStatus(result))
	v.dataProfileDetail.SetText(formatDatasetQueryResult(result))
	v.dataLastQuery = result
	v.dataLastChart = datasetsSvc.ChartResult{}
	v.addActivity("Queried dataset " + result.RelPath + ".")
}

func (v *View) previewDatasetChart() {
	result, ok := v.ensureDatasetQueryForChart()
	if !ok {
		return
	}
	chart, err := datasetsSvc.BuildChart(result)
	if err != nil {
		v.dataProfileStatus.SetText("Chart preview failed for " + result.RelPath)
		dialog.ShowError(err, v.window)
		return
	}
	v.dataLastChart = chart
	v.dataProfileStatus.SetText(chart.Message)
	v.dataProfileDetail.SetText(formatDatasetChart(chart))
	v.addActivity("Previewed chart for " + chart.RelPath + ".")
}

func (v *View) exportDatasetChartArtifact() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before exporting chart artifacts.")
		return
	}
	if v.dataLastChart.SVG == "" {
		v.previewDatasetChart()
		if v.dataLastChart.SVG == "" {
			return
		}
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	artifact, err := store.WriteChartArtifact(chartArtifactInput(v.dataLastChart))
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.persistArtifactRecord(artifact)
	v.dataProfileStatus.SetText("Exported chart " + artifact.RelPath)
	v.addActivity(artifact.Message)
	v.refreshArtifactsWithQuery("kind:chart")
}

func (v *View) ensureDatasetQueryForChart() (datasetsSvc.QueryResult, bool) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before charting data.")
		return datasetsSvc.QueryResult{}, false
	}
	selected := selectedPathOrEmpty(v)
	if v.dataLastQuery.RelPath != "" && (selected == "" || selected == v.dataLastQuery.RelPath) {
		return v.dataLastQuery, true
	}
	if selected == "" {
		v.dataProfileStatus.SetText("Select a CSV, TSV, JSON, or XLSX file before charting data.")
		return datasetsSvc.QueryResult{}, false
	}
	result, err := v.datasetService.Query(workspace.Root, selected, v.dataQueryEntry.Text)
	if err != nil {
		v.dataProfileStatus.SetText("Chart query failed for " + selected)
		dialog.ShowError(err, v.window)
		return datasetsSvc.QueryResult{}, false
	}
	v.dataLastQuery = result
	return result, true
}

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

func sqlRunRecord(result datasetsSvc.SQLResult, relPath string, sqlText string, runErr error) metadataSvc.SQLRunRecord {
	status := "success"
	message := result.Message
	errorText := ""
	completed := result.CompletedAt
	if completed.IsZero() {
		completed = result.StartedAt
	}
	if runErr != nil {
		status = "failed"
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

func (v *View) persistDatasetDependency(record metadataSvc.DatasetDependencyRecord) {
	if v.metadataStore == nil {
		return
	}
	if err := v.metadataStore.SaveDatasetDependency(record); err != nil {
		v.addActivity("Could not persist dataset dependency metadata: " + err.Error())
	}
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
