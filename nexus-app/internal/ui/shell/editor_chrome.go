package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	editorSvc "nexusdesk/internal/services/editor"
)

func (v *View) newEditorPanel(tab editorSvc.Tab, preview domain.FilePreview) fyne.CanvasObject {
	content := newFilePreview(preview)
	path := widget.NewLabel(preview.RelPath)
	path.TextStyle = fyne.TextStyle{Monospace: true}
	state := widget.NewLabel(editorStateText(tab))
	pin := widget.NewButtonWithIcon("", theme.ConfirmIcon(), func() {
		if next, ok := v.editorSession.TogglePinned(tab.ID); ok {
			state.SetText(editorStateText(next))
			v.updateEditorTabState(next)
			v.syncEditorTabOrder()
		}
	})
	pin.Importance = widget.LowImportance
	dirty := widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		nextDirty := true
		if current, ok := v.editorSession.Tab(tab.ID); ok {
			nextDirty = !current.Dirty
		}
		if v.editorSession.MarkDirty(tab.ID, nextDirty) {
			if next, ok := v.editorSession.Tab(tab.ID); ok {
				state.SetText(editorStateText(next))
				v.updateEditorTabState(next)
			}
		}
	})
	dirty.Importance = widget.LowImportance
	tools := container.NewHBox(pin, dirty, state)
	return container.NewBorder(container.NewBorder(nil, nil, path, tools), nil, nil, nil, content)
}

func (v *View) updateEditorTabState(tab editorSvc.Tab) {
	item := v.openTabs[tab.ID]
	if item == nil {
		return
	}
	item.Text = editorTabTitle(tab)
	item.Icon = editorTabIcon(tab)
	item.Content.Refresh()
	v.editorTabs.Refresh()
}

func (v *View) syncEditorTabOrder() {
	ordered := make([]*container.TabItem, 0, len(v.openTabs))
	for _, tab := range v.editorSession.Tabs() {
		if item := v.openTabs[tab.ID]; item != nil {
			ordered = append(ordered, item)
		}
	}
	v.editorTabs.Items = ordered
	v.editorTabs.Refresh()
}

func editorTabTitle(tab editorSvc.Tab) string {
	title := tab.Title
	if tab.Pinned {
		title = "[P] " + title
	}
	if tab.Dirty {
		title = "* " + title
	}
	return title
}

func editorTabIcon(tab editorSvc.Tab) fyne.Resource {
	if tab.Dirty {
		return theme.DocumentSaveIcon()
	}
	if tab.Pinned {
		return theme.ConfirmIcon()
	}
	if tab.Kind == editorSvc.KindFile {
		return theme.FileTextIcon()
	}
	return nil
}

func editorStateText(tab editorSvc.Tab) string {
	state := "Saved"
	if tab.Dirty {
		state = "Modified"
	}
	if tab.Pinned {
		state += " - Pinned"
	}
	return state
}
