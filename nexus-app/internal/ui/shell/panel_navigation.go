package shell

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

const (
	workbenchExpandedOffset   = 0.68
	editorWidthPriorityOffset = 0.82
)

func (v *View) selectBottomTab(title string) bool {
	if v.bottomTabs == nil {
		return false
	}
	if !selectAppTabByTitle(v.bottomTabs, title) {
		return false
	}
	v.enforceEditorWidthPriority()
	v.updateRailActiveStateForTab(title)
	return true
}

func (v *View) collapseBottomPanel() {
	if v == nil {
		return
	}
	v.bottomPanelCollapsed = true
	if v.workbenchSplit != nil {
		v.workbenchSplit.SetOffset(1)
	}
}

func (v *View) expandBottomPanel() {
	if v == nil {
		return
	}
	v.bottomPanelCollapsed = false
	if v.workbenchSplit != nil {
		v.workbenchSplit.SetOffset(workbenchExpandedOffset)
	}
}

func (v *View) newEditorPrioritySplit(rightWorkbench fyne.CanvasObject) *container.Split {
	split := container.NewHSplit(v.editor.tabs, rightWorkbench)
	split.SetOffset(editorWidthPriorityOffset)
	v.mainSplit = split
	return split
}

func (v *View) enforceEditorWidthPriority() {
	if v == nil || v.mainSplit == nil {
		return
	}
	if v.mainSplit.Offset < editorWidthPriorityOffset {
		v.mainSplit.SetOffset(editorWidthPriorityOffset)
	}
}

func selectAppTabByTitle(tabs *container.AppTabs, title string) bool {
	if tabs == nil {
		return false
	}
	for _, item := range tabs.Items {
		if strings.EqualFold(item.Text, title) {
			tabs.Select(item)
			return true
		}
		childTabs, ok := item.Content.(*container.AppTabs)
		if !ok || !selectAppTabByTitle(childTabs, title) {
			continue
		}
		tabs.Select(item)
		return true
	}
	return false
}

func (v *View) isBottomTabSelected(title string) bool {
	if v.bottomTabs == nil {
		return false
	}
	return isAppTabSelected(v.bottomTabs, title)
}

func isAppTabSelected(tabs *container.AppTabs, title string) bool {
	if tabs == nil {
		return false
	}
	selected := tabs.Selected()
	if selected == nil {
		return false
	}
	if strings.EqualFold(selected.Text, title) {
		return true
	}
	childTabs, ok := selected.Content.(*container.AppTabs)
	return ok && isAppTabSelected(childTabs, title)
}
