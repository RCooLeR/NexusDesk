package shell

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type toolWindowSide string

const (
	toolWindowSideLeft   toolWindowSide = "left"
	toolWindowSideRight  toolWindowSide = "right"
	toolWindowSideBottom toolWindowSide = "bottom"
)

type toolWindowRegistration struct {
	ID             string
	Label          string
	Side           toolWindowSide
	Shortcut       string
	ShortcutKey    fyne.KeyName
	TargetTab      string
	Activity       string
	OpenProject    bool
	FocusAssistant bool
	Icon           fyne.Resource
}

func (tool toolWindowRegistration) ButtonLabel() string {
	if tool.Shortcut == "" {
		return tool.Label
	}
	return fmt.Sprintf("%s  %s", tool.Shortcut, tool.Label)
}

type toolWindowRegistry struct {
	ordered []toolWindowRegistration
	byID    map[string]toolWindowRegistration
}

func newToolWindowRegistry(tools []toolWindowRegistration) toolWindowRegistry {
	registry := toolWindowRegistry{
		ordered: make([]toolWindowRegistration, 0, len(tools)),
		byID:    map[string]toolWindowRegistration{},
	}
	for _, tool := range tools {
		if tool.ID == "" {
			continue
		}
		registry.ordered = append(registry.ordered, tool)
		registry.byID[tool.ID] = tool
	}
	return registry
}

func (r toolWindowRegistry) ForSide(side toolWindowSide) []toolWindowRegistration {
	tools := []toolWindowRegistration{}
	for _, tool := range r.ordered {
		if tool.Side == side {
			tools = append(tools, tool)
		}
	}
	return tools
}

func (r toolWindowRegistry) ShortcutTools() []toolWindowRegistration {
	tools := []toolWindowRegistration{}
	for _, tool := range r.ordered {
		if tool.ShortcutKey != "" {
			tools = append(tools, tool)
		}
	}
	return tools
}

func (r toolWindowRegistry) Lookup(id string) (toolWindowRegistration, bool) {
	tool, ok := r.byID[id]
	return tool, ok
}

func defaultToolWindowRegistry() toolWindowRegistry {
	return newToolWindowRegistry([]toolWindowRegistration{
		{ID: "project", Label: "Project", Side: toolWindowSideLeft, Shortcut: "Alt+1", ShortcutKey: fyne.Key1, Activity: "Project selected.", OpenProject: true, Icon: theme.HomeIcon()},
		{ID: "search", Label: "Search", Side: toolWindowSideLeft, Shortcut: "Alt+2", ShortcutKey: fyne.Key2, TargetTab: "Search", Activity: "Search selected.", Icon: theme.SearchIcon()},
		{ID: "problems", Label: "Problems", Side: toolWindowSideLeft, Shortcut: "Alt+3", ShortcutKey: fyne.Key3, TargetTab: "Problems", Activity: "Problems selected.", Icon: theme.WarningIcon()},
		{ID: "git", Label: "Git", Side: toolWindowSideLeft, Shortcut: "Alt+4", ShortcutKey: fyne.Key4, TargetTab: "Git", Activity: "Git selected.", Icon: theme.ContentCopyIcon()},
		{ID: "tasks", Label: "Tasks", Side: toolWindowSideLeft, Shortcut: "Alt+5", ShortcutKey: fyne.Key5, TargetTab: "Tasks", Activity: "Tasks selected.", Icon: theme.MediaPlayIcon()},
		{ID: "jobs", Label: "Jobs", Side: toolWindowSideLeft, Shortcut: "Alt+6", ShortcutKey: fyne.Key6, TargetTab: "Jobs", Activity: "Jobs selected.", Icon: theme.ListIcon()},
		{ID: "data", Label: "Data", Side: toolWindowSideLeft, Shortcut: "Alt+7", ShortcutKey: fyne.Key7, TargetTab: "Data", Activity: "Data & Analytics selected.", Icon: theme.StorageIcon()},
		{ID: "artifacts", Label: "Artifacts", Side: toolWindowSideLeft, Shortcut: "Alt+8", ShortcutKey: fyne.Key8, TargetTab: "Artifacts", Activity: "Artifacts selected.", Icon: theme.DocumentIcon()},
		{ID: "operations", Label: "Operations", Side: toolWindowSideLeft, Shortcut: "Alt+9", ShortcutKey: fyne.Key9, TargetTab: "Operations", Activity: "Operations selected.", Icon: theme.ComputerIcon()},
		{ID: "diagnostics", Label: "Diagnostics", Side: toolWindowSideLeft, Shortcut: "Alt+0", ShortcutKey: fyne.Key0, TargetTab: "Diagnostics", Activity: "Diagnostics selected.", Icon: theme.VisibilityIcon()},
		{ID: "activity", Label: "Activity", Side: toolWindowSideLeft, Shortcut: "Alt+Y", ShortcutKey: fyne.KeyY, TargetTab: "Activity", Activity: "Activity selected.", Icon: theme.HistoryIcon()},
		{ID: "audit", Label: "Audit", Side: toolWindowSideLeft, Shortcut: "Alt+U", ShortcutKey: fyne.KeyU, TargetTab: "Agent Audit", Activity: "Agent audit selected.", Icon: theme.InfoIcon()},
		{ID: "assistant", Label: "Assistant", Side: toolWindowSideRight, Shortcut: "Alt+A", ShortcutKey: fyne.KeyA, Activity: "Assistant selected.", FocusAssistant: true, Icon: theme.MailComposeIcon()},
		{ID: "sources", Label: "Sources", Side: toolWindowSideRight, Shortcut: "Alt+S", ShortcutKey: fyne.KeyS, TargetTab: "Artifacts", Activity: "Assistant sources and artifacts selected.", Icon: theme.SearchIcon()},
		{ID: "lineage", Label: "Lineage", Side: toolWindowSideRight, Shortcut: "Alt+L", ShortcutKey: fyne.KeyL, TargetTab: "Artifacts", Activity: "Artifact lineage selected.", Icon: theme.DocumentIcon()},
		{ID: "monitor", Label: "Monitor", Side: toolWindowSideRight, Shortcut: "Alt+M", ShortcutKey: fyne.KeyM, TargetTab: "Jobs", Activity: "Job monitor selected.", Icon: theme.ListIcon()},
		{ID: "inspector", Label: "Inspector", Side: toolWindowSideRight, Shortcut: "Alt+I", ShortcutKey: fyne.KeyI, TargetTab: "Diagnostics", Activity: "Inspector diagnostics selected.", Icon: theme.VisibilityIcon()},
		{ID: "history", Label: "History", Side: toolWindowSideBottom, TargetTab: "History", Activity: "History selected.", Icon: theme.InfoIcon()},
		{ID: "approvals", Label: "Approvals", Side: toolWindowSideBottom, TargetTab: "Approvals", Activity: "Approvals selected.", Icon: theme.ConfirmIcon()},
	})
}
