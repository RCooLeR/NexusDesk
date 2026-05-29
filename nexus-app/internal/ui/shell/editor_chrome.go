package shell

import (
	"path"
	"path/filepath"

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
		content = v.newTextEditor(tab, preview, func(next editorSvc.Tab, encodingDirty bool) {
			state.SetText(editorStateText(next))
			setSaveEnabled(save, next.Dirty || encodingDirty)
			v.updateEditorTabState(next)
		})
	} else {
		v.removeTextEditor(tab.ID)
	}
	split := widget.NewButtonWithIcon("", theme.ViewFullScreenIcon(), func() {
		v.editorSplitEnabled = !v.editorSplitEnabled
		v.refreshEditorTabPanel(tab.ID)
	})
	split.Importance = widget.LowImportance
	if v.editorSplitEnabled {
		content = v.newSplitEditorContent(tab, content)
	}
	tools := container.NewHBox(pin, split, save, state)
	return container.NewBorder(container.NewBorder(nil, nil, v.newEditorBreadcrumbs(tab.RelPath), tools), nil, nil, nil, content)
}

func (v *View) newSplitEditorContent(active editorSvc.Tab, primary fyne.CanvasObject) fyne.CanvasObject {
	secondaryTab, ok := v.editorSession.ResolveSecondaryFileTab(active.RelPath, v.editorSecondaryRelPath)
	if !ok {
		empty := widget.NewLabel("Open another file tab to use split editor.")
		empty.Wrapping = fyne.TextWrapWord
		split := container.NewHSplit(primary, container.NewPadded(empty))
		split.SetOffset(0.66)
		return split
	}
	v.editorSecondaryRelPath = secondaryTab.RelPath
	secondary := v.newSecondaryEditorPanel(active.RelPath, secondaryTab)
	split := container.NewHSplit(primary, secondary)
	split.SetOffset(0.66)
	return split
}

func (v *View) newSecondaryEditorPanel(activeRelPath string, secondaryTab editorSvc.Tab) fyne.CanvasObject {
	options := v.secondaryEditorOptions(activeRelPath)
	selectSecondary := widget.NewSelect(options, nil)
	selectSecondary.PlaceHolder = "Secondary tab"
	selectSecondary.SetSelected(secondaryTab.RelPath)
	selectSecondary.OnChanged = func(relPath string) {
		v.editorSecondaryRelPath = relPath
		v.refreshActiveEditorTabPanel()
	}
	preview, ok := v.editorPreviews[secondaryTab.ID]
	if !ok {
		preview = domain.FilePreview{
			RelPath: secondaryTab.RelPath,
			Name:    filepath.Base(secondaryTab.RelPath),
			Kind:    domain.PreviewText,
			Text:    secondaryTab.DraftText,
		}
	}
	if secondaryTab.Kind == editorSvc.KindFile {
		preview.Text = secondaryTab.DraftText
	}
	title := widget.NewLabel("Secondary")
	title.TextStyle = fyne.TextStyle{Bold: true}
	header := container.NewBorder(nil, nil, title, nil, selectSecondary)
	return container.NewBorder(header, nil, nil, nil, newFilePreview(preview))
}

func (v *View) secondaryEditorOptions(activeRelPath string) []string {
	options := []string{}
	for _, tab := range v.editorSession.Tabs() {
		if tab.Kind != editorSvc.KindFile || tab.RelPath == activeRelPath {
			continue
		}
		options = append(options, tab.RelPath)
	}
	return options
}

func (v *View) refreshActiveEditorTabPanel() {
	if item := v.editorTabs.Selected(); item != nil {
		if id := v.tabIDs[item]; id != "" {
			v.refreshEditorTabPanel(id)
		}
	}
}

func (v *View) refreshEditorTabPanel(tabID string) {
	tab, ok := v.editorSession.Tab(tabID)
	if !ok || tab.Kind != editorSvc.KindFile {
		return
	}
	preview, ok := v.editorPreviews[tab.ID]
	if !ok {
		return
	}
	if item := v.openTabs[tab.ID]; item != nil {
		item.Content = v.newEditorPanel(tab, preview)
		item.Content.Refresh()
		v.editorTabs.Refresh()
	}
}

func (v *View) newEditorBreadcrumbs(relPath string) fyne.CanvasObject {
	workspace := v.state.Workspace()
	crumbs := editorSvc.BuildBreadcrumbs(relPath, workspace.Name)
	row := container.NewHBox()
	for index, crumb := range crumbs {
		current := crumb
		if index > 0 {
			row.Add(widget.NewLabel(">"))
		}
		button := widget.NewButton(current.Label, func() {
			v.openEditorBreadcrumb(current.RelPath)
		})
		button.Importance = widget.LowImportance
		if current.RelPath == "" {
			button.Disable()
		}
		row.Add(button)
	}
	return container.NewHScroll(row)
}

func (v *View) openEditorBreadcrumb(relPath string) {
	if relPath == "" {
		return
	}
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before using breadcrumbs.")
		return
	}
	v.state.SetSelectedPath(relPath)
	v.refreshStatusBar()
	v.refreshNavigatorTargets(relPath)
	if selectedWorkspaceNodeKind(workspace, relPath) == domain.NodeDirectory || path.Ext(relPath) == "" {
		v.addActivity("Selected folder " + relPath + " from editor breadcrumbs.")
		return
	}
	v.openWorkspaceRelFile(relPath)
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
	encodingDirty := false
	if editor, ok := v.textEditor(tab.ID); ok {
		encodingDirty = editor.encodingDirty()
	}
	if !tab.Dirty && !encodingDirty {
		v.addActivity("No draft changes to save.")
		return
	}
	preview, ok := v.editorPreviews[tab.ID]
	if !ok {
		v.addActivity("Could not resolve editor preview for saving " + tab.Title + ".")
		return
	}
	encoding := preview.Encoding
	if editor, ok := v.textEditor(tab.ID); ok {
		encoding = editor.writeEncoding()
	}
	proposal, err := v.workspaceService.ApplyFileWrite(workspace.Root, workspaceSvc.FileWriteRequest{
		RelPath:  tab.RelPath,
		Content:  tab.DraftText,
		Encoding: encoding,
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
	if editor, ok := v.textEditor(next.ID); ok {
		editor.markEncodingSaved(proposal.Encoding)
	}
	if item := v.openTabs[next.ID]; item != nil {
		item.Content = v.newEditorPanel(next, preview)
	}
	v.updateEditorTabState(next)
	v.refreshStatusBar()
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
