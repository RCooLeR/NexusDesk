package shell

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type rightRailToolWindow struct {
	Label          string
	Shortcut       string
	ShortcutKey    fyne.KeyName
	TargetTab      string
	Activity       string
	FocusAssistant bool
	Icon           fyne.Resource
}

func (tool rightRailToolWindow) ButtonLabel() string {
	if tool.Shortcut == "" {
		return tool.Label
	}
	return fmt.Sprintf("%s  %s", tool.Shortcut, tool.Label)
}

func rightRailToolWindows() []rightRailToolWindow {
	return []rightRailToolWindow{
		{Label: "Assistant", Shortcut: "Alt+A", ShortcutKey: fyne.KeyA, Activity: "Assistant selected.", FocusAssistant: true, Icon: theme.MailComposeIcon()},
		{Label: "Sources", Shortcut: "Alt+S", ShortcutKey: fyne.KeyS, TargetTab: "Artifacts", Activity: "Assistant sources and artifacts selected.", Icon: theme.SearchIcon()},
		{Label: "Lineage", Shortcut: "Alt+L", ShortcutKey: fyne.KeyL, TargetTab: "Artifacts", Activity: "Artifact lineage selected.", Icon: theme.DocumentIcon()},
		{Label: "Monitor", Shortcut: "Alt+M", ShortcutKey: fyne.KeyM, TargetTab: "Jobs", Activity: "Job monitor selected.", Icon: theme.ListIcon()},
		{Label: "Inspector", Shortcut: "Alt+I", ShortcutKey: fyne.KeyI, TargetTab: "Diagnostics", Activity: "Inspector diagnostics selected.", Icon: theme.VisibilityIcon()},
	}
}

func (v *View) openRightRailToolWindow(tool rightRailToolWindow) {
	if tool.FocusAssistant {
		if v.window != nil && v.assistantPrompt != nil {
			v.window.Canvas().Focus(v.assistantPrompt)
		}
		v.addActivity(tool.Activity)
		return
	}
	if tool.TargetTab == "" {
		v.addActivity(tool.Label + " is unavailable.")
		return
	}
	if !v.selectBottomTab(tool.TargetTab) {
		v.addActivity(tool.Label + " panel is unavailable.")
		return
	}
	v.addActivity(tool.Activity)
}
