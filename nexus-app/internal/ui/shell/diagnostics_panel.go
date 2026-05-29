package shell

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	externalagentsSvc "nexusdesk/internal/services/externalagents"
	issueReportSvc "nexusdesk/internal/services/issuereport"
	jobsSvc "nexusdesk/internal/services/jobs"
	llmSvc "nexusdesk/internal/services/llm"
	metadataSvc "nexusdesk/internal/services/metadata"
	perfSvc "nexusdesk/internal/services/perf"
	readinessSvc "nexusdesk/internal/services/readiness"
	settingsSvc "nexusdesk/internal/services/settings"
	startupSvc "nexusdesk/internal/services/startup"
)

const (
	diagnosticsRecentLimit         = 200
	diagnosticsFailureDetailLimit  = 8
	diagnosticsActivityTailLimit   = 24
	diagnosticsRuntimeModelLimit   = 6
	diagnosticsCompactMessageLimit = 180
	diagnosticsPerformanceLimit    = 12
)

type diagnosticsSnapshot struct {
	CollectedAt             time.Time
	WorkspaceRoot           string
	Settings                settingsSvc.Settings
	SettingsError           string
	ProbeResult             *llmSvc.ProbeResult
	ProbeError              string
	MetadataStatus          *metadataSvc.Status
	MetadataError           string
	InMemoryJobs            int
	InMemoryRunningJobs     int
	InMemoryFailedJobs      int
	JobPersistenceIssue     string
	RecentPersistedJobs     int
	RecentPersistedFailures int
	RecentTaskRuns          int
	RecentTaskFailures      int
	RecentSQLRuns           int
	RecentSQLFailures       int
	RecentAgentRuns         int
	RecentAgentFailures     int
	RecentArtifacts         int
	StartupRecovery         startupSvc.Status
	PerformanceTimings      []perfSvc.TimingRecord
	FailureScenarios        []readinessSvc.FailureScenario
	FailureScenarioIssue    string
	RuntimeSummary          []string
	ExternalAgentTools      []externalagentsSvc.ToolStatus
	RecentJobFailures       []string
	RecentTaskFailuresList  []string
	RecentSQLFailuresList   []string
	RecentAgentFailuresList []string
	ActivityTail            []string
	ProviderGuidance        []string
	RecommendedActions      []string
	Warnings                []string
}

type diagnosticsHealthCard struct {
	Label  string
	Status string
	Detail string
	Action string
}

type diagnosticsProber interface {
	Probe(ctx context.Context, config llmSvc.Config) (llmSvc.ProbeResult, error)
}

func (v *View) newDiagnosticsPanel() fyne.CanvasObject {
	refresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), v.refreshDiagnostics)
	copyReport := widget.NewButtonWithIcon("Copy report", theme.ContentCopyIcon(), v.copyDiagnosticsReport)
	exportIssueReport := widget.NewButtonWithIcon("Export issue report", theme.DownloadIcon(), v.exportDiagnosticsIssueReport)
	exportMetadata := widget.NewButtonWithIcon("Export metadata", theme.DownloadIcon(), v.exportDiagnosticsMetadataBackup)
	exportState := widget.NewButtonWithIcon("Export state", theme.DownloadIcon(), v.exportDiagnosticsWorkspaceStateBackup)
	openSettings := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), v.openSettingsTab)
	openJobs := widget.NewButtonWithIcon("Jobs", theme.ListIcon(), func() {
		if !v.selectBottomTab("Jobs") {
			v.addActivity("Jobs panel is unavailable.")
		}
	})
	openAudit := widget.NewButtonWithIcon("Agent Audit", theme.InfoIcon(), func() {
		if !v.selectBottomTab("Agent Audit") {
			v.addActivity("Agent Audit panel is unavailable.")
		}
	})
	actions := container.NewHBox(refresh, copyReport, exportIssueReport, exportMetadata, exportState, openSettings, openJobs, openAudit)
	header := container.NewVBox(v.diagnosticsStatus, actions)
	scroll := container.NewScroll(v.diagnosticsDetail)
	scroll.SetMinSize(fyne.NewSize(260, 120))
	if strings.TrimSpace(v.diagnosticsDetail.Text) == "" {
		v.diagnosticsDetail.SetText("Run diagnostics to inspect provider, metadata, and job health.")
	}
	return container.NewBorder(header, nil, nil, nil, scroll)
}

func (v *View) copyDiagnosticsReport() {
	report := strings.TrimSpace(v.diagnosticsDetail.Text)
	if report == "" {
		v.diagnosticsStatus.SetText("Run diagnostics before copying the report.")
		return
	}
	if app := fyne.CurrentApp(); app != nil && app.Clipboard() != nil {
		app.Clipboard().SetContent(report)
		v.diagnosticsStatus.SetText("Diagnostics report copied to clipboard.")
		return
	}
	if v.window != nil && v.window.Clipboard() != nil {
		v.window.Clipboard().SetContent(report)
		v.diagnosticsStatus.SetText("Diagnostics report copied to clipboard.")
		return
	}
	v.diagnosticsStatus.SetText("Clipboard is unavailable in this runtime.")
}

func (v *View) exportDiagnosticsMetadataBackup() {
	workspace := v.state.Workspace()
	if workspace.Root == "" || v.metadataStore == nil {
		v.diagnosticsStatus.SetText("Open a workspace before exporting metadata backup.")
		return
	}
	result, err := v.metadataStore.ExportBackup()
	if err != nil {
		v.diagnosticsStatus.SetText("Metadata backup export failed.")
		v.addActivity("Metadata backup export failed: " + err.Error())
		return
	}
	v.diagnosticsStatus.SetText("Metadata backup exported: " + result.Path)
	v.addActivity(fmt.Sprintf("Exported metadata backup %s (%d file(s), %d bytes).", result.Path, len(result.Files), result.SizeBytes))
}

func (v *View) exportDiagnosticsWorkspaceStateBackup() {
	workspace := v.state.Workspace()
	if workspace.Root == "" || v.metadataStore == nil {
		v.diagnosticsStatus.SetText("Open a workspace before exporting workspace state backup.")
		return
	}
	result, err := v.metadataStore.ExportWorkspaceStateBackup(metadataSvc.WorkspaceStateBackupOptions{
		SettingsPath:          v.settingsStore.Path(),
		ConnectorProfilesPath: v.connectorProfileStore.Path(),
	})
	if err != nil {
		v.diagnosticsStatus.SetText("Workspace state backup export failed.")
		v.addActivity("Workspace state backup export failed: " + err.Error())
		return
	}
	v.diagnosticsStatus.SetText("Workspace state backup exported: " + result.Path)
	v.addActivity(fmt.Sprintf("Exported workspace state backup %s (%d file(s), %d bytes).", result.Path, len(result.Files), result.SizeBytes))
}

func (v *View) exportDiagnosticsIssueReport() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.diagnosticsStatus.SetText("Open a workspace before exporting an issue report.")
		return
	}
	root := workspace.Root
	v.diagnosticsStatus.SetText("Exporting redacted issue report...")
	activityTail := v.recentActivityLines(diagnosticsActivityTailLimit)
	currentReport := strings.TrimSpace(v.diagnosticsDetail.Text)
	go func() {
		report := currentReport
		if report == "" || strings.Contains(report, "Run diagnostics to inspect") {
			snapshot := v.collectDiagnosticsSnapshot(root, activityTail)
			report = formatDiagnosticsSnapshot(snapshot)
		}
		result, err := issueReportSvc.Export(issueReportSvc.Options{
			WorkspaceRoot:     root,
			DiagnosticsReport: report,
			ActivityTail:      activityTail,
		})
		fyne.Do(func() {
			if err != nil {
				v.diagnosticsStatus.SetText("Issue report export failed.")
				v.addActivity("Issue report export failed: " + err.Error())
				return
			}
			v.diagnosticsStatus.SetText("Redacted issue report exported: " + result.Path)
			v.addActivity(fmt.Sprintf("Exported redacted issue report %s (%d file(s), %d bytes).", result.Path, len(result.Files), result.SizeBytes))
		})
	}()
}

func (v *View) refreshDiagnostics() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.diagnosticsStatus.SetText("Open a workspace before running diagnostics.")
		v.diagnosticsDetail.SetText("Diagnostics are scoped to an open workspace.")
		return
	}
	v.diagnosticsStatus.SetText("Running diagnostics...")
	root := workspace.Root
	activityTail := v.recentActivityLines(diagnosticsActivityTailLimit)
	go func() {
		snapshot := v.collectDiagnosticsSnapshot(root, activityTail)
		detail := formatDiagnosticsSnapshot(snapshot)
		status := diagnosticsStatusLine(snapshot)
		fyne.Do(func() {
			v.diagnosticsDetail.SetText(detail)
			v.diagnosticsStatus.SetText(status)
		})
	}()
}

func (v *View) collectDiagnosticsSnapshot(root string, activityTail []string) diagnosticsSnapshot {
	snapshot := diagnosticsSnapshot{
		CollectedAt:        time.Now().UTC(),
		WorkspaceRoot:      root,
		ActivityTail:       append([]string(nil), activityTail...),
		StartupRecovery:    v.startupStatus,
		PerformanceTimings: v.performanceTimings(diagnosticsPerformanceLimit),
		FailureScenarios:   readinessSvc.ProductionFailureScenarios(),
		ExternalAgentTools: externalagentsSvc.Probe(externalagentsSvc.Options{}),
	}
	if err := readinessSvc.ValidateProductionFailureScenarios(snapshot.FailureScenarios); err != nil {
		snapshot.FailureScenarioIssue = err.Error()
		snapshot.Warnings = append(snapshot.Warnings, "Failure-scenario matrix warning: "+err.Error())
	}
	if snapshot.StartupRecovery.PreviousUnclean {
		snapshot.Warnings = append(snapshot.Warnings, snapshot.StartupRecovery.Message)
	}
	for _, timing := range snapshot.PerformanceTimings {
		if !timing.WithinBudget {
			snapshot.Warnings = append(snapshot.Warnings, performanceTimingWarning(timing))
		}
	}
	inMemoryJobs := v.jobService.List()
	snapshot.InMemoryJobs = len(inMemoryJobs)
	if issue, ok := v.jobService.PersistenceIssue(); ok {
		snapshot.JobPersistenceIssue = fmt.Sprintf("%s: %s", firstNonEmptyString(issue.JobID, "latest job"), issue.Error)
		snapshot.Warnings = append(snapshot.Warnings, "Job persistence warning: "+snapshot.JobPersistenceIssue)
	}
	for _, job := range inMemoryJobs {
		switch job.Status {
		case jobsSvc.StatusRunning:
			snapshot.InMemoryRunningJobs++
		case jobsSvc.StatusFailed, jobsSvc.StatusTimedOut:
			snapshot.InMemoryFailedJobs++
			snapshot.RecentJobFailures = appendDiagnosticsDetail(
				snapshot.RecentJobFailures,
				diagnosticsFailureDetailLimit,
				formatJobFailureDetail(job),
			)
		}
	}

	settings, err := v.settingsStore.Load()
	if err != nil {
		snapshot.SettingsError = err.Error()
		snapshot.Warnings = append(snapshot.Warnings, "Settings load failed: "+err.Error())
	} else {
		snapshot.Settings = settings
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		prober := v.diagnosticsProber
		if prober == nil {
			prober = llmSvc.NewClient()
		}
		probeConfig := llmSvc.ConfigFromSettings(settings)
		probe, probeErr := prober.Probe(ctx, probeConfig)
		cancel()
		snapshot.ProviderGuidance = llmSvc.ProviderGuidance(probeConfig, probe, probeErr)
		if probeErr != nil {
			snapshot.ProbeError = probeErr.Error()
			snapshot.Warnings = append(snapshot.Warnings, "Provider probe failed: "+probeErr.Error())
		} else {
			snapshot.ProbeResult = &probe
			if !probe.OK {
				snapshot.Warnings = append(snapshot.Warnings, "Provider probe returned non-OK status.")
			}
			for _, warning := range probe.Warnings {
				snapshot.Warnings = append(snapshot.Warnings, warning)
			}
			if probe.Runtime != nil {
				snapshot.RuntimeSummary = diagnosticsRuntimeSummary(*probe.Runtime)
				if strings.TrimSpace(probe.Runtime.SelectedModel) != "" && !probe.Runtime.SelectedModelLoaded {
					snapshot.Warnings = append(snapshot.Warnings, "Selected model is not currently loaded in runtime.")
				}
			}
		}
	}

	if v.metadataStore == nil {
		snapshot.MetadataError = "metadata store is unavailable"
		snapshot.Warnings = append(snapshot.Warnings, "Metadata store is unavailable.")
		snapshot.RecommendedActions = diagnosticsRecommendedActions(snapshot)
		return snapshot
	}
	status, err := v.metadataStore.Ensure()
	if err != nil {
		snapshot.MetadataError = err.Error()
		snapshot.Warnings = append(snapshot.Warnings, "Metadata ensure failed: "+err.Error())
		snapshot.RecommendedActions = diagnosticsRecommendedActions(snapshot)
		return snapshot
	}
	snapshot.MetadataStatus = &status

	if jobs, err := v.metadataStore.ListJobs(); err == nil {
		snapshot.RecentPersistedJobs = len(jobs)
		for _, job := range jobs {
			if job.Status == jobsSvc.StatusFailed || job.Status == jobsSvc.StatusTimedOut {
				snapshot.RecentPersistedFailures++
				snapshot.RecentJobFailures = appendDiagnosticsDetail(
					snapshot.RecentJobFailures,
					diagnosticsFailureDetailLimit,
					formatJobFailureDetail(job),
				)
			}
		}
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "Could not list persisted jobs: "+err.Error())
	}
	if runs, err := v.metadataStore.ListTaskRuns(diagnosticsRecentLimit); err == nil {
		snapshot.RecentTaskRuns = len(runs)
		for _, run := range runs {
			if !isSuccessStatus(run.Status) {
				snapshot.RecentTaskFailures++
				snapshot.RecentTaskFailuresList = appendDiagnosticsDetail(
					snapshot.RecentTaskFailuresList,
					diagnosticsFailureDetailLimit,
					formatTaskFailureDetail(run),
				)
			}
		}
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "Could not list task runs: "+err.Error())
	}
	if runs, err := v.metadataStore.ListSQLRuns(diagnosticsRecentLimit); err == nil {
		snapshot.RecentSQLRuns = len(runs)
		for _, run := range runs {
			if !isSuccessStatus(run.Status) {
				snapshot.RecentSQLFailures++
				snapshot.RecentSQLFailuresList = appendDiagnosticsDetail(
					snapshot.RecentSQLFailuresList,
					diagnosticsFailureDetailLimit,
					formatSQLFailureDetail(run),
				)
			}
		}
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "Could not list SQL runs: "+err.Error())
	}
	if runs, err := v.metadataStore.ListAgentRuns(diagnosticsRecentLimit); err == nil {
		snapshot.RecentAgentRuns = len(runs)
		for _, run := range runs {
			if !isSuccessStatus(run.Status) {
				snapshot.RecentAgentFailures++
				snapshot.RecentAgentFailuresList = appendDiagnosticsDetail(
					snapshot.RecentAgentFailuresList,
					diagnosticsFailureDetailLimit,
					formatAgentFailureDetail(run),
				)
			}
		}
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "Could not list agent runs: "+err.Error())
	}
	if artifacts, err := v.metadataStore.ListArtifacts("", true, diagnosticsRecentLimit); err == nil {
		snapshot.RecentArtifacts = len(artifacts)
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "Could not list artifacts: "+err.Error())
	}
	snapshot.RecommendedActions = diagnosticsRecommendedActions(snapshot)
	return snapshot
}

func isSuccessStatus(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "success")
}

func diagnosticsStatusLine(snapshot diagnosticsSnapshot) string {
	provider := "unchecked"
	switch {
	case snapshot.SettingsError != "":
		provider = "settings-error"
	case snapshot.ProbeError != "":
		provider = "probe-error"
	case snapshot.ProbeResult == nil:
		provider = "unknown"
	case snapshot.ProbeResult.OK:
		provider = "ok"
	default:
		provider = "warning"
	}
	metadata := "ok"
	if snapshot.MetadataError != "" || snapshot.MetadataStatus == nil {
		metadata = "warning"
	}
	return fmt.Sprintf(
		"Diagnostics %s | provider %s | metadata %s | jobs %d running %d failed %d",
		snapshot.CollectedAt.Local().Format("15:04"),
		provider,
		metadata,
		snapshot.InMemoryJobs,
		snapshot.InMemoryRunningJobs,
		snapshot.InMemoryFailedJobs,
	)
}

func formatDiagnosticsSnapshot(snapshot diagnosticsSnapshot) string {
	var builder strings.Builder
	builder.WriteString("# Diagnostics\n\n")
	builder.WriteString("Collected: ")
	builder.WriteString(snapshot.CollectedAt.Local().Format("2006-01-02 15:04:05"))
	builder.WriteString("\nWorkspace: ")
	builder.WriteString(firstNonEmptyString(snapshot.WorkspaceRoot, "(none)"))
	builder.WriteString("\n\n## Health Cards\n")
	for _, card := range diagnosticsHealthCards(snapshot) {
		builder.WriteString("- **[")
		builder.WriteString(strings.ToUpper(firstNonEmptyString(card.Status, "unknown")))
		builder.WriteString("] ")
		builder.WriteString(firstNonEmptyString(card.Label, "Check"))
		builder.WriteString(":** ")
		builder.WriteString(firstNonEmptyString(card.Detail, "No detail available."))
		if action := strings.TrimSpace(card.Action); action != "" {
			builder.WriteString(" Next: ")
			builder.WriteString(action)
		}
		builder.WriteString("\n")
	}
	builder.WriteString("\n\n## Provider\n")
	if snapshot.SettingsError != "" {
		builder.WriteString("Settings error: ")
		builder.WriteString(snapshot.SettingsError)
		builder.WriteString("\n")
	} else {
		builder.WriteString("Provider: ")
		builder.WriteString(snapshot.Settings.Provider)
		builder.WriteString("\nBase URL: ")
		builder.WriteString(snapshot.Settings.BaseURL)
		builder.WriteString("\nModel: ")
		builder.WriteString(snapshot.Settings.Model)
		builder.WriteString("\n")
	}
	switch {
	case snapshot.ProbeError != "":
		builder.WriteString("Probe: failed - ")
		builder.WriteString(snapshot.ProbeError)
		builder.WriteString("\n")
	case snapshot.ProbeResult == nil:
		builder.WriteString("Probe: not available\n")
	default:
		builder.WriteString("Probe: ")
		if snapshot.ProbeResult.OK {
			builder.WriteString("ok")
		} else {
			builder.WriteString("warning")
		}
		builder.WriteString(" - ")
		builder.WriteString(snapshot.ProbeResult.Message)
		builder.WriteString(fmt.Sprintf("\nEndpoint: %s\nModels: %d\n", snapshot.ProbeResult.Endpoint, snapshot.ProbeResult.ModelCount))
		if snapshot.ProbeResult.Runtime != nil {
			builder.WriteString("Runtime: ")
			builder.WriteString(snapshot.ProbeResult.Runtime.Message)
			builder.WriteString("\n")
		}
	}
	if len(snapshot.RuntimeSummary) > 0 {
		builder.WriteString("\n## Provider Runtime\n")
		for _, line := range snapshot.RuntimeSummary {
			builder.WriteString("- ")
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}
	if len(snapshot.ProviderGuidance) > 0 {
		builder.WriteString("\n## Provider Guidance\n")
		for _, line := range snapshot.ProviderGuidance {
			builder.WriteString("- ")
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}

	builder.WriteString("\n## External Agent CLIs\n")
	builder.WriteString(externalagentsSvc.FormatMarkdown(snapshot.ExternalAgentTools))
	builder.WriteString("\n")

	builder.WriteString("\n## Startup Recovery\n")
	if snapshot.StartupRecovery.PreviousUnclean {
		builder.WriteString("Status: warning - ")
		builder.WriteString(firstNonEmptyString(snapshot.StartupRecovery.Message, "Previous run did not record a clean exit."))
		builder.WriteString("\n")
		if !snapshot.StartupRecovery.PreviousStartedAt.IsZero() {
			builder.WriteString("Previous start: ")
			builder.WriteString(snapshot.StartupRecovery.PreviousStartedAt.Local().Format("2006-01-02 15:04:05"))
			builder.WriteString("\n")
		}
	} else {
		builder.WriteString("Status: ok - clean-exit markers are active.\n")
	}
	if strings.TrimSpace(snapshot.StartupRecovery.Path) != "" {
		builder.WriteString("Marker: ")
		builder.WriteString(snapshot.StartupRecovery.Path)
		builder.WriteString("\n")
	}

	builder.WriteString("\n## Performance Timings\n")
	if len(snapshot.PerformanceTimings) == 0 {
		builder.WriteString("No startup or folder-open timings captured yet.\n")
	} else {
		for _, timing := range snapshot.PerformanceTimings {
			builder.WriteString("- ")
			builder.WriteString(formatPerformanceTiming(timing))
			builder.WriteString("\n")
		}
	}

	builder.WriteString("\n## Production Failure Gates\n")
	builder.WriteString(readinessSvc.FormatFailureScenarioMatrix(diagnosticsFailureScenarios(snapshot)))

	builder.WriteString("\n## Metadata\n")
	if snapshot.MetadataError != "" {
		builder.WriteString("Status: warning - ")
		builder.WriteString(snapshot.MetadataError)
		builder.WriteString("\n")
	} else if snapshot.MetadataStatus != nil {
		builder.WriteString("Status: ok\nPath: ")
		builder.WriteString(snapshot.MetadataStatus.Path)
		builder.WriteString("\nTables: ")
		builder.WriteString(fmt.Sprintf("%d", len(snapshot.MetadataStatus.Tables)))
		builder.WriteString("\nMessage: ")
		builder.WriteString(snapshot.MetadataStatus.Message)
		builder.WriteString("\n")
	} else {
		builder.WriteString("Status: unknown\n")
	}

	builder.WriteString("\n## Jobs\n")
	builder.WriteString(fmt.Sprintf("In-memory: %d total, %d running, %d failed\n", snapshot.InMemoryJobs, snapshot.InMemoryRunningJobs, snapshot.InMemoryFailedJobs))
	builder.WriteString(fmt.Sprintf("Persisted jobs (recent): %d total, %d non-success\n", snapshot.RecentPersistedJobs, snapshot.RecentPersistedFailures))
	builder.WriteString(fmt.Sprintf("Task runs (recent): %d total, %d non-success\n", snapshot.RecentTaskRuns, snapshot.RecentTaskFailures))
	builder.WriteString(fmt.Sprintf("SQL runs (recent): %d total, %d non-success\n", snapshot.RecentSQLRuns, snapshot.RecentSQLFailures))
	builder.WriteString(fmt.Sprintf("Agent runs (recent): %d total, %d non-success\n", snapshot.RecentAgentRuns, snapshot.RecentAgentFailures))
	builder.WriteString(fmt.Sprintf("Artifacts (recent): %d\n", snapshot.RecentArtifacts))

	builder.WriteString("\n## Recommended Actions\n")
	if len(snapshot.RecommendedActions) == 0 {
		builder.WriteString("1. Diagnostics look healthy. Keep monitoring after major runs.\n")
	} else {
		for index, action := range snapshot.RecommendedActions {
			builder.WriteString(fmt.Sprintf("%d. %s\n", index+1, action))
		}
	}

	builder.WriteString("\n## Recent Failure Triage\n")
	writeDiagnosticsDetailBlock(&builder, "Jobs", snapshot.RecentJobFailures)
	writeDiagnosticsDetailBlock(&builder, "Task runs", snapshot.RecentTaskFailuresList)
	writeDiagnosticsDetailBlock(&builder, "SQL runs", snapshot.RecentSQLFailuresList)
	writeDiagnosticsDetailBlock(&builder, "Agent runs", snapshot.RecentAgentFailuresList)

	builder.WriteString("\n## App Log Tail\n")
	if len(snapshot.ActivityTail) == 0 {
		builder.WriteString("No activity messages captured yet.\n")
	} else {
		for _, line := range snapshot.ActivityTail {
			builder.WriteString("- ")
			builder.WriteString(compactDiagnosticsLine(line, diagnosticsCompactMessageLimit))
			builder.WriteString("\n")
		}
	}

	if len(snapshot.Warnings) > 0 {
		builder.WriteString("\n## Warnings\n")
		for _, warning := range snapshot.Warnings {
			builder.WriteString("- ")
			builder.WriteString(warning)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func diagnosticsHealthCards(snapshot diagnosticsSnapshot) []diagnosticsHealthCard {
	return []diagnosticsHealthCard{
		diagnosticsProviderHealthCard(snapshot),
		diagnosticsExternalAgentsHealthCard(snapshot),
		diagnosticsMetadataHealthCard(snapshot),
		diagnosticsJobsHealthCard(snapshot),
		diagnosticsPerformanceHealthCard(snapshot),
		diagnosticsFailureGatesHealthCard(snapshot),
		diagnosticsStartupHealthCard(snapshot),
		{
			Label:  "Issue report",
			Status: "ok",
			Detail: "Redacted diagnostics export is available and excludes workspace file contents by default.",
			Action: "Use Export issue report before sharing beta bugs or release-candidate failures.",
		},
	}
}

func diagnosticsFailureGatesHealthCard(snapshot diagnosticsSnapshot) diagnosticsHealthCard {
	scenarios := diagnosticsFailureScenarios(snapshot)
	if strings.TrimSpace(snapshot.FailureScenarioIssue) != "" {
		return diagnosticsHealthCard{
			Label:  "Production failure gates",
			Status: "warning",
			Detail: "Failure-scenario matrix is incomplete: " + compactDiagnosticsLine(snapshot.FailureScenarioIssue, diagnosticsCompactMessageLimit),
			Action: "Update readiness failure scenarios before release-candidate smoke.",
		}
	}
	if err := readinessSvc.ValidateProductionFailureScenarios(scenarios); err != nil {
		return diagnosticsHealthCard{
			Label:  "Production failure gates",
			Status: "warning",
			Detail: "Failure-scenario matrix is incomplete: " + compactDiagnosticsLine(err.Error(), diagnosticsCompactMessageLimit),
			Action: "Update readiness failure scenarios before release-candidate smoke.",
		}
	}
	return diagnosticsHealthCard{
		Label:  "Production failure gates",
		Status: "ok",
		Detail: fmt.Sprintf("%d scenario(s) cover crash/hang/provider/metadata/cancel release gates.", len(scenarios)),
		Action: "Use this section during clean-machine and release-candidate smoke.",
	}
}

func diagnosticsFailureScenarios(snapshot diagnosticsSnapshot) []readinessSvc.FailureScenario {
	if len(snapshot.FailureScenarios) > 0 {
		return snapshot.FailureScenarios
	}
	return readinessSvc.ProductionFailureScenarios()
}

func diagnosticsExternalAgentsHealthCard(snapshot diagnosticsSnapshot) diagnosticsHealthCard {
	summary := externalagentsSvc.Summary(snapshot.ExternalAgentTools)
	switch {
	case len(snapshot.ExternalAgentTools) == 0:
		return diagnosticsHealthCard{
			Label:  "External agent CLIs",
			Status: "warning",
			Detail: summary,
			Action: "Keep the catalog updated before exposing execution workflows.",
		}
	case !externalagentsSvc.HasAnyAvailable(snapshot.ExternalAgentTools):
		return diagnosticsHealthCard{
			Label:  "External agent CLIs",
			Status: "action",
			Detail: summary,
			Action: "Install Codex, Claude Code, or OpenCode and ensure the command is on PATH before routing approved external-agent workflows.",
		}
	case externalagentsSvc.HasMissing(snapshot.ExternalAgentTools):
		return diagnosticsHealthCard{
			Label:  "External agent CLIs",
			Status: "warning",
			Detail: summary,
			Action: "Install missing CLIs only if those providers are part of your intended workflow.",
		}
	default:
		return diagnosticsHealthCard{
			Label:  "External agent CLIs",
			Status: "ok",
			Detail: summary,
		}
	}
}

func diagnosticsProviderHealthCard(snapshot diagnosticsSnapshot) diagnosticsHealthCard {
	switch {
	case snapshot.SettingsError != "":
		return diagnosticsHealthCard{
			Label:  "Provider",
			Status: "action",
			Detail: "Settings could not be loaded: " + compactDiagnosticsLine(snapshot.SettingsError, diagnosticsCompactMessageLimit),
			Action: "Open Settings and save the provider configuration again.",
		}
	case snapshot.ProbeError != "":
		return diagnosticsHealthCard{
			Label:  "Provider",
			Status: "action",
			Detail: "Probe failed: " + compactDiagnosticsLine(snapshot.ProbeError, diagnosticsCompactMessageLimit),
			Action: "Verify base URL, credentials, selected model, and runtime availability.",
		}
	case snapshot.ProbeResult == nil:
		return diagnosticsHealthCard{
			Label:  "Provider",
			Status: "warning",
			Detail: "Provider probe has not completed for this diagnostics snapshot.",
			Action: "Refresh Diagnostics after provider settings are saved.",
		}
	case snapshot.ProbeResult.OK:
		return diagnosticsHealthCard{
			Label:  "Provider",
			Status: "ok",
			Detail: firstNonEmptyString(compactDiagnosticsLine(snapshot.ProbeResult.Message, diagnosticsCompactMessageLimit), "Provider probe passed."),
		}
	default:
		return diagnosticsHealthCard{
			Label:  "Provider",
			Status: "warning",
			Detail: firstNonEmptyString(compactDiagnosticsLine(snapshot.ProbeResult.Message, diagnosticsCompactMessageLimit), "Provider returned a warning status."),
			Action: "Review Provider Guidance and rerun Test connection.",
		}
	}
}

func diagnosticsMetadataHealthCard(snapshot diagnosticsSnapshot) diagnosticsHealthCard {
	if snapshot.MetadataError != "" || snapshot.MetadataStatus == nil {
		return diagnosticsHealthCard{
			Label:  "Metadata",
			Status: "action",
			Detail: firstNonEmptyString(compactDiagnosticsLine(snapshot.MetadataError, diagnosticsCompactMessageLimit), "Metadata store is unavailable."),
			Action: "Export metadata backup if possible, then inspect .nexusdesk/metadata recovery state.",
		}
	}
	return diagnosticsHealthCard{
		Label:  "Metadata",
		Status: "ok",
		Detail: fmt.Sprintf("%d table(s). %s", len(snapshot.MetadataStatus.Tables), firstNonEmptyString(compactDiagnosticsLine(snapshot.MetadataStatus.Message, diagnosticsCompactMessageLimit), "SQLite metadata store is active.")),
	}
}

func diagnosticsJobsHealthCard(snapshot diagnosticsSnapshot) diagnosticsHealthCard {
	failures := snapshot.InMemoryFailedJobs + snapshot.RecentPersistedFailures + snapshot.RecentTaskFailures + snapshot.RecentSQLFailures + snapshot.RecentAgentFailures
	total := snapshot.InMemoryJobs + snapshot.RecentPersistedJobs + snapshot.RecentTaskRuns + snapshot.RecentSQLRuns + snapshot.RecentAgentRuns
	if strings.TrimSpace(snapshot.JobPersistenceIssue) != "" {
		return diagnosticsHealthCard{
			Label:  "Jobs and runs",
			Status: "warning",
			Detail: "Latest job metadata save failed: " + compactDiagnosticsLine(snapshot.JobPersistenceIssue, diagnosticsCompactMessageLimit),
			Action: "Check disk space and metadata health, then retry or export an issue report.",
		}
	}
	if failures > 0 {
		return diagnosticsHealthCard{
			Label:  "Jobs and runs",
			Status: "action",
			Detail: fmt.Sprintf("%d non-success item(s) across %d recent/in-memory job, task, SQL, and agent record(s).", failures, total),
			Action: "Open Jobs and Agent Audit, inspect failures, then retry only safe workloads.",
		}
	}
	if snapshot.InMemoryRunningJobs > 0 {
		return diagnosticsHealthCard{
			Label:  "Jobs and runs",
			Status: "warning",
			Detail: fmt.Sprintf("%d job(s) currently running; recent persisted records show no failures.", snapshot.InMemoryRunningJobs),
			Action: "Wait for running jobs to finish before packaging or exporting a release report.",
		}
	}
	return diagnosticsHealthCard{
		Label:  "Jobs and runs",
		Status: "ok",
		Detail: fmt.Sprintf("%d recent/in-memory job, task, SQL, and agent record(s); no non-success records detected.", total),
	}
}

func diagnosticsPerformanceHealthCard(snapshot diagnosticsSnapshot) diagnosticsHealthCard {
	if hasOverBudgetPerformanceTiming(snapshot.PerformanceTimings) {
		return diagnosticsHealthCard{
			Label:  "Performance",
			Status: "warning",
			Detail: "At least one startup or folder-open timing is over budget.",
			Action: "Review Performance Timings before scaling to larger repositories.",
		}
	}
	if len(snapshot.PerformanceTimings) == 0 {
		return diagnosticsHealthCard{
			Label:  "Performance",
			Status: "warning",
			Detail: "No startup or folder-open timings were captured yet.",
			Action: "Open a workspace and rerun Diagnostics before release-candidate smoke.",
		}
	}
	return diagnosticsHealthCard{
		Label:  "Performance",
		Status: "ok",
		Detail: fmt.Sprintf("%d timing record(s) captured and within budget.", len(snapshot.PerformanceTimings)),
	}
}

func diagnosticsStartupHealthCard(snapshot diagnosticsSnapshot) diagnosticsHealthCard {
	if snapshot.StartupRecovery.PreviousUnclean {
		return diagnosticsHealthCard{
			Label:  "Startup recovery",
			Status: "warning",
			Detail: firstNonEmptyString(compactDiagnosticsLine(snapshot.StartupRecovery.Message, diagnosticsCompactMessageLimit), "Previous run did not record a clean exit."),
			Action: "Review Jobs, Agent Audit, and metadata health before repeating long workflows.",
		}
	}
	return diagnosticsHealthCard{
		Label:  "Startup recovery",
		Status: "ok",
		Detail: "Clean-exit markers are active.",
	}
}

func diagnosticsRuntimeSummary(runtime llmSvc.RuntimeStatus) []string {
	summary := []string{
		"Provider: " + firstNonEmptyString(runtime.Provider, "unknown"),
		"Endpoint: " + firstNonEmptyString(runtime.Endpoint, "(none)"),
		"Runtime message: " + firstNonEmptyString(runtime.Message, "(none)"),
	}
	selected := firstNonEmptyString(runtime.SelectedModel, "(none)")
	summary = append(summary, fmt.Sprintf("Selected model: %s (loaded=%t, vram=%s)", selected, runtime.SelectedModelLoaded, formatDiagnosticsBytes(runtime.SelectedModelVRAM)))
	summary = append(summary, fmt.Sprintf("Loaded models: %d", len(runtime.LoadedModels)))
	for index, model := range runtime.LoadedModels {
		if index >= diagnosticsRuntimeModelLimit {
			break
		}
		name := firstNonEmptyString(model.Name, model.Model, fmt.Sprintf("model-%d", index+1))
		context := "unknown"
		if model.ContextLength > 0 {
			context = fmt.Sprintf("%d", model.ContextLength)
		}
		summary = append(
			summary,
			fmt.Sprintf("%s | ctx %s | size %s | vram %s", name, context, formatDiagnosticsBytes(model.Size), formatDiagnosticsBytes(model.SizeVRAM)),
		)
	}
	if len(runtime.LoadedModels) > diagnosticsRuntimeModelLimit {
		summary = append(summary, fmt.Sprintf("... %d more loaded model(s)", len(runtime.LoadedModels)-diagnosticsRuntimeModelLimit))
	}
	return summary
}

func formatDiagnosticsBytes(value int64) string {
	if value <= 0 {
		return "n/a"
	}
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case value >= gb:
		return fmt.Sprintf("%.2f GiB", float64(value)/float64(gb))
	case value >= mb:
		return fmt.Sprintf("%.1f MiB", float64(value)/float64(mb))
	case value >= kb:
		return fmt.Sprintf("%.1f KiB", float64(value)/float64(kb))
	default:
		return fmt.Sprintf("%d B", value)
	}
}

func writeDiagnosticsDetailBlock(builder *strings.Builder, title string, lines []string) {
	builder.WriteString(title)
	builder.WriteString(": ")
	if len(lines) == 0 {
		builder.WriteString("none\n")
		return
	}
	builder.WriteString("\n")
	for _, line := range lines {
		builder.WriteString("- ")
		builder.WriteString(compactDiagnosticsLine(line, diagnosticsCompactMessageLimit))
		builder.WriteString("\n")
	}
}

func appendDiagnosticsDetail(existing []string, limit int, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || limit <= 0 {
		return existing
	}
	if len(existing) >= limit {
		return existing
	}
	return append(existing, value)
}

func formatJobFailureDetail(job jobsSvc.Job) string {
	reason := firstNonEmptyString(job.Error, job.Message)
	if len(job.LogTail) > 0 {
		reason = firstNonEmptyString(reason, job.LogTail[len(job.LogTail)-1])
	}
	return fmt.Sprintf(
		"%s [%s] %s: %s",
		firstNonEmptyString(strings.TrimSpace(job.ID), "job"),
		strings.TrimSpace(string(job.Status)),
		firstNonEmptyString(strings.TrimSpace(job.Label), strings.TrimSpace(job.Kind), "(no label)"),
		firstNonEmptyString(compactDiagnosticsLine(reason, diagnosticsCompactMessageLimit), "(no failure message)"),
	)
}

func formatTaskFailureDetail(run metadataSvc.TaskRunRecord) string {
	status := strings.TrimSpace(run.Status)
	label := firstNonEmptyString(run.Label, run.TaskID, run.ID)
	reason := firstNonEmptyString(run.Message, run.Stderr, run.Stdout)
	return fmt.Sprintf(
		"%s [%s] %s exit %d: %s",
		firstNonEmptyString(run.ID, "task-run"),
		firstNonEmptyString(status, "unknown"),
		label,
		run.ExitCode,
		firstNonEmptyString(compactDiagnosticsLine(reason, diagnosticsCompactMessageLimit), "(no failure message)"),
	)
}

func formatSQLFailureDetail(run metadataSvc.SQLRunRecord) string {
	status := firstNonEmptyString(run.Status, "unknown")
	reason := firstNonEmptyString(run.Error, run.Message)
	return fmt.Sprintf(
		"%s [%s %s] %s: %s",
		firstNonEmptyString(run.ID, "sql-run"),
		firstNonEmptyString(run.Engine, "sql"),
		status,
		firstNonEmptyString(run.RelPath, "(unknown source)"),
		firstNonEmptyString(compactDiagnosticsLine(reason, diagnosticsCompactMessageLimit), "(no failure message)"),
	)
}

func formatAgentFailureDetail(run metadataSvc.AgentRunRecord) string {
	status := firstNonEmptyString(run.Status, "unknown")
	reason := firstNonEmptyString(run.Message, run.StopReason)
	return fmt.Sprintf(
		"%s [%s] iter %d stop %s: %s",
		firstNonEmptyString(run.ID, "agent-run"),
		status,
		run.Iterations,
		firstNonEmptyString(run.StopReason, "unknown"),
		firstNonEmptyString(compactDiagnosticsLine(reason, diagnosticsCompactMessageLimit), "(no failure message)"),
	)
}

func compactDiagnosticsLine(value string, limit int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func diagnosticsRecommendedActions(snapshot diagnosticsSnapshot) []string {
	actions := []string{}
	if snapshot.SettingsError != "" {
		actions = append(actions, "Open Settings and fix provider configuration load errors.")
	} else if snapshot.ProbeError != "" {
		actions = append(actions, "Open Settings and verify provider base URL, credentials, and selected model.")
	} else if snapshot.ProbeResult != nil && !snapshot.ProbeResult.OK {
		actions = append(actions, "Run provider probe again after checking model availability and endpoint health.")
	}
	actions = append(actions, snapshot.ProviderGuidance...)
	if snapshot.ProbeResult != nil && snapshot.ProbeResult.Runtime != nil && strings.TrimSpace(snapshot.ProbeResult.Runtime.SelectedModel) != "" && !snapshot.ProbeResult.Runtime.SelectedModelLoaded {
		actions = append(actions, "Load the selected model in your runtime or switch to an already-loaded model in Settings.")
	}
	if !externalagentsSvc.HasAnyAvailable(snapshot.ExternalAgentTools) {
		actions = append(actions, "Install at least one external coding-agent CLI such as Codex, Claude Code, or OpenCode before enabling future approved external-agent workflows.")
	} else if externalagentsSvc.HasMissing(snapshot.ExternalAgentTools) {
		actions = append(actions, "Review External Agent CLIs and install any missing tools you expect NexusDesk to orchestrate later.")
	}
	if snapshot.MetadataError != "" {
		actions = append(actions, "Inspect metadata health and recover .nexusdesk/metadata before continuing long runs.")
		actions = append(actions, "Use Diagnostics Export metadata to create a backup before attempting manual recovery.")
	}
	if snapshot.RecentPersistedFailures > 0 || snapshot.RecentTaskFailures > 0 || snapshot.RecentSQLFailures > 0 || snapshot.RecentAgentFailures > 0 || snapshot.InMemoryFailedJobs > 0 {
		actions = append(actions, "Open Jobs and Agent Audit tabs to inspect recent failures and retry safe workloads.")
	}
	if snapshot.StartupRecovery.PreviousUnclean {
		actions = append(actions, "Review Startup Recovery, Jobs, Agent Audit, and metadata health before repeating any long workflow from the previous session.")
	}
	if hasOverBudgetPerformanceTiming(snapshot.PerformanceTimings) {
		actions = append(actions, "Review Performance Timings for slow startup or folder-open work before scaling to larger repositories.")
	}
	if len(snapshot.ActivityTail) == 0 {
		actions = append(actions, "Trigger one small task and rerun diagnostics to populate runtime activity context.")
	}
	return actions
}
