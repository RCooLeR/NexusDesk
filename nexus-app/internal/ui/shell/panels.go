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
	logo.SetMinSize(fyne.NewSize(112, 34))
	workspaceButton := widget.NewButtonWithIcon("", theme.HomeIcon(), func() {
		v.addActivity("Workbench selected.")
	})
	dataButton := widget.NewButtonWithIcon("", theme.StorageIcon(), func() {
		v.addPlaceholderTab("Data & Analytics", "Database, CSV, Excel, and analysis workflows will live here.")
	})
	artifactsButton := widget.NewButtonWithIcon("", theme.DocumentIcon(), func() {
		v.addPlaceholderTab("Artifacts", "Generated reports, exports, lineage, and comparisons will live here.")
	})
	settingsButton := widget.NewButtonWithIcon("", theme.SettingsIcon(), v.openSettingsTab)
	return container.NewVBox(logo, widget.NewSeparator(), workspaceButton, dataButton, artifactsButton, layout.NewSpacer(), settingsButton)
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
	activity.SetMinSize(fyne.NewSize(200, 110))
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Activity", theme.HistoryIcon(), activity),
		container.NewTabItemWithIcon("Search", theme.SearchIcon(), v.newSearchPanel()),
		container.NewTabItemWithIcon("Problems", theme.WarningIcon(), v.newProblemsPanel()),
		container.NewTabItemWithIcon("Git", theme.ContentCopyIcon(), v.newGitPanel()),
		container.NewTabItemWithIcon("Tasks", theme.MediaPlayIcon(), v.newTasksPanel()),
		container.NewTabItemWithIcon("Jobs", theme.ListIcon(), v.newJobsPanel()),
		container.NewTabItemWithIcon("Rollbacks", theme.ContentUndoIcon(), v.newRollbackPanel()),
		container.NewTabItemWithIcon("Approvals", theme.ConfirmIcon(), v.newApprovalsPanel()),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}
