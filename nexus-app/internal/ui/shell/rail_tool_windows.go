package shell

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type leftRailToolWindow struct {
	Label       string
	Shortcut    string
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
		{Label: "Project", Shortcut: "1", Activity: "Project selected.", OpenProject: true, Icon: theme.HomeIcon()},
		{Label: "Search", Shortcut: "2", TargetTab: "Search", Activity: "Search selected.", Icon: theme.SearchIcon()},
		{Label: "Problems", Shortcut: "3", TargetTab: "Problems", Activity: "Problems selected.", Icon: theme.WarningIcon()},
		{Label: "Git", Shortcut: "4", TargetTab: "Git", Activity: "Git selected.", Icon: theme.ContentCopyIcon()},
		{Label: "Tasks", Shortcut: "5", TargetTab: "Tasks", Activity: "Tasks selected.", Icon: theme.MediaPlayIcon()},
		{Label: "Jobs", Shortcut: "6", TargetTab: "Jobs", Activity: "Jobs selected.", Icon: theme.ListIcon()},
		{Label: "Data", Shortcut: "7", TargetTab: "Data", Activity: "Data & Analytics selected.", Icon: theme.StorageIcon()},
		{Label: "Artifacts", Shortcut: "8", TargetTab: "Artifacts", Activity: "Artifacts selected.", Icon: theme.DocumentIcon()},
		{Label: "Operations", Shortcut: "9", TargetTab: "Operations", Activity: "Operations selected.", Icon: theme.ComputerIcon()},
		{Label: "Diagnostics", Shortcut: "0", TargetTab: "Diagnostics", Activity: "Diagnostics selected.", Icon: theme.VisibilityIcon()},
	}
}

func (v *View) openLeftRailToolWindow(tool leftRailToolWindow) {
	if tool.OpenProject {
		v.openHomeTab()
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
