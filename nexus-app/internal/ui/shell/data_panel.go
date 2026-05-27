package shell

import (
	"fmt"
	"sort"
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
	dashboardButton := widget.NewButtonWithIcon("Preview dashboard", theme.ColorPaletteIcon(), v.previewDatasetDashboard)
	exportDashboardButton := widget.NewButtonWithIcon("Export dashboard", theme.DocumentSaveIcon(), v.exportDatasetDashboardArtifact)
	historyButton := widget.NewButtonWithIcon("SQL history", theme.HistoryIcon(), v.showDatasetSQLHistory)
	saveNotebookButton := widget.NewButtonWithIcon("Save notebook", theme.DocumentSaveIcon(), v.saveSelectedDatasetNotebook)
	loadNotebookButton := widget.NewButtonWithIcon("Load notebook", theme.FolderOpenIcon(), v.loadSelectedDatasetNotebook)
	actions := container.NewHBox(profileButton, queryButton, sqlButton, saveNotebookButton, loadNotebookButton, chartButton, exportChartButton, dashboardButton, exportDashboardButton, historyButton)
	queryBar := container.NewBorder(nil, nil, nil, actions, v.dataQueryEntry)
	header := container.NewVBox(v.dataProfileStatus, queryBar)
	detail := container.NewScroll(v.dataProfileDetail)
	detail.SetMinSize(fyne.NewSize(320, 130))
	return container.NewBorder(header, nil, nil, nil, detail)
}

func (v *View) saveSelectedDatasetNotebook() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before saving SQL notebooks.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.dataProfileStatus.SetText("Select a dataset before saving a SQL notebook.")
		return
	}
	sqlText := strings.TrimSpace(v.dataQueryEntry.Text)
	if sqlText == "" {
		v.dataProfileStatus.SetText("Write a SELECT query before saving a SQL notebook.")
		return
	}
	saved, err := v.datasetService.SaveNotebook(workspace.Root, datasetsSvc.NotebookSaveRequest{
		RelPath: selected,
		Label:   notebookLabelForDataset(selected),
		Cells: []datasetsSvc.NotebookCell{{
			ID:    "cell-1",
			Kind:  "sql",
			Label: "Query",
			SQL:   sqlText,
		}},
	})
	if err != nil {
		v.dataProfileStatus.SetText("Notebook save failed for " + selected)
		dialog.ShowError(err, v.window)
		return
	}
	v.persistDatasetDependency(notebookDependencyRecord(selected, saved))
	v.dataProfileStatus.SetText(fmt.Sprintf("Saved SQL notebook %s with %d cell(s).", saved.Label, len(saved.Cells)))
	v.dataProfileDetail.SetText(formatDatasetNotebooks([]datasetsSvc.Notebook{saved}))
	v.addActivity("Saved SQL notebook for " + selected + ".")
}

func (v *View) loadSelectedDatasetNotebook() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before loading SQL notebooks.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.dataProfileStatus.SetText("Select a dataset before loading SQL notebooks.")
		return
	}
	notebooks, err := v.datasetService.ListNotebooks(workspace.Root, selected)
	if err != nil {
		v.dataProfileStatus.SetText("Notebook load failed for " + selected)
		dialog.ShowError(err, v.window)
		return
	}
	if len(notebooks) == 0 {
		v.dataProfileStatus.SetText("No saved SQL notebooks for " + selected + ".")
		v.dataProfileDetail.SetText(formatDatasetNotebooks(nil))
		return
	}
	if sqlText := firstNotebookSQL(notebooks[0]); sqlText != "" {
		v.dataQueryEntry.SetText(sqlText)
	}
	v.dataProfileStatus.SetText(fmt.Sprintf("Loaded %d SQL notebook(s) for %s.", len(notebooks), selected))
	v.dataProfileDetail.SetText(formatDatasetNotebooks(notebooks))
	v.addActivity("Loaded SQL notebooks for " + selected + ".")
}

func (v *View) showDatasetSQLHistory() {
	if v.metadataStore == nil {
		v.dataProfileStatus.SetText("Open a workspace before inspecting dataset SQL history.")
		return
	}
	selected := selectedPathOrEmpty(v)
	runs, err := v.metadataStore.ListSQLRuns(50)
	if err != nil {
		v.dataProfileStatus.SetText("SQL history unavailable.")
		dialog.ShowError(err, v.window)
		return
	}
	dependencies, err := v.metadataStore.ListDatasetDependencies(selected, 50)
	if err != nil {
		v.dataProfileStatus.SetText("Dataset dependency history unavailable.")
		dialog.ShowError(err, v.window)
		return
	}
	v.dataProfileStatus.SetText(datasetHistoryStatus(selected, runs, dependencies))
	v.dataProfileDetail.SetText(formatDatasetHistory(selected, runs, dependencies))
	v.addActivity("Loaded dataset SQL history.")
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
	v.dataLastDashboard = datasetsSvc.DashboardResult{}
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
	v.dataLastDashboard = datasetsSvc.DashboardResult{}
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
	v.dataLastDashboard = datasetsSvc.DashboardResult{}
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

func (v *View) previewDatasetDashboard() {
	result, ok := v.ensureDatasetQueryForChart()
	if !ok {
		return
	}
	dashboard, err := datasetsSvc.BuildDashboard(result)
	if err != nil {
		v.dataProfileStatus.SetText("Dashboard preview failed for " + result.RelPath)
		dialog.ShowError(err, v.window)
		return
	}
	v.dataLastChart = dashboard.Chart
	v.dataLastDashboard = dashboard
	v.dataProfileStatus.SetText(dashboard.Message)
	v.dataProfileDetail.SetText(formatDatasetDashboard(dashboard))
	v.addActivity("Previewed dashboard for " + dashboard.RelPath + ".")
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

func (v *View) exportDatasetDashboardArtifact() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before exporting dashboard artifacts.")
		return
	}
	if v.dataLastDashboard.SVG == "" {
		v.previewDatasetDashboard()
		if v.dataLastDashboard.SVG == "" {
			return
		}
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	artifact, err := store.WriteChartArtifact(dashboardArtifactInput(v.dataLastDashboard))
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.persistArtifactRecord(artifact)
	v.dataProfileStatus.SetText("Exported dashboard " + artifact.RelPath)
	v.addActivity(artifact.Message)
	v.refreshArtifactsWithQuery("kind:dashboard")
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

func notebookDependencyRecord(source string, notebook datasetsSvc.Notebook) metadataSvc.DatasetDependencyRecord {
	return metadataSvc.DatasetDependencyRecord{
		SourcePath:    source,
		DependentKind: "sql-notebook",
		DependentRef:  notebook.ID,
		Relation:      "saves",
		Metadata: map[string]string{
			"label": notebook.Label,
			"cells": fmt.Sprintf("%d", len(notebook.Cells)),
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
