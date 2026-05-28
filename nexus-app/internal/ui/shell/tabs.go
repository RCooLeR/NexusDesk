package shell

import (
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
)

func newEditorTabs(welcomeTitle string, openWorkspace func(), openFile func()) *container.DocTabs {
	tabs := container.NewDocTabs(container.NewTabItem(welcomeTitle, welcomePanel(openWorkspace, openFile)))
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

func welcomePanel(openWorkspace func(), openFile func()) fyne.CanvasObject {
	title := widget.NewRichTextFromMarkdown("# Nexus Augentic Studio\n\nNative local-first workbench.")
	openWorkspaceButton := widget.NewButtonWithIcon("Open Workspace", theme.FolderOpenIcon(), openWorkspace)
	openFileButton := widget.NewButtonWithIcon("Open File", theme.FileTextIcon(), openFile)
	return container.NewCenter(container.NewVBox(title, container.NewHBox(openWorkspaceButton, openFileButton)))
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
