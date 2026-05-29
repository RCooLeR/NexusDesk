package shell

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type leftRailToolWindow struct {
	Label       string
	Shortcut    string
	ShortcutKey fyne.KeyName
	TargetTab   string
	Activity    string
	OpenProject bool
	Icon        fyne.Resource
}

func (tool leftRailToolWindow) ButtonLabel() string {
	if tool.Shortcut == "" {
		return tool.Label
	}
	return fmt.Sprintf("%s  %s", tool.Shortcut, tool.Label)
}

func leftRailToolWindows() []leftRailToolWindow {
	return []leftRailToolWindow{
		{Label: "Project", Shortcut: "Alt+1", ShortcutKey: fyne.Key1, Activity: "Project selected.", OpenProject: true, Icon: theme.HomeIcon()},
		{Label: "Search", Shortcut: "Alt+2", ShortcutKey: fyne.Key2, TargetTab: "Search", Activity: "Search selected.", Icon: theme.SearchIcon()},
		{Label: "Problems", Shortcut: "Alt+3", ShortcutKey: fyne.Key3, TargetTab: "Problems", Activity: "Problems selected.", Icon: theme.WarningIcon()},
		{Label: "Git", Shortcut: "Alt+4", ShortcutKey: fyne.Key4, TargetTab: "Git", Activity: "Git selected.", Icon: theme.ContentCopyIcon()},
		{Label: "Tasks", Shortcut: "Alt+5", ShortcutKey: fyne.Key5, TargetTab: "Tasks", Activity: "Tasks selected.", Icon: theme.MediaPlayIcon()},
		{Label: "Jobs", Shortcut: "Alt+6", ShortcutKey: fyne.Key6, TargetTab: "Jobs", Activity: "Jobs selected.", Icon: theme.ListIcon()},
		{Label: "Data", Shortcut: "Alt+7", ShortcutKey: fyne.Key7, TargetTab: "Data", Activity: "Data & Analytics selected.", Icon: theme.StorageIcon()},
		{Label: "Artifacts", Shortcut: "Alt+8", ShortcutKey: fyne.Key8, TargetTab: "Artifacts", Activity: "Artifacts selected.", Icon: theme.DocumentIcon()},
		{Label: "Operations", Shortcut: "Alt+9", ShortcutKey: fyne.Key9, TargetTab: "Operations", Activity: "Operations selected.", Icon: theme.ComputerIcon()},
		{Label: "Diagnostics", Shortcut: "Alt+0", ShortcutKey: fyne.Key0, TargetTab: "Diagnostics", Activity: "Diagnostics selected.", Icon: theme.VisibilityIcon()},
	}
}

func (v *View) openLeftRailToolWindow(tool leftRailToolWindow) {
	if tool.OpenProject {
		v.openHomeTab()
		v.setLeftRailActive(tool.Label)
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
	v.setLeftRailActive(tool.Label)
	v.addActivity(tool.Activity)
}
