package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	agentSvc "nexusdesk/internal/services/agent"
	approvalsSvc "nexusdesk/internal/services/approvals"
	artifactsSvc "nexusdesk/internal/services/artifacts"
	assistantSvc "nexusdesk/internal/services/assistant"
	datasetsSvc "nexusdesk/internal/services/datasets"
	editorSvc "nexusdesk/internal/services/editor"
	gitSvc "nexusdesk/internal/services/git"
	historySvc "nexusdesk/internal/services/history"
	jobsSvc "nexusdesk/internal/services/jobs"
	llmSvc "nexusdesk/internal/services/llm"
	metadataSvc "nexusdesk/internal/services/metadata"
	operationsSvc "nexusdesk/internal/services/operations"
	settingsSvc "nexusdesk/internal/services/settings"
	tasksSvc "nexusdesk/internal/services/tasks"
	toolsSvc "nexusdesk/internal/services/tools"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

type View struct {
	window                  fyne.Window
	state                   *State
	workspaceService        *workspaceSvc.Service
	gitService              *gitSvc.Service
	jobService              *jobsSvc.Service
	approvalService         *approvalsSvc.Service
	assistantService        *assistantSvc.Service
	agentService            *agentSvc.Service
	datasetService          *datasetsSvc.Service
	operationsService       *operationsSvc.Service
	metadataStore           *metadataSvc.Store
	settingsStore           *settingsSvc.Store
	taskService             *tasksSvc.Service
	editorSession           *editorSvc.Session
	status                  *widget.Label
	navigator               *fyne.Container
	editorTabs              *container.DocTabs
	openTabs                map[string]*container.TabItem
	tabIDs                  map[*container.TabItem]string
	activityLog             *widget.RichText
	activityText            string
	searchResults           *fyne.Container
	searchStatus            *widget.Label
	problemResults          *fyne.Container
	problemStatus           *widget.Label
	dataProfileStatus       *widget.Label
	dataProfileDetail       *widget.Entry
	dataQueryEntry          *widget.Entry
	dataLastQuery           datasetsSvc.QueryResult
	dataLastChart           datasetsSvc.ChartResult
	operationsResults       *fyne.Container
	operationsStatus        *widget.Label
	operationsDetail        *widget.Entry
	gitResults              *fyne.Container
	gitStatus               *widget.Label
	gitDiffText             *widget.Entry
	gitDiffStatus           *widget.Label
	gitDiffMode             gitDiffMode
	gitLastDiff             gitSvc.FileDiff
	gitFileBadges           map[string]string
	gitHunkStatus           *widget.Label
	gitActiveHunk           int
	taskResults             *fyne.Container
	taskStatus              *widget.Label
	taskOutput              *widget.Entry
	jobResults              *fyne.Container
	jobStatus               *widget.Label
	rollbackResults         *fyne.Container
	rollbackStatus          *widget.Label
	artifactResults         *fyne.Container
	artifactStatus          *widget.Label
	artifactPreview         *widget.Entry
	artifactSourceStatus    *widget.Label
	artifactSources         *fyne.Container
	artifactIncludeArchived bool
	artifactCompareLeft     artifactsCompareSelection
	artifactLastComparison  artifactsSvc.ArtifactComparison
	chatHistoryResults      *fyne.Container
	chatHistoryStatus       *widget.Label
	chatHistoryDetail       *widget.Entry
	historyResults          *fyne.Container
	historyStatus           *widget.Label
	historyDetail           *widget.Entry
	agentAuditResults       *fyne.Container
	agentAuditStatus        *widget.Label
	agentAuditDetail        *widget.Entry
	approvalResults         *fyne.Container
	approvalStatus          *widget.Label
	accessStatus            *widget.Label
	assistantContextStatus  *widget.Label
	assistantContextList    *fyne.Container
	assistantHistoryStatus  *widget.Label
	assistantHistoryList    *fyne.Container
	assistantPrompt         *widget.Entry
	assistantMode           *widget.Select
}

type artifactsCompareSelection struct {
	RelPath string
	Kind    string
	Title   string
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
	historyDetail := widget.NewMultiLineEntry()
	historyDetail.TextStyle = fyne.TextStyle{Monospace: true}
	historyDetail.Wrapping = fyne.TextWrapWord
	historyDetail.Disable()
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
	datasetService := datasetsSvc.New(workspaceService)
	dataProfileDetail := widget.NewMultiLineEntry()
	dataProfileDetail.TextStyle = fyne.TextStyle{Monospace: true}
	dataProfileDetail.Wrapping = fyne.TextWrapWord
	dataProfileDetail.Disable()
	dataQueryEntry := widget.NewEntry()
	dataQueryEntry.SetPlaceHolder("Search, e.g. channel=paid order by spend desc limit 20")
	operationsDetail := widget.NewMultiLineEntry()
	operationsDetail.TextStyle = fyne.TextStyle{Monospace: true}
	operationsDetail.Wrapping = fyne.TextWrapWord
	operationsDetail.Disable()
	view := &View{
		window:            window,
		state:             NewState(),
		workspaceService:  workspaceService,
		gitService:        gitService,
		jobService:        jobsSvc.New(),
		approvalService:   approvalsSvc.New(),
		assistantService:  assistantService,
		agentService:      agentService,
		datasetService:    datasetService,
		operationsService: operationsSvc.New(),
		settingsStore:     settingsStore,
		taskService:       taskService,
		editorSession:     editorSession,
		status:            widget.NewLabel("No workspace open"),
		navigator:         container.NewStack(widget.NewLabel("Open a workspace to browse files.")),
		editorTabs:        editorTabs,
		openTabs:          map[string]*container.TabItem{welcome.ID: editorTabs.Items[0]},
		tabIDs:            map[*container.TabItem]string{editorTabs.Items[0]: welcome.ID},
		activityLog:       widget.NewRichTextFromMarkdown("Ready."),
		activityText:      "Ready.",
		searchResults:     container.NewVBox(widget.NewLabel("Search results will appear here.")),
		searchStatus:      widget.NewLabel("No search yet."),
		problemResults:    container.NewVBox(widget.NewLabel("Run a scan to inspect lightweight workspace problems.")),
		problemStatus:     widget.NewLabel("No problem scan yet."),
		dataProfileStatus: widget.NewLabel(
			"Select a CSV, TSV, or JSON file, then profile or query it.",
		),
		dataProfileDetail: dataProfileDetail,
		dataQueryEntry:    dataQueryEntry,
		operationsResults: container.NewVBox(widget.NewLabel("Scan the workspace to inspect Docker, Compose, env, config, script, and log files.")),
		operationsStatus:  widget.NewLabel("Operations scan has not been run."),
		operationsDetail:  operationsDetail,
		gitResults:        container.NewVBox(widget.NewLabel("Press Refresh git to inspect repository status.")),
		gitStatus:         widget.NewLabel("Git status has not been loaded."),
		gitDiffText:       gitDiffText,
		gitDiffStatus:     widget.NewLabel("Select a changed file to load a read-only diff."),
		gitDiffMode:       gitDiffModeUnified,
		gitFileBadges:     map[string]string{},
		gitHunkStatus:     widget.NewLabel("No hunk selected."),
		taskResults:       container.NewVBox(widget.NewLabel("Discover workspace tasks to run tests, scripts, or Compose checks.")),
		taskStatus:        widget.NewLabel("No tasks discovered."),
		taskOutput:        taskOutput,
		jobResults:        container.NewVBox(widget.NewLabel("Run a task to create a job record.")),
		jobStatus:         widget.NewLabel("No jobs yet."),
		rollbackResults:   container.NewVBox(widget.NewLabel("Refresh rollback records to inspect undo points.")),
		rollbackStatus:    widget.NewLabel("Rollback records have not been loaded."),
		artifactResults:   container.NewVBox(widget.NewLabel("Refresh artifacts to inspect generated task reports.")),
		artifactStatus:    widget.NewLabel("Artifacts have not been loaded."),
		artifactPreview:   artifactPreview,
		artifactSourceStatus: widget.NewLabel(
			"Artifact sources have not been loaded.",
		),
		artifactSources: container.NewVBox(widget.NewLabel("Preview an artifact to inspect cited sources.")),
		chatHistoryResults: container.NewVBox(
			widget.NewLabel("Open a workspace to search persisted chat messages."),
		),
		chatHistoryStatus: widget.NewLabel("Chat history has not been loaded."),
		chatHistoryDetail: chatHistoryDetail,
		historyResults: container.NewVBox(
			widget.NewLabel("Open a workspace to inspect unified history."),
		),
		historyStatus: widget.NewLabel("History has not been loaded."),
		historyDetail: historyDetail,
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

func (v *View) historyService() (*historySvc.Service, error) {
	workspace := v.state.Workspace()
	if workspace.Root == "" || v.metadataStore == nil {
		return historySvc.New(v.metadataStore, nil), nil
	}
	artifactStore, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		return nil, err
	}
	return historySvc.New(v.metadataStore, artifactStore), nil
}
