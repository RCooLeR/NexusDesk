package shell

import (
	"path"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
)

const (
	navigatorActionCreate     = "Create file near selection"
	navigatorActionCopy       = "Copy selected file"
	navigatorActionRename     = "Rename or move selected file"
	navigatorActionDelete     = "Delete selected file"
	navigatorActionCopyPath   = "Copy relative path"
	navigatorActionUseContext = "Use as assistant context"
)

func (v *View) newWorkspaceNavigator() fyne.CanvasObject {
	summary := widget.NewLabel(navigatorSelectionSummary(""))
	summary.Truncation = fyne.TextTruncateEllipsis

	quickActions := container.NewHBox(
		widget.NewButtonWithIcon("", theme.FileIcon(), v.promptCreateFile),
		widget.NewButtonWithIcon("", theme.ContentCopyIcon(), v.promptCopyFile),
		widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), v.promptRenameFile),
		widget.NewButtonWithIcon("", theme.DeleteIcon(), v.confirmDeleteFile),
	)
	header := container.NewVBox(summary, quickActions)
	tree := newWorkspaceTree(v.state, v.workspaceService, func(node domain.WorkspaceNode) {
		summary.SetText(navigatorSelectionSummary(node.RelPath))
		v.openWorkspaceNode(node)
	}, func(node domain.WorkspaceNode, event *fyne.PointEvent) {
		v.state.SetSelectedPath(node.RelPath)
		summary.SetText(navigatorSelectionSummary(node.RelPath))
		v.showNavigatorContextMenu(node, event)
	})
	return container.NewBorder(header, nil, nil, nil, tree)
}

func selectedPathOrEmpty(v *View) string {
	return v.state.SelectedPath()
}

func defaultCreatePath(selected string) string {
	if selected == "" {
		return "new-file.txt"
	}
	if path.Ext(selected) == "" {
		return path.Join(selected, "new-file.txt")
	}
	return path.Join(path.Dir(selected), "new-file.txt")
}

func navigatorSelectionSummary(selected string) string {
	if selected == "" {
		return "No file selected"
	}
	return selected
}

func navigatorActionOptions(selected string, kind domain.WorkspaceNodeKind) []string {
	if selected == "" {
		return []string{navigatorActionCreate}
	}
	if kind == domain.NodeDirectory {
		return []string{
			navigatorActionCreate,
			navigatorActionCopyPath,
			navigatorActionUseContext,
		}
	}
	return []string{
		navigatorActionCreate,
		navigatorActionCopy,
		navigatorActionRename,
		navigatorActionDelete,
		navigatorActionCopyPath,
		navigatorActionUseContext,
	}
}

func navigatorContextMenuItems(options []string, onAction func(string)) []*fyne.MenuItem {
	items := make([]*fyne.MenuItem, 0, len(options)+1)
	for index, option := range options {
		if index == 1 {
			items = append(items, fyne.NewMenuItemSeparator())
		}
		action := option
		items = append(items, fyne.NewMenuItem(option, func() {
			onAction(action)
		}))
	}
	return items
}

func (v *View) showNavigatorContextMenu(node domain.WorkspaceNode, event *fyne.PointEvent) {
	options := navigatorActionOptions(node.RelPath, node.Kind)
	menu := fyne.NewMenu("", navigatorContextMenuItems(options, v.handleNavigatorAction)...)
	widget.ShowPopUpMenuAtPosition(menu, v.window.Canvas(), event.AbsolutePosition)
}

func (v *View) handleNavigatorAction(action string) {
	switch action {
	case navigatorActionCreate:
		v.promptCreateFile()
	case navigatorActionCopy:
		v.promptCopyFile()
	case navigatorActionRename:
		v.promptRenameFile()
	case navigatorActionDelete:
		v.confirmDeleteFile()
	case navigatorActionCopyPath:
		v.copySelectedWorkspacePath()
	case navigatorActionUseContext:
		v.useSelectedPathForAssistantContext()
	}
}

func (v *View) copySelectedWorkspacePath() {
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.addActivity("Select a file or folder before copying its path.")
		return
	}
	v.window.Clipboard().SetContent(selected)
	v.addActivity("Copied workspace path " + selected + ".")
}

func (v *View) useSelectedPathForAssistantContext() {
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.addActivity("Select a file or folder before using it as assistant context.")
		return
	}
	v.pinAssistantContextPath(selected)
}
