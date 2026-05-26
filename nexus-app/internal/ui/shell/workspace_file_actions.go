package shell

import (
	"fmt"
	"path"

	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	workspaceSvc "nexusdesk/internal/services/workspace"
)

func (v *View) promptCreateFile() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before creating files.")
		return
	}
	target := widget.NewEntry()
	target.SetText(defaultCreatePath(selectedPathOrEmpty(v)))
	dialog.ShowForm("Create file", "Create", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Path", target),
	}, func(confirm bool) {
		if !confirm {
			return
		}
		proposal, err := v.workspaceService.ApplyFileCreate(workspace.Root, workspaceSvc.FileCreateRequest{RelPath: target.Text})
		v.finishFileOperation(proposal.Message, err)
	}, v.window)
}

func (v *View) promptCopyFile() {
	source := selectedPathOrEmpty(v)
	if source == "" {
		v.addActivity("Select a file before copying.")
		return
	}
	target := widget.NewEntry()
	target.SetText(defaultCopyPath(source))
	dialog.ShowForm("Copy file", "Copy", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Source", widget.NewLabel(source)),
		widget.NewFormItem("Target", target),
	}, func(confirm bool) {
		if !confirm {
			return
		}
		workspace := v.state.Workspace()
		proposal, err := v.workspaceService.ApplyFileCopy(workspace.Root, workspaceSvc.FileCopyRequest{SourceRelPath: source, TargetRelPath: target.Text})
		v.finishFileOperation(proposal.Message, err)
	}, v.window)
}

func (v *View) promptRenameFile() {
	source := selectedPathOrEmpty(v)
	if source == "" {
		v.addActivity("Select a file before renaming or moving.")
		return
	}
	target := widget.NewEntry()
	target.SetText(source)
	dialog.ShowForm("Rename or move file", "Apply", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Source", widget.NewLabel(source)),
		widget.NewFormItem("Target", target),
	}, func(confirm bool) {
		if !confirm {
			return
		}
		workspace := v.state.Workspace()
		proposal, err := v.workspaceService.ApplyFileRename(workspace.Root, workspaceSvc.FileMoveRequest{SourceRelPath: source, TargetRelPath: target.Text})
		v.finishFileOperation(proposal.Message, err)
	}, v.window)
}

func (v *View) confirmDeleteFile() {
	source := selectedPathOrEmpty(v)
	if source == "" {
		v.addActivity("Select a file before deleting.")
		return
	}
	dialog.ShowConfirm("Delete file", "Delete "+source+"?", func(confirm bool) {
		if !confirm {
			return
		}
		workspace := v.state.Workspace()
		proposal, err := v.workspaceService.ApplyFileDelete(workspace.Root, source)
		v.finishFileOperation(proposal.Message, err)
	}, v.window)
}

func (v *View) finishFileOperation(message string, err error) {
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.addActivity(message)
	v.refreshWorkspace()
}

func defaultCopyPath(source string) string {
	dir := path.Dir(source)
	base := path.Base(source)
	extension := path.Ext(base)
	name := base[:len(base)-len(extension)]
	copyName := fmt.Sprintf("%s-copy%s", name, extension)
	if dir == "." || dir == "/" {
		return copyName
	}
	return path.Join(dir, copyName)
}
