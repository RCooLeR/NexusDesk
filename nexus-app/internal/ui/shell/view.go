package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	editorSvc "nexusdesk/internal/services/editor"
	gitSvc "nexusdesk/internal/services/git"
	jobsSvc "nexusdesk/internal/services/jobs"
	settingsSvc "nexusdesk/internal/services/settings"
	tasksSvc "nexusdesk/internal/services/tasks"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

type View struct {
	window           fyne.Window
	state            *State
	workspaceService *workspaceSvc.Service
	gitService       *gitSvc.Service
	jobService       *jobsSvc.Service
	settingsStore    *settingsSvc.Store
	taskService      *tasksSvc.Service
	editorSession    *editorSvc.Session
	status           *widget.Label
	navigator        *fyne.Container
	editorTabs       *container.DocTabs
	openTabs         map[string]*container.TabItem
	tabIDs           map[*container.TabItem]string
	activityLog      *widget.RichText
	activityText     string
	searchResults    *fyne.Container
	searchStatus     *widget.Label
	problemResults   *fyne.Container
	problemStatus    *widget.Label
	gitResults       *fyne.Container
	gitStatus        *widget.Label
	gitDiffText      *widget.Entry
	gitDiffStatus    *widget.Label
	gitDiffMode      gitDiffMode
	gitLastDiff      gitSvc.FileDiff
	gitHunkStatus    *widget.Label
	gitActiveHunk    int
	taskResults      *fyne.Container
	taskStatus       *widget.Label
	taskOutput       *widget.Entry
	jobResults       *fyne.Container
	jobStatus        *widget.Label
	rollbackResults  *fyne.Container
	rollbackStatus   *widget.Label
}

func New(window fyne.Window) *View {
	editorSession := editorSvc.NewSession()
	welcome := editorSession.OpenWelcome("Welcome")
	editorTabs := newEditorTabs(welcome.Title)
	settingsStore, err := settingsSvc.NewStore()
	if err != nil {
		settingsStore = settingsSvc.NewFileStore("nexus-settings.json")
	}
	gitDiffText := widget.NewMultiLineEntry()
	gitDiffText.TextStyle = fyne.TextStyle{Monospace: true}
	gitDiffText.Wrapping = fyne.TextWrapOff
	gitDiffText.Disable()
	taskOutput := widget.NewMultiLineEntry()
	taskOutput.TextStyle = fyne.TextStyle{Monospace: true}
	taskOutput.Wrapping = fyne.TextWrapOff
	taskOutput.Disable()
	view := &View{
		window:           window,
		state:            NewState(),
		workspaceService: workspaceSvc.New(),
		gitService:       gitSvc.New(),
		jobService:       jobsSvc.New(),
		settingsStore:    settingsStore,
		taskService:      tasksSvc.New(),
		editorSession:    editorSession,
		status:           widget.NewLabel("No workspace open"),
		navigator:        container.NewStack(widget.NewLabel("Open a workspace to browse files.")),
		editorTabs:       editorTabs,
		openTabs:         map[string]*container.TabItem{welcome.ID: editorTabs.Items[0]},
		tabIDs:           map[*container.TabItem]string{editorTabs.Items[0]: welcome.ID},
		activityLog:      widget.NewRichTextFromMarkdown("Ready."),
		activityText:     "Ready.",
		searchResults:    container.NewVBox(widget.NewLabel("Search results will appear here.")),
		searchStatus:     widget.NewLabel("No search yet."),
		problemResults:   container.NewVBox(widget.NewLabel("Run a scan to inspect lightweight workspace problems.")),
		problemStatus:    widget.NewLabel("No problem scan yet."),
		gitResults:       container.NewVBox(widget.NewLabel("Press Refresh git to inspect repository status.")),
		gitStatus:        widget.NewLabel("Git status has not been loaded."),
		gitDiffText:      gitDiffText,
		gitDiffStatus:    widget.NewLabel("Select a changed file to load a read-only diff."),
		gitDiffMode:      gitDiffModeUnified,
		gitHunkStatus:    widget.NewLabel("No hunk selected."),
		taskResults:      container.NewVBox(widget.NewLabel("Discover workspace tasks to run tests, scripts, or Compose checks.")),
		taskStatus:       widget.NewLabel("No tasks discovered."),
		taskOutput:       taskOutput,
		jobResults:       container.NewVBox(widget.NewLabel("Run a task to create a job record.")),
		jobStatus:        widget.NewLabel("No jobs yet."),
		rollbackResults:  container.NewVBox(widget.NewLabel("Refresh rollback records to inspect undo points.")),
		rollbackStatus:   widget.NewLabel("Rollback records have not been loaded."),
	}
	view.configureEditorTabs()
	return view
}

func (v *View) Canvas() fyne.CanvasObject {
	rail := v.newRail()
	workbench := container.NewBorder(v.newToolbar(), v.newBottomPanel(), v.navigator, v.newAssistantPanel(), v.editorTabs)
	return container.NewBorder(nil, v.status, rail, nil, workbench)
}
