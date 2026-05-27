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
	tasksSvc "nexusdesk/internal/services/tasks"
)

func (v *View) newTasksPanel() fyne.CanvasObject {
	discover := widget.NewButtonWithIcon("Discover", theme.SearchIcon(), v.discoverTasks)
	header := container.NewBorder(nil, nil, v.taskStatus, discover)
	taskScroll := container.NewScroll(v.taskResults)
	taskScroll.SetMinSize(fyne.NewSize(260, 110))
	output := container.NewBorder(widget.NewLabel("Last task output"), nil, nil, nil, v.taskOutput)
	split := container.NewVSplit(taskScroll, output)
	split.Offset = 0.52
	return container.NewBorder(header, nil, nil, nil, split)
}

func (v *View) discoverTasks() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.taskStatus.SetText("Open a workspace before discovering tasks.")
		v.addActivity("Open a workspace before discovering tasks.")
		return
	}
	summary, err := v.taskService.Discover(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.taskStatus.SetText(summary.Message)
	v.taskResults.Objects = taskRows(summary.Tasks, v.confirmRunTask)
	v.taskResults.Refresh()
	v.addActivity(summary.Message)
}

func (v *View) confirmRunTask(task tasksSvc.Task) {
	message := fmt.Sprintf("Run %s in %s?", task.Label, task.Cwd)
	dialog.ShowConfirm("Run task", message, func(confirm bool) {
		if !confirm {
			return
		}
		v.runTask(task)
	}, v.window)
}

func (v *View) runTask(task tasksSvc.Task) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.taskStatus.SetText("Open a workspace before running tasks.")
		v.addActivity("Open a workspace before running tasks.")
		return
	}
	job, ctx := v.jobService.Start("task", task.Label)
	v.jobService.AppendLog(job.ID, "Command: "+task.Command)
	v.taskStatus.SetText("Running " + task.Label + " as " + job.ID + ".")
	v.taskOutput.SetText("Running " + task.Label + " as " + job.ID + "...\n")
	v.addActivity("Started " + job.ID + ": " + task.Label + ".")
	v.refreshJobs()
	root := workspace.Root
	go func() {
		result, err := v.taskService.RunContext(ctx, root, task.ID)
		fyne.Do(func() {
			v.finishTaskJob(job.ID, result, err)
		})
	}()
}

func (v *View) finishTaskJob(jobID string, result tasksSvc.RunResult, err error) {
	if err != nil {
		v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Task failed before execution.", err)
		dialog.ShowError(err, v.window)
		v.taskStatus.SetText("Task failed before execution.")
		v.refreshJobs()
		return
	}
	v.jobService.AppendLog(jobID, taskRunLogLine(result))
	v.jobService.Finish(jobID, jobStatusFromTask(result), result.Message, nil)
	v.persistTaskRun(jobID, result)
	v.taskStatus.SetText(result.Message)
	v.taskOutput.SetText(formatTaskRun(result))
	v.addActivity(result.Message)
	v.refreshTaskRows()
	v.refreshJobs()
}

func (v *View) persistTaskRun(jobID string, result tasksSvc.RunResult) {
	if v.metadataStore == nil {
		return
	}
	record := v.metadataStore.NormalizeTaskRunRecord(taskRunRecord(jobID, result))
	if artifact, err := writeTaskRunArtifact(v.state.Workspace().Root, record); err == nil {
		record.ArtifactPath = artifact.RelPath
		v.persistArtifactRecord(artifact)
		v.addActivity(artifact.Message)
	} else {
		v.addActivity("Could not write task report artifact: " + err.Error())
	}
	if err := v.metadataStore.SaveTaskRun(record); err != nil {
		v.addActivity("Could not persist task run: " + err.Error())
	}
	v.refreshArtifacts()
}

func writeTaskRunArtifact(root string, record metadataSvc.TaskRunRecord) (artifactsSvc.Artifact, error) {
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	return store.WriteTaskRunReport(taskRunArtifactInput(record))
}

func taskRunArtifactInput(record metadataSvc.TaskRunRecord) artifactsSvc.TaskRunReport {
	return artifactsSvc.TaskRunReport{
		ID:          record.ID,
		JobID:       record.JobID,
		TaskID:      record.TaskID,
		Kind:        record.Kind,
		Label:       record.Label,
		Command:     record.Command,
		Cwd:         record.Cwd,
		Source:      record.Source,
		Status:      record.Status,
		ExitCode:    record.ExitCode,
		Stdout:      record.Stdout,
		Stderr:      record.Stderr,
		Message:     record.Message,
		StartedAt:   record.StartedAt,
		CompletedAt: record.CompletedAt,
		DurationMs:  record.DurationMs,
	}
}

func taskRunRecord(jobID string, result tasksSvc.RunResult) metadataSvc.TaskRunRecord {
	return metadataSvc.TaskRunRecord{
		JobID:       jobID,
		TaskID:      result.Task.ID,
		Kind:        result.Task.Kind,
		Label:       result.Task.Label,
		Command:     result.Task.Command,
		Cwd:         result.Task.Cwd,
		Source:      result.Task.Source,
		Status:      result.Status,
		ExitCode:    result.ExitCode,
		Stdout:      result.Stdout,
		Stderr:      result.Stderr,
		Message:     result.Message,
		StartedAt:   result.StartedAt,
		CompletedAt: result.CompletedAt,
		DurationMs:  result.Duration.Milliseconds(),
	}
}

func (v *View) refreshTaskRows() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		return
	}
	summary, err := v.taskService.Discover(workspace.Root)
	if err != nil {
		return
	}
	v.taskResults.Objects = taskRows(summary.Tasks, v.confirmRunTask)
	v.taskResults.Refresh()
}

func taskRows(tasks []tasksSvc.Task, onRun func(tasksSvc.Task)) []fyne.CanvasObject {
	if len(tasks) == 0 {
		return []fyne.CanvasObject{widget.NewLabel("No package scripts, Go tests, or Compose checks detected.")}
	}
	rows := make([]fyne.CanvasObject, 0, len(tasks))
	for _, task := range tasks {
		task := task
		run := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
			onRun(task)
		})
		run.Importance = widget.LowImportance
		title := widget.NewLabel(task.Label)
		title.TextStyle = fyne.TextStyle{Bold: true}
		meta := widget.NewLabel(fmt.Sprintf("%s - cwd: %s - source: %s", task.Kind, task.Cwd, task.Source))
		meta.Truncation = fyne.TextTruncateEllipsis
		command := widget.NewLabel(task.Command)
		command.Truncation = fyne.TextTruncateEllipsis
		rows = append(rows, container.NewBorder(nil, nil, run, nil, container.NewVBox(title, meta, command)))
	}
	return rows
}

func formatTaskRun(result tasksSvc.RunResult) string {
	var builder strings.Builder
	builder.WriteString(result.Message)
	builder.WriteString("\n")
	builder.WriteString("Status: ")
	builder.WriteString(result.Status)
	builder.WriteString("\nExit code: ")
	builder.WriteString(fmt.Sprintf("%d", result.ExitCode))
	builder.WriteString("\nCommand: ")
	builder.WriteString(result.Task.Command)
	builder.WriteString("\nCwd: ")
	builder.WriteString(result.Task.Cwd)
	builder.WriteString("\nDuration: ")
	builder.WriteString(result.Duration.String())
	builder.WriteString("\n\nStdout\n")
	builder.WriteString(result.Stdout)
	if !strings.HasSuffix(result.Stdout, "\n") {
		builder.WriteString("\n")
	}
	builder.WriteString("\nStderr\n")
	builder.WriteString(result.Stderr)
	if !strings.HasSuffix(result.Stderr, "\n") {
		builder.WriteString("\n")
	}
	return builder.String()
}

func taskRunLogLine(result tasksSvc.RunResult) string {
	return fmt.Sprintf("%s exit=%d duration=%s", result.Status, result.ExitCode, result.Duration)
}

func jobStatusFromTask(result tasksSvc.RunResult) jobsSvc.Status {
	switch result.Status {
	case "success":
		return jobsSvc.StatusSuccess
	case "timeout":
		return jobsSvc.StatusTimedOut
	case "canceled":
		return jobsSvc.StatusCanceled
	default:
		return jobsSvc.StatusFailed
	}
}
