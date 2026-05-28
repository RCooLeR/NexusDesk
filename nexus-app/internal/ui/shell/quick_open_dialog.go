package shell

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	workspaceSvc "nexusdesk/internal/services/workspace"
)

const quickOpenResultLimit = 40

func (v *View) openQuickOpenDialog() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before using quick open.")
		return
	}

	entry := widget.NewEntry()
	entry.SetPlaceHolder("Type a file name or path...")
	status := widget.NewLabel("")
	results := []workspaceSvc.QuickOpenFile{}
	var picker dialog.Dialog

	openCandidate := func(index int) {
		if index < 0 || index >= len(results) {
			return
		}
		if picker != nil {
			picker.Hide()
		}
		v.openWorkspaceRelFile(results[index].RelPath)
	}
	list := widget.NewList(
		func() int { return len(results) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Truncation = fyne.TextTruncateEllipsis
			return label
		},
		func(id widget.ListItemID, object fyne.CanvasObject) {
			object.(*widget.Label).SetText(results[id].RelPath)
		},
	)
	list.OnSelected = openCandidate

	refresh := func(query string) {
		files, err := v.workspaceService.QuickOpenFiles(workspace.Root, query, quickOpenResultLimit)
		if err != nil {
			status.SetText("Quick open unavailable: " + err.Error())
			results = nil
			list.Refresh()
			return
		}
		results = files
		status.SetText(quickOpenStatusText(len(results), query))
		list.Refresh()
	}
	entry.OnChanged = refresh
	entry.OnSubmitted = func(string) {
		openCandidate(0)
	}

	content := container.NewBorder(entry, status, nil, nil, list)
	content.Resize(fyne.NewSize(560, 360))
	picker = dialog.NewCustom("Quick Open", "Close", content, v.window)
	refresh("")
	picker.Show()
	v.window.Canvas().Focus(entry)
}

func (v *View) openWorkspaceRelFile(relPath string) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before opening files.")
		return
	}
	preview, err := v.workspaceService.PreviewFile(workspace.Root, relPath)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.openPreviewTab(preview)
	v.addActivity("Opened " + relPath)
	v.refreshAssistantContextPins()
}

func quickOpenStatusText(count int, query string) string {
	if count == 0 {
		return fmt.Sprintf("No matches for %q.", query)
	}
	if query == "" {
		return fmt.Sprintf("%d file(s) available. Type to filter, Enter to open the first match.", count)
	}
	return fmt.Sprintf("%d match(es). Enter opens the first match.", count)
}
