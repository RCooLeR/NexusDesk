package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	agentSvc "nexusdesk/internal/services/agent"
	approvalsSvc "nexusdesk/internal/services/approvals"
	assistantSvc "nexusdesk/internal/services/assistant"
	editorSvc "nexusdesk/internal/services/editor"
	gitSvc "nexusdesk/internal/services/git"
	jobsSvc "nexusdesk/internal/services/jobs"
	llmSvc "nexusdesk/internal/services/llm"
	metadataSvc "nexusdesk/internal/services/metadata"
	settingsSvc "nexusdesk/internal/services/settings"
	tasksSvc "nexusdesk/internal/services/tasks"
	toolsSvc "nexusdesk/internal/services/tools"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

type View struct {
	window                 fyne.Window
	state                  *State
	workspaceService       *workspaceSvc.Service
	gitService             *gitSvc.Service
	jobService             *jobsSvc.Service
	approvalService        *approvalsSvc.Service
	assistantService       *assistantSvc.Service
	agentService           *agentSvc.Service
	metadataStore          *metadataSvc.Store
	settingsStore          *settingsSvc.Store
	taskService            *tasksSvc.Service
	editorSession          *editorSvc.Session
	status                 *widget.Label
	navigator              *fyne.Container
	editorTabs             *container.DocTabs
	openTabs               map[string]*container.TabItem
	tabIDs                 map[*container.TabItem]string
	activityLog            *widget.RichText
	activityText           string
	searchResults          *fyne.Container
	searchStatus           *widget.Label
	problemResults         *fyne.Container
	problemStatus          *widget.Label
	gitResults             *fyne.Container
	gitStatus              *widget.Label
	gitDiffText            *widget.Entry
	gitDiffStatus          *widget.Label
	gitDiffMode            gitDiffMode
	gitLastDiff            gitSvc.FileDiff
	gitHunkStatus          *widget.Label
	gitActiveHunk          int
	taskResults            *fyne.Container
	taskStatus             *widget.Label
	taskOutput             *widget.Entry
	jobResults             *fyne.Container
	jobStatus              *widget.Label
	rollbackResults        *fyne.Container
	rollbackStatus         *widget.Label
	artifactResults        *fyne.Container
	artifactStatus         *widget.Label
	artifactPreview        *widget.Entry
	chatHistoryResults     *fyne.Container
	chatHistoryStatus      *widget.Label
	chatHistoryDetail      *widget.Entry
	agentAuditResults      *fyne.Container
	agentAuditStatus       *widget.Label
	agentAuditDetail       *widget.Entry
	approvalResults        *fyne.Container
	approvalStatus         *widget.Label
	accessStatus           *widget.Label
	assistantContextStatus *widget.Label
	assistantContextList   *fyne.Container
	assistantHistoryStatus *widget.Label
	assistantHistoryList   *fyne.Container
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
	artifactPreview := widget.NewMultiLineEntry()
	artifactPreview.TextStyle = fyne.TextStyle{Monospace: true}
	artifactPreview.Wrapping = fyne.TextWrapWord
	artifactPreview.Disable()
	chatHistoryDetail := widget.NewMultiLineEntry()
	chatHistoryDetail.TextStyle = fyne.TextStyle{Monospace: true}
	chatHistoryDetail.Wrapping = fyne.TextWrapWord
	chatHistoryDetail.Disable()
	agentAuditDetail := widget.NewMultiLineEntry()
	agentAuditDetail.TextStyle = fyne.TextStyle{Monospace: true}
	agentAuditDetail.Wrapping = fyne.TextWrapWord
	agentAuditDetail.Disable()
	workspaceService := workspaceSvc.New()
	llmClient := llmSvc.NewClient()
	assistantService := assistantSvc.New(settingsStore, workspaceService, llmClient)
	gitService := gitSvc.New()
	taskService := tasksSvc.New()
	toolDispatcher := toolsSvc.NewDefaultDispatcher(toolsSvc.Dependencies{
		Workspace: workspaceService,
		Git:       gitService,
		Tasks:     taskService,
	})
	agentService := agentSvc.New(settingsStore, llmClient, toolDispatcher)
	view := &View{
		window:           window,
		state:            NewState(),
		workspaceService: workspaceService,
		gitService:       gitService,
		jobService:       jobsSvc.New(),
		approvalService:  approvalsSvc.New(),
		assistantService: assistantService,
		agentService:     agentService,
		settingsStore:    settingsStore,
		taskService:      taskService,
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
		artifactResults:  container.NewVBox(widget.NewLabel("Refresh artifacts to inspect generated task reports.")),
		artifactStatus:   widget.NewLabel("Artifacts have not been loaded."),
		artifactPreview:  artifactPreview,
		chatHistoryResults: container.NewVBox(
			widget.NewLabel("Open a workspace to search persisted chat messages."),
		),
		chatHistoryStatus: widget.NewLabel("Chat history has not been loaded."),
		chatHistoryDetail: chatHistoryDetail,
		agentAuditResults: container.NewVBox(
			widget.NewLabel("Open a workspace to inspect persisted agent runs."),
		),
		agentAuditStatus: widget.NewLabel("Agent audit has not been loaded."),
		agentAuditDetail: agentAuditDetail,
		approvalResults:  container.NewVBox(widget.NewLabel("Open a workspace to inspect approval records.")),
		approvalStatus:   widget.NewLabel("Approval records have not been loaded."),
		accessStatus:     widget.NewLabel("Full project access: inactive"),
	}
	view.configureEditorTabs()
	return view
}

func (v *View) Canvas() fyne.CanvasObject {
	rail := v.newRail()
	workbench := container.NewBorder(v.newToolbar(), v.newBottomPanel(), v.navigator, v.newAssistantPanel(), v.editorTabs)
	return container.NewBorder(nil, v.status, rail, nil, workbench)
}
