package shell

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	gitSvc "nexusdesk/internal/services/git"
	settingsSvc "nexusdesk/internal/services/settings"
)

type toolbarStatusSnapshot struct {
	Workspace     domain.Workspace
	GitStatus     gitSvc.Status
	Settings      settingsSvc.Settings
	SettingsError string
}

func (v *View) refreshToolbarStatus() {
	if v == nil || v.toolbarWorkspaceStatus == nil || v.toolbarBranchStatus == nil || v.toolbarProviderStatus == nil {
		return
	}
	settings := settingsSvc.Defaults()
	settingsError := ""
	if v.settingsStore != nil {
		loaded, err := v.settingsStore.LoadForDisplay()
		if err != nil {
			settingsError = err.Error()
		} else {
			settings = loaded
		}
	}
	workspace := domain.Workspace{}
	if v.state != nil {
		workspace = v.state.Workspace()
	}
	snapshot := toolbarStatusSnapshot{
		Workspace:     workspace,
		GitStatus:     v.gitStatusSnapshot,
		Settings:      settings,
		SettingsError: settingsError,
	}
	v.toolbarWorkspaceStatus.SetText(toolbarWorkspaceText(snapshot))
	v.toolbarBranchStatus.SetText(toolbarBranchText(snapshot))
	v.toolbarProviderStatus.SetText(toolbarProviderText(snapshot))
}

func newToolbarStatusLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Truncation = fyne.TextTruncateEllipsis
	return label
}

func toolbarWorkspaceText(snapshot toolbarStatusSnapshot) string {
	name := strings.TrimSpace(snapshot.Workspace.Name)
	if name == "" {
		return "Workspace: none"
	}
	return "Workspace: " + name
}

func toolbarBranchText(snapshot toolbarStatusSnapshot) string {
	if !snapshot.GitStatus.Available || strings.TrimSpace(snapshot.GitStatus.Branch) == "" {
		return "Branch: refresh Git"
	}
	branch := strings.TrimSpace(snapshot.GitStatus.Branch)
	if strings.TrimSpace(snapshot.GitStatus.Head) != "" {
		head := strings.TrimSpace(snapshot.GitStatus.Head)
		if len(head) > 7 {
			head = head[:7]
		}
		branch += " @ " + head
	}
	if strings.TrimSpace(snapshot.GitStatus.AheadBehind) != "" {
		branch += " " + strings.TrimSpace(snapshot.GitStatus.AheadBehind)
	}
	return "Branch: " + branch
}

func toolbarProviderText(snapshot toolbarStatusSnapshot) string {
	if strings.TrimSpace(snapshot.SettingsError) != "" {
		return "Model: settings error"
	}
	provider := strings.TrimSpace(snapshot.Settings.Provider)
	if provider == "" {
		provider = "provider?"
	}
	model := strings.TrimSpace(snapshot.Settings.Model)
	if model == "" {
		model = "model not selected"
	}
	return "Model: " + provider + "/" + model
}
