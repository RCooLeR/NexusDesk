package shell

import (
	"fmt"
	"path"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
)

const (
	navigatorActionCreate     = "Create file near selection"
	navigatorActionCreateDir  = "Create folder near selection"
	navigatorActionCopy       = "Copy selected file"
	navigatorActionCut        = "Cut selected file"
	navigatorActionPaste      = "Paste file into selection"
	navigatorActionRename     = "Rename or move selected file"
	navigatorActionDelete     = "Delete selected file"
	navigatorActionCopyPath   = "Copy relative path"
	navigatorActionUseContext = "Use as assistant context"
)

type navigatorClipboard struct {
	Mode          string
	SourceRelPath string
}

func (v *View) newWorkspaceNavigator() fyne.CanvasObject {
	summary := widget.NewLabel(navigatorSelectionSummary(""))
	summary.Truncation = fyne.TextTruncateEllipsis
	visibility := widget.NewLabel("")
	visibility.Truncation = fyne.TextTruncateEllipsis

	quickActions := container.NewHBox(
		widget.NewButtonWithIcon("", theme.FileIcon(), v.promptCreateFile),
		widget.NewButtonWithIcon("", theme.FolderNewIcon(), v.promptCreateFolder),
		widget.NewButtonWithIcon("", theme.ContentCopyIcon(), v.promptCopyFile),
		widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), v.promptRenameFile),
		widget.NewButtonWithIcon("", theme.DeleteIcon(), v.confirmDeleteFile),
	)
	var refreshVisibility func()
	tree, store := newWorkspaceTree(v.state, v.workspaceService, v.gitFileBadges, func(node domain.WorkspaceNode) {
		summary.SetText(navigatorSelectionSummary(node.RelPath))
		v.openWorkspaceNode(node)
	}, func(node domain.WorkspaceNode, event *fyne.PointEvent) {
		v.state.SetSelectedPath(node.RelPath)
		summary.SetText(navigatorSelectionSummary(node.RelPath))
		v.showNavigatorContextMenu(node, event)
	}, func(string, domain.ScanSummary) {
		if refreshVisibility != nil {
			refreshVisibility()
		}
	})
	refreshVisibility = func() {
		visibility.SetText(navigatorVisibilitySummary(store.includeIgnored, store.visibleSummary()))
	}
	refreshVisibility()
	v.navigatorTree = tree
	v.navigatorStore = store
	v.navigatorRefreshSummary = refreshVisibility
	showIgnored := widget.NewCheck("Show ignored", func(include bool) {
		if err := store.setIncludeIgnored(include); err != nil {
			v.addActivity("Could not reload project tree: " + err.Error())
			return
		}
		tree.CloseAllBranches()
		tree.Refresh()
		refreshVisibility()
	})
	revealButton := widget.NewButtonWithIcon("", theme.ZoomFitIcon(), func() {
		branches := store.branchPathForSelection(v.state.SelectedPath())
		if len(branches) == 0 {
			for _, rootID := range store.roots {
				tree.OpenBranch(rootID)
			}
			return
		}
		for _, branch := range branches {
			tree.OpenBranch(branch)
		}
	})
	collapseButton := widget.NewButtonWithIcon("", theme.MenuDropUpIcon(), func() {
		tree.CloseAllBranches()
	})
	treeControls := container.NewHBox(showIgnored, revealButton, collapseButton)
	header := container.NewVBox(summary, quickActions, treeControls, visibility)
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

func navigatorVisibilitySummary(includeIgnored bool, summary domain.ScanSummary) string {
	truncation := ""
	if summary.EntryCap > 0 {
		truncation = fmt.Sprintf(", %d folder(s) clipped by entry cap", summary.EntryCap)
	}
	if includeIgnored {
		return fmt.Sprintf("%d shown, %d ignored visible where safe%s", summary.Included, summary.Ignored, truncation)
	}
	if summary.Ignored == 0 {
		return fmt.Sprintf("%d shown%s", summary.Included, truncation)
	}
	return fmt.Sprintf("%d shown, %d ignored hidden%s", summary.Included, summary.Ignored, truncation)
}

func navigatorActionOptions(selected string, kind domain.WorkspaceNodeKind, hasClipboard bool) []string {
	if selected == "" {
		return []string{navigatorActionCreate, navigatorActionCreateDir}
	}
	if kind == domain.NodeDirectory {
		options := []string{
			navigatorActionCreate,
			navigatorActionCreateDir,
			navigatorActionCopyPath,
			navigatorActionUseContext,
		}
		if hasClipboard {
			options = append([]string{navigatorActionCreate, navigatorActionCreateDir, navigatorActionPaste}, options[2:]...)
		}
		return options
	}
	options := []string{
		navigatorActionCreate,
		navigatorActionCreateDir,
		navigatorActionCopy,
		navigatorActionCut,
		navigatorActionRename,
		navigatorActionDelete,
		navigatorActionCopyPath,
		navigatorActionUseContext,
	}
	if hasClipboard {
		options = append(options[:4], append([]string{navigatorActionPaste}, options[4:]...)...)
	}
	return options
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
	options := navigatorActionOptions(node.RelPath, node.Kind, v.hasNavigatorClipboard())
	menu := fyne.NewMenu("", navigatorContextMenuItems(options, v.handleNavigatorAction)...)
	widget.ShowPopUpMenuAtPosition(menu, v.window.Canvas(), event.AbsolutePosition)
}

func (v *View) handleNavigatorAction(action string) {
	switch action {
	case navigatorActionCreate:
		v.promptCreateFile()
	case navigatorActionCreateDir:
		v.promptCreateFolder()
	case navigatorActionCopy:
		v.setNavigatorClipboard("copy")
	case navigatorActionCut:
		v.setNavigatorClipboard("cut")
	case navigatorActionPaste:
		v.pasteNavigatorClipboard()
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
