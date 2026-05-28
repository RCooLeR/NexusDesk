package shell

import (
	"context"
	"errors"
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
	dbconnectorSvc "nexusdesk/internal/services/dbconnector"
	jobsSvc "nexusdesk/internal/services/jobs"
	metadataSvc "nexusdesk/internal/services/metadata"
)

const (
	datasetProfileJobKind        = "dataset-profile"
	datasetQueryJobKind          = "dataset-query"
	datasetSQLJobKind            = "dataset-sql"
	datasetNotebookRunJobKind    = "dataset-notebook-run"
	datasetNotebookExportJobKind = "dataset-notebook-export"
	datasetChartPreviewJobKind   = "dataset-chart-preview"
	datasetChartExportJobKind    = "dataset-chart-export"
	datasetDashboardPreviewKind  = "dataset-dashboard-preview"
	datasetDashboardExportKind   = "dataset-dashboard-export"
	datasetSQLiteArtifactJobKind = "sqlite-artifact-export"
)

func (v *View) newDataPanel() fyne.CanvasObject {
	profileButton := widget.NewButtonWithIcon("Profile selected", theme.SearchIcon(), v.profileSelectedDataset)
	queryButton := widget.NewButtonWithIcon("Run query", theme.MediaPlayIcon(), func() {
		v.querySelectedDataset(v.dataQueryEntry.Text)
	})
	sqlButton := widget.NewButtonWithIcon("Run SQL", theme.ComputerIcon(), func() {
		v.runSelectedDatasetSQL(v.dataQueryEntry.Text)
	})
	sqliteButton := widget.NewButtonWithIcon("Inspect SQLite", theme.StorageIcon(), v.inspectSelectedSQLite)
	sqliteQueryButton := widget.NewButtonWithIcon("Run SQLite", theme.MediaPlayIcon(), func() {
		v.runSelectedSQLiteQuery(v.dataQueryEntry.Text)
	})
	cancelSQLiteButton := widget.NewButtonWithIcon("Cancel SQLite", theme.CancelIcon(), v.cancelActiveSQLiteQuery)
	v.dataQueryEntry.OnSubmitted = func(query string) {
		v.querySelectedDataset(query)
	}
	chartButton := widget.NewButtonWithIcon("Preview chart", theme.ViewFullScreenIcon(), v.previewDatasetChart)
	exportChartButton := widget.NewButtonWithIcon("Export chart", theme.DocumentSaveIcon(), v.exportDatasetChartArtifact)
	dashboardButton := widget.NewButtonWithIcon("Preview dashboard", theme.ColorPaletteIcon(), v.previewDatasetDashboard)
	exportDashboardButton := widget.NewButtonWithIcon("Export dashboard", theme.DocumentSaveIcon(), v.exportDatasetDashboardArtifact)
	historyButton := widget.NewButtonWithIcon("SQL history", theme.HistoryIcon(), v.showDatasetSQLHistory)
	saveSQLiteQueryButton := widget.NewButtonWithIcon("Save SQLite query", theme.DocumentSaveIcon(), v.saveSelectedSQLiteQuery)
	savedSQLiteQueriesButton := widget.NewButtonWithIcon("Saved SQLite", theme.ListIcon(), v.showSavedSQLiteQueries)
	exportSQLiteCSVButton := widget.NewButtonWithIcon("Export SQLite CSV", theme.DownloadIcon(), v.exportSQLiteQueryCSVArtifact)
	exportSQLiteReportButton := widget.NewButtonWithIcon("Export SQLite report", theme.DocumentSaveIcon(), v.exportSQLiteQueryMarkdownArtifact)
	copyRowButton := widget.NewButtonWithIcon("Copy row", theme.ContentCopyIcon(), v.copySelectedDataRow)
	copyCellButton := widget.NewButtonWithIcon("Copy cell", theme.ContentCopyIcon(), v.copySelectedDataCell)
	addSQLCellButton := widget.NewButtonWithIcon("Add SQL cell", theme.ContentAddIcon(), func() {
		v.insertNotebookCellTemplate("cell")
	})
	addChartCellButton := widget.NewButtonWithIcon("Add chart cell", theme.ContentAddIcon(), func() {
		v.insertNotebookCellTemplate("chart")
	})
	saveNotebookButton := widget.NewButtonWithIcon("Save notebook", theme.DocumentSaveIcon(), v.saveSelectedDatasetNotebook)
	loadNotebookButton := widget.NewButtonWithIcon("Load notebook", theme.FolderOpenIcon(), v.loadSelectedDatasetNotebook)
	runNotebookButton := widget.NewButtonWithIcon("Run notebook", theme.MediaPlayIcon(), v.runLatestDatasetNotebook)
	exportNotebookButton := widget.NewButtonWithIcon("Export notebook", theme.DocumentSaveIcon(), v.exportDatasetNotebookArtifact)
	refreshNotebookCellsButton := widget.NewButtonWithIcon("Cells", theme.ViewRefreshIcon(), v.refreshNotebookCellSelector)
	moveNotebookCellUpButton := widget.NewButtonWithIcon("Move up", theme.MoveUpIcon(), func() {
		v.moveSelectedNotebookCell(-1)
	})
	moveNotebookCellDownButton := widget.NewButtonWithIcon("Move down", theme.MoveDownIcon(), func() {
		v.moveSelectedNotebookCell(1)
	})
	deleteNotebookCellButton := widget.NewButtonWithIcon("Delete cell", theme.DeleteIcon(), v.deleteSelectedNotebookCell)
	notebookLabel := widget.NewEntry()
	notebookLabel.SetPlaceHolder("Notebook label")
	notebookLabel.SetText("SQL Notebook")
	notebookCellSelect := widget.NewSelect([]string{}, func(choice string) {
		v.dataNotebookCellIndex = notebookCellOptionIndex(v.dataNotebookCellSelect.Options, choice)
	})
	notebookCellSelect.PlaceHolder = "Notebook cells"
	v.dataNotebookLabel = notebookLabel
	v.dataNotebookCellSelect = notebookCellSelect
	reuseSQLButton := widget.NewButtonWithIcon("Use latest SQL", theme.ContentPasteIcon(), v.reuseLatestDatasetSQLRun)
	rerunSQLButton := widget.NewButtonWithIcon("Rerun latest SQL", theme.MediaReplayIcon(), v.rerunLatestDatasetSQLRun)
	connectorSelect := widget.NewSelect([]string{}, func(choice string) {
		v.dataConnectorProfileID = v.dataConnectorOptions[choice]
	})
	connectorSelect.PlaceHolder = "External profile"
	v.dataConnectorProfile = connectorSelect
	listConnectorProfilesButton := widget.NewButtonWithIcon("List profiles", theme.ViewRefreshIcon(), v.listConnectorProfiles)
	saveConnectorProfileButton := widget.NewButtonWithIcon("Save profile", theme.DocumentCreateIcon(), v.promptSaveConnectorProfile)
	deleteConnectorProfileButton := widget.NewButtonWithIcon("Delete profile", theme.DeleteIcon(), v.deleteSelectedConnectorProfile)
	validateConnectorSQLButton := widget.NewButtonWithIcon("Validate SQL", theme.ConfirmIcon(), v.validateExternalConnectorSQL)
	inspectConnectorProfileButton := widget.NewButtonWithIcon("Inspect profile", theme.InfoIcon(), v.inspectSelectedConnectorProfile)
	testConnectorProfileButton := widget.NewButtonWithIcon("Test profile", theme.SearchIcon(), v.testSelectedConnectorProfile)
	runConnectorQueryButton := widget.NewButtonWithIcon("Run profile query", theme.MediaPlayIcon(), v.runSelectedConnectorProfileQuery)
	cancelConnectorQueryButton := widget.NewButtonWithIcon("Cancel profile query", theme.CancelIcon(), v.cancelActiveConnectorQuery)
	notebookControls := container.NewVBox(
		dataActionStrip(addSQLCellButton, addChartCellButton, saveNotebookButton, loadNotebookButton, runNotebookButton, exportNotebookButton),
		container.NewGridWithColumns(2, notebookLabel, notebookCellSelect),
		dataActionStrip(refreshNotebookCellsButton, moveNotebookCellUpButton, moveNotebookCellDownButton, deleteNotebookCellButton),
	)
	externalConnectorControls := container.NewVBox(
		container.NewGridWithColumns(2, widget.NewLabel("Profile"), connectorSelect),
		dataActionStrip(listConnectorProfilesButton, saveConnectorProfileButton, deleteConnectorProfileButton, validateConnectorSQLButton, inspectConnectorProfileButton, testConnectorProfileButton, runConnectorQueryButton, cancelConnectorQueryButton),
	)
	actions := container.NewAppTabs(
		container.NewTabItem("Source", dataActionStrip(profileButton, queryButton, sqlButton, sqliteButton, sqliteQueryButton, cancelSQLiteButton)),
		container.NewTabItem("Notebook", notebookControls),
		container.NewTabItem("Visuals", dataActionStrip(chartButton, exportChartButton, dashboardButton, exportDashboardButton)),
		container.NewTabItem("History", dataActionStrip(historyButton, reuseSQLButton, rerunSQLButton, copyCellButton, copyRowButton, saveSQLiteQueryButton, savedSQLiteQueriesButton, exportSQLiteCSVButton, exportSQLiteReportButton)),
		container.NewTabItem("External DB", externalConnectorControls),
	)
	actions.SetTabLocation(container.TabLocationTop)
	queryBar := container.NewBorder(nil, nil, nil, nil, v.dataQueryEntry)
	header := container.NewVBox(v.dataProfileStatus, queryBar, actions)
	summary := container.NewScroll(v.dataProfileDetail)
	v.dataRowsContainer = container.NewMax(v.dataRowsDetail)
	rows := container.NewScroll(v.dataRowsContainer)
	plan := container.NewScroll(v.dataPlanDetail)
	charts := container.NewScroll(v.dataChartDetail)
	for _, scroll := range []*container.Scroll{summary, rows, plan, charts} {
		scroll.SetMinSize(fyne.NewSize(320, 130))
	}
	v.dataResultTabs = container.NewAppTabs(
		container.NewTabItem("Summary", summary),
		container.NewTabItem("Rows", rows),
		container.NewTabItem("Plan", plan),
		container.NewTabItem("Charts", charts),
	)
	if profiles, err := v.loadConnectorProfiles(); err == nil {
		v.refreshConnectorProfileSelect(profiles)
	}
	return container.NewBorder(header, nil, nil, nil, v.dataResultTabs)
}

func dataActionStrip(actions ...fyne.CanvasObject) fyne.CanvasObject {
	row := container.NewHBox(actions...)
	scroll := container.NewHScroll(row)
	scroll.SetMinSize(fyne.NewSize(320, 44))
	return scroll
}

func (v *View) setDataSummary(summary string) {
	v.dataProfileDetail.SetText(summary)
	v.setDataRowsText("")
	if v.dataPlanDetail != nil {
		v.dataPlanDetail.SetText("")
	}
	if v.dataChartDetail != nil {
		v.dataChartDetail.SetText("")
	}
	if v.dataResultTabs != nil && len(v.dataResultTabs.Items) > 0 {
		v.dataResultTabs.Select(v.dataResultTabs.Items[0])
	}
}

func (v *View) setDataNotebookRunTabs(result datasetsSvc.NotebookRunResult) {
	v.dataProfileDetail.SetText(formatNotebookRunResult(result))
	v.setDataRowsText(formatNotebookRowsTab(result))
	if v.dataPlanDetail != nil {
		v.dataPlanDetail.SetText(formatNotebookPlanTab(result))
	}
	if v.dataChartDetail != nil {
		v.dataChartDetail.SetText(formatNotebookChartsTab(result))
	}
	if v.dataResultTabs != nil && len(v.dataResultTabs.Items) > 0 {
		v.dataResultTabs.Select(v.dataResultTabs.Items[0])
	}
}

func (v *View) insertNotebookCellTemplate(kind string) {
	updated := appendNotebookCellTemplate(v.dataQueryEntry.Text, kind)
	v.dataQueryEntry.SetText(updated)
	if kind == "chart" {
		v.dataProfileStatus.SetText("Added chart notebook cell template.")
		v.refreshNotebookCellSelector()
		return
	}
	v.dataProfileStatus.SetText("Added SQL notebook cell template.")
	v.refreshNotebookCellSelector()
}

func (v *View) runSelectedSQLiteQuery(sqlText string) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before running SQLite queries.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.dataProfileStatus.SetText("Select a .sqlite, .sqlite3, or .db file before running SQLite queries.")
		return
	}
	request := dbconnectorSvc.NormalizeSQLiteQueryRequest(dbconnectorSvc.SQLiteQueryRequest{
		RelPath:        selected,
		SQL:            sqlText,
		ResultLimit:    dbconnectorSvc.DefaultSQLiteRows,
		TimeoutSeconds: dbconnectorSvc.DefaultSQLiteTimeoutSeconds,
	})
	started := time.Now().UTC()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	queryID := v.startSQLiteQuery(cancel)
	v.dataProfileStatus.SetText(fmt.Sprintf("%s: SQLite query running, cap %d, timeout %ds.", selected, request.ResultLimit, request.TimeoutSeconds))
	v.addActivity("Started read-only SQLite query for " + selected + ".")
	go func() {
		defer cancel()
		result, err := v.dbconnectorService.QueryWorkspaceSQLiteContext(ctx, workspace.Root, request)
		fyne.Do(func() {
			v.finishSelectedSQLiteQuery(queryID, selected, sqlText, started, result, err)
		})
	}()
}

func (v *View) finishSelectedSQLiteQuery(queryID string, selected string, sqlText string, started time.Time, result dbconnectorSvc.SQLiteQueryResult, err error) {
	v.clearSQLiteQuery(queryID)
	if strings.TrimSpace(sqlText) != "" {
		record := sqliteSQLRunRecord(result, selected, sqlText, started, err)
		if v.metadataStore != nil {
			record = v.metadataStore.NormalizeSQLRunRecord(record)
			if saveErr := v.metadataStore.SaveSQLRun(record); saveErr != nil {
				v.addActivity("Could not persist SQLite query metadata: " + saveErr.Error())
			} else if err == nil {
				v.persistDatasetDependency(sqliteDependencyRecord(selected, record))
			}
		}
	}
	if err != nil {
		if isSQLiteQueryCanceled(err) {
			v.dataProfileStatus.SetText("SQLite query cancelled for " + selected)
			v.addActivity("Cancelled read-only SQLite query for " + selected + ".")
			return
		}
		v.dataProfileStatus.SetText("SQLite query failed for " + selected)
		dialog.ShowError(err, v.window)
		return
	}
	v.dataProfileStatus.SetText(sqliteQueryStatus(result))
	v.setDataSummary(formatSQLiteQueryResult(result))
	v.setDataRowsGrid(result.Columns, result.Rows)
	v.dataLastQuery = sqliteQueryAsDatasetResult(result)
	v.dataLastSQLiteQuery = result
	v.dataLastChart = datasetsSvc.ChartResult{}
	v.dataLastDashboard = datasetsSvc.DashboardResult{}
	v.addActivity("Ran read-only SQLite query for " + result.RelPath + ".")
}

func (v *View) startSQLiteQuery(cancel context.CancelFunc) string {
	id := fmt.Sprintf("sqlite-%d", time.Now().UTC().UnixNano())
	v.dataSQLiteQueryMu.Lock()
	previousCancel := v.dataSQLiteCancel
	v.dataSQLiteCancel = cancel
	v.dataSQLiteQueryID = id
	v.dataSQLiteQueryMu.Unlock()
	if previousCancel != nil {
		previousCancel()
	}
	return id
}

func (v *View) clearSQLiteQuery(id string) {
	v.dataSQLiteQueryMu.Lock()
	defer v.dataSQLiteQueryMu.Unlock()
	if v.dataSQLiteQueryID != id {
		return
	}
	v.dataSQLiteCancel = nil
	v.dataSQLiteQueryID = ""
}

func (v *View) cancelActiveSQLiteQuery() {
	v.dataSQLiteQueryMu.Lock()
	cancel := v.dataSQLiteCancel
	queryID := v.dataSQLiteQueryID
	v.dataSQLiteQueryMu.Unlock()
	if cancel == nil {
		v.dataProfileStatus.SetText("No SQLite connector query is currently running.")
		return
	}
	cancel()
	v.dataProfileStatus.SetText("SQLite query cancellation requested.")
	v.addActivity("SQLite query cancel requested: " + queryID + ".")
}

func isSQLiteQueryCanceled(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "canceled") || strings.Contains(message, "cancelled")
}

func (v *View) inspectSelectedSQLite() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before inspecting SQLite databases.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.dataProfileStatus.SetText("Select a .sqlite, .sqlite3, or .db file before inspecting schema.")
		return
	}
	metadata, err := v.dbconnectorService.InspectWorkspaceSQLite(workspace.Root, selected)
	if err != nil {
		v.dataProfileStatus.SetText("SQLite inspection failed for " + selected)
		dialog.ShowError(err, v.window)
		return
	}
	v.dataProfileStatus.SetText(metadata.Message)
	v.setDataSummary(formatSQLiteMetadata(metadata))
	v.dataLastQuery = datasetsSvc.QueryResult{}
	v.dataLastSQLiteQuery = dbconnectorSvc.SQLiteQueryResult{}
	v.dataLastChart = datasetsSvc.ChartResult{}
	v.dataLastDashboard = datasetsSvc.DashboardResult{}
	v.addActivity("Inspected SQLite database " + metadata.RelPath + ".")
}

func (v *View) saveSelectedSQLiteQuery() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before saving SQLite queries.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.dataProfileStatus.SetText("Select a .sqlite, .sqlite3, or .db file before saving SQLite queries.")
		return
	}
	sqlText := strings.TrimSpace(v.dataQueryEntry.Text)
	if sqlText == "" {
		v.dataProfileStatus.SetText("Enter a read-only SQLite query before saving it.")
		return
	}
	saved, err := v.datasetService.SaveQuery(workspace.Root, selected, sqlText, "", "sqlite-sql")
	if err != nil {
		v.dataProfileStatus.SetText("SQLite query save failed for " + selected)
		dialog.ShowError(err, v.window)
		return
	}
	v.persistDatasetDependency(sqliteSavedQueryDependencyRecord(selected, saved))
	v.dataProfileStatus.SetText("Saved SQLite query " + saved.Label + ".")
	v.setDataSummary(formatSavedQueries("Saved SQLite Queries", []datasetsSvc.SavedQuery{saved}))
	v.addActivity("Saved SQLite query for " + selected + ".")
}

func (v *View) showSavedSQLiteQueries() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before listing SQLite queries.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.dataProfileStatus.SetText("Select a SQLite source before listing saved queries.")
		return
	}
	queries, err := v.datasetService.ListSavedQueries(workspace.Root, selected, "sqlite-sql")
	if err != nil {
		v.dataProfileStatus.SetText("Saved SQLite queries unavailable for " + selected)
		dialog.ShowError(err, v.window)
		return
	}
	v.dataProfileStatus.SetText(fmt.Sprintf("%s: %d saved SQLite query snippet(s).", selected, len(queries)))
	v.setDataSummary(formatSavedQueries("Saved SQLite Queries", queries))
	v.addActivity("Listed saved SQLite queries for " + selected + ".")
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
	label := notebookLabelForDataset(selected)
	if v.dataNotebookLabel != nil && strings.TrimSpace(v.dataNotebookLabel.Text) != "" {
		label = strings.TrimSpace(v.dataNotebookLabel.Text)
	}
	saved, err := v.datasetService.SaveNotebook(workspace.Root, datasetsSvc.NotebookSaveRequest{
		ID:      v.dataActiveNotebookID,
		RelPath: selected,
		Label:   label,
		Cells:   notebookCellsFromEditor(sqlText),
	})
	if err != nil {
		v.dataProfileStatus.SetText("Notebook save failed for " + selected)
		dialog.ShowError(err, v.window)
		return
	}
	v.persistDatasetDependency(notebookDependencyRecord(selected, saved))
	v.dataActiveNotebookID = saved.ID
	if v.dataNotebookLabel != nil {
		v.dataNotebookLabel.SetText(saved.Label)
	}
	v.refreshNotebookCellSelector()
	v.dataLastNotebookRun = datasetsSvc.NotebookRunResult{}
	v.dataProfileStatus.SetText(fmt.Sprintf("Saved SQL notebook %s with %d cell(s).", saved.Label, len(saved.Cells)))
	v.setDataSummary(formatDatasetNotebooks([]datasetsSvc.Notebook{saved}))
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
		v.setDataSummary(formatDatasetNotebooks(nil))
		return
	}
	v.dataActiveNotebookID = notebooks[0].ID
	if v.dataNotebookLabel != nil {
		v.dataNotebookLabel.SetText(notebooks[0].Label)
	}
	v.dataQueryEntry.SetText(formatNotebookForEditor(notebooks[0]))
	v.refreshNotebookCellSelector()
	v.dataLastNotebookRun = datasetsSvc.NotebookRunResult{}
	v.dataProfileStatus.SetText(fmt.Sprintf("Loaded %d SQL notebook(s) for %s.", len(notebooks), selected))
	v.setDataSummary(formatDatasetNotebooks(notebooks))
	v.addActivity("Loaded SQL notebooks for " + selected + ".")
}

func (v *View) runLatestDatasetNotebook() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before running SQL notebooks.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.dataProfileStatus.SetText("Select a dataset before running SQL notebooks.")
		return
	}
	jobLabel := datasetNotebookRunJobLabel(selected)
	job, ctx := v.jobService.Start(datasetNotebookRunJobKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Path: "+selected)
	v.dataProfileStatus.SetText("Running SQL notebook as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	go func() {
		result, notebookLabel, err := v.runNotebookForDataset(ctx, root, selected)
		fyne.Do(func() {
			v.finishDatasetNotebookRunJob(job.ID, selected, notebookLabel, result, err)
		})
	}()
}

func (v *View) showDatasetSQLHistory() {
	if v.metadataStore == nil {
		v.dataProfileStatus.SetText("Open a workspace before inspecting dataset SQL history.")
		return
	}
	selected := primarySQLHistorySource(selectedPathOrEmpty(v), v.dataConnectorProfileID)
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
	v.setDataSummary(formatDatasetHistory(selected, runs, dependencies))
	v.addActivity("Loaded dataset SQL history.")
}

func (v *View) reuseLatestDatasetSQLRun() {
	run, ok := v.latestReusableSQLRun()
	if !ok {
		return
	}
	v.dataQueryEntry.SetText(run.SQL)
	v.dataProfileStatus.SetText("Loaded latest SQL history entry for " + run.RelPath + ".")
	v.setDataSummary(formatSQLRunReuse("Loaded latest SQL for editing", run))
	v.addActivity("Loaded SQL history entry for " + run.RelPath + ".")
}

func (v *View) rerunLatestDatasetSQLRun() {
	run, ok := v.latestReusableSQLRun()
	if !ok {
		return
	}
	v.dataQueryEntry.SetText(run.SQL)
	v.dataProfileStatus.SetText("Rerunning latest SQL history entry for " + run.RelPath + ".")
	if isConnectorRun(run) {
		connectorID := connectorProfileIDFromSourcePath(run.RelPath)
		if connectorID == "" {
			v.dataProfileStatus.SetText("Connector SQL history entry is missing a profile reference.")
			return
		}
		v.dataConnectorProfileID = connectorID
		if v.dataConnectorProfile != nil {
			for label, id := range v.dataConnectorOptions {
				if id == connectorID {
					v.dataConnectorProfile.SetSelected(label)
					break
				}
			}
		}
		v.runSelectedConnectorProfileQuery()
		return
	}
	if isSQLiteRun(run) {
		v.runSelectedSQLiteQuery(run.SQL)
		return
	}
	v.runSelectedDatasetSQL(run.SQL)
}

func (v *View) latestReusableSQLRun() (metadataSvc.SQLRunRecord, bool) {
	if v.metadataStore == nil {
		v.dataProfileStatus.SetText("Open a workspace before reusing SQL history.")
		return metadataSvc.SQLRunRecord{}, false
	}
	sources := sqlHistorySources(selectedPathOrEmpty(v), v.dataConnectorProfileID)
	if len(sources) == 0 {
		v.dataProfileStatus.SetText("Select a dataset, SQLite source, or external connector profile before reusing SQL history.")
		return metadataSvc.SQLRunRecord{}, false
	}
	runs, err := v.metadataStore.ListSQLRuns(100)
	if err != nil {
		v.dataProfileStatus.SetText("SQL history unavailable.")
		dialog.ShowError(err, v.window)
		return metadataSvc.SQLRunRecord{}, false
	}
	for _, source := range sources {
		run, ok := latestReusableSQLRun(runs, source)
		if ok {
			return run, true
		}
	}
	v.dataProfileStatus.SetText("No reusable SQL history entry found for " + sources[0] + ".")
	v.setDataSummary(formatSQLRunReuseEmpty(sources[0]))
	return metadataSvc.SQLRunRecord{}, false
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
	jobLabel := datasetSQLJobLabel(selected)
	job, ctx := v.jobService.Start(datasetSQLJobKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Path: "+selected)
	v.jobService.AppendLog(job.ID, fmt.Sprintf("SQL bytes: %d", len(strings.TrimSpace(sqlText))))
	v.dataProfileStatus.SetText("Running native dataset SQL as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	go func() {
		result, err := v.datasetService.QuerySQLContext(ctx, root, selected, sqlText)
		fyne.Do(func() {
			v.finishDatasetSQLJob(job.ID, selected, sqlText, result, err)
		})
	}()
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
	jobLabel := datasetProfileJobLabel(selected)
	job, ctx := v.jobService.Start(datasetProfileJobKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Path: "+selected)
	v.dataProfileStatus.SetText("Profiling dataset as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	go func() {
		profile, err := v.datasetService.ProfileContext(ctx, root, selected)
		fyne.Do(func() {
			v.finishDatasetProfileJob(job.ID, selected, profile, err)
		})
	}()
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
	jobLabel := datasetQueryJobLabel(selected)
	job, ctx := v.jobService.Start(datasetQueryJobKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Path: "+selected)
	if strings.TrimSpace(query) != "" {
		v.jobService.AppendLog(job.ID, fmt.Sprintf("Query bytes: %d", len(strings.TrimSpace(query))))
	}
	v.dataProfileStatus.SetText("Running dataset query as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	go func() {
		result, err := v.datasetService.QueryContext(ctx, root, selected, query)
		fyne.Do(func() {
			v.finishDatasetQueryJob(job.ID, selected, result, err)
		})
	}()
}

func (v *View) finishDatasetProfileJob(jobID string, selected string, profile datasetsSvc.Profile, err error) {
	if err != nil {
		if isDataJobCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Dataset profile cancelled.", nil)
			v.dataProfileStatus.SetText("Dataset profile cancelled for " + selected + ".")
			v.addActivity("Cancelled dataset profile for " + selected + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Dataset profile failed.", err)
			v.dataProfileStatus.SetText("Profile failed for " + selected)
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	v.jobService.AppendLog(jobID, fmt.Sprintf("Rows=%d columns=%d", profile.Rows, len(profile.Columns)))
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, firstNonEmptyString(profileStatus(profile), "Dataset profile completed."), nil)
	v.dataProfileStatus.SetText(profileStatus(profile))
	v.setDataSummary(formatDatasetProfile(profile))
	v.dataLastQuery = datasetsSvc.QueryResult{}
	v.dataLastSQLiteQuery = dbconnectorSvc.SQLiteQueryResult{}
	v.dataLastChart = datasetsSvc.ChartResult{}
	v.dataLastDashboard = datasetsSvc.DashboardResult{}
	v.addActivity("Profiled dataset " + profile.RelPath + ".")
	v.refreshJobs()
}

func (v *View) finishDatasetQueryJob(jobID string, selected string, result datasetsSvc.QueryResult, err error) {
	if err != nil {
		if isDataJobCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Dataset query cancelled.", nil)
			v.dataProfileStatus.SetText("Dataset query cancelled for " + selected + ".")
			v.addActivity("Cancelled dataset query for " + selected + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Dataset query failed.", err)
			v.dataProfileStatus.SetText("Query failed for " + selected)
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	v.jobService.AppendLog(jobID, fmt.Sprintf("Rows: shown=%d matched=%d", len(result.Rows), result.MatchedRows))
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, firstNonEmptyString(result.Message, "Dataset query completed."), nil)
	v.dataProfileStatus.SetText(queryStatus(result))
	v.setDataSummary(formatDatasetQueryResult(result))
	v.setDataRowsGrid(result.Columns, result.Rows)
	v.dataLastQuery = result
	v.dataLastSQLiteQuery = dbconnectorSvc.SQLiteQueryResult{}
	v.dataLastChart = datasetsSvc.ChartResult{}
	v.dataLastDashboard = datasetsSvc.DashboardResult{}
	v.addActivity("Queried dataset " + result.RelPath + ".")
	v.refreshJobs()
}

func (v *View) finishDatasetSQLJob(jobID string, selected string, sqlText string, result datasetsSvc.SQLResult, err error) {
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
		if isDataJobCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Native dataset SQL cancelled.", nil)
			v.dataProfileStatus.SetText("Native dataset SQL cancelled for " + selected + ".")
			v.addActivity("Cancelled native dataset SQL for " + selected + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Native dataset SQL failed.", err)
			v.dataProfileStatus.SetText("SQL failed for " + selected)
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	v.jobService.AppendLog(jobID, fmt.Sprintf("Rows: shown=%d matched=%d duration=%dms", len(result.QueryResult.Rows), result.QueryResult.MatchedRows, result.DurationMs))
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, firstNonEmptyString(result.Message, "Native dataset SQL completed."), nil)
	v.dataProfileStatus.SetText(sqlStatus(result))
	v.setDataSummary(formatDatasetSQLResult(result))
	v.setDataRowsGrid(result.QueryResult.Columns, result.QueryResult.Rows)
	v.dataLastQuery = result.QueryResult
	v.dataLastSQLiteQuery = dbconnectorSvc.SQLiteQueryResult{}
	v.dataLastChart = datasetsSvc.ChartResult{}
	v.dataLastDashboard = datasetsSvc.DashboardResult{}
	v.addActivity("Ran native dataset SQL for " + result.RelPath + ".")
	v.refreshJobs()
}

func datasetProfileJobLabel(relPath string) string {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "Dataset profile"
	}
	return "Dataset profile (" + relPath + ")"
}

func datasetQueryJobLabel(relPath string) string {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "Dataset query"
	}
	return "Dataset query (" + relPath + ")"
}

func datasetSQLJobLabel(relPath string) string {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "Dataset SQL query"
	}
	return "Dataset SQL query (" + relPath + ")"
}

func datasetNotebookRunJobLabel(relPath string) string {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "SQL notebook run"
	}
	return "SQL notebook run (" + relPath + ")"
}

func datasetNotebookExportJobLabel(relPath string) string {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "SQL notebook export"
	}
	return "SQL notebook export (" + relPath + ")"
}

func datasetChartPreviewJobLabel(relPath string) string {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "Dataset chart preview"
	}
	return "Dataset chart preview (" + relPath + ")"
}

func datasetChartExportJobLabel(relPath string) string {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "Dataset chart export"
	}
	return "Dataset chart export (" + relPath + ")"
}

func datasetDashboardPreviewJobLabel(relPath string) string {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "Dataset dashboard preview"
	}
	return "Dataset dashboard preview (" + relPath + ")"
}

func datasetDashboardExportJobLabel(relPath string) string {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "Dataset dashboard export"
	}
	return "Dataset dashboard export (" + relPath + ")"
}

func sqliteArtifactExportJobLabel(relPath string, kind string) string {
	base := "SQLite query export"
	kind = strings.TrimSpace(strings.ToLower(kind))
	if kind == "csv" {
		base = "SQLite CSV export"
	}
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return base
	}
	return base + " (" + relPath + ")"
}

func isDataJobCanceled(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func (v *View) previewDatasetChart() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before charting data.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if strings.TrimSpace(selected) == "" && strings.TrimSpace(v.dataLastQuery.RelPath) == "" {
		v.dataProfileStatus.SetText("Select a CSV, TSV, JSON, or XLSX file before charting data.")
		return
	}
	jobLabel := datasetChartPreviewJobLabel(selected)
	job, ctx := v.jobService.Start(datasetChartPreviewJobKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Path: "+firstNonEmptyString(selected, v.dataLastQuery.RelPath))
	v.dataProfileStatus.SetText("Previewing chart as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	cachedQuery := v.dataLastQuery
	queryText := v.dataQueryEntry.Text
	go func() {
		result, chart, err := v.buildDatasetChartPreview(ctx, root, selected, cachedQuery, queryText)
		fyne.Do(func() {
			v.finishDatasetChartPreviewJob(job.ID, selected, result, chart, err)
		})
	}()
}

func (v *View) previewDatasetDashboard() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before charting data.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if strings.TrimSpace(selected) == "" && strings.TrimSpace(v.dataLastQuery.RelPath) == "" {
		v.dataProfileStatus.SetText("Select a CSV, TSV, JSON, or XLSX file before charting data.")
		return
	}
	jobLabel := datasetDashboardPreviewJobLabel(selected)
	job, ctx := v.jobService.Start(datasetDashboardPreviewKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Path: "+firstNonEmptyString(selected, v.dataLastQuery.RelPath))
	v.dataProfileStatus.SetText("Previewing dashboard as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	cachedQuery := v.dataLastQuery
	queryText := v.dataQueryEntry.Text
	go func() {
		result, dashboard, err := v.buildDatasetDashboardPreview(ctx, root, selected, cachedQuery, queryText)
		fyne.Do(func() {
			v.finishDatasetDashboardPreviewJob(job.ID, selected, result, dashboard, err)
		})
	}()
}

func (v *View) exportDatasetChartArtifact() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before exporting chart artifacts.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if strings.TrimSpace(selected) == "" && strings.TrimSpace(v.dataLastQuery.RelPath) == "" {
		v.dataProfileStatus.SetText("Select a CSV, TSV, JSON, or XLSX file before exporting chart artifacts.")
		return
	}
	jobLabel := datasetChartExportJobLabel(selected)
	job, ctx := v.jobService.Start(datasetChartExportJobKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Path: "+firstNonEmptyString(selected, v.dataLastQuery.RelPath))
	v.dataProfileStatus.SetText("Exporting chart artifact as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	cachedQuery := v.dataLastQuery
	cachedChart := v.dataLastChart
	queryText := v.dataQueryEntry.Text
	go func() {
		result, chart, artifact, err := v.buildDatasetChartArtifactForExport(ctx, root, selected, cachedQuery, cachedChart, queryText)
		fyne.Do(func() {
			v.finishDatasetChartExportJob(job.ID, selected, result, chart, artifact, err)
		})
	}()
}

func (v *View) exportDatasetDashboardArtifact() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before exporting dashboard artifacts.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if strings.TrimSpace(selected) == "" && strings.TrimSpace(v.dataLastQuery.RelPath) == "" {
		v.dataProfileStatus.SetText("Select a CSV, TSV, JSON, or XLSX file before exporting dashboard artifacts.")
		return
	}
	jobLabel := datasetDashboardExportJobLabel(selected)
	job, ctx := v.jobService.Start(datasetDashboardExportKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Path: "+firstNonEmptyString(selected, v.dataLastQuery.RelPath))
	v.dataProfileStatus.SetText("Exporting dashboard artifact as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	cachedQuery := v.dataLastQuery
	cachedDashboard := v.dataLastDashboard
	queryText := v.dataQueryEntry.Text
	go func() {
		result, dashboard, artifact, err := v.buildDatasetDashboardArtifactForExport(ctx, root, selected, cachedQuery, cachedDashboard, queryText)
		fyne.Do(func() {
			v.finishDatasetDashboardExportJob(job.ID, selected, result, dashboard, artifact, err)
		})
	}()
}

func (v *View) exportDatasetNotebookArtifact() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before exporting SQL notebook artifacts.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.dataProfileStatus.SetText("Select a dataset before exporting SQL notebook artifacts.")
		return
	}
	cachedRun := v.dataLastNotebookRun
	jobLabel := datasetNotebookExportJobLabel(selected)
	job, ctx := v.jobService.Start(datasetNotebookExportJobKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Path: "+selected)
	v.dataProfileStatus.SetText("Exporting SQL notebook artifact as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	go func() {
		result, notebookLabel, artifact, reranNotebook, err := v.buildNotebookArtifactForExport(ctx, root, selected, cachedRun)
		fyne.Do(func() {
			v.finishDatasetNotebookExportJob(job.ID, selected, notebookLabel, result, artifact, reranNotebook, err)
		})
	}()
}

func (v *View) runNotebookForDataset(ctx context.Context, root string, selected string) (datasetsSvc.NotebookRunResult, string, error) {
	notebooks, err := v.datasetService.ListNotebooks(root, selected)
	if err != nil {
		return datasetsSvc.NotebookRunResult{}, "", err
	}
	if len(notebooks) == 0 {
		return datasetsSvc.NotebookRunResult{}, "", fmt.Errorf("no saved SQL notebooks for %s", selected)
	}
	notebook := notebooks[0]
	result, err := v.datasetService.RunNotebookContext(ctx, root, notebook)
	if err != nil {
		return datasetsSvc.NotebookRunResult{}, notebook.Label, err
	}
	return result, notebook.Label, nil
}

func (v *View) finishDatasetNotebookRunJob(jobID string, selected string, notebookLabel string, result datasetsSvc.NotebookRunResult, err error) {
	if err != nil {
		if isDataJobCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "SQL notebook run cancelled.", nil)
			v.dataProfileStatus.SetText("SQL notebook run cancelled for " + selected + ".")
			v.addActivity("Cancelled SQL notebook run for " + selected + ".")
		} else if strings.Contains(strings.ToLower(err.Error()), "no saved sql notebooks") {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "SQL notebook run failed.", err)
			v.dataProfileStatus.SetText("No saved SQL notebooks for " + selected + ".")
			v.setDataSummary(formatDatasetNotebooks(nil))
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "SQL notebook run failed.", err)
			v.dataProfileStatus.SetText("Notebook run failed for " + selected)
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	v.jobService.AppendLog(jobID, fmt.Sprintf("Cells=%d duration=%dms", len(result.Cells), result.DurationMs))
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, firstNonEmptyString(result.Message, "SQL notebook run completed."), nil)
	v.persistNotebookRunSQL(result)
	v.dataProfileStatus.SetText(result.Message)
	v.setDataNotebookRunTabs(result)
	v.dataLastQuery = lastNotebookQueryResult(result)
	v.dataLastChart = lastNotebookChartResult(result)
	v.dataLastDashboard = datasetsSvc.DashboardResult{}
	v.dataLastNotebookRun = result
	v.addActivity("Ran SQL notebook " + firstNonEmptyString(notebookLabel, result.Label) + ".")
	v.refreshJobs()
}

func (v *View) buildNotebookArtifactForExport(
	ctx context.Context,
	root string,
	selected string,
	cachedRun datasetsSvc.NotebookRunResult,
) (datasetsSvc.NotebookRunResult, string, artifactsSvc.Artifact, bool, error) {
	result := cachedRun
	notebookLabel := ""
	reranNotebook := false
	if result.RelPath == "" || (selected != "" && result.RelPath != selected) {
		rerunResult, label, err := v.runNotebookForDataset(ctx, root, selected)
		if err != nil {
			return datasetsSvc.NotebookRunResult{}, label, artifactsSvc.Artifact{}, false, err
		}
		result = rerunResult
		notebookLabel = label
		reranNotebook = true
	}
	select {
	case <-ctx.Done():
		return datasetsSvc.NotebookRunResult{}, notebookLabel, artifactsSvc.Artifact{}, reranNotebook, ctx.Err()
	default:
	}
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		return datasetsSvc.NotebookRunResult{}, notebookLabel, artifactsSvc.Artifact{}, reranNotebook, err
	}
	artifact, err := store.WriteNotebookRunReport(notebookRunArtifactInput(result))
	if err != nil {
		return datasetsSvc.NotebookRunResult{}, notebookLabel, artifactsSvc.Artifact{}, reranNotebook, err
	}
	select {
	case <-ctx.Done():
		return datasetsSvc.NotebookRunResult{}, notebookLabel, artifactsSvc.Artifact{}, reranNotebook, ctx.Err()
	default:
	}
	return result, notebookLabel, artifact, reranNotebook, nil
}

func (v *View) finishDatasetNotebookExportJob(
	jobID string,
	selected string,
	notebookLabel string,
	result datasetsSvc.NotebookRunResult,
	artifact artifactsSvc.Artifact,
	reranNotebook bool,
	err error,
) {
	if err != nil {
		if isDataJobCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "SQL notebook artifact export cancelled.", nil)
			v.dataProfileStatus.SetText("SQL notebook artifact export cancelled for " + selected + ".")
			v.addActivity("Cancelled SQL notebook artifact export for " + selected + ".")
		} else if strings.Contains(strings.ToLower(err.Error()), "no saved sql notebooks") {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "SQL notebook artifact export failed.", err)
			v.dataProfileStatus.SetText("No saved SQL notebooks for " + selected + ".")
			v.setDataSummary(formatDatasetNotebooks(nil))
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "SQL notebook artifact export failed.", err)
			v.dataProfileStatus.SetText("SQL notebook artifact export failed for " + selected)
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	artifact.JobID = jobID
	v.jobService.AppendLog(jobID, "Artifact: "+artifact.RelPath)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Created "+artifact.RelPath+".", nil)
	if reranNotebook {
		v.persistNotebookRunSQL(result)
		v.dataLastQuery = lastNotebookQueryResult(result)
		v.dataLastChart = lastNotebookChartResult(result)
		v.dataLastDashboard = datasetsSvc.DashboardResult{}
		v.dataLastNotebookRun = result
		v.addActivity("Ran SQL notebook " + firstNonEmptyString(notebookLabel, result.Label) + " before export.")
	}
	v.persistArtifactRecord(artifact)
	v.dataProfileStatus.SetText("Exported SQL notebook run " + artifact.RelPath)
	v.addActivity(artifact.Message)
	v.refreshArtifactsWithQuery("kind:sql-notebook-run")
	v.refreshJobs()
}

func (v *View) exportSQLiteQueryCSVArtifact() {
	v.exportSQLiteQueryArtifact("csv")
}

func (v *View) exportSQLiteQueryMarkdownArtifact() {
	v.exportSQLiteQueryArtifact("markdown")
}

func (v *View) exportSQLiteQueryArtifact(kind string) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before exporting SQLite query artifacts.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" && strings.TrimSpace(v.dataLastSQLiteQuery.RelPath) == "" {
		v.dataProfileStatus.SetText("Select a SQLite source before exporting query artifacts.")
		return
	}
	jobLabel := sqliteArtifactExportJobLabel(selected, kind)
	job, ctx := v.jobService.Start(datasetSQLiteArtifactJobKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Path: "+firstNonEmptyString(selected, v.dataLastSQLiteQuery.RelPath))
	v.jobService.AppendLog(job.ID, "Format: "+kind)
	v.dataProfileStatus.SetText("Exporting SQLite query artifact as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	cachedSQLite := v.dataLastSQLiteQuery
	sqlText := v.dataQueryEntry.Text
	go func() {
		result, artifact, err := v.buildSQLiteArtifactForExport(ctx, root, selected, kind, cachedSQLite, sqlText)
		fyne.Do(func() {
			v.finishSQLiteArtifactExportJob(job.ID, selected, kind, result, artifact, err)
		})
	}()
}

func (v *View) resolveDatasetQueryForChart(
	ctx context.Context,
	root string,
	selected string,
	cached datasetsSvc.QueryResult,
	queryText string,
) (datasetsSvc.QueryResult, error) {
	if strings.TrimSpace(cached.RelPath) != "" && (selected == "" || selected == cached.RelPath) {
		return cached, nil
	}
	if selected == "" {
		return datasetsSvc.QueryResult{}, fmt.Errorf("select a CSV, TSV, JSON, or XLSX file before charting data")
	}
	result, err := v.datasetService.QueryContext(ctx, root, selected, queryText)
	if err != nil {
		return datasetsSvc.QueryResult{}, err
	}
	select {
	case <-ctx.Done():
		return datasetsSvc.QueryResult{}, ctx.Err()
	default:
	}
	return result, nil
}

func (v *View) buildDatasetChartPreview(
	ctx context.Context,
	root string,
	selected string,
	cached datasetsSvc.QueryResult,
	queryText string,
) (datasetsSvc.QueryResult, datasetsSvc.ChartResult, error) {
	result, err := v.resolveDatasetQueryForChart(ctx, root, selected, cached, queryText)
	if err != nil {
		return datasetsSvc.QueryResult{}, datasetsSvc.ChartResult{}, err
	}
	chart, err := datasetsSvc.BuildChart(result)
	if err != nil {
		return datasetsSvc.QueryResult{}, datasetsSvc.ChartResult{}, err
	}
	select {
	case <-ctx.Done():
		return datasetsSvc.QueryResult{}, datasetsSvc.ChartResult{}, ctx.Err()
	default:
	}
	return result, chart, nil
}

func (v *View) buildDatasetDashboardPreview(
	ctx context.Context,
	root string,
	selected string,
	cached datasetsSvc.QueryResult,
	queryText string,
) (datasetsSvc.QueryResult, datasetsSvc.DashboardResult, error) {
	result, err := v.resolveDatasetQueryForChart(ctx, root, selected, cached, queryText)
	if err != nil {
		return datasetsSvc.QueryResult{}, datasetsSvc.DashboardResult{}, err
	}
	dashboard, err := datasetsSvc.BuildDashboard(result)
	if err != nil {
		return datasetsSvc.QueryResult{}, datasetsSvc.DashboardResult{}, err
	}
	select {
	case <-ctx.Done():
		return datasetsSvc.QueryResult{}, datasetsSvc.DashboardResult{}, ctx.Err()
	default:
	}
	return result, dashboard, nil
}

func (v *View) finishDatasetChartPreviewJob(
	jobID string,
	selected string,
	result datasetsSvc.QueryResult,
	chart datasetsSvc.ChartResult,
	err error,
) {
	if err != nil {
		if isDataJobCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Chart preview cancelled.", nil)
			v.dataProfileStatus.SetText("Chart preview cancelled for " + firstNonEmptyString(selected, "selection") + ".")
			v.addActivity("Cancelled chart preview for " + firstNonEmptyString(selected, "selection") + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Chart preview failed.", err)
			v.dataProfileStatus.SetText("Chart preview failed for " + firstNonEmptyString(selected, result.RelPath))
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	v.jobService.AppendLog(jobID, fmt.Sprintf("Points=%d", len(chart.Points)))
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, firstNonEmptyString(chart.Message, "Chart preview completed."), nil)
	v.dataLastQuery = result
	v.dataLastChart = chart
	v.dataLastDashboard = datasetsSvc.DashboardResult{}
	v.dataProfileStatus.SetText(chart.Message)
	v.setDataSummary(formatDatasetChart(chart))
	v.addActivity("Previewed chart for " + chart.RelPath + ".")
	v.refreshJobs()
}

func (v *View) finishDatasetDashboardPreviewJob(
	jobID string,
	selected string,
	result datasetsSvc.QueryResult,
	dashboard datasetsSvc.DashboardResult,
	err error,
) {
	if err != nil {
		if isDataJobCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Dashboard preview cancelled.", nil)
			v.dataProfileStatus.SetText("Dashboard preview cancelled for " + firstNonEmptyString(selected, "selection") + ".")
			v.addActivity("Cancelled dashboard preview for " + firstNonEmptyString(selected, "selection") + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Dashboard preview failed.", err)
			v.dataProfileStatus.SetText("Dashboard preview failed for " + firstNonEmptyString(selected, result.RelPath))
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	v.jobService.AppendLog(jobID, fmt.Sprintf("Metrics=%d points=%d", len(dashboard.Metrics), len(dashboard.Chart.Points)))
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, firstNonEmptyString(dashboard.Message, "Dashboard preview completed."), nil)
	v.dataLastQuery = result
	v.dataLastChart = dashboard.Chart
	v.dataLastDashboard = dashboard
	v.dataProfileStatus.SetText(dashboard.Message)
	v.setDataSummary(formatDatasetDashboard(dashboard))
	v.addActivity("Previewed dashboard for " + dashboard.RelPath + ".")
	v.refreshJobs()
}

func (v *View) buildDatasetChartArtifactForExport(
	ctx context.Context,
	root string,
	selected string,
	cachedQuery datasetsSvc.QueryResult,
	cachedChart datasetsSvc.ChartResult,
	queryText string,
) (datasetsSvc.QueryResult, datasetsSvc.ChartResult, artifactsSvc.Artifact, error) {
	chart := cachedChart
	queryResult := cachedQuery
	if strings.TrimSpace(chart.SVG) == "" || (selected != "" && chart.RelPath != selected) {
		result, builtChart, err := v.buildDatasetChartPreview(ctx, root, selected, cachedQuery, queryText)
		if err != nil {
			return datasetsSvc.QueryResult{}, datasetsSvc.ChartResult{}, artifactsSvc.Artifact{}, err
		}
		queryResult = result
		chart = builtChart
	}
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		return datasetsSvc.QueryResult{}, datasetsSvc.ChartResult{}, artifactsSvc.Artifact{}, err
	}
	artifact, err := store.WriteChartArtifact(chartArtifactInput(chart))
	if err != nil {
		return datasetsSvc.QueryResult{}, datasetsSvc.ChartResult{}, artifactsSvc.Artifact{}, err
	}
	select {
	case <-ctx.Done():
		return datasetsSvc.QueryResult{}, datasetsSvc.ChartResult{}, artifactsSvc.Artifact{}, ctx.Err()
	default:
	}
	return queryResult, chart, artifact, nil
}

func (v *View) buildDatasetDashboardArtifactForExport(
	ctx context.Context,
	root string,
	selected string,
	cachedQuery datasetsSvc.QueryResult,
	cachedDashboard datasetsSvc.DashboardResult,
	queryText string,
) (datasetsSvc.QueryResult, datasetsSvc.DashboardResult, artifactsSvc.Artifact, error) {
	dashboard := cachedDashboard
	queryResult := cachedQuery
	if strings.TrimSpace(dashboard.SVG) == "" || (selected != "" && dashboard.RelPath != selected) {
		result, builtDashboard, err := v.buildDatasetDashboardPreview(ctx, root, selected, cachedQuery, queryText)
		if err != nil {
			return datasetsSvc.QueryResult{}, datasetsSvc.DashboardResult{}, artifactsSvc.Artifact{}, err
		}
		queryResult = result
		dashboard = builtDashboard
	}
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		return datasetsSvc.QueryResult{}, datasetsSvc.DashboardResult{}, artifactsSvc.Artifact{}, err
	}
	artifact, err := store.WriteChartArtifact(dashboardArtifactInput(dashboard))
	if err != nil {
		return datasetsSvc.QueryResult{}, datasetsSvc.DashboardResult{}, artifactsSvc.Artifact{}, err
	}
	select {
	case <-ctx.Done():
		return datasetsSvc.QueryResult{}, datasetsSvc.DashboardResult{}, artifactsSvc.Artifact{}, ctx.Err()
	default:
	}
	return queryResult, dashboard, artifact, nil
}

func (v *View) finishDatasetChartExportJob(
	jobID string,
	selected string,
	result datasetsSvc.QueryResult,
	chart datasetsSvc.ChartResult,
	artifact artifactsSvc.Artifact,
	err error,
) {
	if err != nil {
		if isDataJobCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Chart export cancelled.", nil)
			v.dataProfileStatus.SetText("Chart export cancelled for " + firstNonEmptyString(selected, "selection") + ".")
			v.addActivity("Cancelled chart export for " + firstNonEmptyString(selected, "selection") + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Chart export failed.", err)
			v.dataProfileStatus.SetText("Chart export failed for " + firstNonEmptyString(selected, result.RelPath))
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	artifact.JobID = jobID
	v.jobService.AppendLog(jobID, "Artifact: "+artifact.RelPath)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Created "+artifact.RelPath+".", nil)
	v.dataLastQuery = result
	v.dataLastChart = chart
	v.dataLastDashboard = datasetsSvc.DashboardResult{}
	v.persistArtifactRecord(artifact)
	v.dataProfileStatus.SetText("Exported chart " + artifact.RelPath)
	v.addActivity(artifact.Message)
	v.refreshArtifactsWithQuery("kind:chart")
	v.refreshJobs()
}

func (v *View) finishDatasetDashboardExportJob(
	jobID string,
	selected string,
	result datasetsSvc.QueryResult,
	dashboard datasetsSvc.DashboardResult,
	artifact artifactsSvc.Artifact,
	err error,
) {
	if err != nil {
		if isDataJobCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Dashboard export cancelled.", nil)
			v.dataProfileStatus.SetText("Dashboard export cancelled for " + firstNonEmptyString(selected, "selection") + ".")
			v.addActivity("Cancelled dashboard export for " + firstNonEmptyString(selected, "selection") + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Dashboard export failed.", err)
			v.dataProfileStatus.SetText("Dashboard export failed for " + firstNonEmptyString(selected, result.RelPath))
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	artifact.JobID = jobID
	v.jobService.AppendLog(jobID, "Artifact: "+artifact.RelPath)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Created "+artifact.RelPath+".", nil)
	v.dataLastQuery = result
	v.dataLastChart = dashboard.Chart
	v.dataLastDashboard = dashboard
	v.persistArtifactRecord(artifact)
	v.dataProfileStatus.SetText("Exported dashboard " + artifact.RelPath)
	v.addActivity(artifact.Message)
	v.refreshArtifactsWithQuery("kind:dashboard")
	v.refreshJobs()
}

func (v *View) resolveSQLiteQueryForArtifact(
	ctx context.Context,
	root string,
	selected string,
	cached dbconnectorSvc.SQLiteQueryResult,
	sqlText string,
) (dbconnectorSvc.SQLiteQueryResult, error) {
	if strings.TrimSpace(cached.RelPath) != "" && (selected == "" || selected == cached.RelPath) {
		return cached, nil
	}
	if selected == "" {
		return dbconnectorSvc.SQLiteQueryResult{}, fmt.Errorf("select a SQLite source before exporting query artifacts")
	}
	request := dbconnectorSvc.NormalizeSQLiteQueryRequest(dbconnectorSvc.SQLiteQueryRequest{
		RelPath:        selected,
		SQL:            sqlText,
		ResultLimit:    dbconnectorSvc.DefaultSQLiteRows,
		TimeoutSeconds: dbconnectorSvc.DefaultSQLiteTimeoutSeconds,
	})
	result, err := v.dbconnectorService.QueryWorkspaceSQLiteContext(ctx, root, request)
	if err != nil {
		return dbconnectorSvc.SQLiteQueryResult{}, err
	}
	select {
	case <-ctx.Done():
		return dbconnectorSvc.SQLiteQueryResult{}, ctx.Err()
	default:
	}
	return result, nil
}

func (v *View) buildSQLiteArtifactForExport(
	ctx context.Context,
	root string,
	selected string,
	kind string,
	cached dbconnectorSvc.SQLiteQueryResult,
	sqlText string,
) (dbconnectorSvc.SQLiteQueryResult, artifactsSvc.Artifact, error) {
	result, err := v.resolveSQLiteQueryForArtifact(ctx, root, selected, cached, sqlText)
	if err != nil {
		return dbconnectorSvc.SQLiteQueryResult{}, artifactsSvc.Artifact{}, err
	}
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		return dbconnectorSvc.SQLiteQueryResult{}, artifactsSvc.Artifact{}, err
	}
	input := sqliteQueryArtifactInput(result)
	var artifact artifactsSvc.Artifact
	switch kind {
	case "csv":
		artifact, err = store.WriteSQLiteQueryCSVArtifact(input)
	default:
		artifact, err = store.WriteSQLiteQueryMarkdownArtifact(input)
	}
	if err != nil {
		return dbconnectorSvc.SQLiteQueryResult{}, artifactsSvc.Artifact{}, err
	}
	select {
	case <-ctx.Done():
		return dbconnectorSvc.SQLiteQueryResult{}, artifactsSvc.Artifact{}, ctx.Err()
	default:
	}
	return result, artifact, nil
}

func (v *View) finishSQLiteArtifactExportJob(
	jobID string,
	selected string,
	kind string,
	result dbconnectorSvc.SQLiteQueryResult,
	artifact artifactsSvc.Artifact,
	err error,
) {
	if err != nil {
		if isDataJobCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "SQLite query artifact export cancelled.", nil)
			v.dataProfileStatus.SetText("SQLite query artifact export cancelled for " + firstNonEmptyString(selected, "selection") + ".")
			v.addActivity("Cancelled SQLite query artifact export for " + firstNonEmptyString(selected, "selection") + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "SQLite query artifact export failed.", err)
			v.dataProfileStatus.SetText("SQLite export query failed for " + firstNonEmptyString(selected, result.RelPath))
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	artifact.JobID = jobID
	v.jobService.AppendLog(jobID, "Artifact: "+artifact.RelPath)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Created "+artifact.RelPath+".", nil)
	v.dataLastSQLiteQuery = result
	v.dataLastQuery = sqliteQueryAsDatasetResult(result)
	v.persistArtifactRecord(artifact)
	v.persistSQLiteArtifactLineage(result, artifact)
	v.dataProfileStatus.SetText("Exported SQLite query artifact " + artifact.RelPath)
	v.addActivity(artifact.Message)
	if kind == "csv" {
		v.refreshArtifactsWithQuery("kind:sqlite-query-csv")
		v.refreshJobs()
		return
	}
	v.refreshArtifactsWithQuery("kind:sqlite-query-report")
	v.refreshJobs()
}
