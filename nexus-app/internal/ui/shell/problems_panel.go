package shell

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	workspaceSvc "nexusdesk/internal/services/workspace"
)

func (v *View) newProblemsPanel() fyne.CanvasObject {
	refresh := widget.NewButtonWithIcon("Scan", theme.ViewRefreshIcon(), v.scanProblems)
	header := container.NewBorder(nil, nil, v.problemStatus, refresh)
	scroll := container.NewScroll(v.problemResults)
	scroll.SetMinSize(fyne.NewSize(240, 110))
	return container.NewBorder(header, nil, nil, nil, scroll)
}

func (v *View) scanProblems() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.problemStatus.SetText("Open a workspace before scanning.")
		v.addActivity("Open a workspace before scanning problems.")
		return
	}
	summary, err := v.workspaceService.ScanProblems(workspace.Root, 80)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.problemStatus.SetText(summary.Message)
	v.problemResults.Objects = problemRows(summary.Problems, v.openProblem)
	v.problemResults.Refresh()
	v.addActivity(summary.Message)
}

func (v *View) openProblem(problem workspaceSvc.WorkspaceProblem) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		return
	}
	preview, err := v.workspaceService.PreviewFile(workspace.Root, problem.RelPath)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.openPreviewTab(preview)
	v.addActivity(fmt.Sprintf("Opened problem %s:%d.", problem.RelPath, problem.Line))
}

func problemRows(problems []workspaceSvc.WorkspaceProblem, onOpen func(workspaceSvc.WorkspaceProblem)) []fyne.CanvasObject {
	if len(problems) == 0 {
		return []fyne.CanvasObject{widget.NewLabel("No lightweight problems found.")}
	}
	rows := make([]fyne.CanvasObject, 0, len(problems))
	for _, problem := range problems {
		problem := problem
		open := widget.NewButtonWithIcon("", theme.FileTextIcon(), func() {
			onOpen(problem)
		})
		open.Importance = widget.LowImportance
		title := fmt.Sprintf("%s:%d", problem.RelPath, problem.Line)
		meta := widget.NewLabel(fmt.Sprintf("%s - %s", problem.Severity, problem.Source))
		meta.Truncation = fyne.TextTruncateEllipsis
		message := widget.NewLabel(problem.Message)
		message.Truncation = fyne.TextTruncateEllipsis
		rows = append(rows, container.NewBorder(nil, nil, open, nil, container.NewVBox(widget.NewLabel(title), meta, message)))
	}
	return rows
}
