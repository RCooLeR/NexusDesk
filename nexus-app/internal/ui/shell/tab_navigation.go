package shell

import "fyne.io/fyne/v2/container"

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
	if _, ok := v.editorSession.Close(id, false); !ok {
		v.addActivity("Close blocked because the tab has unsaved changes.")
		v.editorTabs.Select(item)
		return
	}
	delete(v.openTabs, id)
	delete(v.tabIDs, item)
	v.editorTabs.Remove(item)
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
