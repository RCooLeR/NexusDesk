package shell

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"nexusdesk/internal/domain"
	metadataSvc "nexusdesk/internal/services/metadata"
)

func (v *View) openWorkspaceDialog() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		if uri == nil {
			return
		}
		v.openWorkspace(uri.Path())
	}, v.window)
}

func (v *View) refreshWorkspace() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("No workspace to refresh.")
		return
	}
	v.openWorkspace(workspace.Root)
}

func (v *View) openWorkspace(root string) {
	workspace, err := v.workspaceService.Open(root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.state.SetWorkspace(workspace)
	if store, err := metadataSvc.NewStore(workspace.Root); err == nil {
		if status, err := store.Ensure(); err == nil {
			v.metadataStore = store
			v.jobService.SetRepository(store, true)
			v.addActivity(status.Message)
			v.refreshJobs()
			v.loadAssistantChatHistory()
			v.refreshAgentAudit()
		} else {
			v.metadataStore = nil
			v.loadAssistantChatHistory()
			v.refreshAgentAudit()
			v.addActivity("Metadata store unavailable: " + err.Error())
		}
	} else {
		v.metadataStore = nil
		v.loadAssistantChatHistory()
		v.refreshAgentAudit()
		v.addActivity("Metadata store unavailable: " + err.Error())
	}
	v.navigator.Objects = []fyne.CanvasObject{v.newWorkspaceNavigator()}
	v.navigator.Refresh()
	v.refreshAssistantContextPins()
	v.status.SetText(fmt.Sprintf("%s: %d indexed, %d ignored, %d unreadable", workspace.Name, workspace.Summary.Included, workspace.Summary.Ignored, workspace.Summary.Unreadable))
	v.addActivity("Opened workspace " + workspace.Root)
	v.refreshApprovals()
}

func (v *View) openWorkspaceNode(node domain.WorkspaceNode) {
	if node.Kind == domain.NodeDirectory {
		v.addActivity("Selected folder " + node.RelPath)
		v.refreshAssistantContextPins()
		return
	}
	workspace := v.state.Workspace()
	preview, err := v.workspaceService.PreviewFile(workspace.Root, node.RelPath)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.openPreviewTab(preview)
	v.addActivity("Opened " + node.RelPath)
	v.refreshAssistantContextPins()
}
