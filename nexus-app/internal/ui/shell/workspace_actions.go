package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"nexusdesk/internal/domain"
	jobsSvc "nexusdesk/internal/services/jobs"
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

func (v *View) openFileDialog() {
	dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		if uri == nil {
			return
		}
		defer uri.Close()
		v.openSingleFile(uri.URI().Path())
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

func (v *View) openSingleFile(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		v.addActivity("No file selected.")
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	if info.IsDir() {
		dialog.ShowError(fmt.Errorf("selected path is a folder, not a file"), v.window)
		return
	}
	root := filepath.Dir(path)
	relPath := filepath.Base(path)
	v.openWorkspace(root)
	if v.state.Workspace().Root == "" {
		return
	}
	preview, err := v.workspaceService.PreviewFile(root, relPath)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.openPreviewTab(preview)
	v.addActivity("Opened file " + path + ".")
}

func (v *View) openWorkspace(root string) {
	workspace, err := v.workspaceService.Open(root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.state.SetWorkspace(workspace)
	v.gitFileBadges = map[string]string{}
	store, err := v.metadataStoreForWorkspace(workspace.Root)
	if err == nil {
		if status, err := store.Ensure(); err == nil {
			v.metadataStore = store
			v.approvalService.SetRepository(newApprovalMetadataRepository(store))
			v.jobService.SetRepository(store, true)
			v.addActivity(status.Message)
			v.runWorkspaceOpenAction(workspaceOpenActionJobsRefresh, v.refreshJobs)
			v.runWorkspaceOpenAction(workspaceOpenActionChatHistoryRefresh, func() {
				v.loadAssistantChatHistory()
				v.refreshChatHistory("")
			})
			v.runWorkspaceOpenAction(workspaceOpenActionAgentAuditRefresh, v.refreshAgentAudit)
			v.runWorkspaceOpenAction(workspaceOpenActionUnifiedHistoryRefresh, func() {
				v.refreshHistory("", "")
			})
			v.runWorkspaceOpenAction(workspaceOpenActionCompatibilityImport, func() {
				v.startCompatibilityImport(workspace.Root, store)
			})
		} else {
			v.metadataStore = nil
			v.approvalService.SetRepository(nil)
			v.runWorkspaceOpenAction(workspaceOpenActionChatHistoryRefresh, func() {
				v.loadAssistantChatHistory()
				v.refreshChatHistory("")
			})
			v.runWorkspaceOpenAction(workspaceOpenActionAgentAuditRefresh, v.refreshAgentAudit)
			v.runWorkspaceOpenAction(workspaceOpenActionUnifiedHistoryRefresh, func() {
				v.refreshHistory("", "")
			})
			v.addActivity("Metadata store unavailable: " + err.Error())
		}
	} else {
		v.metadataStore = nil
		v.approvalService.SetRepository(nil)
		v.runWorkspaceOpenAction(workspaceOpenActionChatHistoryRefresh, func() {
			v.loadAssistantChatHistory()
			v.refreshChatHistory("")
		})
		v.runWorkspaceOpenAction(workspaceOpenActionAgentAuditRefresh, v.refreshAgentAudit)
		v.runWorkspaceOpenAction(workspaceOpenActionUnifiedHistoryRefresh, func() {
			v.refreshHistory("", "")
		})
		v.addActivity("Metadata store unavailable: " + err.Error())
	}
	v.runWorkspaceOpenAction(workspaceOpenActionNavigatorRefresh, v.refreshNavigator)
	v.runWorkspaceOpenAction(workspaceOpenActionAssistantPinsRefresh, v.refreshAssistantContextPins)
	v.status.SetText(fmt.Sprintf("%s: %d indexed, %d ignored, %d unreadable", workspace.Name, workspace.Summary.Included, workspace.Summary.Ignored, workspace.Summary.Unreadable))
	v.addActivity("Opened workspace " + workspace.Root)
	v.closeWelcomeTabs()
	v.runWorkspaceOpenAction(workspaceOpenActionApprovalsRefresh, v.refreshApprovals)
}

func (v *View) metadataStoreForWorkspace(root string) (*metadataSvc.Store, error) {
	if v.metadataStore != nil && sameWorkspaceRoot(v.metadataStore.Root(), root) {
		return v.metadataStore, nil
	}
	if v.metadataStore != nil {
		_ = v.metadataStore.Close()
		v.metadataStore = nil
	}
	return metadataSvc.NewStore(root)
}

func sameWorkspaceRoot(left string, right string) bool {
	left = filepath.Clean(strings.TrimSpace(left))
	right = filepath.Clean(strings.TrimSpace(right))
	return strings.EqualFold(left, right)
}

func (v *View) startCompatibilityImport(workspaceRoot string, store *metadataSvc.Store) {
	if !v.beginCompatibilityImport(workspaceRoot) {
		return
	}
	pending, err := store.CompatibilityImportPending()
	if err != nil {
		v.endCompatibilityImport(workspaceRoot)
		v.addActivity("Compatibility import state check failed: " + err.Error())
		return
	}
	if !pending {
		v.endCompatibilityImport(workspaceRoot)
		return
	}
	jobLabel := compatibilityImportJobLabel()
	job, ctx := v.jobService.Start("metadata-compat-import", jobLabel)
	v.jobService.AppendLog(job.ID, "Workspace: "+workspaceRoot)
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	go func() {
		report, err := store.ImportCompatibilityDataContext(ctx, metadataSvc.CompatibilityImportOptions{})
		fyne.Do(func() {
			defer v.endCompatibilityImport(workspaceRoot)
			current := v.state.Workspace()
			if current.Root != workspaceRoot || v.metadataStore != store {
				v.jobService.Finish(job.ID, jobsSvc.StatusCanceled, "Compatibility metadata import cancelled after workspace switch.", nil)
				v.refreshJobs()
				return
			}
			if err != nil {
				if isDataJobCanceled(err) {
					v.jobService.Finish(job.ID, jobsSvc.StatusCanceled, "Compatibility metadata import cancelled.", nil)
					v.addActivity("Compatibility metadata import cancelled.")
				} else {
					v.jobService.Finish(job.ID, jobsSvc.StatusFailed, "Compatibility metadata import failed.", err)
					v.addActivity("Compatibility metadata import skipped: " + err.Error())
				}
				v.refreshJobs()
				return
			}
			v.jobService.Finish(job.ID, jobsSvc.StatusSuccess, report.Message, nil)
			v.jobService.AppendLog(job.ID, report.Message)
			v.addActivity(report.Message)
			v.refreshJobs()
			v.loadAssistantChatHistory()
			v.refreshChatHistory("")
			v.refreshAgentAudit()
			v.refreshHistory("", "")
			v.refreshApprovals()
		})
	}()
}

func (v *View) beginCompatibilityImport(workspaceRoot string) bool {
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return false
	}
	v.compatibilityImportMu.Lock()
	defer v.compatibilityImportMu.Unlock()
	if v.compatibilityImportByWS == nil {
		v.compatibilityImportByWS = map[string]bool{}
	}
	if v.compatibilityImportByWS[workspaceRoot] {
		return false
	}
	v.compatibilityImportByWS[workspaceRoot] = true
	return true
}

func (v *View) endCompatibilityImport(workspaceRoot string) {
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return
	}
	v.compatibilityImportMu.Lock()
	defer v.compatibilityImportMu.Unlock()
	delete(v.compatibilityImportByWS, workspaceRoot)
}

func compatibilityImportJobLabel() string {
	return "Compatibility metadata import"
}

func (v *View) refreshNavigator() {
	v.navigator.Objects = []fyne.CanvasObject{v.newWorkspaceNavigator()}
	v.navigator.Refresh()
}

func (v *View) openWorkspaceNode(node domain.WorkspaceNode) {
	if node.Kind == domain.NodeDirectory {
		v.addActivity("Selected folder " + node.RelPath)
		v.refreshAssistantContextPins()
		return
	}
	v.openWorkspaceRelFile(node.RelPath)
}
