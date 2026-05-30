package shell

import (
	"path"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	editorSvc "nexusdesk/internal/services/editor"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

const (
	editorBreadcrumbVisibleLimit = 5
	editorBreadcrumbLabelLimit   = 28
)

func (v *View) newEditorPanel(tab editorSvc.Tab, preview domain.FilePreview) fyne.CanvasObject {
	v.editor.previews[tab.ID] = preview
	content := newFilePreview(preview)
	saving := v.editorSaving(tab.ID)
	state := widget.NewLabel(editorStateTextWithSaving(tab, saving))
	save := widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		v.saveEditorDraft(tab.ID)
	})
	save.Importance = widget.MediumImportance
	setSaveEnabled(save, !saving && editorSaveAllowed(tab, preview, false, !preview.EncodingAmbiguous))
	pin := widget.NewButtonWithIcon("", theme.ConfirmIcon(), func() {
		if next, ok := v.editorSession.TogglePinned(tab.ID); ok {
			state.SetText(editorStateText(next))
			v.updateEditorTabState(next)
			v.syncEditorTabOrder()
		}
	})
	pin.Importance = widget.LowImportance
	if preview.Kind == domain.PreviewText {
		content = v.newTextEditor(tab, preview, func(next editorSvc.Tab, encodingDirty bool, encodingExplicit bool) {
			state.SetText(editorStateTextWithSaving(next, v.editorSaving(next.ID)))
			setSaveEnabled(save, !v.editorSaving(next.ID) && editorSaveAllowed(next, preview, encodingDirty, encodingExplicit))
			v.updateEditorTabState(next)
		})
		if editor, ok := v.textEditor(tab.ID); ok {
			editor.tabState = state
			editor.saveButton = save
			editor.saving = saving
			if saving {
				editor.status.SetText("Saving draft...")
			}
		}
	} else {
		v.removeTextEditor(tab.ID)
	}
	split := widget.NewButtonWithIcon("", theme.ViewFullScreenIcon(), func() {
		v.editor.splitEnabled = !v.editor.splitEnabled
		v.refreshEditorTabPanel(tab.ID)
	})
	split.Importance = widget.LowImportance
	if v.editor.splitEnabled {
		content = v.newSplitEditorContent(tab, content)
	}
	tools := container.NewHBox(pin, split, save, state)
	return container.NewBorder(container.NewBorder(nil, nil, v.newEditorBreadcrumbs(tab.RelPath), tools), nil, nil, nil, content)
}

func (v *View) newSplitEditorContent(active editorSvc.Tab, primary fyne.CanvasObject) fyne.CanvasObject {
	secondaryTab, ok := v.editorSession.ResolveSecondaryFileTab(active.RelPath, v.editor.secondaryRelPath)
	if !ok {
		empty := widget.NewLabel("Open another file tab to use split editor.")
		empty.Wrapping = fyne.TextWrapWord
		split := container.NewHSplit(primary, container.NewPadded(empty))
		split.SetOffset(0.66)
		return split
	}
	v.editor.secondaryRelPath = secondaryTab.RelPath
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
		v.editor.secondaryRelPath = relPath
		v.refreshActiveEditorTabPanel()
	}
	preview, ok := v.editor.previews[secondaryTab.ID]
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
	if item := v.editor.tabs.Selected(); item != nil {
		if id := v.editor.tabIDs[item]; id != "" {
			v.refreshEditorTabPanel(id)
		}
	}
}

func (v *View) refreshEditorTabPanel(tabID string) {
	tab, ok := v.editorSession.Tab(tabID)
	if !ok || tab.Kind != editorSvc.KindFile {
		return
	}
	preview, ok := v.editor.previews[tab.ID]
	if !ok {
		return
	}
	if item := v.editor.openTabs[tab.ID]; item != nil {
		item.Content = v.newEditorPanel(tab, preview)
		item.Content.Refresh()
		v.editor.tabs.Refresh()
	}
}

func (v *View) newEditorBreadcrumbs(relPath string) fyne.CanvasObject {
	workspace := v.state.Workspace()
	crumbs := compactEditorBreadcrumbs(editorSvc.BuildBreadcrumbs(relPath, workspace.Name))
	row := container.NewHBox()
	for index, crumb := range crumbs {
		current := crumb
		if index > 0 {
			row.Add(widget.NewLabel("/"))
		}
		button := widget.NewButton(compactEditorBreadcrumbLabel(current.Label), func() {
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

func compactEditorBreadcrumbs(crumbs []editorSvc.Breadcrumb) []editorSvc.Breadcrumb {
	if len(crumbs) <= editorBreadcrumbVisibleLimit {
		return append([]editorSvc.Breadcrumb{}, crumbs...)
	}
	compact := []editorSvc.Breadcrumb{crumbs[0], {Label: "..."}}
	tailCount := editorBreadcrumbVisibleLimit - len(compact)
	compact = append(compact, crumbs[len(crumbs)-tailCount:]...)
	return compact
}

func compactEditorBreadcrumbLabel(label string) string {
	label = strings.TrimSpace(label)
	if len(label) <= editorBreadcrumbLabelLimit {
		return label
	}
	return label[:editorBreadcrumbLabelLimit-3] + "..."
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

type editorSaveRequest struct {
	tabID         string
	title         string
	relPath       string
	draftText     string
	encoding      string
	workspaceRoot string
	preview       domain.FilePreview
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
	if v.editorSaving(tab.ID) {
		v.addActivity("Save already in progress for " + tab.RelPath + ".")
		return
	}
	encodingDirty := false
	encodingExplicit := true
	if editor, ok := v.textEditor(tab.ID); ok {
		encodingDirty = editor.encodingDirty()
		encodingExplicit = editor.hasExplicitEncoding()
	}
	if !tab.Dirty && !encodingDirty {
		v.addActivity("No draft changes to save.")
		return
	}
	preview, ok := v.editor.previews[tab.ID]
	if !ok {
		v.addActivity("Could not resolve editor preview for saving " + tab.Title + ".")
		return
	}
	if preview.Truncated {
		v.addActivity("Save blocked for " + tab.RelPath + ": preview is truncated, so inline editing is read-only.")
		dialog.ShowInformation("Save blocked", "This file preview is truncated. Inline editing is disabled so NexusDesk does not overwrite the full file with a capped prefix.", v.window)
		return
	}
	if preview.EncodingAmbiguous && !encodingExplicit {
		v.addActivity("Save blocked for " + tab.RelPath + ": choose an explicit encoding before saving.")
		dialog.ShowInformation("Save blocked", "NexusDesk detected this file with low charset confidence. Choose a save encoding before saving so the file is not rewritten with an unintended encoding.", v.window)
		return
	}
	encoding := preview.Encoding
	if editor, ok := v.textEditor(tab.ID); ok {
		encoding = editor.writeEncoding()
	}
	request := editorSaveRequest{
		tabID:         tab.ID,
		title:         tab.Title,
		relPath:       tab.RelPath,
		draftText:     tab.DraftText,
		encoding:      encoding,
		workspaceRoot: workspace.Root,
		preview:       preview,
	}
	v.setEditorSaveState(tab.ID, true, "Saving draft...")
	go v.applyEditorSave(request)
}

func (v *View) applyEditorSave(request editorSaveRequest) {
	proposal, err := v.workspaceService.ApplyFileWrite(request.workspaceRoot, workspaceSvc.FileWriteRequest{
		RelPath:  request.relPath,
		Content:  request.draftText,
		Encoding: request.encoding,
	})
	fyne.Do(func() {
		v.finishEditorSave(request, proposal, err)
	})
}

func (v *View) finishEditorSave(request editorSaveRequest, proposal workspaceSvc.FileWriteProposal, err error) {
	v.setEditorSaveState(request.tabID, false, "")
	if err != nil {
		v.setEditorSaveState(request.tabID, false, "Save failed: "+err.Error()+" Retry Save after fixing the problem.")
		v.addActivity("Save failed for " + request.relPath + ": " + err.Error())
		dialog.ShowError(err, v.window)
		return
	}
	next, ok := v.editorSession.MarkDraftSavedAs(request.tabID, request.draftText)
	if !ok {
		v.addActivity("Saved file, but editor state could not be refreshed.")
		return
	}
	preview := request.preview
	preview.Text = next.SourceText
	preview.Size = int64(proposal.Size)
	preview.Encoding = proposal.Encoding
	v.refreshEditorAfterSave(next, preview)
	v.updateEditorTabState(next)
	v.refreshStatusBar()
	message := proposal.Message
	if next.Dirty {
		message += " Draft has newer unsaved changes."
	}
	v.addActivity(message)
}

func (v *View) refreshEditorAfterSave(next editorSvc.Tab, preview domain.FilePreview) {
	v.editor.previews[next.ID] = preview
	if editor, ok := v.textEditor(next.ID); ok {
		editor.markEncodingSaved(preview.Encoding)
		editor.applyTabState(next)
		return
	}
	if item := v.editor.openTabs[next.ID]; item != nil {
		item.Content = v.newEditorPanel(next, preview)
	}
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

func editorSaveAllowed(tab editorSvc.Tab, preview domain.FilePreview, encodingDirty bool, encodingExplicit bool) bool {
	return !preview.Truncated && (!preview.EncodingAmbiguous || encodingExplicit) && (tab.Dirty || encodingDirty)
}

func (v *View) updateEditorTabState(tab editorSvc.Tab) {
	item := v.editor.openTabs[tab.ID]
	if item == nil {
		return
	}
	item.Text = editorTabTitle(tab)
	item.Icon = editorTabIcon(tab)
	item.Content.Refresh()
	v.editor.tabs.Refresh()
}

func (v *View) editorSaving(tabID string) bool {
	return v != nil && v.editor != nil && v.editor.savingTabs != nil && v.editor.savingTabs[tabID]
}

func (v *View) setEditorSaveState(tabID string, saving bool, message string) {
	if v == nil || v.editor == nil {
		return
	}
	if v.editor.savingTabs == nil {
		v.editor.savingTabs = map[string]bool{}
	}
	if saving {
		v.editor.savingTabs[tabID] = true
	} else {
		delete(v.editor.savingTabs, tabID)
	}
	if v.editorSession == nil {
		return
	}
	tab, ok := v.editorSession.Tab(tabID)
	if !ok {
		return
	}
	editor, hasEditor := v.textEditor(tabID)
	if hasEditor {
		editor.saving = saving
		if editor.status != nil {
			if message != "" {
				editor.status.SetText(message)
			} else {
				editor.status.SetText(draftStatusTextWithEncoding(tab, editor.encodingDirty(), !editor.hasExplicitEncoding()))
			}
		}
		if editor.saveButton != nil {
			preview := v.editor.previews[tabID]
			setSaveEnabled(editor.saveButton, !saving && editorSaveAllowed(tab, preview, editor.encodingDirty(), editor.hasExplicitEncoding()))
		}
	}
	if hasEditor && editor.tabState != nil {
		editor.tabState.SetText(editorStateTextWithSaving(tab, saving))
	}
}

func (v *View) syncEditorTabOrder() {
	ordered := make([]*container.TabItem, 0, len(v.editor.openTabs))
	for _, tab := range v.editorSession.Tabs() {
		if item := v.editor.openTabs[tab.ID]; item != nil {
			ordered = append(ordered, item)
		}
	}
	v.editor.tabs.Items = ordered
	v.editor.tabs.Refresh()
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
	return editorStateTextWithSaving(tab, false)
}

func editorStateTextWithSaving(tab editorSvc.Tab, saving bool) string {
	if saving {
		return "Saving..."
	}
	state := "Saved"
	if tab.Dirty {
		state = "Modified"
	}
	if tab.Pinned {
		state += " - Pinned"
	}
	return state
}
