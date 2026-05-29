package shell

import (
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
)

func (v *View) closeSelectedTab() {
	item := v.editor.tabs.Selected()
	if item == nil {
		return
	}
	v.requestCloseTab(item)
}

func (v *View) requestCloseTab(item *container.TabItem) {
	id := v.editor.tabIDs[item]
	if id == "" {
		v.editor.tabs.Remove(item)
		return
	}
	if tab, ok := v.editorSession.Tab(id); ok && tab.Dirty {
		v.editor.tabs.Select(item)
		dialog.ShowConfirm("Discard unsaved changes?", dirtyTabCloseMessage(tab.Title), func(confirm bool) {
			v.handleDirtyTabCloseDecision(item, id, tab.Title, confirm)
		}, v.window)
		return
	}
	v.closeEditorTabItem(item, id, false)
}

func (v *View) handleDirtyTabCloseDecision(item *container.TabItem, id string, title string, confirm bool) {
	if !confirm {
		v.addActivity("Kept modified tab " + title + " open.")
		v.editor.tabs.Select(item)
		return
	}
	v.closeEditorTabItem(item, id, true)
}

func (v *View) closeEditorTabItem(item *container.TabItem, id string, force bool) {
	if _, ok := v.editorSession.Close(id, force); !ok {
		v.addActivity("Close blocked because the tab has unsaved changes.")
		v.editor.tabs.Select(item)
		return
	}
	delete(v.editor.openTabs, id)
	delete(v.editor.tabIDs, item)
	delete(v.editor.previews, id)
	v.removeTextEditor(id)
	v.editor.tabs.Remove(item)
}

func dirtyTabCloseMessage(tabTitle string) string {
	if tabTitle == "" {
		tabTitle = "this tab"
	}
	return "Discard unsaved changes in " + tabTitle + "?"
}

func (v *View) selectNextTab() {
	v.selectRelativeTab(1)
}

func (v *View) selectPreviousTab() {
	v.selectRelativeTab(-1)
}

func (v *View) selectRelativeTab(delta int) {
	if len(v.editor.tabs.Items) == 0 {
		return
	}
	current := v.editor.tabs.Selected()
	index := 0
	for i, item := range v.editor.tabs.Items {
		if item == current {
			index = i
			break
		}
	}
	next := (index + delta + len(v.editor.tabs.Items)) % len(v.editor.tabs.Items)
	v.editor.tabs.Select(v.editor.tabs.Items[next])
}
