package shell

import (
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
)

func (v *View) closeSelectedTab() {
	item := v.editorTabs.Selected()
	if item == nil {
		return
	}
	v.requestCloseTab(item)
}

func (v *View) requestCloseTab(item *container.TabItem) {
	id := v.tabIDs[item]
	if id == "" {
		v.editorTabs.Remove(item)
		return
	}
	if tab, ok := v.editorSession.Tab(id); ok && tab.Dirty {
		v.editorTabs.Select(item)
		dialog.ShowConfirm("Discard unsaved changes?", dirtyTabCloseMessage(tab.Title), func(confirm bool) {
			if !confirm {
				v.addActivity("Kept modified tab " + tab.Title + " open.")
				v.editorTabs.Select(item)
				return
			}
			v.closeEditorTabItem(item, id, true)
		}, v.window)
		return
	}
	v.closeEditorTabItem(item, id, false)
}

func (v *View) closeEditorTabItem(item *container.TabItem, id string, force bool) {
	if _, ok := v.editorSession.Close(id, force); !ok {
		v.addActivity("Close blocked because the tab has unsaved changes.")
		v.editorTabs.Select(item)
		return
	}
	delete(v.openTabs, id)
	delete(v.tabIDs, item)
	v.editorTabs.Remove(item)
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
	if len(v.editorTabs.Items) == 0 {
		return
	}
	current := v.editorTabs.Selected()
	index := 0
	for i, item := range v.editorTabs.Items {
		if item == current {
			index = i
			break
		}
	}
	next := (index + delta + len(v.editorTabs.Items)) % len(v.editorTabs.Items)
	v.editorTabs.Select(v.editorTabs.Items[next])
}
