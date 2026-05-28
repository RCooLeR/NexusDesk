package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/brand"
)

func (v *View) newRail() fyne.CanvasObject {
	logo := canvas.NewImageFromResource(brand.HorizontalLogo())
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(128, 38))
	workspaceButton := widget.NewButtonWithIcon("Workbench", theme.HomeIcon(), func() {
		v.addActivity("Workbench selected.")
	})
	dataButton := widget.NewButtonWithIcon("Data", theme.StorageIcon(), func() {
		if !v.selectBottomTab("Data") {
			v.addActivity("Data panel is unavailable.")
			return
		}
		v.addActivity("Data & Analytics selected.")
	})
	artifactsButton := widget.NewButtonWithIcon("Artifacts", theme.DocumentIcon(), func() {
		if !v.selectBottomTab("Artifacts") {
			v.addActivity("Artifacts panel is unavailable.")
			return
		}
		v.addActivity("Artifacts selected.")
	})
	settingsButton := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), v.openSettingsTab)
	return container.NewPadded(container.NewVBox(logo, widget.NewSeparator(), workspaceButton, dataButton, artifactsButton, layout.NewSpacer(), settingsButton))
}

func (v *View) newToolbar() fyne.CanvasObject {
	openButton := widget.NewButtonWithIcon("Open Workspace", theme.FolderOpenIcon(), v.openWorkspaceDialog)
	refreshButton := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), v.refreshWorkspace)
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search workspace")
	searchEntry.OnSubmitted = v.searchWorkspace
	searchButton := widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		v.searchWorkspace(searchEntry.Text)
	})
	return container.NewBorder(nil, nil, container.NewHBox(openButton, refreshButton), searchButton, searchEntry)
}

func (v *View) newBottomPanel() fyne.CanvasObject {
	activity := container.NewScroll(v.activityLog)
	activity.SetMinSize(fyne.NewSize(200, 90))
	tabs := container.NewAppTabs(
		bottomTabGroup("Workbench", theme.HomeIcon(),
			container.NewTabItemWithIcon("Activity", theme.HistoryIcon(), activity),
			container.NewTabItemWithIcon("Search", theme.SearchIcon(), v.newSearchPanel()),
			container.NewTabItemWithIcon("Problems", theme.WarningIcon(), v.newProblemsPanel()),
			container.NewTabItemWithIcon("Git", theme.ContentCopyIcon(), v.newGitPanel()),
			container.NewTabItemWithIcon("Tasks", theme.MediaPlayIcon(), v.newTasksPanel()),
			container.NewTabItemWithIcon("Jobs", theme.ListIcon(), v.newJobsPanel()),
			container.NewTabItemWithIcon("Rollbacks", theme.ContentUndoIcon(), v.newRollbackPanel()),
		),
		bottomTabGroup("Data Studio", theme.StorageIcon(),
			container.NewTabItemWithIcon("Data", theme.StorageIcon(), v.newDataPanel()),
			container.NewTabItemWithIcon("Operations", theme.ComputerIcon(), v.newOperationsPanel()),
			container.NewTabItemWithIcon("Artifacts", theme.DocumentIcon(), v.newArtifactsPanel()),
		),
		bottomTabGroup("Knowledge", theme.InfoIcon(),
			container.NewTabItemWithIcon("History", theme.InfoIcon(), v.newHistoryPanel()),
			container.NewTabItemWithIcon("Chat", theme.MailComposeIcon(), v.newChatHistoryPanel()),
			container.NewTabItemWithIcon("Agent Audit", theme.InfoIcon(), v.newAgentAuditPanel()),
		),
		bottomTabGroup("System", theme.VisibilityIcon(),
			container.NewTabItemWithIcon("Diagnostics", theme.VisibilityIcon(), v.newDiagnosticsPanel()),
			container.NewTabItemWithIcon("Approvals", theme.ConfirmIcon(), v.newApprovalsPanel()),
		),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	v.bottomTabs = tabs
	return tabs
}

func bottomTabGroup(title string, icon fyne.Resource, items ...*container.TabItem) *container.TabItem {
	tabs := container.NewAppTabs(items...)
	tabs.SetTabLocation(container.TabLocationTop)
	return container.NewTabItemWithIcon(title, icon, tabs)
}
