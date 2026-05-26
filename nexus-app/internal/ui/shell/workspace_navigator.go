package shell

import (
	"path"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (v *View) newWorkspaceNavigator() fyne.CanvasObject {
	actions := container.NewHBox(
		widget.NewButtonWithIcon("", theme.FileIcon(), v.promptCreateFile),
		widget.NewButtonWithIcon("", theme.ContentCopyIcon(), v.promptCopyFile),
		widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), v.promptRenameFile),
		widget.NewButtonWithIcon("", theme.DeleteIcon(), v.confirmDeleteFile),
	)
	tree := newWorkspaceTree(v.state, v.workspaceService, v.openWorkspaceNode)
	return container.NewBorder(actions, nil, nil, nil, tree)
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
