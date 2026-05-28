package shell

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	editorSvc "nexusdesk/internal/services/editor"
)

type textEditorBinding struct {
	source        *widget.Entry
	status        *widget.Label
	rendered      *previewPane
	outlineStatus *widget.Label
	outlineList   *fyne.Container
	relPath       string
	onState       func(editorSvc.Tab)
}

func (b *textEditorBinding) applyTabState(tab editorSvc.Tab) {
	if b == nil {
		return
	}
	b.status.SetText(draftStatusText(tab))
	b.rendered.SetText(tab.DraftText)
	b.setOutline(tab.DraftText)
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
	outlineStatus := widget.NewLabel("")
	outlineStatus.Wrapping = fyne.TextWrapWord
	outlineList := container.NewVBox()
	binding := &textEditorBinding{
		source:        source,
		status:        status,
		rendered:      rendered,
		outlineStatus: outlineStatus,
		outlineList:   outlineList,
		relPath:       tab.RelPath,
		onState:       onState,
	}
	binding.setOutline(tab.DraftText)
	source.OnChanged = func(text string) {
		if !v.editorSession.UpdateDraft(tab.ID, text) {
			return
		}
		if next, ok := v.editorSession.Tab(tab.ID); ok {
			status.SetText(draftStatusText(next))
			rendered.SetText(next.DraftText)
			binding.setOutline(next.DraftText)
			onState(next)
		}
	}
	revert := widget.NewButtonWithIcon("Revert draft", theme.ContentUndoIcon(), func() {
		v.revertEditorDraft(tab.ID)
	})
	revert.Importance = widget.LowImportance

	v.bindTextEditor(tab.ID, binding)

	sourcePanel := container.NewBorder(container.NewBorder(nil, nil, status, revert), nil, nil, nil, source)
	previewPanel := container.NewBorder(widget.NewLabel(previewHeader(preview)), nil, nil, nil, rendered.Canvas())
	outlinePanel := container.NewBorder(outlineStatus, nil, nil, nil, container.NewVScroll(outlineList))
	tabs := container.NewAppTabs(
		container.NewTabItem("Source", sourcePanel),
		container.NewTabItem("Preview", previewPanel),
		container.NewTabItem("Outline", outlinePanel),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

func (b *textEditorBinding) setOutline(text string) {
	if b == nil || b.outlineList == nil || b.outlineStatus == nil {
		return
	}
	items := editorSvc.BuildOutline(b.relPath, text)
	b.outlineList.Objects = b.outlineList.Objects[:0]
	if len(items) == 0 {
		b.outlineStatus.SetText("Outline: no symbols detected for this file.")
		b.outlineList.Add(widget.NewLabel("No outline symbols detected."))
		b.outlineList.Refresh()
		return
	}
	b.outlineStatus.SetText(fmt.Sprintf("Outline: %d symbol(s). Select one to move the editor cursor.", len(items)))
	for _, item := range items {
		current := item
		button := widget.NewButton(outlineItemText(current), func() {
			editorSetCursorLine(b.source, current.Line)
			b.outlineStatus.SetText(fmt.Sprintf("Moved cursor to %s on line %d.", current.Label, current.Line))
		})
		button.Alignment = widget.ButtonAlignLeading
		button.Importance = widget.LowImportance
		b.outlineList.Add(button)
	}
	b.outlineList.Refresh()
}

func outlineItemText(item editorSvc.OutlineItem) string {
	indent := strings.Repeat("  ", item.Level)
	return fmt.Sprintf("%s%s  %s  L%d", indent, item.Kind, item.Label, item.Line)
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
