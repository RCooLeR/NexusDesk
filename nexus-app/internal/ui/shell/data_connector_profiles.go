package shell

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	datasetsSvc "nexusdesk/internal/services/datasets"
	dbconnectorSvc "nexusdesk/internal/services/dbconnector"
	jobsSvc "nexusdesk/internal/services/jobs"
	metadataSvc "nexusdesk/internal/services/metadata"
)

const (
	connectorQueryJobKind   = "connector-query"
	connectorTestJobKind    = "connector-test"
	connectorInspectJobKind = "connector-inspect"
)

func (v *View) listConnectorProfiles() {
	profiles, err := v.loadConnectorProfiles()
	if err != nil {
		v.dataProfileStatus.SetText("External connector profiles are unavailable.")
		dialog.ShowError(err, v.window)
		return
	}
	v.refreshConnectorProfileSelect(profiles)
	v.dataProfileStatus.SetText(fmt.Sprintf("Loaded %d external connector profile(s).", len(profiles)))
	v.setDataSummary(formatConnectorProfiles(profiles))
	v.addActivity(fmt.Sprintf("Loaded %d external connector profile(s).", len(profiles)))
}

func (v *View) promptSaveConnectorProfile() {
	if v.connectorProfileStore == nil {
		v.dataProfileStatus.SetText("External connector profile store is unavailable.")
		return
	}
	base := v.selectedConnectorProfile()
	name := widget.NewEntry()
	name.SetText(base.Name)
	kind := widget.NewSelect([]string{"postgres", "mysql", "mariadb", "sqlserver", "duckdb", "sqlite"}, nil)
	kind.SetSelected(firstNonEmptyString(base.Kind, "postgres"))
	host := widget.NewEntry()
	host.SetText(base.Host)
	port := widget.NewEntry()
	port.SetText(strconv.Itoa(base.Port))
	database := widget.NewEntry()
	database.SetText(base.Database)
	username := widget.NewEntry()
	username.SetText(base.Username)
	password := widget.NewPasswordEntry()
	password.SetText(base.Password)
	sslMode := widget.NewEntry()
	sslMode.SetText(firstNonEmptyString(base.SSLMode, "prefer"))
	resultLimit := widget.NewEntry()
	resultLimit.SetText(strconv.Itoa(valueOrDefault(base.ResultLimit, dbconnectorDefaultResultLimit())))
	timeout := widget.NewEntry()
	timeout.SetText(strconv.Itoa(valueOrDefault(base.TimeoutSeconds, dbconnectorDefaultTimeoutSeconds())))
	workspaceScope := strings.TrimSpace(base.WorkspaceScope)
	if workspaceScope == "" {
		workspaceScope = strings.TrimSpace(v.state.Workspace().Root)
	}

	dialog.ShowForm("Save external connector profile", "Save", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Name", name),
		widget.NewFormItem("Kind", kind),
		widget.NewFormItem("Host", host),
		widget.NewFormItem("Port", port),
		widget.NewFormItem("Database", database),
		widget.NewFormItem("Username", username),
		widget.NewFormItem("Password", password),
		widget.NewFormItem("SSL mode", sslMode),
		widget.NewFormItem("Result limit", resultLimit),
		widget.NewFormItem("Timeout seconds", timeout),
	}, func(confirm bool) {
		if !confirm {
			return
		}
		portValue, err := strconv.Atoi(strings.TrimSpace(port.Text))
		if err != nil {
			dialog.ShowError(fmt.Errorf("port must be numeric"), v.window)
			return
		}
		resultLimitValue, err := strconv.Atoi(strings.TrimSpace(resultLimit.Text))
		if err != nil {
			dialog.ShowError(fmt.Errorf("result limit must be numeric"), v.window)
			return
		}
		timeoutValue, err := strconv.Atoi(strings.TrimSpace(timeout.Text))
		if err != nil {
			dialog.ShowError(fmt.Errorf("timeout must be numeric"), v.window)
			return
		}
		saved, err := v.connectorProfileStore.Save(dbconnectorSvc.ConnectorProfile{
			ID:             base.ID,
			Name:           name.Text,
			Kind:           kind.Selected,
			Host:           host.Text,
			Port:           portValue,
			Database:       database.Text,
			Username:       username.Text,
			Password:       password.Text,
			CredentialRef:  base.CredentialRef,
			SSLMode:        sslMode.Text,
			WorkspaceScope: workspaceScope,
			ResultLimit:    resultLimitValue,
			TimeoutSeconds: timeoutValue,
		})
		if err != nil {
			v.dataProfileStatus.SetText("External connector profile save failed.")
			dialog.ShowError(err, v.window)
			return
		}
		v.dataConnectorProfileID = saved.ID
		v.dataProfileStatus.SetText("Saved external connector profile " + saved.Name + ".")
		v.addActivity("Saved external connector profile " + saved.Name + ".")
		v.listConnectorProfiles()
	}, v.window)
}

func (v *View) deleteSelectedConnectorProfile() {
	if v.connectorProfileStore == nil {
		v.dataProfileStatus.SetText("External connector profile store is unavailable.")
		return
	}
	selected := strings.TrimSpace(v.dataConnectorProfileID)
	if selected == "" {
		v.dataProfileStatus.SetText("Select an external connector profile before deleting.")
		return
	}
	dialog.ShowConfirm("Delete external connector profile", "Delete selected profile?", func(confirm bool) {
		if !confirm {
			return
		}
		if err := v.connectorProfileStore.Delete(selected); err != nil {
			v.dataProfileStatus.SetText("External connector profile delete failed.")
			dialog.ShowError(err, v.window)
			return
		}
		v.dataConnectorProfileID = ""
		v.dataProfileStatus.SetText("Deleted external connector profile.")
		v.addActivity("Deleted external connector profile " + selected + ".")
		v.listConnectorProfiles()
	}, v.window)
}

func (v *View) validateExternalConnectorSQL() {
	selected := strings.TrimSpace(v.dataConnectorProfileID)
	if selected == "" {
		v.dataProfileStatus.SetText("Select an external connector profile before validating SQL.")
		return
	}
	normalized, err := dbconnectorSvc.NormalizeExternalReadOnlySQL(v.dataQueryEntry.Text)
	if err != nil {
		v.dataProfileStatus.SetText("External connector SQL validation failed.")
		dialog.ShowError(err, v.window)
		return
	}
	profile := v.selectedConnectorProfile()
	name := firstNonEmptyString(profile.Name, selected)
	v.dataProfileStatus.SetText("External connector SQL is read-only and valid for " + name + ".")
	v.setDataSummary(formatConnectorSQLValidation(name, normalized))
	v.addActivity("Validated read-only external connector SQL for " + name + ".")
}

func (v *View) testSelectedConnectorProfile() {
	profile, ok := v.selectedConnectorProfileWithGuard()
	if !ok {
		return
	}
	jobLabel := connectorProfileTestJobLabel(profile)
	job, jobCtx := v.jobService.Start(connectorTestJobKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Profile: "+firstNonEmptyString(profile.Name, profile.ID))
	v.jobService.AppendLog(job.ID, "Kind: "+profile.Kind)
	timeout := valueOrDefault(profile.TimeoutSeconds, dbconnectorDefaultTimeoutSeconds())
	ctx, cancel := context.WithTimeout(jobCtx, time.Duration(timeout)*time.Second)
	v.dataProfileStatus.SetText("Testing external connector profile " + profile.Name + " as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	go func() {
		defer cancel()
		status, err := v.dbconnectorService.TestConnectorProfileContext(ctx, profile)
		fyne.Do(func() {
			v.finishConnectorProfileTestJob(job.ID, profile, status, err)
		})
	}()
}

func (v *View) inspectSelectedConnectorProfile() {
	profile, ok := v.selectedConnectorProfileWithGuard()
	if !ok {
		return
	}
	jobLabel := connectorProfileInspectJobLabel(profile)
	job, jobCtx := v.jobService.Start(connectorInspectJobKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Profile: "+firstNonEmptyString(profile.Name, profile.ID))
	v.jobService.AppendLog(job.ID, "Kind: "+profile.Kind)
	timeout := valueOrDefault(profile.TimeoutSeconds, dbconnectorDefaultTimeoutSeconds())
	ctx, cancel := context.WithTimeout(jobCtx, time.Duration(timeout)*time.Second)
	v.dataProfileStatus.SetText("Inspecting external connector profile " + profile.Name + " as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	go func() {
		defer cancel()
		metadata, err := v.dbconnectorService.InspectConnectorProfileContext(ctx, profile)
		fyne.Do(func() {
			v.finishConnectorProfileInspectJob(job.ID, profile, metadata, err)
		})
	}()
}

func (v *View) runSelectedConnectorProfileQuery() {
	profile, ok := v.selectedConnectorProfileWithGuard()
	if !ok {
		return
	}
	request := dbconnectorSvc.NormalizeConnectorQueryRequest(dbconnectorSvc.ConnectorQueryRequest{
		ProfileID:      profile.ID,
		SQL:            v.dataQueryEntry.Text,
		ResultLimit:    profile.ResultLimit,
		TimeoutSeconds: profile.TimeoutSeconds,
	})
	jobLabel := connectorQueryJobLabel(profile)
	job, jobCtx := v.jobService.Start(connectorQueryJobKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Profile: "+firstNonEmptyString(profile.Name, profile.ID))
	v.jobService.AppendLog(job.ID, fmt.Sprintf("SQL bytes: %d", len(strings.TrimSpace(request.SQL))))
	started := time.Now().UTC()
	ctx, cancel := context.WithTimeout(jobCtx, time.Duration(request.TimeoutSeconds)*time.Second)
	queryID := v.startConnectorQuery(cancel)
	v.dataProfileStatus.SetText(fmt.Sprintf("%s: external query running, cap %d, timeout %ds.", profile.Name, request.ResultLimit, request.TimeoutSeconds))
	v.addActivity("Started external connector query for " + profile.Name + ".")
	v.refreshJobs()
	go func() {
		defer cancel()
		result, err := v.dbconnectorService.QueryConnectorProfileContext(ctx, profile, request)
		fyne.Do(func() {
			v.finishConnectorQuery(job.ID, queryID, profile, request.SQL, started, result, err)
		})
	}()
}

func (v *View) cancelActiveConnectorQuery() {
	v.dataConnectorQueryMu.Lock()
	cancel := v.dataConnectorCancel
	queryID := v.dataConnectorQueryID
	v.dataConnectorQueryMu.Unlock()
	if cancel == nil {
		v.dataProfileStatus.SetText("No external connector query is currently running.")
		return
	}
	cancel()
	v.dataProfileStatus.SetText("External connector query cancellation requested.")
	v.addActivity("External connector query cancel requested: " + queryID + ".")
}

func (v *View) finishConnectorQuery(jobID string, queryID string, profile dbconnectorSvc.ConnectorProfile, sqlText string, started time.Time, result dbconnectorSvc.ConnectorQueryResult, err error) {
	v.clearConnectorQuery(queryID)
	if strings.TrimSpace(sqlText) != "" {
		record := connectorSQLRunRecord(result, profile, sqlText, started, err)
		if v.metadataStore != nil {
			record = v.metadataStore.NormalizeSQLRunRecord(record)
			if saveErr := v.metadataStore.SaveSQLRun(record); saveErr != nil {
				v.addActivity("Could not persist external connector SQL metadata: " + saveErr.Error())
			} else if err == nil {
				v.persistDatasetDependency(connectorDependencyRecord(profile, record))
			}
		}
	}
	if err != nil {
		if isSQLiteQueryCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "External connector query cancelled.", nil)
			v.dataProfileStatus.SetText("External connector query cancelled for " + profile.Name)
			v.addActivity("Cancelled external connector query for " + profile.Name + ".")
			v.refreshJobs()
			return
		}
		v.jobService.Finish(jobID, jobsSvc.StatusFailed, "External connector query failed.", err)
		v.dataProfileStatus.SetText("External connector query failed for " + profile.Name)
		dialog.ShowError(err, v.window)
		v.refreshJobs()
		return
	}
	v.jobService.AppendLog(jobID, fmt.Sprintf("Rows: shown=%d total=%d duration=%dms", len(result.Rows), result.TotalRows, result.DurationMs))
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, firstNonEmptyString(result.Message, "External connector query completed."), nil)
	v.dataProfileStatus.SetText(result.Message)
	v.setDataSummary(formatConnectorQueryResult(result))
	v.setDataRowsGrid(result.Columns, result.Rows)
	v.dataLastConnectorQuery = result
	v.dataLastQuery = datasetsQueryFromConnectorQuery(result)
	v.dataLastSQLiteQuery = dbconnectorSvc.SQLiteQueryResult{}
	v.dataLastChart = datasetsSvc.ChartResult{}
	v.dataLastDashboard = datasetsSvc.DashboardResult{}
	v.addActivity("Ran external connector query for " + profile.Name + ".")
	v.refreshJobs()
}

func (v *View) loadConnectorProfiles() ([]dbconnectorSvc.ConnectorProfile, error) {
	if v.connectorProfileStore == nil {
		return nil, fmt.Errorf("external connector profile store is not available")
	}
	workspaceRoot := strings.TrimSpace(v.state.Workspace().Root)
	if workspaceRoot == "" {
		return v.connectorProfileStore.List()
	}
	return v.connectorProfileStore.ListForWorkspace(workspaceRoot)
}

func (v *View) refreshConnectorProfileSelect(profiles []dbconnectorSvc.ConnectorProfile) {
	if v.dataConnectorProfile == nil {
		return
	}
	v.dataConnectorOptions = map[string]string{}
	options := make([]string, 0, len(profiles))
	selectedLabel := ""
	for _, profile := range profiles {
		label := fmt.Sprintf("%s [%s]", firstNonEmptyString(profile.Name, profile.ID), profile.Kind)
		options = append(options, label)
		v.dataConnectorOptions[label] = profile.ID
		if profile.ID == v.dataConnectorProfileID {
			selectedLabel = label
		}
	}
	v.dataConnectorProfile.Options = options
	v.dataConnectorProfile.Refresh()
	if selectedLabel != "" {
		v.dataConnectorProfile.SetSelected(selectedLabel)
		return
	}
	if len(options) == 0 {
		v.dataConnectorProfile.ClearSelected()
		v.dataConnectorProfileID = ""
		return
	}
	v.dataConnectorProfile.SetSelected(options[0])
	v.dataConnectorProfileID = v.dataConnectorOptions[options[0]]
}

func (v *View) selectedConnectorProfile() dbconnectorSvc.ConnectorProfile {
	profiles, err := v.loadConnectorProfiles()
	if err != nil {
		return dbconnectorSvc.ConnectorProfile{}
	}
	selected := strings.TrimSpace(v.dataConnectorProfileID)
	if selected == "" {
		return dbconnectorSvc.ConnectorProfile{}
	}
	for _, profile := range profiles {
		if profile.ID == selected {
			return profile
		}
	}
	return dbconnectorSvc.ConnectorProfile{}
}

func (v *View) selectedConnectorProfileWithGuard() (dbconnectorSvc.ConnectorProfile, bool) {
	selected := strings.TrimSpace(v.dataConnectorProfileID)
	if selected == "" {
		v.dataProfileStatus.SetText("Select an external connector profile first.")
		return dbconnectorSvc.ConnectorProfile{}, false
	}
	profile := v.selectedConnectorProfile()
	if strings.TrimSpace(profile.ID) == "" {
		v.dataProfileStatus.SetText("Selected external connector profile is unavailable.")
		return dbconnectorSvc.ConnectorProfile{}, false
	}
	return profile, true
}

func (v *View) startConnectorQuery(cancel context.CancelFunc) string {
	id := fmt.Sprintf("connector-%d", time.Now().UTC().UnixNano())
	v.dataConnectorQueryMu.Lock()
	previousCancel := v.dataConnectorCancel
	v.dataConnectorCancel = cancel
	v.dataConnectorQueryID = id
	v.dataConnectorQueryMu.Unlock()
	if previousCancel != nil {
		previousCancel()
	}
	return id
}

func (v *View) clearConnectorQuery(id string) {
	v.dataConnectorQueryMu.Lock()
	defer v.dataConnectorQueryMu.Unlock()
	if v.dataConnectorQueryID != id {
		return
	}
	v.dataConnectorCancel = nil
	v.dataConnectorQueryID = ""
}

func dbconnectorDefaultResultLimit() int {
	return 1000
}

func dbconnectorDefaultTimeoutSeconds() int {
	return 30
}

func formatConnectorProfiles(profiles []dbconnectorSvc.ConnectorProfile) string {
	if len(profiles) == 0 {
		return "# External connector profiles\n\nNo external connector profiles configured yet.\n"
	}
	var builder strings.Builder
	builder.WriteString("# External connector profiles\n\n")
	for _, profile := range profiles {
		builder.WriteString(fmt.Sprintf("- %s [%s]\n", profile.Name, profile.Kind))
		builder.WriteString(fmt.Sprintf("  Host: %s  Port: %d  DB: %s\n", profile.Host, profile.Port, profile.Database))
		builder.WriteString(fmt.Sprintf("  User: %s  SSL: %s  Read-only: %t\n", profile.Username, profile.SSLMode, profile.ReadOnly))
		builder.WriteString(fmt.Sprintf("  Cap: %d rows  Timeout: %ds\n", profile.ResultLimit, profile.TimeoutSeconds))
		scope := strings.TrimSpace(profile.WorkspaceScope)
		if scope == "" {
			scope = "global"
		}
		builder.WriteString("  Scope: ")
		builder.WriteString(scope)
		builder.WriteString("\n")
		if profile.UpdatedAt != "" {
			builder.WriteString("  Updated: ")
			builder.WriteString(profile.UpdatedAt)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func formatConnectorSQLValidation(profileName string, query string) string {
	return "# External connector SQL validation\n\nProfile: " + profileName + "\n\nValidated read-only SQL:\n\n" + query + "\n"
}

func formatConnectorProfileStatus(status dbconnectorSvc.ConnectorProfileStatus) string {
	return "# External connector profile test\n\n" +
		"Profile: " + status.Name + "\n" +
		"Kind: " + status.Kind + "\n" +
		"Engine: " + status.Engine + "\n" +
		"Read-only: " + fmt.Sprintf("%t", status.ReadOnly) + "\n\n" +
		status.Message + "\n"
}

func formatConnectorQueryResult(result dbconnectorSvc.ConnectorQueryResult) string {
	var builder strings.Builder
	builder.WriteString("# External connector query\n\n")
	builder.WriteString("Profile: ")
	builder.WriteString(result.Name)
	builder.WriteString("\nKind: ")
	builder.WriteString(result.Kind)
	builder.WriteString("\nEngine: ")
	builder.WriteString(result.Engine)
	builder.WriteString(fmt.Sprintf("\nShown rows: %d\n", len(result.Rows)))
	builder.WriteString(fmt.Sprintf("Total rows: %d\n", result.TotalRows))
	builder.WriteString(fmt.Sprintf("Cap: %d\n", result.ResultLimit))
	builder.WriteString(fmt.Sprintf("Timeout: %ds\n", result.TimeoutSeconds))
	builder.WriteString(fmt.Sprintf("Duration: %d ms\n", result.DurationMs))
	if strings.TrimSpace(result.Message) != "" {
		builder.WriteString("\n")
		builder.WriteString(result.Message)
		builder.WriteString("\n")
	}
	if strings.TrimSpace(result.SQL) != "" {
		builder.WriteString("\nSQL\n\n")
		builder.WriteString(result.SQL)
		builder.WriteString("\n")
	}
	if len(result.Columns) > 0 {
		builder.WriteString("\nRows\n\n")
		builder.WriteString(strings.Join(result.Columns, "\t"))
		builder.WriteString("\n")
		for _, row := range result.Rows {
			builder.WriteString(strings.Join(row, "\t"))
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func formatConnectorMetadata(metadata dbconnectorSvc.ConnectorMetadata) string {
	var builder strings.Builder
	builder.WriteString("# External connector metadata\n\n")
	builder.WriteString("Profile: ")
	builder.WriteString(metadata.Name)
	builder.WriteString("\nKind: ")
	builder.WriteString(metadata.Kind)
	builder.WriteString("\nEngine: ")
	builder.WriteString(metadata.Engine)
	builder.WriteString("\nRead-only: ")
	builder.WriteString(fmt.Sprintf("%t", metadata.ReadOnly))
	builder.WriteString(fmt.Sprintf("\nTables: %d\nViews: %d\nIndexes: %d\nRelationships: %d\n", len(metadata.Tables), len(metadata.Views), len(metadata.Indexes), len(metadata.Relationships)))
	if strings.TrimSpace(metadata.Message) != "" {
		builder.WriteString("\n")
		builder.WriteString(metadata.Message)
		builder.WriteString("\n")
	}
	if len(metadata.Tables) > 0 {
		builder.WriteString("\nTables\n\n")
		for _, table := range metadata.Tables {
			builder.WriteString(fmt.Sprintf("- %s | rows=%d | columns=%d | indexes=%d\n", table.Name, table.RowCount, len(table.Columns), len(table.Indexes)))
		}
	}
	if len(metadata.Views) > 0 {
		builder.WriteString("\nViews\n\n")
		for _, view := range metadata.Views {
			builder.WriteString(fmt.Sprintf("- %s | columns=%d\n", view.Name, len(view.Columns)))
		}
	}
	if len(metadata.Relationships) > 0 {
		builder.WriteString("\nRelationships\n\n")
		limit := len(metadata.Relationships)
		if limit > 25 {
			limit = 25
		}
		for _, relationship := range metadata.Relationships[:limit] {
			builder.WriteString(fmt.Sprintf("- %s.%s -> %s.%s [%s]\n", relationship.FromTable, relationship.FromColumn, relationship.ToTable, relationship.ToColumn, relationship.Kind))
		}
		if len(metadata.Relationships) > limit {
			builder.WriteString(fmt.Sprintf("- ... %d more relationship(s)\n", len(metadata.Relationships)-limit))
		}
	}
	return builder.String()
}

func datasetsQueryFromConnectorQuery(result dbconnectorSvc.ConnectorQueryResult) datasetsSvc.QueryResult {
	return datasetsSvc.QueryResult{
		RelPath:     "connector:" + result.ProfileID,
		Format:      "Connector",
		Query:       result.SQL,
		Columns:     result.Columns,
		Rows:        result.Rows,
		TotalRows:   result.TotalRows,
		MatchedRows: result.TotalRows,
		Truncated:   result.Truncated,
		Message:     result.Message,
	}
}

func connectorSQLRunRecord(result dbconnectorSvc.ConnectorQueryResult, profile dbconnectorSvc.ConnectorProfile, sqlText string, started time.Time, runErr error) metadataSvc.SQLRunRecord {
	status := "success"
	message := result.Message
	errorText := ""
	completed := time.Now().UTC()
	if runErr != nil {
		if isDataJobCanceled(runErr) {
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
		RelPath:     firstNonEmptyString("connector:"+firstNonEmptyString(result.ProfileID, profile.ID), "connector"),
		SQL:         strings.TrimSpace(firstNonEmptyString(result.SQL, sqlText)),
		Engine:      firstNonEmptyString(result.Engine, "connector-"+firstNonEmptyString(result.Kind, profile.Kind), "connector-readonly"),
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

func connectorDependencyRecord(profile dbconnectorSvc.ConnectorProfile, sqlRun metadataSvc.SQLRunRecord) metadataSvc.DatasetDependencyRecord {
	return metadataSvc.DatasetDependencyRecord{
		SourcePath:    firstNonEmptyString("connector:"+strings.TrimSpace(profile.ID), "connector"),
		DependentKind: "connector-query",
		DependentRef:  sqlRun.ID,
		Relation:      "reads",
		Metadata: map[string]string{
			"profile": firstNonEmptyString(profile.Name, profile.ID),
			"kind":    profile.Kind,
			"engine":  sqlRun.Engine,
			"sql":     sqlRun.SQL,
		},
		CreatedAt: sqlRun.StartedAt,
		UpdatedAt: sqlRun.CompletedAt,
	}
}

func valueOrDefault(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func connectorQueryJobLabel(profile dbconnectorSvc.ConnectorProfile) string {
	name := strings.TrimSpace(firstNonEmptyString(profile.Name, profile.ID, "connector"))
	return "Connector query (" + name + ")"
}

func connectorProfileTestJobLabel(profile dbconnectorSvc.ConnectorProfile) string {
	name := strings.TrimSpace(firstNonEmptyString(profile.Name, profile.ID, "connector"))
	return "Connector test (" + name + ")"
}

func connectorProfileInspectJobLabel(profile dbconnectorSvc.ConnectorProfile) string {
	name := strings.TrimSpace(firstNonEmptyString(profile.Name, profile.ID, "connector"))
	return "Connector inspect (" + name + ")"
}

func (v *View) finishConnectorProfileTestJob(jobID string, profile dbconnectorSvc.ConnectorProfile, status dbconnectorSvc.ConnectorProfileStatus, err error) {
	if err != nil {
		if isSQLiteQueryCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "External connector profile test cancelled.", nil)
			v.persistDatasetDependency(connectorProfileTestDependencyRecord(jobID, profile, status, "canceled", err))
			v.dataProfileStatus.SetText("External connector profile test cancelled for " + profile.Name + ".")
			v.addActivity("Cancelled external connector profile test for " + profile.Name + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "External connector profile test failed.", err)
			v.persistDatasetDependency(connectorProfileTestDependencyRecord(jobID, profile, status, "failed", err))
			v.dataProfileStatus.SetText("External connector profile test failed for " + profile.Name + ".")
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	v.jobService.AppendLog(jobID, "Engine: "+status.Engine)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, firstNonEmptyString(status.Message, "External connector profile test succeeded."), nil)
	v.persistDatasetDependency(connectorProfileTestDependencyRecord(jobID, profile, status, "success", nil))
	v.dataProfileStatus.SetText(status.Message)
	v.setDataSummary(formatConnectorProfileStatus(status))
	v.addActivity("Tested external connector profile " + profile.Name + ".")
	v.refreshJobs()
}

func (v *View) finishConnectorProfileInspectJob(jobID string, profile dbconnectorSvc.ConnectorProfile, metadata dbconnectorSvc.ConnectorMetadata, err error) {
	if err != nil {
		if isSQLiteQueryCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "External connector profile inspection cancelled.", nil)
			v.persistDatasetDependency(connectorProfileInspectDependencyRecord(jobID, profile, metadata, "canceled", err))
			v.dataProfileStatus.SetText("External connector profile inspection cancelled for " + profile.Name + ".")
			v.addActivity("Cancelled external connector profile inspection for " + profile.Name + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "External connector profile inspection failed.", err)
			v.persistDatasetDependency(connectorProfileInspectDependencyRecord(jobID, profile, metadata, "failed", err))
			v.dataProfileStatus.SetText("External connector profile inspection failed for " + profile.Name + ".")
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	v.jobService.AppendLog(jobID, fmt.Sprintf("Tables=%d Views=%d Relationships=%d", len(metadata.Tables), len(metadata.Views), len(metadata.Relationships)))
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, firstNonEmptyString(metadata.Message, "External connector profile inspection succeeded."), nil)
	v.persistDatasetDependency(connectorProfileInspectDependencyRecord(jobID, profile, metadata, "success", nil))
	v.dataProfileStatus.SetText(metadata.Message)
	v.setDataSummary(formatConnectorMetadata(metadata))
	v.addActivity("Inspected external connector profile " + profile.Name + ".")
	v.refreshJobs()
}

func connectorProfileTestDependencyRecord(jobID string, profile dbconnectorSvc.ConnectorProfile, status dbconnectorSvc.ConnectorProfileStatus, runStatus string, runErr error) metadataSvc.DatasetDependencyRecord {
	now := time.Now().UTC()
	return metadataSvc.DatasetDependencyRecord{
		SourcePath:    firstNonEmptyString("connector:"+strings.TrimSpace(profile.ID), "connector"),
		DependentKind: "connector-profile-test",
		DependentRef:  firstNonEmptyString(strings.TrimSpace(jobID), fmt.Sprintf("connector-test-%d", now.UnixNano())),
		Relation:      "checks",
		Metadata: map[string]string{
			"profile": firstNonEmptyString(profile.Name, profile.ID),
			"kind":    firstNonEmptyString(profile.Kind, status.Kind),
			"engine":  firstNonEmptyString(status.Engine, "connector-readonly"),
			"status":  firstNonEmptyString(runStatus, "unknown"),
			"message": firstNonEmptyString(compactDataLine(status.Message, 220), compactDataLine(errText(runErr), 220)),
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func connectorProfileInspectDependencyRecord(jobID string, profile dbconnectorSvc.ConnectorProfile, metadata dbconnectorSvc.ConnectorMetadata, runStatus string, runErr error) metadataSvc.DatasetDependencyRecord {
	now := time.Now().UTC()
	return metadataSvc.DatasetDependencyRecord{
		SourcePath:    firstNonEmptyString("connector:"+strings.TrimSpace(profile.ID), "connector"),
		DependentKind: "connector-profile-inspect",
		DependentRef:  firstNonEmptyString(strings.TrimSpace(jobID), fmt.Sprintf("connector-inspect-%d", now.UnixNano())),
		Relation:      "inspects",
		Metadata: map[string]string{
			"profile":       firstNonEmptyString(profile.Name, profile.ID, metadata.Name),
			"kind":          firstNonEmptyString(profile.Kind, metadata.Kind),
			"engine":        firstNonEmptyString(metadata.Engine, "connector-readonly"),
			"status":        firstNonEmptyString(runStatus, "unknown"),
			"message":       firstNonEmptyString(compactDataLine(metadata.Message, 220), compactDataLine(errText(runErr), 220)),
			"tables":        fmt.Sprintf("%d", len(metadata.Tables)),
			"views":         fmt.Sprintf("%d", len(metadata.Views)),
			"indexes":       fmt.Sprintf("%d", len(metadata.Indexes)),
			"relationships": fmt.Sprintf("%d", len(metadata.Relationships)),
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
