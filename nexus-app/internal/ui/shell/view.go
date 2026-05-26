package shell

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

type View struct {
	window           fyne.Window
	state            *State
	workspaceService *workspaceSvc.Service
	status           *widget.Label
	navigator        *fyne.Container
	editorTabs       *container.DocTabs
	openTabs         map[string]*container.TabItem
	activityLog      *widget.RichText
	activityText     string
}

func New(window fyne.Window) *View {
	view := &View{
		window:           window,
		state:            NewState(),
		workspaceService: workspaceSvc.New(),
		status:           widget.NewLabel("No workspace open"),
		navigator:        container.NewStack(widget.NewLabel("Open a workspace to browse files.")),
		editorTabs:       newEditorTabs(),
		openTabs:         map[string]*container.TabItem{},
		activityLog:      widget.NewRichTextFromMarkdown("Ready."),
		activityText:     "Ready.",
	}
	view.configureEditorTabs()
	return view
}

func (v *View) Canvas() fyne.CanvasObject {
	rail := v.newRail()
	workbench := container.NewBorder(v.newToolbar(), v.newBottomPanel(), v.navigator, v.newAssistantPanel(), v.editorTabs)
	return container.NewBorder(nil, v.status, rail, nil, workbench)
}

func (v *View) newRail() fyne.CanvasObject {
	logo := widget.NewLabelWithStyle("Nexus", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
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

func (v *View) openWorkspaceDialog() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		if uri == nil {
			return
		}
		v.openWorkspace(uri.Path())
	}, v.window)
}

func (v *View) refreshWorkspace() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("No workspace to refresh.")
		return
	}
	v.openWorkspace(workspace.Root)
}

func (v *View) openWorkspace(root string) {
	workspace, err := v.workspaceService.Open(root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.state.SetWorkspace(workspace)
	v.navigator.Objects = []fyne.CanvasObject{newWorkspaceTree(v.state, v.workspaceService, v.openWorkspaceNode)}
	v.navigator.Refresh()
	v.status.SetText(fmt.Sprintf("%s: %d indexed, %d ignored, %d unreadable", workspace.Name, workspace.Summary.Included, workspace.Summary.Ignored, workspace.Summary.Unreadable))
	v.addActivity("Opened workspace " + workspace.Root)
}

func (v *View) addActivity(message string) {
	v.activityText += "\n\n" + message
	v.activityLog.ParseMarkdown(v.activityText)
}

func welcomePanel() fyne.CanvasObject {
	return container.NewCenter(widget.NewRichTextFromMarkdown("# Nexus Augentic Studio\n\nFyne-native migration shell. Open a workspace to begin."))
}

func (v *View) openWorkspaceNode(node domain.WorkspaceNode) {
	if node.Kind == domain.NodeDirectory {
		v.addActivity("Selected folder " + node.RelPath)
		return
	}
	workspace := v.state.Workspace()
	preview, err := v.workspaceService.PreviewFile(workspace.Root, node.RelPath)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.openPreviewTab(preview)
	v.addActivity("Opened " + node.RelPath)
}
