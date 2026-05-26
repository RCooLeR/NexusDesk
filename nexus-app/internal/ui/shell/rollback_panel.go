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

func (v *View) newRollbackPanel() fyne.CanvasObject {
	refresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), v.refreshRollbacks)
	header := container.NewBorder(nil, nil, v.rollbackStatus, refresh)
	scroll := container.NewScroll(v.rollbackResults)
	scroll.SetMinSize(fyne.NewSize(240, 110))
	return container.NewBorder(header, nil, nil, nil, scroll)
}

func (v *View) refreshRollbacks() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.rollbackStatus.SetText("Open a workspace before reading rollback records.")
		v.addActivity("Open a workspace before reading rollback records.")
		return
	}
	records, err := v.workspaceService.ListRollbacks(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.rollbackStatus.SetText(fmt.Sprintf("%d rollback record(s)", len(records)))
	v.rollbackResults.Objects = rollbackRows(records, v.confirmRollback)
	v.rollbackResults.Refresh()
	v.addActivity(fmt.Sprintf("Loaded %d rollback record(s).", len(records)))
}

func (v *View) confirmRollback(record workspaceSvc.RollbackRecord) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		return
	}
	if record.Status == "applied" {
		v.addActivity("Rollback already applied: " + record.ID)
		return
	}
	dialog.ShowConfirm("Apply rollback", rollbackConfirmText(record), func(confirm bool) {
		if !confirm {
			return
		}
		result, err := v.workspaceService.ApplyRollback(workspace.Root, record.ID)
		if err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		v.addActivity(result.Message)
		v.refreshRollbacks()
		v.refreshWorkspace()
	}, v.window)
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
