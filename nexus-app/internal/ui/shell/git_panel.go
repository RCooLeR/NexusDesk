package shell

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	gitSvc "nexusdesk/internal/services/git"
)

func (v *View) newGitPanel() fyne.CanvasObject {
	refresh := widget.NewButtonWithIcon("Refresh git", theme.ViewRefreshIcon(), v.refreshGitStatus)
	header := container.NewBorder(nil, nil, v.gitStatus, refresh)
	scroll := container.NewScroll(v.gitResults)
	scroll.SetMinSize(fyne.NewSize(240, 110))
	return container.NewBorder(header, nil, nil, nil, scroll)
}

func (v *View) refreshGitStatus() {
	workspace := v.state.Workspace()
	status, err := v.gitService.Status(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.gitStatus.SetText(gitStatusLabel(status))
	v.gitResults.Objects = gitRows(status)
	v.gitResults.Refresh()
	v.addActivity(status.Message)
}

func gitStatusLabel(status gitSvc.Status) string {
	if !status.Available {
		return status.Message
	}
	head := status.Head
	if head == "" {
		head = "no HEAD"
	}
	return fmt.Sprintf("%s @ %s - %d changed", status.Branch, head, len(status.ChangedFiles))
}

func gitRows(status gitSvc.Status) []fyne.CanvasObject {
	if !status.Available {
		return []fyne.CanvasObject{widget.NewLabel(status.Message)}
	}
	rows := []fyne.CanvasObject{
		widget.NewLabel(status.Message),
		widget.NewLabel(fmt.Sprintf("%d staged / %d unstaged", len(status.StagedFiles), len(status.UnstagedFiles))),
	}
	if status.AheadBehind != "" {
		rows = append(rows, widget.NewLabel(status.AheadBehind))
	}
	if len(status.ChangedFiles) == 0 {
		rows = append(rows, widget.NewLabel("Working tree is clean."))
		return rows
	}
	for _, change := range status.ChangedFiles {
		label := change.Path
		if change.OldPath != "" {
			label = change.OldPath + " -> " + change.Path
		}
		rows = append(rows, container.NewBorder(nil, nil, widget.NewIcon(theme.FileTextIcon()), nil, widget.NewLabel(fmt.Sprintf("%s - %s", change.Summary, label))))
	}
	return rows
}
