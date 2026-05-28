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
	source         *widget.Entry
	status         *widget.Label
	rendered       *previewPane
	outlineStatus  *widget.Label
	outlineList    *fyne.Container
	relPath        string
	sourceEncoding string
	saveEncoding   string
	onEncoding     func()
	onState        func(editorSvc.Tab, bool)
}

func (b *textEditorBinding) applyTabState(tab editorSvc.Tab) {
	if b == nil {
		return
	}
	b.status.SetText(draftStatusText(tab))
	b.rendered.SetText(tab.DraftText)
	b.setOutline(tab.DraftText)
	if b.onState != nil {
		b.onState(tab, b.encodingDirty())
	}
}

func (v *View) newTextEditor(tab editorSvc.Tab, preview domain.FilePreview, onState func(editorSvc.Tab, bool)) fyne.CanvasObject {
	source := widget.NewMultiLineEntry()
	source.SetText(tab.DraftText)
	source.Wrapping = fyne.TextWrapOff
	source.TextStyle = fyne.TextStyle{Monospace: true}
	status := widget.NewLabel(draftStatusText(tab))
	rendered := newPreviewPane(preview, tab.DraftText)
	initialEncoding := editorWriteEncoding(preview.Encoding)
	outlineStatus := widget.NewLabel("")
	outlineStatus.Wrapping = fyne.TextWrapWord
	outlineList := container.NewVBox()
	binding := &textEditorBinding{
		source:         source,
		status:         status,
		rendered:       rendered,
		outlineStatus:  outlineStatus,
		outlineList:    outlineList,
		relPath:        tab.RelPath,
		sourceEncoding: initialEncoding,
		saveEncoding:   initialEncoding,
		onState:        onState,
	}
	encodingSelect := widget.NewSelect(editorEncodingOptions(), func(value string) {
		binding.saveEncoding = editorWriteEncoding(value)
		status.SetText(draftStatusTextWithEncoding(tab, binding.encodingDirty()))
		if binding.onEncoding != nil {
			binding.onEncoding()
		}
	})
	encodingSelect.SetSelected(initialEncoding)
	binding.onEncoding = func() {
		if next, ok := v.editorSession.Tab(tab.ID); ok {
			onState(next, binding.encodingDirty())
		}
	}
	binding.setOutline(tab.DraftText)
	source.OnChanged = func(text string) {
		if !v.editorSession.UpdateDraft(tab.ID, text) {
			return
		}
		if next, ok := v.editorSession.Tab(tab.ID); ok {
			status.SetText(draftStatusTextWithEncoding(next, binding.encodingDirty()))
			rendered.SetText(next.DraftText)
			binding.setOutline(next.DraftText)
			onState(next, binding.encodingDirty())
		}
	}
	revert := widget.NewButtonWithIcon("Revert draft", theme.ContentUndoIcon(), func() {
		v.revertEditorDraft(tab.ID)
	})
	revert.Importance = widget.LowImportance
	format := widget.NewButtonWithIcon("Format", theme.DocumentCreateIcon(), func() {
		result, err := editorSvc.FormatDocument(tab.RelPath, source.Text)
		if err != nil {
			status.SetText(err.Error())
			return
		}
		if result.Changed {
			source.SetText(result.Content)
		}
		status.SetText(result.Message)
	})
	format.Importance = widget.LowImportance

	v.bindTextEditor(tab.ID, binding)

	encodingControl := container.NewHBox(widget.NewLabel("Save as"), encodingSelect, format, revert)
	sourcePanel := container.NewBorder(container.NewBorder(nil, nil, status, encodingControl), nil, nil, nil, source)
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

func (b *textEditorBinding) writeEncoding() string {
	if b == nil {
		return "utf-8"
	}
	return editorWriteEncoding(b.saveEncoding)
}

func (b *textEditorBinding) encodingDirty() bool {
	if b == nil {
		return false
	}
	return editorWriteEncoding(b.sourceEncoding) != editorWriteEncoding(b.saveEncoding)
}

func (b *textEditorBinding) markEncodingSaved(encoding string) {
	if b == nil {
		return
	}
	next := editorWriteEncoding(encoding)
	b.sourceEncoding = next
	b.saveEncoding = next
}

func editorEncodingOptions() []string {
	return []string{"utf-8", "utf-8-bom", "utf-16le", "utf-16be", "windows-1251", "windows-1252"}
}

func editorWriteEncoding(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "utf8", "utf-8":
		return "utf-8"
	case "utf8-bom", "utf-8-bom", "utf-8 bom":
		return "utf-8-bom"
	case "utf16le", "utf-16le", "utf-16 le":
		return "utf-16le"
	case "utf16be", "utf-16be", "utf-16 be":
		return "utf-16be"
	case "cp1251", "windows1251", "windows-1251":
		return "windows-1251"
	case "cp1252", "windows1252", "windows-1252":
		return "windows-1252"
	default:
		return value
	}
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
	return draftStatusTextWithEncoding(tab, false)
}

func draftStatusTextWithEncoding(tab editorSvc.Tab, encodingDirty bool) string {
	if encodingDirty && tab.Dirty {
		return "Draft modified and save encoding changed. Save applies through the safe write service and creates a rollback snapshot."
	}
	if encodingDirty {
		return "Save encoding changed. Save applies through the safe write service and creates a rollback snapshot."
	}
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
