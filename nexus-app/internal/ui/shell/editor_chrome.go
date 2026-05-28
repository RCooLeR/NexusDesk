package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	editorSvc "nexusdesk/internal/services/editor"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

func (v *View) newEditorPanel(tab editorSvc.Tab, preview domain.FilePreview) fyne.CanvasObject {
	v.editorPreviews[tab.ID] = preview
	content := newFilePreview(preview)
	path := widget.NewLabel(preview.RelPath)
	path.TextStyle = fyne.TextStyle{Monospace: true}
	state := widget.NewLabel(editorStateText(tab))
	save := widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		v.saveEditorDraft(tab.ID)
	})
	save.Importance = widget.MediumImportance
	setSaveEnabled(save, tab.Dirty)
	pin := widget.NewButtonWithIcon("", theme.ConfirmIcon(), func() {
		if next, ok := v.editorSession.TogglePinned(tab.ID); ok {
			state.SetText(editorStateText(next))
			v.updateEditorTabState(next)
			v.syncEditorTabOrder()
		}
	})
	pin.Importance = widget.LowImportance
	if preview.Kind == domain.PreviewText {
		content = v.newTextEditor(tab, preview, func(next editorSvc.Tab) {
			state.SetText(editorStateText(next))
			setSaveEnabled(save, next.Dirty)
			v.updateEditorTabState(next)
		})
	} else {
		v.removeTextEditor(tab.ID)
	}
	tools := container.NewHBox(pin, save, state)
	return container.NewBorder(container.NewBorder(nil, nil, path, tools), nil, nil, nil, content)
}

func (v *View) saveEditorDraft(tabID string) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before saving.")
		return
	}
	tab, ok := v.editorSession.Tab(tabID)
	if !ok {
		v.addActivity("Editor tab is no longer available.")
		return
	}
	if !tab.Dirty {
		v.addActivity("No draft changes to save.")
		return
	}
	preview, ok := v.editorPreviews[tab.ID]
	if !ok {
		v.addActivity("Could not resolve editor preview for saving " + tab.Title + ".")
		return
	}
	proposal, err := v.workspaceService.ApplyFileWrite(workspace.Root, workspaceSvc.FileWriteRequest{
		RelPath:  tab.RelPath,
		Content:  tab.DraftText,
		Encoding: preview.Encoding,
	})
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	next, ok := v.editorSession.MarkDraftSaved(tab.ID)
	if !ok {
		v.addActivity("Saved file, but editor state could not be refreshed.")
		return
	}
	preview.Text = next.SourceText
	preview.Size = int64(proposal.Size)
	preview.Encoding = proposal.Encoding
	v.editorPreviews[next.ID] = preview
	if item := v.openTabs[next.ID]; item != nil {
		item.Content = v.newEditorPanel(next, preview)
	}
	v.updateEditorTabState(next)
	v.addActivity(proposal.Message)
}

func (v *View) saveActiveEditorDraft() {
	tabID, _, ok := v.activeTextEditor()
	if !ok {
		v.addActivity("Select a text editor tab before saving.")
		return
	}
	v.saveEditorDraft(tabID)
}

func (v *View) revertEditorDraft(tabID string) {
	next, ok := v.editorSession.RevertDraft(tabID)
	if !ok {
		return
	}
	editor, ok := v.textEditor(tabID)
	if !ok {
		return
	}
	editor.source.SetText(next.DraftText)
	editor.applyTabState(next)
	v.updateEditorTabState(next)
}

func (v *View) revertActiveEditorDraft() {
	tabID, _, ok := v.activeTextEditor()
	if !ok {
		v.addActivity("Select a text editor tab before reverting.")
		return
	}
	v.revertEditorDraft(tabID)
}

func setSaveEnabled(button *widget.Button, enabled bool) {
	if enabled {
		button.Enable()
		return
	}
	button.Disable()
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
