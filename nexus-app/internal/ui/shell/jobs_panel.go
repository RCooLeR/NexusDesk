package shell

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	jobsSvc "nexusdesk/internal/services/jobs"
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
	v.jobResults.Objects = jobRows(jobs, v.cancelJob)
	v.jobResults.Refresh()
}

func (v *View) cancelJob(id string) {
	if v.jobService.Cancel(id) {
		v.addActivity("Cancel requested for " + id + ".")
	}
	v.refreshJobs()
}

func jobRows(jobs []jobsSvc.Job, onCancel func(string)) []fyne.CanvasObject {
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
		title := widget.NewLabel(fmt.Sprintf("%s - %s", job.ID, job.Label))
		title.TextStyle = fyne.TextStyle{Bold: true}
		meta := widget.NewLabel(fmt.Sprintf("%s - %s", job.Kind, job.Status))
		meta.Truncation = fyne.TextTruncateEllipsis
		message := widget.NewLabel(jobSummary(job))
		message.Truncation = fyne.TextTruncateEllipsis
		rows = append(rows, container.NewBorder(nil, nil, cancel, nil, container.NewVBox(title, meta, message)))
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
