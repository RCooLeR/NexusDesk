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

type rollbackController struct {
	view    *View
	results *fyne.Container
	status  *widget.Label
}

func newRollbackController(view *View) *rollbackController {
	return &rollbackController{
		view:    view,
		results: container.NewVBox(widget.NewLabel("Refresh rollback records to inspect undo points.")),
		status:  widget.NewLabel("Rollback records have not been loaded."),
	}
}

func (v *View) newRollbackPanel() fyne.CanvasObject {
	return v.rollbacks.Panel()
}

func (v *View) refreshRollbacks() {
	v.rollbacks.Refresh()
}

func (v *View) confirmRollback(record workspaceSvc.RollbackRecord) {
	v.rollbacks.Confirm(record)
}

func (c *rollbackController) Panel() fyne.CanvasObject {
	refresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), c.Refresh)
	header := container.NewBorder(nil, nil, c.status, refresh)
	scroll := container.NewScroll(c.results)
	scroll.SetMinSize(fyne.NewSize(240, 110))
	return container.NewBorder(header, nil, nil, nil, scroll)
}

func (c *rollbackController) Refresh() {
	workspace := c.view.state.Workspace()
	if workspace.Root == "" {
		c.status.SetText("Open a workspace before reading rollback records.")
		c.view.addActivity("Open a workspace before reading rollback records.")
		return
	}
	records, err := c.view.workspaceService.ListRollbacks(workspace.Root)
	if err != nil {
		dialog.ShowError(err, c.view.window)
		return
	}
	c.status.SetText(fmt.Sprintf("%d rollback record(s)", len(records)))
	c.results.Objects = rollbackRows(records, c.Confirm)
	c.results.Refresh()
	c.view.addActivity(fmt.Sprintf("Loaded %d rollback record(s).", len(records)))
}

func (c *rollbackController) Confirm(record workspaceSvc.RollbackRecord) {
	workspace := c.view.state.Workspace()
	if workspace.Root == "" {
		return
	}
	if record.Status == "applied" {
		c.view.addActivity("Rollback already applied: " + record.ID)
		return
	}
	dialog.ShowConfirm("Apply rollback", rollbackConfirmText(record), func(confirm bool) {
		if !confirm {
			return
		}
		result, err := c.view.workspaceService.ApplyRollback(workspace.Root, record.ID)
		if err != nil {
			dialog.ShowError(err, c.view.window)
			return
		}
		c.view.addActivity(result.Message)
		c.Refresh()
		c.view.refreshWorkspace()
	}, c.view.window)
}

func rollbackRows(records []workspaceSvc.RollbackRecord, onApply func(workspaceSvc.RollbackRecord)) []fyne.CanvasObject {
	if len(records) == 0 {
		return []fyne.CanvasObject{widget.NewLabel("No rollback records.")}
	}
	rows := make([]fyne.CanvasObject, 0, len(records))
	for _, record := range records {
		record := record
		apply := widget.NewButtonWithIcon("", theme.ContentUndoIcon(), func() {
			onApply(record)
		})
		apply.Importance = widget.LowImportance
		if record.Status == "applied" {
			apply.Disable()
		}
		rows = append(rows, container.NewBorder(nil, nil, apply, nil, rollbackRecordBody(record)))
	}
	return rows
}

func rollbackRecordBody(record workspaceSvc.RollbackRecord) fyne.CanvasObject {
	title := widget.NewLabel(fmt.Sprintf("%s - %s", record.Action, record.Target))
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Truncation = fyne.TextTruncateEllipsis
	meta := widget.NewLabel(fmt.Sprintf("%s - %d path(s) - %s", record.Status, len(record.Entries), record.CreatedAt.Format("2006-01-02 15:04:05")))
	meta.Truncation = fyne.TextTruncateEllipsis
	message := widget.NewLabel(record.Message)
	message.Truncation = fyne.TextTruncateEllipsis
	return container.NewVBox(title, meta, message)
}

func rollbackConfirmText(record workspaceSvc.RollbackRecord) string {
	return fmt.Sprintf("Restore/remove %d path(s) for %s?\n\n%s", len(record.Entries), record.Target, record.Message)
}
