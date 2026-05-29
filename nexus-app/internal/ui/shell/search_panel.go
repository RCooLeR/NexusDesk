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

type searchController struct {
	view    *View
	results *fyne.Container
	status  *widget.Label
}

func newSearchController(view *View) *searchController {
	return &searchController{
		view:    view,
		results: container.NewVBox(widget.NewLabel("Search results will appear here.")),
		status:  widget.NewLabel("No search yet."),
	}
}

func (v *View) newSearchPanel() fyne.CanvasObject {
	return v.search.Panel()
}

func (v *View) searchWorkspace(query string) {
	v.search.Search(query)
}

func (v *View) openSearchResult(result workspaceSvc.SearchResult) {
	v.search.OpenResult(result)
}

func (c *searchController) Panel() fyne.CanvasObject {
	scroll := container.NewScroll(c.results)
	scroll.SetMinSize(fyne.NewSize(240, 110))
	return container.NewBorder(c.status, nil, nil, nil, scroll)
}

func (c *searchController) Search(query string) {
	workspace := c.view.state.Workspace()
	if workspace.Root == "" {
		c.status.SetText("Open a workspace before searching.")
		c.view.addActivity("Open a workspace before searching.")
		return
	}
	results, metadata, err := c.view.workspaceService.SearchWithMetadata(workspace.Root, query, workspaceSvc.SearchOptions{MaxResults: 80})
	if err != nil {
		dialog.ShowError(err, c.view.window)
		return
	}
	export, exportErr := c.view.workspaceService.WriteSearchMetadata(workspace.Root, metadata)
	status := fmt.Sprintf("%d result(s) for %q. Indexed %d file(s). Metadata: %s.", len(results), query, metadata.FilesScanned, export.RelPath)
	if exportErr != nil {
		status = fmt.Sprintf("%d result(s) for %q. Metadata export failed: %v.", len(results), query, exportErr)
	} else if export.Recovered {
		status = fmt.Sprintf("%s Recovered corrupt metadata to %s.", status, export.RecoveredRelPath)
	}
	c.status.SetText(status)
	c.results.Objects = searchResultRows(results, c.OpenResult)
	c.results.Refresh()
	c.view.addActivity(status)
}

func (c *searchController) OpenResult(result workspaceSvc.SearchResult) {
	workspace := c.view.state.Workspace()
	if workspace.Root == "" || result.Kind == "directory" {
		return
	}
	preview, err := c.view.workspaceService.PreviewFile(workspace.Root, result.RelPath)
	if err != nil {
		dialog.ShowError(err, c.view.window)
		return
	}
	c.view.openPreviewTab(preview)
	c.view.addActivity(fmt.Sprintf("Opened search result %s.", result.RelPath))
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
