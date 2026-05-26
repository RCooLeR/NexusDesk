package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
)

func (v *View) InstallWindowActions() {
	v.window.SetMainMenu(v.mainMenu())
	v.installShortcuts()
}

func (v *View) mainMenu() *fyne.MainMenu {
	openWorkspace := menuItem("Open Workspace", shortcutOpenWorkspace(), v.openWorkspaceDialog)
	refresh := menuItem("Refresh Workspace", shortcutRefreshWorkspace(), v.refreshWorkspace)
	closeTab := menuItem("Close Tab", shortcutCloseTab(), v.closeSelectedTab)
	settings := menuItem("Settings", shortcutSettings(), func() {
		v.addPlaceholderTab("Settings", "Provider, access policy, model, and connector settings will live here.")
	})
	about := fyne.NewMenuItemWithIcon("About Nexus", theme.InfoIcon(), v.showAbout)

	return fyne.NewMainMenu(
		fyne.NewMenu("File", openWorkspace, refresh, fyne.NewMenuItemSeparator(), closeTab),
		fyne.NewMenu("Edit",
			disabledMenuItem("Save Draft"),
			disabledMenuItem("Revert Draft"),
			fyne.NewMenuItemSeparator(),
			disabledMenuItem("Find"),
		),
		fyne.NewMenu("View",
			fyne.NewMenuItem("Workbench", func() { v.addActivity("Workbench selected.") }),
			fyne.NewMenuItem("Data & Analytics", func() {
				v.addPlaceholderTab("Data & Analytics", "Database, CSV, Excel, and analysis workflows will live here.")
			}),
			fyne.NewMenuItem("Artifacts", func() {
				v.addPlaceholderTab("Artifacts", "Generated reports, exports, lineage, and comparisons will live here.")
			}),
			settings,
		),
		fyne.NewMenu("Navigate",
			menuItem("Next Tab", shortcutNextTab(), v.selectNextTab),
			menuItem("Previous Tab", shortcutPreviousTab(), v.selectPreviousTab),
		),
		fyne.NewMenu("Tools",
			fyne.NewMenuItem("Refresh Activity", func() { v.activityLog.Refresh() }),
			disabledMenuItem("Command Palette"),
		),
		fyne.NewMenu("Help", about),
	)
}

func menuItem(label string, shortcut fyne.Shortcut, action func()) *fyne.MenuItem {
	item := fyne.NewMenuItem(label, action)
	item.Shortcut = shortcut
	return item
}

func disabledMenuItem(label string) *fyne.MenuItem {
	item := fyne.NewMenuItem(label, nil)
	item.Disabled = true
	return item
}

func (v *View) showAbout() {
	dialog.ShowInformation(
		"About Nexus",
		"Nexus Augentic Studio\nAgentic work. Augmented by context.\n\nFyne-native migration build.",
		v.window,
	)
}
