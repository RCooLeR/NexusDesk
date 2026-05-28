package shell

import (
	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	editorSvc "nexusdesk/internal/services/editor"
	recentWorkspacesSvc "nexusdesk/internal/services/recentworkspaces"
)

func newEditorTabs(welcomeItem *container.TabItem) *container.DocTabs {
	tabs := container.NewDocTabs(welcomeItem)
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

func (v *View) newWelcomePanel() fyne.CanvasObject {
	title := widget.NewRichTextFromMarkdown("# Nexus Augentic Studio\n\nNative local-first workbench.")
	openWorkspaceButton := widget.NewButtonWithIcon("Open Workspace", theme.FolderOpenIcon(), v.openWorkspaceDialog)
	openFileButton := widget.NewButtonWithIcon("Open File", theme.FileTextIcon(), v.openFileDialog)
	recent := v.recentWorkspaceRows()
	return container.NewCenter(container.NewVBox(title, container.NewHBox(openWorkspaceButton, openFileButton), widget.NewSeparator(), recent))
}

func (v *View) recentWorkspaceRows() fyne.CanvasObject {
	items, err := v.listRecentWorkspaces()
	if err != nil {
		return widget.NewLabel("Recent workspaces are unavailable: " + err.Error())
	}
	if len(items) == 0 {
		return widget.NewLabel("No recent workspaces yet.")
	}
	rows := []fyne.CanvasObject{widget.NewLabel("Recent workspaces")}
	for _, item := range items {
		item := item
		open := widget.NewButtonWithIcon(item.Name, theme.FolderOpenIcon(), func() {
			v.openWorkspace(item.Path)
		})
		remove := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			v.removeRecentWorkspace(item.Path)
		})
		remove.Importance = widget.LowImportance
		pathLabel := widget.NewLabel(item.Path)
		pathLabel.Truncation = fyne.TextTruncateEllipsis
		rows = append(rows, container.NewBorder(nil, nil, open, remove, pathLabel))
	}
	clear := widget.NewButtonWithIcon("Clear recent workspaces", theme.DeleteIcon(), v.clearRecentWorkspaces)
	clear.Importance = widget.LowImportance
	rows = append(rows, container.NewHBox(layout.NewSpacer(), clear))
	return container.NewVBox(rows...)
}

func (v *View) listRecentWorkspaces() ([]recentWorkspacesSvc.Workspace, error) {
	if v.recentWorkspaceStore == nil {
		return nil, fmt.Errorf("recent workspace store is unavailable")
	}
	return v.recentWorkspaceStore.List()
}

func (v *View) recordRecentWorkspace(root string) {
	if v.recentWorkspaceStore == nil {
		return
	}
	if _, err := v.recentWorkspaceStore.Add(root); err != nil {
		v.addActivity("Could not update recent workspaces: " + err.Error())
	}
}

func (v *View) removeRecentWorkspace(root string) {
	if v.recentWorkspaceStore == nil {
		v.addActivity("Recent workspace store is unavailable.")
		return
	}
	if _, err := v.recentWorkspaceStore.Remove(root); err != nil {
		v.addActivity("Could not remove recent workspace: " + err.Error())
		return
	}
	v.refreshWelcomeTabs()
}

func (v *View) clearRecentWorkspaces() {
	if v.recentWorkspaceStore == nil {
		v.addActivity("Recent workspace store is unavailable.")
		return
	}
	if _, err := v.recentWorkspaceStore.Clear(); err != nil {
		v.addActivity("Could not clear recent workspaces: " + err.Error())
		return
	}
	v.refreshWelcomeTabs()
}

func (v *View) openHomeTab() {
	for item, id := range v.tabIDs {
		tab, ok := v.editorSession.Tab(id)
		if !ok || tab.Kind != editorSvc.KindWelcome {
			continue
		}
		item.Content = v.newWelcomePanel()
		v.editorTabs.Select(item)
		return
	}
	tabState := v.editorSession.OpenWelcome("Home")
	item := container.NewTabItemWithIcon(editorTabTitle(tabState), theme.HomeIcon(), v.newWelcomePanel())
	v.openTabs[tabState.ID] = item
	v.tabIDs[item] = tabState.ID
	v.editorTabs.Append(item)
	v.editorTabs.Select(item)
}

func (v *View) refreshWelcomeTabs() {
	for item, id := range v.tabIDs {
		tab, ok := v.editorSession.Tab(id)
		if !ok || tab.Kind != editorSvc.KindWelcome {
			continue
		}
		item.Content = v.newWelcomePanel()
		item.Content.Refresh()
	}
}

func (v *View) configureEditorTabs() {
	v.editorTabs.CloseIntercept = func(item *container.TabItem) {
		v.requestCloseTab(item)
	}
}

func (v *View) openPreviewTab(preview domain.FilePreview) {
	tabState := v.editorSession.OpenFileWithSource(preview.RelPath, filepath.Base(preview.RelPath), preview.Text)
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

func (v *View) closeWelcomeTabs() {
	for item, id := range v.tabIDs {
		tab, ok := v.editorSession.Tab(id)
		if !ok || tab.Kind != editorSvc.KindWelcome {
			continue
		}
		v.closeEditorTabItem(item, id, true)
	}
}
