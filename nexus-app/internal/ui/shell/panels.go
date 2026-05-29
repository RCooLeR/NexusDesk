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
	railButtons := make([]fyne.CanvasObject, 0, len(leftRailToolWindows())+4)
	railButtons = append(railButtons, widget.NewLabelWithStyle("Tool Windows", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	for _, tool := range leftRailToolWindows() {
		tool := tool
		railButtons = append(railButtons, widget.NewButtonWithIcon(tool.ButtonLabel(), tool.Icon, func() {
			v.openLeftRailToolWindow(tool)
		}))
	}
	settingsButton := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), v.openSettingsTab)
	return container.NewPadded(container.NewVBox(
		logo,
		widget.NewSeparator(),
		container.NewVBox(railButtons...),
		layout.NewSpacer(),
		settingsButton,
	))
}

func (v *View) newRightRail() fyne.CanvasObject {
	railButtons := make([]fyne.CanvasObject, 0, len(rightRailToolWindows())+1)
	railButtons = append(railButtons, widget.NewLabelWithStyle("AI Tools", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
	for _, tool := range rightRailToolWindows() {
		tool := tool
		railButtons = append(railButtons, widget.NewButtonWithIcon(tool.ButtonLabel(), tool.Icon, func() {
			v.openRightRailToolWindow(tool)
		}))
	}
	return container.NewPadded(container.NewVBox(railButtons...))
}

func (v *View) newToolbar() fyne.CanvasObject {
	openButton := widget.NewButtonWithIcon("Open Workspace", theme.FolderOpenIcon(), v.openWorkspaceDialog)
	refreshButton := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), v.refreshWorkspace)
	tasksButton := widget.NewButtonWithIcon("Tasks", theme.MediaPlayIcon(), func() {
		if !v.selectBottomTab("Tasks") {
			v.addActivity("Tasks panel is unavailable.")
			return
		}
		v.addActivity("Tasks selected.")
	})
	gitButton := widget.NewButtonWithIcon("Git", theme.ContentCopyIcon(), func() {
		if !v.selectBottomTab("Git") {
			v.addActivity("Git panel is unavailable.")
			return
		}
		v.addActivity("Git selected.")
	})
	commandButton := widget.NewButtonWithIcon("Command", theme.MenuIcon(), v.openCommandPaletteDialog)
	settingsButton := widget.NewButtonWithIcon("", theme.SettingsIcon(), v.openSettingsTab)
	v.toolbarWorkspaceStatus = newToolbarStatusLabel("Workspace: none")
	v.toolbarBranchStatus = newToolbarStatusLabel("Branch: refresh Git")
	v.toolbarProviderStatus = newToolbarStatusLabel("Model: provider?/model not selected")
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search workspace")
	searchEntry.OnSubmitted = v.searchWorkspace
	searchButton := widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		v.searchWorkspace(searchEntry.Text)
	})
	v.refreshToolbarStatus()
	left := container.NewHBox(
		openButton,
		refreshButton,
		gitButton,
		tasksButton,
		widget.NewSeparator(),
		v.toolbarWorkspaceStatus,
		v.toolbarBranchStatus,
	)
	right := container.NewHBox(v.toolbarProviderStatus, commandButton, settingsButton, searchButton)
	return container.NewPadded(container.NewBorder(nil, nil, left, right, searchEntry))
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
