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
	jobsSvc "nexusdesk/internal/services/jobs"
	metadataSvc "nexusdesk/internal/services/metadata"
)

type jobsController struct {
	view    *View
	results *fyne.Container
	status  *widget.Label
}

func newJobsController(view *View) *jobsController {
	return &jobsController{
		view:    view,
		results: container.NewVBox(widget.NewLabel("Run a task to create a job record.")),
		status:  widget.NewLabel("No jobs yet."),
	}
}

func (v *View) newJobsPanel() fyne.CanvasObject {
	return v.jobs.Panel()
}

func (v *View) refreshJobs() {
	v.jobs.Refresh()
}

func (v *View) cancelJob(id string) {
	v.jobs.Cancel(id)
}

func (v *View) confirmPruneJobs() {
	v.jobs.ConfirmPrune()
}

func (v *View) confirmRetryJob(id string) {
	v.jobs.ConfirmRetry(id)
}

func (v *View) openJobOutput(id string) {
	v.jobs.OpenOutput(id)
}

func (c *jobsController) Panel() fyne.CanvasObject {
	refresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), c.Refresh)
	cleanup := widget.NewButtonWithIcon("Clean Up", theme.DeleteIcon(), c.ConfirmPrune)
	header := container.NewBorder(nil, nil, c.status, container.NewHBox(refresh, cleanup))
	scroll := container.NewScroll(c.results)
	scroll.SetMinSize(fyne.NewSize(240, 110))
	return container.NewBorder(header, nil, nil, nil, scroll)
}

func (c *jobsController) Refresh() {
	jobs := c.view.jobService.List()
	status := fmt.Sprintf("%d job(s)", len(jobs))
	if issue, ok := c.view.jobService.PersistenceIssue(); ok {
		status = fmt.Sprintf("%s - persistence warning on %s: %s", status, firstNonEmptyString(issue.JobID, "latest job"), issue.Error)
	}
	c.status.SetText(status)
	c.results.Objects = jobRows(jobs, c.Cancel, c.ConfirmRetry, c.OpenOutput, c.view.taskRunsByJob(), c.view.artifactOutputsByJob())
	c.results.Refresh()
	c.view.refreshStatusBar()
}

func (c *jobsController) Cancel(id string) {
	if c.view.jobService.Cancel(id) {
		c.view.addActivity("Cancel requested for " + id + ".")
	}
	c.Refresh()
}

func (c *jobsController) ConfirmPrune() {
	policy := jobsSvc.DefaultRetentionPolicy()
	message := fmt.Sprintf(
		"Remove completed successful/canceled jobs older than %s or beyond the latest %d retained records. Running jobs and failed/timed-out jobs are kept.",
		formatRetentionAge(policy.MaxAge),
		policy.KeepRecent,
	)
	dialog.ShowConfirm("Clean up job history", message, func(confirm bool) {
		if !confirm {
			return
		}
		result, err := c.view.jobService.Prune(policy)
		if err != nil {
			dialog.ShowError(err, c.view.window)
			c.view.addActivity("Job history cleanup failed: " + err.Error())
			return
		}
		c.status.SetText(fmt.Sprintf("Removed %d job(s); kept %d.", result.Removed, result.Kept+result.RunningKept+result.FailuresKept))
		c.view.addActivity(fmt.Sprintf("Cleaned up %d completed job(s); running and failed jobs were preserved.", result.Removed))
		c.Refresh()
	}, c.view.window)
}

func (c *jobsController) ConfirmRetry(id string) {
	record, ok := c.view.latestTaskRunForJob(id)
	if !ok {
		c.status.SetText("No persisted task run found for " + id + ".")
		return
	}
	workspace := c.view.state.Workspace()
	if workspace.Root == "" {
		c.status.SetText("Open a workspace before retrying a job.")
		return
	}
	task, found, err := c.view.taskService.Find(workspace.Root, record.TaskID)
	if err != nil {
		dialog.ShowError(err, c.view.window)
		return
	}
	if !found {
		c.status.SetText("Task is no longer discoverable: " + record.TaskID + ".")
		c.view.addActivity("Retry blocked because task is no longer discoverable: " + record.TaskID + ".")
		return
	}
	dialog.ShowConfirm("Retry task", "Run "+task.Label+" again?", func(confirm bool) {
		if !confirm {
			return
		}
		c.view.runTask(task)
	}, c.view.window)
}

func (c *jobsController) OpenOutput(id string) {
	record, ok := c.view.latestTaskRunForJob(id)
	if !ok {
		if artifact, hasArtifact := c.view.latestArtifactOutputForJob(id); hasArtifact {
			if c.view.openArtifactOutputByPath(artifact.RelPath, "Opened artifact output "+artifact.RelPath+".") {
				return
			}
		}
		if job, hasJob := c.view.jobService.Get(id); hasJob {
			c.view.taskOutput.SetText(formatJobRecord(job))
			c.view.taskStatus.SetText("Opened job output for " + id + ".")
			c.view.addActivity("Opened job output for " + id + ".")
			return
		}
		c.status.SetText("No persisted output found for " + id + ".")
		return
	}
	if strings.TrimSpace(record.ArtifactPath) != "" && c.view.openTaskRunArtifactOutput(record) {
		return
	}
	c.view.taskOutput.SetText(formatTaskRunRecord(record))
	c.view.taskStatus.SetText("Opened task output for " + id + ".")
	c.view.addActivity("Opened task output for " + id + ".")
}

func (v *View) openTaskRunArtifactOutput(record metadataSvc.TaskRunRecord) bool {
	if strings.TrimSpace(record.ArtifactPath) == "" {
		return false
	}
	return v.openArtifactOutputByPath(record.ArtifactPath, "Opened task report artifact "+record.ArtifactPath+".")
}

func (v *View) openArtifactOutputByPath(relPath string, activity string) bool {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		return false
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return false
	}
	artifacts, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: relPath, IncludeArchived: true})
	if err != nil {
		dialog.ShowError(err, v.window)
		return false
	}
	for _, artifact := range artifacts {
		if artifact.RelPath != relPath {
			continue
		}
		v.previewArtifact(artifact)
		if preview, err := v.workspaceService.PreviewFile(workspace.Root, artifact.RelPath); err == nil {
			v.openPreviewTab(preview)
		}
		v.refreshArtifactsWithQuery(relPath)
		if strings.TrimSpace(activity) != "" {
			v.addActivity(activity)
		}
		return true
	}
	return false
}

func (v *View) latestTaskRunForJob(id string) (metadataSvc.TaskRunRecord, bool) {
	if v.metadataStore == nil {
		return metadataSvc.TaskRunRecord{}, false
	}
	record, ok, err := v.metadataStore.LatestTaskRunForJob(id)
	if err != nil {
		v.addActivity("Could not read task output for " + id + ": " + err.Error())
		return metadataSvc.TaskRunRecord{}, false
	}
	return record, ok
}

func (v *View) taskRunsByJob() map[string]metadataSvc.TaskRunRecord {
	out := map[string]metadataSvc.TaskRunRecord{}
	if v.metadataStore == nil {
		return out
	}
	runs, err := v.metadataStore.ListTaskRuns(200)
	if err != nil {
		v.addActivity("Could not read task-run records: " + err.Error())
		return out
	}
	for _, run := range runs {
		if run.JobID == "" {
			continue
		}
		if _, exists := out[run.JobID]; !exists {
			out[run.JobID] = run
		}
	}
	return out
}

func (v *View) artifactOutputsByJob() map[string]metadataSvc.ArtifactRecord {
	out := map[string]metadataSvc.ArtifactRecord{}
	if v.metadataStore == nil {
		return out
	}
	artifacts, err := v.metadataStore.ListArtifacts("", true, 400)
	if err != nil {
		v.addActivity("Could not read artifact records: " + err.Error())
		return out
	}
	for _, artifact := range artifacts {
		jobID := strings.TrimSpace(artifact.JobID)
		if jobID == "" || strings.TrimSpace(artifact.RelPath) == "" {
			continue
		}
		if existing, ok := out[jobID]; !ok || artifactRecordSortTime(artifact).After(artifactRecordSortTime(existing)) {
			out[jobID] = artifact
		}
	}
	return out
}

func (v *View) latestArtifactOutputForJob(jobID string) (metadataSvc.ArtifactRecord, bool) {
	artifacts := v.artifactOutputsByJob()
	record, ok := artifacts[strings.TrimSpace(jobID)]
	return record, ok
}

func artifactRecordSortTime(record metadataSvc.ArtifactRecord) time.Time {
	if !record.GeneratedAt.IsZero() {
		return record.GeneratedAt
	}
	if !record.CreatedAt.IsZero() {
		return record.CreatedAt
	}
	if !record.UpdatedAt.IsZero() {
		return record.UpdatedAt
	}
	return time.Time{}
}

func jobRows(
	jobs []jobsSvc.Job,
	onCancel func(string),
	onRetry func(string),
	onOpenOutput func(string),
	taskRuns map[string]metadataSvc.TaskRunRecord,
	artifactOutputs map[string]metadataSvc.ArtifactRecord,
) []fyne.CanvasObject {
	if len(jobs) == 0 {
		return []fyne.CanvasObject{widget.NewLabel("No jobs yet.")}
	}
	rows := make([]fyne.CanvasObject, 0, len(jobs))
	for _, job := range jobs {
		job := job
		cancel := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
			onCancel(job.ID)
		})
		cancel.Importance = widget.LowImportance
		if job.Status != jobsSvc.StatusRunning {
			cancel.Disable()
		}
		retry := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
			onRetry(job.ID)
		})
		retry.Importance = widget.LowImportance
		output := widget.NewButtonWithIcon("", theme.DocumentIcon(), func() {
			onOpenOutput(job.ID)
		})
		output.Importance = widget.LowImportance
		taskRun, hasTaskRun := taskRuns[job.ID]
		_, hasArtifactOutput := artifactOutputs[job.ID]
		if job.Status == jobsSvc.StatusRunning || !hasTaskRun || strings.TrimSpace(taskRun.TaskID) == "" {
			retry.Disable()
		}
		if (!hasTaskRun || !taskRunHasOutput(taskRun)) && !hasArtifactOutput && !jobHasOutput(job) {
			output.Disable()
		}
		title := widget.NewLabel(fmt.Sprintf("%s - %s", job.ID, job.Label))
		title.TextStyle = fyne.TextStyle{Bold: true}
		meta := widget.NewLabel(fmt.Sprintf("%s - %s", job.Kind, job.Status))
		meta.Truncation = fyne.TextTruncateEllipsis
		message := widget.NewLabel(jobSummary(job))
		message.Truncation = fyne.TextTruncateEllipsis
		rows = append(rows, container.NewBorder(nil, nil, container.NewHBox(cancel, retry, output), nil, container.NewVBox(title, meta, message)))
	}
	return rows
}

func jobSummary(job jobsSvc.Job) string {
	parts := []string{}
	if job.Message != "" {
		parts = append(parts, job.Message)
	}
	if job.Error != "" {
		parts = append(parts, job.Error)
	}
	if len(job.LogTail) > 0 {
		parts = append(parts, job.LogTail[len(job.LogTail)-1])
	}
	if len(parts) == 0 {
		return string(job.Status)
	}
	return strings.Join(parts, " - ")
}

func taskRunHasOutput(record metadataSvc.TaskRunRecord) bool {
	return strings.TrimSpace(record.ArtifactPath) != "" ||
		strings.TrimSpace(record.Stdout) != "" ||
		strings.TrimSpace(record.Stderr) != "" ||
		strings.TrimSpace(record.Message) != ""
}

func jobHasOutput(job jobsSvc.Job) bool {
	return strings.TrimSpace(job.Message) != "" || strings.TrimSpace(job.Error) != "" || len(job.LogTail) > 0
}

func formatRetentionAge(value time.Duration) string {
	if value <= 0 {
		return "the configured age"
	}
	hours := int(value.Hours())
	if hours%24 == 0 {
		days := hours / 24
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
	return value.String()
}

func formatJobRecord(job jobsSvc.Job) string {
	var builder strings.Builder
	builder.WriteString(firstNonEmptyString(strings.TrimSpace(job.Message), "Job output"))
	builder.WriteString("\n")
	builder.WriteString("Status: ")
	builder.WriteString(strings.TrimSpace(string(job.Status)))
	builder.WriteString("\nKind: ")
	builder.WriteString(strings.TrimSpace(job.Kind))
	builder.WriteString("\nLabel: ")
	builder.WriteString(strings.TrimSpace(job.Label))
	if !job.StartedAt.IsZero() {
		builder.WriteString("\nStarted: ")
		builder.WriteString(job.StartedAt.Local().Format("2006-01-02 15:04:05"))
	}
	if !job.CompletedAt.IsZero() {
		builder.WriteString("\nCompleted: ")
		builder.WriteString(job.CompletedAt.Local().Format("2006-01-02 15:04:05"))
		builder.WriteString("\nDuration: ")
		builder.WriteString(job.CompletedAt.Sub(job.StartedAt).String())
	}
	if strings.TrimSpace(job.Error) != "" {
		builder.WriteString("\n\nError\n")
		builder.WriteString(job.Error)
		if !strings.HasSuffix(job.Error, "\n") {
			builder.WriteString("\n")
		}
	}
	if len(job.LogTail) > 0 {
		builder.WriteString("\nLog tail\n")
		for _, line := range job.LogTail {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			builder.WriteString("- ")
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func formatTaskRunRecord(record metadataSvc.TaskRunRecord) string {
	var builder strings.Builder
	builder.WriteString(record.Message)
	builder.WriteString("\n")
	builder.WriteString("Status: ")
	builder.WriteString(record.Status)
	builder.WriteString("\nExit code: ")
	builder.WriteString(fmt.Sprintf("%d", record.ExitCode))
	builder.WriteString("\nCommand: ")
	builder.WriteString(record.Command)
	builder.WriteString("\nCwd: ")
	builder.WriteString(record.Cwd)
	if strings.TrimSpace(record.ArtifactPath) != "" {
		builder.WriteString("\nArtifact: ")
		builder.WriteString(record.ArtifactPath)
	}
	builder.WriteString(fmt.Sprintf("\nDuration: %d ms", record.DurationMs))
	builder.WriteString("\n\nStdout\n")
	builder.WriteString(record.Stdout)
	if !strings.HasSuffix(record.Stdout, "\n") {
		builder.WriteString("\n")
	}
	builder.WriteString("\nStderr\n")
	builder.WriteString(record.Stderr)
	if !strings.HasSuffix(record.Stderr, "\n") {
		builder.WriteString("\n")
	}
	return builder.String()
}
