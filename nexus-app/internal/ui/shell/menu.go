package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"

	"nexusdesk/internal/buildinfo"
	userGuideSvc "nexusdesk/internal/services/userguide"
)

func (v *View) InstallWindowActions() {
	v.window.SetMainMenu(v.mainMenu())
	v.installShortcuts()
}

func (v *View) mainMenu() *fyne.MainMenu {
	openWorkspace := menuItem("Open Workspace", shortcutOpenWorkspace(), v.openWorkspaceDialog)
	openFile := fyne.NewMenuItemWithIcon("Open File", theme.FileTextIcon(), v.openFileDialog)
	refresh := menuItem("Refresh Workspace", shortcutRefreshWorkspace(), v.refreshWorkspace)
	closeTab := menuItem("Close Tab", shortcutCloseTab(), v.closeSelectedTab)
	settings := menuItem("Settings", shortcutSettings(), v.openSettingsTab)
	copySelection := copyDataCellMenuItem(v.copySelection)
	saveDraft := menuItem("Save Draft", shortcutSaveDraft(), v.saveActiveEditorDraft)
	revertDraft := menuItem("Revert Draft", shortcutRevertDraft(), v.revertActiveEditorDraft)
	formatDraft := fyne.NewMenuItemWithIcon("Format Draft", theme.DocumentCreateIcon(), v.formatActiveEditorDraft)
	symbols := fyne.NewMenuItemWithIcon("Go to Symbol", theme.SearchIcon(), v.openActiveEditorSymbols)
	references := fyne.NewMenuItemWithIcon("Find References", theme.SearchIcon(), v.openActiveEditorReferences)
	findReplace := menuItem("Find / Replace", shortcutFindReplace(), v.openFindReplaceDialog)
	renameMove := fyne.NewMenuItemWithIcon("Rename / Move Selected", theme.DocumentCreateIcon(), v.promptRenameFile)
	copyFile := fyne.NewMenuItemWithIcon("Copy Selected File", theme.ContentCopyIcon(), v.promptCopyFile)
	deleteFile := fyne.NewMenuItemWithIcon("Delete Selected File", theme.DeleteIcon(), v.confirmDeleteFile)
	discoverTasks := fyne.NewMenuItemWithIcon("Discover Tasks", theme.SearchIcon(), v.discoverTasks)
	tasks := fyne.NewMenuItemWithIcon("Tasks", theme.MediaPlayIcon(), func() {
		if !v.selectBottomTab("Tasks") {
			v.addActivity("Tasks panel is unavailable.")
		}
	})
	jobs := fyne.NewMenuItemWithIcon("Jobs", theme.ListIcon(), func() {
		if !v.selectBottomTab("Jobs") {
			v.addActivity("Jobs panel is unavailable.")
		}
	})
	safeAgentGuide := fyne.NewMenuItemWithIcon("Safe Agent Guide", theme.HelpIcon(), v.openSafeAgentGuideTab)
	betaFeedbackGuide := fyne.NewMenuItemWithIcon("Beta Feedback & Release Notes", theme.DocumentIcon(), v.openBetaFeedbackGuideTab)
	smokeChecklistGuide := fyne.NewMenuItemWithIcon("Clean-Machine Smoke Checklist", theme.ConfirmIcon(), v.openSmokeChecklistGuideTab)
	appDataCleanupGuide := fyne.NewMenuItemWithIcon("App Data & Uninstall Cleanup", theme.StorageIcon(), v.openAppDataCleanupGuideTab)
	releaseHygieneGuide := fyne.NewMenuItemWithIcon("Release Hygiene & Antivirus Notes", theme.WarningIcon(), v.openReleaseHygieneGuideTab)
	packageOwnershipGuide := fyne.NewMenuItemWithIcon("Internal Package Ownership", theme.ListIcon(), v.openPackageOwnershipGuideTab)
	contributorGuide := fyne.NewMenuItemWithIcon("Contributor Setup & Standards", theme.DocumentIcon(), v.openContributorGuideTab)
	about := fyne.NewMenuItemWithIcon("About Nexus", theme.InfoIcon(), v.showAbout)

	return fyne.NewMainMenu(
		fyne.NewMenu("File", openWorkspace, openFile, refresh, fyne.NewMenuItemSeparator(), closeTab),
		fyne.NewMenu("Edit",
			copySelection,
			fyne.NewMenuItemSeparator(),
			saveDraft,
			revertDraft,
			fyne.NewMenuItemSeparator(),
			findReplace,
		),
		fyne.NewMenu("View",
			fyne.NewMenuItem("Workbench", func() { v.addActivity("Workbench selected.") }),
			fyne.NewMenuItem("Data & Analytics", func() {
				if !v.selectBottomTab("Data") {
					v.addActivity("Data panel is unavailable.")
					return
				}
				v.addActivity("Data & Analytics selected.")
			}),
			fyne.NewMenuItem("Artifacts", func() {
				if !v.selectBottomTab("Artifacts") {
					v.addActivity("Artifacts panel is unavailable.")
					return
				}
				v.addActivity("Artifacts selected.")
			}),
			settings,
		),
		fyne.NewMenu("Navigate",
			menuItem("Quick Open", shortcutQuickOpen(), v.openQuickOpenDialog),
			symbols,
			references,
			fyne.NewMenuItemSeparator(),
			menuItem("Next Tab", shortcutNextTab(), v.selectNextTab),
			menuItem("Previous Tab", shortcutPreviousTab(), v.selectPreviousTab),
		),
		fyne.NewMenu("Code", saveDraft, revertDraft, formatDraft, fyne.NewMenuItemSeparator(), findReplace),
		fyne.NewMenu("Refactor", renameMove, copyFile, deleteFile),
		fyne.NewMenu("Run", discoverTasks, tasks, jobs),
		fyne.NewMenu("Tools",
			fyne.NewMenuItem("Refresh Activity", func() { v.activityLog.Refresh() }),
			menuItem("Command Palette", shortcutCommandPalette(), v.openCommandPaletteDialog),
		),
		fyne.NewMenu("Help", safeAgentGuide, betaFeedbackGuide, smokeChecklistGuide, appDataCleanupGuide, releaseHygieneGuide, packageOwnershipGuide, contributorGuide, fyne.NewMenuItemSeparator(), about),
	)
}

func menuItem(label string, shortcut fyne.Shortcut, action func()) *fyne.MenuItem {
	item := fyne.NewMenuItem(label, action)
	item.Shortcut = shortcut
	return item
}

func copyDataCellMenuItem(action func()) *fyne.MenuItem {
	return fyne.NewMenuItem("Copy Data Cell", action)
}

func (v *View) showAbout() {
	dialog.ShowInformation(
		"About Nexus",
		buildinfo.AboutText(),
		v.window,
	)
}

func (v *View) openSafeAgentGuideTab() {
	v.addPlaceholderTab("Safe Agent Guide", userGuideSvc.SafeAgentMarkdown())
}

func (v *View) openBetaFeedbackGuideTab() {
	v.addPlaceholderTab("Beta Feedback", userGuideSvc.BetaFeedbackMarkdown())
}

func (v *View) openSmokeChecklistGuideTab() {
	v.addPlaceholderTab("Smoke Checklist", userGuideSvc.CleanMachineSmokeChecklistMarkdown())
}

func (v *View) openAppDataCleanupGuideTab() {
	v.addPlaceholderTab("App Data Cleanup", userGuideSvc.AppDataCleanupMarkdown())
}

func (v *View) openReleaseHygieneGuideTab() {
	v.addPlaceholderTab("Release Hygiene", userGuideSvc.ReleaseHygieneMarkdown())
}

func (v *View) openPackageOwnershipGuideTab() {
	v.addPlaceholderTab("Package Ownership", userGuideSvc.PackageOwnershipMarkdown())
}

func (v *View) openContributorGuideTab() {
	v.addPlaceholderTab("Contributor Guide", userGuideSvc.ContributorMarkdown())
}
