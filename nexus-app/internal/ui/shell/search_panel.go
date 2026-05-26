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

func (v *View) newSearchPanel() fyne.CanvasObject {
	scroll := container.NewScroll(v.searchResults)
	scroll.SetMinSize(fyne.NewSize(240, 110))
	return container.NewBorder(v.searchStatus, nil, nil, nil, scroll)
}

func (v *View) searchWorkspace(query string) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.searchStatus.SetText("Open a workspace before searching.")
		v.addActivity("Open a workspace before searching.")
		return
	}
	results, err := v.workspaceService.Search(workspace.Root, query, workspaceSvc.SearchOptions{MaxResults: 80})
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.searchStatus.SetText(fmt.Sprintf("%d result(s) for %q", len(results), query))
	v.searchResults.Objects = searchResultRows(results, v.openSearchResult)
	v.searchResults.Refresh()
	v.addActivity(fmt.Sprintf("Search found %d result(s) for %q.", len(results), query))
}

func (v *View) openSearchResult(result workspaceSvc.SearchResult) {
	workspace := v.state.Workspace()
	if workspace.Root == "" || result.Kind == "directory" {
		return
	}
	preview, err := v.workspaceService.PreviewFile(workspace.Root, result.RelPath)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.openPreviewTab(preview)
	v.addActivity(fmt.Sprintf("Opened search result %s.", result.RelPath))
}

func searchResultRows(results []workspaceSvc.SearchResult, onOpen func(workspaceSvc.SearchResult)) []fyne.CanvasObject {
	if len(results) == 0 {
		return []fyne.CanvasObject{widget.NewLabel("No results.")}
	}
	rows := make([]fyne.CanvasObject, 0, len(results))
	for _, result := range results {
		result := result
		title := result.RelPath
		if result.Line > 0 {
			title = fmt.Sprintf("%s:%d", result.RelPath, result.Line)
		}
		open := widget.NewButtonWithIcon("", theme.FileTextIcon(), func() {
			onOpen(result)
		})
		open.Importance = widget.LowImportance
		meta := widget.NewLabel(fmt.Sprintf("%s - %s", result.MatchType, result.MediaType))
		meta.Truncation = fyne.TextTruncateEllipsis
		snippet := widget.NewLabel(result.Snippet)
		snippet.Truncation = fyne.TextTruncateEllipsis
		rows = append(rows, container.NewBorder(nil, nil, open, nil, container.NewVBox(widget.NewLabel(title), meta, snippet)))
	}
	return rows
}
