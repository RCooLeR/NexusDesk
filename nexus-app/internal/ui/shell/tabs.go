package shell

import (
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
)

func newEditorTabs(welcomeTitle string) *container.DocTabs {
	tabs := container.NewDocTabs(container.NewTabItem(welcomeTitle, welcomePanel()))
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

func welcomePanel() fyne.CanvasObject {
	return container.NewCenter(widget.NewRichTextFromMarkdown("# Nexus Augentic Studio\n\nFyne-native migration shell. Open a workspace to begin."))
}

func (v *View) configureEditorTabs() {
	v.editorTabs.CloseIntercept = func(item *container.TabItem) {
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
}

func (v *View) openPreviewTab(preview domain.FilePreview) {
	tabState := v.editorSession.OpenFile(preview.RelPath, filepath.Base(preview.RelPath))
	if existing := v.openTabs[tabState.ID]; existing != nil {
		existing.Content = v.newEditorPanel(tabState, preview)
		v.updateEditorTabState(tabState)
		v.editorTabs.Select(existing)
		return
	}
	tab := container.NewTabItemWithIcon(editorTabTitle(tabState), editorTabIcon(tabState), v.newEditorPanel(tabState, preview))
	v.openTabs[tabState.ID] = tab
	v.tabIDs[tab] = tabState.ID
	v.editorTabs.Append(tab)
	v.editorTabs.Select(tab)
}

func (v *View) addPlaceholderTab(title string, body string) {
	tabState := v.editorSession.OpenPlaceholder(title)
	tab := container.NewTabItemWithIcon(editorTabTitle(tabState), editorTabIcon(tabState), widget.NewRichTextFromMarkdown(body))
	v.openTabs[tabState.ID] = tab
	v.tabIDs[tab] = tabState.ID
	v.editorTabs.Append(tab)
	v.editorTabs.Select(tab)
}
