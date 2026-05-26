package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	editorSvc "nexusdesk/internal/services/editor"
)

func (v *View) newTextEditor(tab editorSvc.Tab, preview domain.FilePreview, onState func(editorSvc.Tab)) fyne.CanvasObject {
	source := widget.NewMultiLineEntry()
	source.SetText(tab.DraftText)
	source.Wrapping = fyne.TextWrapOff
	source.TextStyle = fyne.TextStyle{Monospace: true}
	status := widget.NewLabel("Draft only. Save is disabled until safe write preview/apply lands.")
	source.OnChanged = func(text string) {
		if !v.editorSession.UpdateDraft(tab.ID, text) {
			return
		}
		if next, ok := v.editorSession.Tab(tab.ID); ok {
			status.SetText(draftStatusText(next))
			onState(next)
		}
	}
	revert := widget.NewButtonWithIcon("Revert draft", theme.ContentUndoIcon(), func() {
		if next, ok := v.editorSession.RevertDraft(tab.ID); ok {
			source.SetText(next.DraftText)
			status.SetText(draftStatusText(next))
			onState(next)
		}
	})
	revert.Importance = widget.LowImportance

	sourcePanel := container.NewBorder(container.NewBorder(nil, nil, status, revert), nil, nil, nil, source)
	previewPanel := container.NewBorder(widget.NewLabel(previewHeader(preview)), nil, nil, nil, readOnlyText(preview.Text))
	tabs := container.NewAppTabs(
		container.NewTabItem("Source", sourcePanel),
		container.NewTabItem("Preview", previewPanel),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

func readOnlyText(text string) fyne.CanvasObject {
	content := widget.NewMultiLineEntry()
	content.SetText(text)
	content.Wrapping = fyne.TextWrapOff
	content.Disable()
	return content
}

func draftStatusText(tab editorSvc.Tab) string {
	if tab.Dirty {
		return "Draft modified. Save is disabled until safe write preview/apply lands."
	}
	return "Draft matches source. Save is disabled until safe write preview/apply lands."
}
