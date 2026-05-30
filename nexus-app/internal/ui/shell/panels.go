package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/brand"
)

func (v *View) newRail() fyne.CanvasObject {
	logo := canvas.NewImageFromResource(brand.AppIcon())
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(42, 42))
	railButtons := make([]fyne.CanvasObject, 0, len(leftRailToolWindows())+4)
	v.leftRailButtons = map[string]*railToolButton{}
	if v.activeLeftRailTool == "" {
		v.activeLeftRailTool = defaultLeftRailTool
	}
	for _, tool := range leftRailToolWindows() {
		tool := tool
		button := v.newRailIconButton(tool, func() {
			v.openLeftRailToolWindow(tool)
		})
		v.leftRailButtons[tool.Label] = button
		railButtons = append(railButtons, button)
	}
	v.refreshRailActiveState()
	settingsButton := widget.NewButtonWithIcon("", theme.SettingsIcon(), v.openSettingsTab)
	buttonList := container.NewVScroll(container.NewVBox(railButtons...))
	buttonList.SetMinSize(fyne.NewSize(46, 320))
	header := container.NewVBox(logo, widget.NewSeparator())
	return container.NewPadded(container.NewBorder(header, settingsButton, nil, nil, buttonList))
}

func (v *View) newRightRail() fyne.CanvasObject {
	railButtons := make([]fyne.CanvasObject, 0, len(rightRailToolWindows())+1)
	v.rightRailButtons = map[string]*railToolButton{}
	if v.activeRightRailTool == "" {
		v.activeRightRailTool = defaultRightRailTool
	}
	for _, tool := range rightRailToolWindows() {
		tool := tool
		button := v.newRailIconButton(tool, func() {
			v.openRightRailToolWindow(tool)
		})
		v.rightRailButtons[tool.Label] = button
		railButtons = append(railButtons, button)
	}
	v.refreshRailActiveState()
	return container.NewPadded(container.NewVBox(railButtons...))
}

func (v *View) newRailIconButton(tool toolWindowRegistration, action func()) *railToolButton {
	return newRailIconButton(tool, action, func(text string) {
		if v != nil && v.status != nil {
			v.status.SetText("Tool: " + text)
		}
	}, func() {
		if v != nil {
			v.refreshStatusBar()
		}
	})
}

func (v *View) newToolbar() fyne.CanvasObject {
	openButton := widget.NewButtonWithIcon("Open Workspace", theme.FolderOpenIcon(), v.openWorkspaceDialog)
	refreshButton := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), v.refreshWorkspace)
	tasksButton := widget.NewButtonWithIcon("Tasks", theme.MediaPlayIcon(), func() {
		v.rememberCurrentToolPanelOffset()
		v.expandToolPanelFor("Tasks")
		if !v.selectBottomTab("Tasks") {
			v.addActivity("Tasks panel is unavailable.")
			return
		}
		v.addActivity("Tasks selected.")
	})
	gitButton := widget.NewButtonWithIcon("Git", theme.ContentCopyIcon(), func() {
		v.rememberCurrentToolPanelOffset()
		v.expandToolPanelFor("Git")
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
	project := container.NewScroll(v.navigator)
	project.SetMinSize(fyne.NewSize(220, 240))
	tabs := container.NewAppTabs(
		v.bottomTabGroup("Workbench", theme.HomeIcon(),
			container.NewTabItemWithIcon("Project", theme.HomeIcon(), project),
			container.NewTabItemWithIcon("Activity", theme.HistoryIcon(), activity),
			container.NewTabItemWithIcon("Search", theme.SearchIcon(), v.newSearchPanel()),
			container.NewTabItemWithIcon("Problems", theme.WarningIcon(), v.newProblemsPanel()),
			container.NewTabItemWithIcon("Git", theme.ContentCopyIcon(), v.newGitPanel()),
			container.NewTabItemWithIcon("Tasks", theme.MediaPlayIcon(), v.newTasksPanel()),
			container.NewTabItemWithIcon("Jobs", theme.ListIcon(), v.newJobsPanel()),
			container.NewTabItemWithIcon("Rollbacks", theme.ContentUndoIcon(), v.newRollbackPanel()),
		),
		v.bottomTabGroup("Data Studio", theme.StorageIcon(),
			container.NewTabItemWithIcon("Data", theme.StorageIcon(), v.newDataPanel()),
			container.NewTabItemWithIcon("Operations", theme.ComputerIcon(), v.newOperationsPanel()),
			container.NewTabItemWithIcon("Artifacts", theme.DocumentIcon(), v.newArtifactsPanel()),
		),
		v.bottomTabGroup("Knowledge", theme.InfoIcon(),
			container.NewTabItemWithIcon("History", theme.InfoIcon(), v.newHistoryPanel()),
			container.NewTabItemWithIcon("Chat", theme.MailComposeIcon(), v.newChatHistoryPanel()),
			container.NewTabItemWithIcon("Agent Audit", theme.InfoIcon(), v.newAgentAuditPanel()),
		),
		v.bottomTabGroup("System", theme.VisibilityIcon(),
			container.NewTabItemWithIcon("Diagnostics", theme.VisibilityIcon(), v.newDiagnosticsPanel()),
			container.NewTabItemWithIcon("Approvals", theme.ConfirmIcon(), v.newApprovalsPanel()),
		),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	tabs.OnSelected = func(item *container.TabItem) {
		if item != nil {
			v.updateRailActiveStateForTab(item.Text)
		}
	}
	v.bottomTabs = tabs
	return tabs
}

func (v *View) bottomTabGroup(title string, icon fyne.Resource, items ...*container.TabItem) *container.TabItem {
	tabs := container.NewAppTabs(items...)
	tabs.SetTabLocation(container.TabLocationTop)
	tabs.OnSelected = func(item *container.TabItem) {
		if item != nil {
			v.updateRailActiveStateForTab(item.Text)
		}
	}
	return container.NewTabItemWithIcon(title, icon, tabs)
}
