package shell

import (
	"fmt"
	"path"
	"strings"

	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
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

func (v *View) promptCreateFolder() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before creating folders.")
		return
	}
	target := widget.NewEntry()
	target.SetText(defaultCreateFolderPath(selectedPathOrEmpty(v)))
	dialog.ShowForm("Create folder", "Create", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Path", target),
	}, func(confirm bool) {
		if !confirm {
			return
		}
		proposal, err := v.workspaceService.ApplyDirectoryCreate(workspace.Root, workspaceSvc.DirectoryCreateRequest{RelPath: target.Text})
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

func (v *View) setNavigatorClipboard(mode string) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode != "cut" {
		mode = "copy"
	}
	source := selectedPathOrEmpty(v)
	if source == "" {
		v.addActivity("Select a file before using the file clipboard.")
		return
	}
	if selectedWorkspaceNodeKind(v.state.Workspace(), source) != domain.NodeFile {
		v.addActivity("File clipboard actions currently support files only.")
		return
	}
	v.navigatorClipboard = navigatorClipboard{Mode: mode, SourceRelPath: source}
	if mode == "cut" {
		v.addActivity("Cut " + source + ". Choose Paste on a target folder or file.")
		return
	}
	v.addActivity("Copied " + source + ". Choose Paste on a target folder or file.")
}

func (v *View) hasNavigatorClipboard() bool {
	return strings.TrimSpace(v.navigatorClipboard.SourceRelPath) != ""
}

func (v *View) pasteNavigatorClipboard() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before pasting files.")
		return
	}
	clipboard := v.navigatorClipboard
	if strings.TrimSpace(clipboard.SourceRelPath) == "" {
		v.addActivity("Copy or cut a file before pasting.")
		return
	}
	targetDirectory := navigatorPasteDirectory(workspace, selectedPathOrEmpty(v))
	targetRelPath := path.Join(targetDirectory, path.Base(clipboard.SourceRelPath))
	if clipboard.Mode == "cut" && targetRelPath == clipboard.SourceRelPath {
		v.addActivity("Paste target is the same as the cut file location.")
		return
	}
	var proposal workspaceSvc.FileOperationProposal
	var err error
	if clipboard.Mode == "cut" {
		proposal, err = v.workspaceService.ApplyFileMove(workspace.Root, workspaceSvc.FileMoveRequest{
			SourceRelPath: clipboard.SourceRelPath,
			TargetRelPath: targetRelPath,
		})
		if err == nil {
			v.navigatorClipboard = navigatorClipboard{}
		}
		v.finishFileOperation(proposal.Message, err)
		return
	}
	targetRelPath = navigatorUniqueCopyPath(workspace, targetRelPath)
	proposal, err = v.workspaceService.ApplyFileCopy(workspace.Root, workspaceSvc.FileCopyRequest{
		SourceRelPath: clipboard.SourceRelPath,
		TargetRelPath: targetRelPath,
	})
	v.finishFileOperation(proposal.Message, err)
}

func navigatorPasteDirectory(workspace domain.Workspace, selected string) string {
	selected = strings.TrimSpace(selected)
	if selected == "" {
		return ""
	}
	if selectedWorkspaceNodeKind(workspace, selected) == domain.NodeDirectory {
		return selected
	}
	directory := path.Dir(selected)
	if directory == "." || directory == "/" {
		return ""
	}
	return directory
}

func navigatorUniqueCopyPath(workspace domain.Workspace, targetRelPath string) string {
	targetRelPath = path.Clean(strings.TrimSpace(targetRelPath))
	if targetRelPath == "." {
		targetRelPath = "copy"
	}
	if !workspaceContainsPath(workspace, targetRelPath) {
		return targetRelPath
	}
	dir := path.Dir(targetRelPath)
	base := path.Base(targetRelPath)
	extension := path.Ext(base)
	name := strings.TrimSuffix(base, extension)
	for index := 2; index < 1000; index++ {
		candidateName := fmt.Sprintf("%s-copy-%d%s", name, index, extension)
		candidate := candidateName
		if dir != "." && dir != "/" {
			candidate = path.Join(dir, candidateName)
		}
		if !workspaceContainsPath(workspace, candidate) {
			return candidate
		}
	}
	return targetRelPath
}

func selectedWorkspaceNodeKind(workspace domain.Workspace, relPath string) domain.WorkspaceNodeKind {
	if node, ok := findWorkspaceNode(workspace.Tree, relPath); ok {
		return node.Kind
	}
	return domain.NodeFile
}

func workspaceContainsPath(workspace domain.Workspace, relPath string) bool {
	_, ok := findWorkspaceNode(workspace.Tree, relPath)
	return ok
}

func findWorkspaceNode(nodes []domain.WorkspaceNode, relPath string) (domain.WorkspaceNode, bool) {
	relPath = path.Clean(strings.TrimSpace(relPath))
	if relPath == "." {
		relPath = ""
	}
	for _, node := range nodes {
		if path.Clean(node.RelPath) == relPath {
			return node, true
		}
		if found, ok := findWorkspaceNode(node.Children, relPath); ok {
			return found, true
		}
	}
	return domain.WorkspaceNode{}, false
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

func defaultCreateFolderPath(selected string) string {
	if selected == "" {
		return "new-folder"
	}
	if path.Ext(selected) == "" {
		return path.Join(selected, "new-folder")
	}
	return path.Join(path.Dir(selected), "new-folder")
}
