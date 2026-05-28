package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	editorSvc "nexusdesk/internal/services/editor"
)

type textEditorBinding struct {
	source   *widget.Entry
	status   *widget.Label
	rendered *previewPane
	onState  func(editorSvc.Tab)
}

func (b *textEditorBinding) applyTabState(tab editorSvc.Tab) {
	if b == nil {
		return
	}
	b.status.SetText(draftStatusText(tab))
	b.rendered.SetText(tab.DraftText)
	if b.onState != nil {
		b.onState(tab)
	}
}

func (v *View) newTextEditor(tab editorSvc.Tab, preview domain.FilePreview, onState func(editorSvc.Tab)) fyne.CanvasObject {
	source := widget.NewMultiLineEntry()
	source.SetText(tab.DraftText)
	source.Wrapping = fyne.TextWrapOff
	source.TextStyle = fyne.TextStyle{Monospace: true}
	status := widget.NewLabel(draftStatusText(tab))
	rendered := newPreviewPane(preview, tab.DraftText)
	source.OnChanged = func(text string) {
		if !v.editorSession.UpdateDraft(tab.ID, text) {
			return
		}
		if next, ok := v.editorSession.Tab(tab.ID); ok {
			status.SetText(draftStatusText(next))
			rendered.SetText(next.DraftText)
			onState(next)
		}
	}
	revert := widget.NewButtonWithIcon("Revert draft", theme.ContentUndoIcon(), func() {
		v.revertEditorDraft(tab.ID)
	})
	revert.Importance = widget.LowImportance

	v.bindTextEditor(tab.ID, &textEditorBinding{
		source:   source,
		status:   status,
		rendered: rendered,
		onState:  onState,
	})

	sourcePanel := container.NewBorder(container.NewBorder(nil, nil, status, revert), nil, nil, nil, source)
	previewPanel := container.NewBorder(widget.NewLabel(previewHeader(preview)), nil, nil, nil, rendered.Canvas())
	tabs := container.NewAppTabs(
		container.NewTabItem("Source", sourcePanel),
		container.NewTabItem("Preview", previewPanel),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

func draftStatusText(tab editorSvc.Tab) string {
	if tab.Dirty {
		return "Draft modified. Save applies through the safe write service and creates a rollback snapshot."
	}
	return "Draft matches source."
}

func (v *View) bindTextEditor(tabID string, binding *textEditorBinding) {
	if v.textEditors == nil {
		v.textEditors = map[string]*textEditorBinding{}
	}
	v.textEditors[tabID] = binding
}

func (v *View) removeTextEditor(tabID string) {
	if len(v.textEditors) == 0 {
		return
	}
	delete(v.textEditors, tabID)
}

func (v *View) textEditor(tabID string) (*textEditorBinding, bool) {
	binding := v.textEditors[tabID]
	return binding, binding != nil
}
