package shell

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const maxCommandPaletteResults = 60

type commandPaletteAction struct {
	ID       string
	Title    string
	Detail   string
	Group    string
	Shortcut string
	Disabled bool
	Run      func()
}

func (v *View) openCommandPaletteDialog() {
	commands := v.commandPaletteActions()
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Run command...")
	status := widget.NewLabel("")
	results := []commandPaletteAction{}
	var picker dialog.Dialog

	runCommand := func(index int) {
		if index < 0 || index >= len(results) {
			return
		}
		command := results[index]
		if command.Disabled || command.Run == nil {
			status.SetText(command.Title + " is unavailable right now.")
			return
		}
		if picker != nil {
			picker.Hide()
		}
		command.Run()
	}
	list := widget.NewList(
		func() int { return len(results) },
		func() fyne.CanvasObject {
			title := widget.NewLabel("")
			title.TextStyle = fyne.TextStyle{Bold: true}
			title.Truncation = fyne.TextTruncateEllipsis
			detail := widget.NewLabel("")
			detail.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(title, detail)
		},
		func(id widget.ListItemID, object fyne.CanvasObject) {
			box := object.(*fyne.Container)
			title := box.Objects[0].(*widget.Label)
			detail := box.Objects[1].(*widget.Label)
			command := results[id]
			title.SetText(commandPaletteTitle(command))
			detail.SetText(commandPaletteDetail(command))
		},
	)
	list.OnSelected = runCommand

	refresh := func(query string) {
		results = filterCommandPaletteActions(commands, query)
		status.SetText(commandPaletteStatusText(len(results), query))
		list.Refresh()
	}
	entry.OnChanged = refresh
	entry.OnSubmitted = func(string) {
		runCommand(0)
	}

	content := container.NewBorder(entry, status, nil, nil, list)
	content.Resize(fyne.NewSize(620, 400))
	picker = dialog.NewCustom("Command Palette", "Close", content, v.window)
	refresh("")
	picker.Show()
	v.window.Canvas().Focus(entry)
}

func (v *View) commandPaletteActions() []commandPaletteAction {
	hasWorkspace := strings.TrimSpace(v.state.Workspace().Root) != ""
	return []commandPaletteAction{
		{
			ID:       "file.open_workspace",
			Title:    "Open Workspace",
			Detail:   "Choose a folder and open it as the active workspace.",
			Group:    "File",
			Shortcut: "Ctrl+O",
			Run:      v.openWorkspaceDialog,
		},
		{
			ID:     "file.open_file",
			Title:  "Open File",
			Detail: "Open a single file by using its parent folder as the workspace.",
			Group:  "File",
			Run:    v.openFileDialog,
		},
		{
			ID:       "workspace.refresh",
			Title:    "Refresh Workspace",
			Detail:   "Reload the active workspace tree and metadata surfaces.",
			Group:    "Workspace",
			Shortcut: "Ctrl+R",
			Disabled: !hasWorkspace,
			Run:      v.refreshWorkspace,
		},
		{
			ID:       "navigate.quick_open",
			Title:    "Quick Open",
			Detail:   "Search workspace files and open the first match.",
			Group:    "Navigate",
			Shortcut: "Ctrl+P",
			Disabled: !hasWorkspace,
			Run:      v.openQuickOpenDialog,
		},
		{
			ID:       "edit.find_replace",
			Title:    "Find / Replace",
			Detail:   "Search or replace text inside the active editor draft.",
			Group:    "Edit",
			Shortcut: "Ctrl+F",
			Run:      v.openFindReplaceDialog,
		},
		{
			ID:       "edit.save_draft",
			Title:    "Save Draft",
			Detail:   "Apply the active text editor draft through the safe write service.",
			Group:    "Edit",
			Shortcut: "Ctrl+S",
			Run:      v.saveActiveEditorDraft,
		},
		{
			ID:       "edit.revert_draft",
			Title:    "Revert Draft",
			Detail:   "Discard the active draft and restore the last loaded file content.",
			Group:    "Edit",
			Shortcut: "Ctrl+Shift+R",
			Run:      v.revertActiveEditorDraft,
		},
		{
			ID:       "tabs.close",
			Title:    "Close Tab",
			Detail:   "Close the active editor tab with dirty-draft protection.",
			Group:    "Navigate",
			Shortcut: "Ctrl+W",
			Run:      v.closeSelectedTab,
		},
		{
			ID:       "tabs.next",
			Title:    "Next Tab",
			Detail:   "Move focus to the next editor tab.",
			Group:    "Navigate",
			Shortcut: "Ctrl+Tab",
			Run:      v.selectNextTab,
		},
		{
			ID:       "tabs.previous",
			Title:    "Previous Tab",
			Detail:   "Move focus to the previous editor tab.",
			Group:    "Navigate",
			Shortcut: "Ctrl+Shift+Tab",
			Run:      v.selectPreviousTab,
		},
		{
			ID:       "settings.open",
			Title:    "Settings",
			Detail:   "Open provider, model, credential, and diagnostic settings.",
			Group:    "View",
			Shortcut: "Ctrl+,",
			Run:      v.openSettingsTab,
		},
		{
			ID:     "help.safe_agent",
			Title:  "Safe Agent Guide",
			Detail: "Open user guidance for approvals, rollbacks, local data, credentials, connectors, and jobs.",
			Group:  "Help",
			Run:    v.openSafeAgentGuideTab,
		},
		{
			ID:     "help.beta_feedback",
			Title:  "Beta Feedback & Release Notes",
			Detail: "Open private-beta reporting guidance, release-note expectations, and redacted issue-report instructions.",
			Group:  "Help",
			Run:    v.openBetaFeedbackGuideTab,
		},
		{
			ID:     "help.smoke_checklist",
			Title:  "Clean-Machine Smoke Checklist",
			Detail: "Open release-candidate smoke checks for install, launch, workspace, assistant, data, diagnostics, and uninstall.",
			Group:  "Help",
			Run:    v.openSmokeChecklistGuideTab,
		},
		{
			ID:     "help.app_data_cleanup",
			Title:  "App Data & Uninstall Cleanup",
			Detail: "Open app data paths, protected-secret storage, workspace state, uninstall, and manual cleanup guidance.",
			Group:  "Help",
			Run:    v.openAppDataCleanupGuideTab,
		},
		v.bottomPanelCommand("view.search", "Search", "Open workspace path and content search results.", "Workbench"),
		v.bottomPanelCommand("view.problems", "Problems", "Open syntax and workspace diagnostics.", "Workbench"),
		v.bottomPanelCommand("view.git", "Git", "Open status, diff, history, blame, and hunk actions.", "Workbench"),
		v.bottomPanelCommand("view.tasks", "Tasks", "Open discovered project task actions and output.", "Workbench"),
		v.bottomPanelCommand("view.jobs", "Jobs", "Open durable job monitor and retry/cancel controls.", "Workbench"),
		v.bottomPanelCommand("view.data", "Data", "Open dataset, SQL, notebook, connector, and chart tools.", "Data"),
		v.bottomPanelCommand("view.operations", "Operations", "Open Docker, Compose, env, config, and runbook inspection.", "Data"),
		v.bottomPanelCommand("view.artifacts", "Artifacts", "Open generated artifacts, lineage, compare, and regeneration.", "Data"),
		v.bottomPanelCommand("view.history", "History", "Open unified history across chats, artifacts, jobs, and agent runs.", "History"),
		v.bottomPanelCommand("view.chat", "Chat", "Open persisted assistant chat search and history.", "History"),
		v.bottomPanelCommand("view.agent_audit", "Agent Audit", "Open persisted agent and tool run audit records.", "History"),
		v.bottomPanelCommand("view.diagnostics", "Diagnostics", "Open provider, metadata, app log, and failure diagnostics.", "System"),
		v.bottomPanelCommand("view.approvals", "Approvals", "Open approval queue and audit records.", "System"),
	}
}

func (v *View) bottomPanelCommand(id string, title string, detail string, group string) commandPaletteAction {
	tabTitle := title
	return commandPaletteAction{
		ID:     id,
		Title:  "Show " + title,
		Detail: detail,
		Group:  group,
		Run: func() {
			if !v.selectBottomTab(tabTitle) {
				v.addActivity(tabTitle + " panel is unavailable.")
				return
			}
			v.addActivity(tabTitle + " panel selected from command palette.")
		},
	}
}

func filterCommandPaletteActions(commands []commandPaletteAction, query string) []commandPaletteAction {
	query = strings.TrimSpace(query)
	scored := make([]scoredCommand, 0, len(commands))
	for index, command := range commands {
		score := scoreCommandPaletteAction(command, query)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredCommand{command: command, index: index, score: score})
	}
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if commandPaletteLess(scored[j], scored[i]) {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
	if len(scored) > maxCommandPaletteResults {
		scored = scored[:maxCommandPaletteResults]
	}
	results := make([]commandPaletteAction, 0, len(scored))
	for _, item := range scored {
		results = append(results, item.command)
	}
	return results
}

func commandPaletteLess(left scoredCommand, right scoredCommand) bool {
	if left.score != right.score {
		return left.score > right.score
	}
	if left.command.Disabled != right.command.Disabled {
		return !left.command.Disabled
	}
	return left.index < right.index
}

type scoredCommand struct {
	command commandPaletteAction
	index   int
	score   int
}

func scoreCommandPaletteAction(command commandPaletteAction, query string) int {
	if query == "" {
		if command.Disabled {
			return 30
		}
		return 80
	}
	needle := strings.ToLower(query)
	title := strings.ToLower(command.Title)
	detail := strings.ToLower(command.Detail)
	group := strings.ToLower(command.Group)
	shortcut := strings.ToLower(command.Shortcut)
	compactNeedle := strings.Join(strings.Fields(needle), "")
	compactTitle := strings.Join(strings.Fields(title), "")
	switch {
	case title == needle:
		return 240
	case strings.HasPrefix(title, needle):
		return 200
	case strings.Contains(group, needle):
		return 160
	case strings.Contains(title, needle):
		return 140
	case strings.Contains(detail, needle) || strings.Contains(shortcut, needle):
		return 100
	case compactNeedle != "" && strings.Contains(compactTitle, compactNeedle):
		return 75
	default:
		return 0
	}
}

func commandPaletteTitle(command commandPaletteAction) string {
	title := command.Title
	if command.Disabled {
		title += " (unavailable)"
	}
	if strings.TrimSpace(command.Group) != "" {
		title += " - " + command.Group
	}
	if strings.TrimSpace(command.Shortcut) != "" {
		title += " - " + command.Shortcut
	}
	return title
}

func commandPaletteDetail(command commandPaletteAction) string {
	detail := strings.TrimSpace(command.Detail)
	if detail == "" {
		return "No details available."
	}
	return detail
}

func commandPaletteStatusText(count int, query string) string {
	if count == 0 {
		return fmt.Sprintf("No matching commands for %q.", query)
	}
	if strings.TrimSpace(query) == "" {
		return fmt.Sprintf("%d command(s) available. Type to filter, Enter runs the first match.", count)
	}
	return fmt.Sprintf("%d command match(es). Enter runs the first match.", count)
}
