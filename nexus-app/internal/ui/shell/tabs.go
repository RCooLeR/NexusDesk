package shell

import (
	"path/filepath"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
)

func newEditorTabs() *container.DocTabs {
	tabs := container.NewDocTabs(container.NewTabItem("Welcome", welcomePanel()))
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

func (v *View) configureEditorTabs() {
	v.editorTabs.OnClosed = func(item *container.TabItem) {
		for relPath, tab := range v.openTabs {
			if tab == item {
				delete(v.openTabs, relPath)
				return
			}
		}
	}
}

func (v *View) openPreviewTab(preview domain.FilePreview) {
	if existing := v.openTabs[preview.RelPath]; existing != nil {
		existing.Content = newFilePreview(preview)
		v.editorTabs.Select(existing)
		return
	}
	tab := container.NewTabItem(filepath.Base(preview.RelPath), newFilePreview(preview))
	v.openTabs[preview.RelPath] = tab
	v.editorTabs.Append(tab)
	v.editorTabs.Select(tab)
}

func (v *View) addPlaceholderTab(title string, body string) {
	tab := container.NewTabItem(title, widget.NewRichTextFromMarkdown(body))
	v.editorTabs.Append(tab)
	v.editorTabs.Select(tab)
}
