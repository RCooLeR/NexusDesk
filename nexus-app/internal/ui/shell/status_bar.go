package shell

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/buildinfo"
	"nexusdesk/internal/domain"
	editorSvc "nexusdesk/internal/services/editor"
	gitSvc "nexusdesk/internal/services/git"
	jobsSvc "nexusdesk/internal/services/jobs"
	settingsSvc "nexusdesk/internal/services/settings"
)

type statusBarSnapshot struct {
	Workspace     domain.Workspace
	Settings      settingsSvc.Settings
	SettingsError string
	GitStatus     gitSvc.Status
	SelectedPath  string
	SaveState     string
	Encoding      string
	LineEnding    string
	Jobs          []jobsSvc.Job
	BuildInfo     buildinfo.Info
}

func (v *View) newStatusBar() fyne.CanvasObject {
	v.status.Truncation = fyne.TextTruncateEllipsis
	v.status.TextStyle = fyne.TextStyle{Monospace: true}
	v.refreshStatusBar()
	return container.NewBorder(widget.NewSeparator(), nil, nil, nil, container.NewPadded(v.status))
}

func (v *View) refreshStatusBar() {
	if v == nil || v.status == nil {
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
	encoding, lineEnding, saveState := v.activeEditorFileStatus()
	workspace := domain.Workspace{}
	selectedPath := ""
	if v.state != nil {
		workspace = v.state.Workspace()
		selectedPath = v.state.SelectedPath()
	}
	jobs := []jobsSvc.Job{}
	if v.jobService != nil {
		jobs = v.jobService.List()
	}
	v.status.SetText(statusBarText(statusBarSnapshot{
		Workspace:     workspace,
		Settings:      settings,
		SettingsError: settingsError,
		GitStatus:     v.gitStatusSnapshot,
		SelectedPath:  selectedPath,
		SaveState:     saveState,
		Encoding:      encoding,
		LineEnding:    lineEnding,
		Jobs:          jobs,
		BuildInfo:     buildinfo.Current(),
	}))
	v.refreshToolbarStatus()
}

func (v *View) activeEditorFileStatus() (string, string, string) {
	if v == nil || v.editor == nil || v.editor.tabs == nil {
		return "n/a", "n/a", "n/a"
	}
	item := v.editor.tabs.Selected()
	if item == nil {
		return "n/a", "n/a", "n/a"
	}
	tabID := strings.TrimSpace(v.editor.tabIDs[item])
	if tabID == "" {
		return "n/a", "n/a", "n/a"
	}
	tab := editorSvc.Tab{}
	if v.editorSession != nil {
		tab, _ = v.editorSession.Tab(tabID)
	}
	if editor, ok := v.editor.textEditors[tabID]; ok && editor != nil {
		encoding := editor.writeEncoding()
		if editor.encodingDirty() {
			encoding += "*"
		}
		text := ""
		if editor.source != nil {
			text = editor.source.Text
		}
		return fallbackStatusValue(encoding, "utf-8"), detectLineEnding(text), editorSaveStateText(tab, editor)
	}
	if preview, ok := v.editor.previews[tabID]; ok {
		return fallbackStatusValue(preview.Encoding, "n/a"), detectLineEnding(preview.Text), "read-only"
	}
	return "n/a", "n/a", "n/a"
}

func editorSaveStateText(tab editorSvc.Tab, editor *textEditorBinding) string {
	if editor != nil && editor.saving {
		return "saving"
	}
	if editor != nil && !editor.hasExplicitEncoding() {
		return "encoding required"
	}
	if tab.Dirty {
		return "modified"
	}
	if editor != nil && editor.encodingDirty() {
		return "encoding changed"
	}
	return "saved"
}

func statusBarText(snapshot statusBarSnapshot) string {
	workspace := "Workspace: none"
	if strings.TrimSpace(snapshot.Workspace.Name) != "" {
		workspace = "Workspace: " + strings.TrimSpace(snapshot.Workspace.Name)
	}
	if strings.TrimSpace(snapshot.Workspace.Root) != "" {
		workspace += fmt.Sprintf(" (%d indexed, %d ignored, %d unreadable)", snapshot.Workspace.Summary.Included, snapshot.Workspace.Summary.Ignored, snapshot.Workspace.Summary.Unreadable)
	}

	provider := fallbackStatusValue(snapshot.Settings.Provider, "provider?")
	model := fallbackStatusValue(snapshot.Settings.Model, "model not selected")
	if strings.TrimSpace(snapshot.SettingsError) != "" {
		model = "settings error"
	}

	branch := "not loaded"
	if snapshot.GitStatus.Available && strings.TrimSpace(snapshot.GitStatus.Branch) != "" {
		branch = strings.TrimSpace(snapshot.GitStatus.Branch)
		if strings.TrimSpace(snapshot.GitStatus.AheadBehind) != "" {
			branch += " " + strings.TrimSpace(snapshot.GitStatus.AheadBehind)
		}
	}

	selected := fallbackStatusValue(snapshot.SelectedPath, "none")
	running, failed := jobStatusCounts(snapshot.Jobs)
	warnings := statusBarWarningCount(snapshot, failed)
	encoding := fallbackStatusValue(snapshot.Encoding, "n/a")
	lineEnding := fallbackStatusValue(snapshot.LineEnding, "n/a")
	saveState := fallbackStatusValue(snapshot.SaveState, "n/a")
	version := fallbackStatusValue(snapshot.BuildInfo.Version, "dev")

	return strings.Join([]string{
		workspace,
		"Provider: " + provider + "/" + model,
		"Branch: " + branch,
		fmt.Sprintf("Jobs: %d running, %d failed", running, failed),
		fmt.Sprintf("Warnings: %d", warnings),
		"Selected: " + selected,
		"Save: " + saveState,
		"Encoding: " + encoding,
		"Line: " + lineEnding,
		"Version: " + version,
	}, "  |  ")
}

func jobStatusCounts(jobs []jobsSvc.Job) (int, int) {
	running := 0
	failed := 0
	for _, job := range jobs {
		switch job.Status {
		case jobsSvc.StatusRunning:
			running++
		case jobsSvc.StatusFailed, jobsSvc.StatusTimedOut:
			failed++
		}
	}
	return running, failed
}

func statusBarWarningCount(snapshot statusBarSnapshot, failedJobs int) int {
	warnings := snapshot.Workspace.Summary.Unreadable + failedJobs
	if strings.TrimSpace(snapshot.SettingsError) != "" {
		warnings++
	}
	if strings.TrimSpace(snapshot.SettingsError) == "" && strings.TrimSpace(snapshot.Settings.Model) == "" {
		warnings++
	}
	return warnings
}

func detectLineEnding(text string) string {
	if text == "" {
		return "n/a"
	}
	hasCRLF := strings.Contains(text, "\r\n")
	withoutCRLF := strings.ReplaceAll(text, "\r\n", "")
	hasLF := strings.Contains(withoutCRLF, "\n")
	hasCR := strings.Contains(withoutCRLF, "\r")
	count := 0
	for _, has := range []bool{hasCRLF, hasLF, hasCR} {
		if has {
			count++
		}
	}
	if count > 1 {
		return "mixed"
	}
	switch {
	case hasCRLF:
		return "CRLF"
	case hasCR:
		return "CR"
	default:
		return "LF"
	}
}

func fallbackStatusValue(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
