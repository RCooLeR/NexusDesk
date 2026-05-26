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
	settingsButton := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		v.addPlaceholderTab("Settings", "Provider, access policy, model, and connector settings will live here.")
	})
	return container.NewVBox(logo, widget.NewSeparator(), workspaceButton, dataButton, artifactsButton, layout.NewSpacer(), settingsButton)
}

func (v *View) newToolbar() fyne.CanvasObject {
	openButton := widget.NewButtonWithIcon("Open Workspace", theme.FolderOpenIcon(), v.openWorkspaceDialog)
	refreshButton := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), v.refreshWorkspace)
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search workspace")
	return container.NewBorder(nil, nil, container.NewHBox(openButton, refreshButton), nil, searchEntry)
}

func (v *View) newAssistantPanel() fyne.CanvasObject {
	prompt := widget.NewMultiLineEntry()
	prompt.SetPlaceHolder("Ask Nexus about this workspace")
	prompt.Wrapping = fyne.TextWrapWord
	response := widget.NewRichTextFromMarkdown("Assistant output will stream here once the LLM service is ported.")
	mode := widget.NewSelect([]string{"Ask", "Agent"}, func(string) {})
	mode.SetSelected("Ask")
	send := widget.NewButtonWithIcon("", theme.MailSendIcon(), func() {
		v.addActivity("Assistant request queued for future LLM port.")
	})
	composer := container.NewBorder(nil, nil, mode, send, prompt)
	card := widget.NewCard("Assistant", "Local-first context and tool mediation", container.NewBorder(nil, composer, nil, nil, response))
	return container.NewPadded(card)
}

func (v *View) newBottomPanel() fyne.CanvasObject {
	activity := container.NewScroll(v.activityLog)
	activity.SetMinSize(fyne.NewSize(200, 110))
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Activity", theme.HistoryIcon(), activity),
		container.NewTabItemWithIcon("Git", theme.ContentCopyIcon(), widget.NewLabel("Git diff/status service will be ported from app-wails/internal/gitservice.")),
		container.NewTabItemWithIcon("Approvals", theme.ConfirmIcon(), widget.NewLabel("Approval queue and access policy UI will live here.")),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}
