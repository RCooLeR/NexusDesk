package shell

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	artifactsSvc "nexusdesk/internal/services/artifacts"
	jobsSvc "nexusdesk/internal/services/jobs"
	metadataSvc "nexusdesk/internal/services/metadata"
)

func (v *View) newJobsPanel() fyne.CanvasObject {
	refresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), v.refreshJobs)
	header := container.NewBorder(nil, nil, v.jobStatus, refresh)
	scroll := container.NewScroll(v.jobResults)
	scroll.SetMinSize(fyne.NewSize(240, 110))
	return container.NewBorder(header, nil, nil, nil, scroll)
}

func (v *View) refreshJobs() {
	jobs := v.jobService.List()
	v.jobStatus.SetText(fmt.Sprintf("%d job(s)", len(jobs)))
	v.jobResults.Objects = jobRows(jobs, v.cancelJob, v.confirmRetryJob, v.openJobOutput, v.taskRunsByJob())
	v.jobResults.Refresh()
}

func (v *View) cancelJob(id string) {
	if v.jobService.Cancel(id) {
		v.addActivity("Cancel requested for " + id + ".")
	}
	v.refreshJobs()
}

func (v *View) confirmRetryJob(id string) {
	record, ok := v.latestTaskRunForJob(id)
	if !ok {
		v.jobStatus.SetText("No persisted task run found for " + id + ".")
		return
	}
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.jobStatus.SetText("Open a workspace before retrying a job.")
		return
	}
	task, found, err := v.taskService.Find(workspace.Root, record.TaskID)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	if !found {
		v.jobStatus.SetText("Task is no longer discoverable: " + record.TaskID + ".")
		v.addActivity("Retry blocked because task is no longer discoverable: " + record.TaskID + ".")
		return
	}
	dialog.ShowConfirm("Retry task", "Run "+task.Label+" again?", func(confirm bool) {
		if !confirm {
			return
		}
		v.runTask(task)
	}, v.window)
}

func (v *View) openJobOutput(id string) {
	record, ok := v.latestTaskRunForJob(id)
	if !ok {
		v.jobStatus.SetText("No persisted task output found for " + id + ".")
		return
	}
	if strings.TrimSpace(record.ArtifactPath) != "" && v.openTaskRunArtifactOutput(record) {
		return
	}
	v.taskOutput.SetText(formatTaskRunRecord(record))
	v.taskStatus.SetText("Opened task output for " + id + ".")
	v.addActivity("Opened task output for " + id + ".")
}

func (v *View) openTaskRunArtifactOutput(record metadataSvc.TaskRunRecord) bool {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		return false
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return false
	}
	artifacts, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: record.ArtifactPath, IncludeArchived: true})
	if err != nil {
		dialog.ShowError(err, v.window)
		return false
	}
	for _, artifact := range artifacts {
		if artifact.RelPath != record.ArtifactPath {
			continue
		}
		v.previewArtifact(artifact)
		if preview, err := v.workspaceService.PreviewFile(workspace.Root, artifact.RelPath); err == nil {
			v.openPreviewTab(preview)
		}
		v.refreshArtifactsWithQuery(record.ArtifactPath)
		v.addActivity("Opened task report artifact " + record.ArtifactPath + ".")
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

func jobRows(jobs []jobsSvc.Job, onCancel func(string), onRetry func(string), onOpenOutput func(string), taskRuns map[string]metadataSvc.TaskRunRecord) []fyne.CanvasObject {
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
		if job.Status == jobsSvc.StatusRunning || !hasTaskRun || strings.TrimSpace(taskRun.TaskID) == "" {
			retry.Disable()
		}
		if !hasTaskRun || !taskRunHasOutput(taskRun) {
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
