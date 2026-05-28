package shell

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	agentSvc "nexusdesk/internal/services/agent"
	approvalsSvc "nexusdesk/internal/services/approvals"
	artifactsSvc "nexusdesk/internal/services/artifacts"
	assistantSvc "nexusdesk/internal/services/assistant"
	datasetsSvc "nexusdesk/internal/services/datasets"
	dbconnectorSvc "nexusdesk/internal/services/dbconnector"
	editorSvc "nexusdesk/internal/services/editor"
	gitSvc "nexusdesk/internal/services/git"
	historySvc "nexusdesk/internal/services/history"
	jobsSvc "nexusdesk/internal/services/jobs"
	llmSvc "nexusdesk/internal/services/llm"
	metadataSvc "nexusdesk/internal/services/metadata"
	operationsSvc "nexusdesk/internal/services/operations"
	recentWorkspacesSvc "nexusdesk/internal/services/recentworkspaces"
	settingsSvc "nexusdesk/internal/services/settings"
	tasksSvc "nexusdesk/internal/services/tasks"
	toolsSvc "nexusdesk/internal/services/tools"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

type View struct {
	window                   fyne.Window
	state                    *State
	workspaceService         *workspaceSvc.Service
	gitService               *gitSvc.Service
	jobService               *jobsSvc.Service
	approvalService          *approvalsSvc.Service
	assistantService         *assistantSvc.Service
	agentService             *agentSvc.Service
	datasetService           *datasetsSvc.Service
	dbconnectorService       *dbconnectorSvc.Service
	connectorProfileStore    *dbconnectorSvc.ConnectorProfileStore
	operationsService        *operationsSvc.Service
	metadataStore            *metadataSvc.Store
	recentWorkspaceStore     *recentWorkspacesSvc.Store
	settingsStore            *settingsSvc.Store
	taskService              *tasksSvc.Service
	editorSession            *editorSvc.Session
	status                   *widget.Label
	navigator                *fyne.Container
	navigatorTree            *widget.Tree
	navigatorStore           *treeStore
	navigatorRefreshSummary  func()
	editorTabs               *container.DocTabs
	bottomTabs               *container.AppTabs
	openTabs                 map[string]*container.TabItem
	tabIDs                   map[*container.TabItem]string
	editorPreviews           map[string]domain.FilePreview
	textEditors              map[string]*textEditorBinding
	navigatorClipboard       navigatorClipboard
	activityLog              *widget.RichText
	activityText             string
	activityLines            []string
	searchResults            *fyne.Container
	searchStatus             *widget.Label
	problemResults           *fyne.Container
	problemStatus            *widget.Label
	dataProfileStatus        *widget.Label
	dataProfileDetail        *widget.Entry
	dataRowsDetail           *widget.Entry
	dataRowsContainer        *fyne.Container
	dataRowsTable            *widget.Table
	dataRowsColumnWidths     []float32
	dataRowsRenderPolicy     dataGridRenderPolicy
	dataRowsColumns          []string
	dataRowsValues           [][]string
	dataRowsSelectedRow      int
	dataRowsSelectedCol      int
	dataPlanDetail           *widget.Entry
	dataChartDetail          *widget.Entry
	dataResultTabs           *container.AppTabs
	dataQueryEntry           *widget.Entry
	dataLastQuery            datasetsSvc.QueryResult
	dataLastSQLiteQuery      dbconnectorSvc.SQLiteQueryResult
	dataLastConnectorQuery   dbconnectorSvc.ConnectorQueryResult
	dataLastChart            datasetsSvc.ChartResult
	dataLastDashboard        datasetsSvc.DashboardResult
	dataLastNotebookRun      datasetsSvc.NotebookRunResult
	dataSQLiteQueryMu        sync.Mutex
	dataSQLiteCancel         func()
	dataSQLiteQueryID        string
	dataConnectorQueryMu     sync.Mutex
	dataConnectorCancel      func()
	dataConnectorQueryID     string
	dataNotebookLabel        *widget.Entry
	dataNotebookCellSelect   *widget.Select
	dataNotebookCellIndex    int
	dataActiveNotebookID     string
	dataConnectorProfileID   string
	dataConnectorProfile     *widget.Select
	dataConnectorOptions     map[string]string
	compatibilityImportMu    sync.Mutex
	compatibilityImportByWS  map[string]bool
	operationsResults        *fyne.Container
	operationsStatus         *widget.Label
	operationsDetail         *widget.Entry
	gitResults               *fyne.Container
	gitStatus                *widget.Label
	gitDiffText              *widget.Entry
	gitDiffStatus            *widget.Label
	gitDiffMode              gitDiffMode
	gitLastDiff              gitSvc.FileDiff
	gitFileBadges            map[string]string
	gitHunkStatus            *widget.Label
	gitActiveHunk            int
	taskResults              *fyne.Container
	taskStatus               *widget.Label
	taskOutput               *widget.Entry
	jobResults               *fyne.Container
	jobStatus                *widget.Label
	rollbackResults          *fyne.Container
	rollbackStatus           *widget.Label
	artifactResults          *fyne.Container
	artifactStatus           *widget.Label
	artifactPreview          *widget.Entry
	artifactSourceStatus     *widget.Label
	artifactSources          *fyne.Container
	artifactIncludeArchived  bool
	artifactCompareLeft      artifactsCompareSelection
	artifactLastComparison   artifactsSvc.ArtifactComparison
	chatHistoryResults       *fyne.Container
	chatHistoryStatus        *widget.Label
	chatHistoryDetail        *widget.Entry
	historyResults           *fyne.Container
	historyStatus            *widget.Label
	historyDetail            *widget.Entry
	agentAuditResults        *fyne.Container
	agentAuditStatus         *widget.Label
	agentAuditDetail         *widget.Entry
	diagnosticsStatus        *widget.Label
	diagnosticsDetail        *widget.Entry
	approvalResults          *fyne.Container
	approvalStatus           *widget.Label
	accessStatus             *widget.Label
	assistantContextStatus   *widget.Label
	assistantContextList     *fyne.Container
	assistantHistoryStatus   *widget.Label
	assistantHistoryList     *fyne.Container
	assistantPrompt          *widget.Entry
	assistantMode            *widget.Select
	assistantRunTaskApproval *widget.Check
	diagnosticsProber        diagnosticsProber
}

type artifactsCompareSelection struct {
	RelPath string
	Kind    string
	Title   string
}

func New(window fyne.Window) *View {
	editorSession := editorSvc.NewSession()
	welcome := editorSession.OpenWelcome("Welcome")
	var view *View
	welcomeItem := container.NewTabItem(welcome.Title, widget.NewLabel("Loading home..."))
	editorTabs := newEditorTabs(welcomeItem)
	settingsStore, err := settingsSvc.NewStore()
	if err != nil {
		settingsStore = settingsSvc.NewFileStore("nexus-settings.json")
	}
	recentWorkspaceStore, err := recentWorkspacesSvc.NewStore()
	if err != nil {
		recentWorkspaceStore = recentWorkspacesSvc.NewFileStore("nexus-recent-workspaces.json")
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
	diagnosticsDetail := widget.NewMultiLineEntry()
	diagnosticsDetail.TextStyle = fyne.TextStyle{Monospace: true}
	diagnosticsDetail.Wrapping = fyne.TextWrapWord
	diagnosticsDetail.Disable()
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
	dbconnectorService := dbconnectorSvc.New()
	connectorProfileStore, err := dbconnectorSvc.NewDefaultConnectorProfileStore()
	if err != nil {
		connectorProfileStore = dbconnectorSvc.NewConnectorProfileStore("nexus-connector-profiles.json")
	}
	dataProfileDetail := widget.NewMultiLineEntry()
	dataProfileDetail.TextStyle = fyne.TextStyle{Monospace: true}
	dataProfileDetail.Wrapping = fyne.TextWrapWord
	dataProfileDetail.Disable()
	dataRowsDetail := widget.NewMultiLineEntry()
	dataRowsDetail.TextStyle = fyne.TextStyle{Monospace: true}
	dataRowsDetail.Wrapping = fyne.TextWrapOff
	dataRowsDetail.Disable()
	dataPlanDetail := widget.NewMultiLineEntry()
	dataPlanDetail.TextStyle = fyne.TextStyle{Monospace: true}
	dataPlanDetail.Wrapping = fyne.TextWrapWord
	dataPlanDetail.Disable()
	dataChartDetail := widget.NewMultiLineEntry()
	dataChartDetail.TextStyle = fyne.TextStyle{Monospace: true}
	dataChartDetail.Wrapping = fyne.TextWrapWord
	dataChartDetail.Disable()
	dataQueryEntry := widget.NewMultiLineEntry()
	dataQueryEntry.SetMinRowsVisible(2)
	dataQueryEntry.SetPlaceHolder("Search/filter, SQL, or notebook cells. Use -- cell: Label and -- chart: Label to save multiple cells.")
	operationsDetail := widget.NewMultiLineEntry()
	operationsDetail.TextStyle = fyne.TextStyle{Monospace: true}
	operationsDetail.Wrapping = fyne.TextWrapWord
	operationsDetail.Disable()
	view = &View{
		window:                window,
		state:                 NewState(),
		workspaceService:      workspaceService,
		gitService:            gitService,
		jobService:            jobsSvc.New(),
		approvalService:       approvalsSvc.New(),
		assistantService:      assistantService,
		agentService:          agentService,
		datasetService:        datasetService,
		dbconnectorService:    dbconnectorService,
		connectorProfileStore: connectorProfileStore,
		operationsService:     operationsSvc.New(),
		settingsStore:         settingsStore,
		recentWorkspaceStore:  recentWorkspaceStore,
		taskService:           taskService,
		editorSession:         editorSession,
		status:                widget.NewLabel("No workspace open"),
		navigator:             container.NewStack(widget.NewLabel("Open a workspace to browse files.")),
		editorTabs:            editorTabs,
		openTabs:              map[string]*container.TabItem{welcome.ID: editorTabs.Items[0]},
		tabIDs:                map[*container.TabItem]string{editorTabs.Items[0]: welcome.ID},
		editorPreviews:        map[string]domain.FilePreview{},
		textEditors:           map[string]*textEditorBinding{},
		activityLog:           widget.NewRichTextFromMarkdown("Ready."),
		activityText:          "Ready.",
		activityLines:         []string{"Ready."},
		searchResults:         container.NewVBox(widget.NewLabel("Search results will appear here.")),
		searchStatus:          widget.NewLabel("No search yet."),
		problemResults:        container.NewVBox(widget.NewLabel("Run a scan to inspect lightweight workspace problems.")),
		problemStatus:         widget.NewLabel("No problem scan yet."),
		dataProfileStatus: widget.NewLabel(
			"Select a CSV, TSV, or JSON file, then profile or query it.",
		),
		dataProfileDetail:       dataProfileDetail,
		dataRowsDetail:          dataRowsDetail,
		dataRowsSelectedRow:     -1,
		dataRowsSelectedCol:     -1,
		dataPlanDetail:          dataPlanDetail,
		dataChartDetail:         dataChartDetail,
		dataQueryEntry:          dataQueryEntry,
		operationsResults:       container.NewVBox(widget.NewLabel("Scan the workspace to inspect Docker, Compose, env, config, script, and log files.")),
		operationsStatus:        widget.NewLabel("Operations scan has not been run."),
		operationsDetail:        operationsDetail,
		dataConnectorOptions:    map[string]string{},
		compatibilityImportByWS: map[string]bool{},
		gitResults:              container.NewVBox(widget.NewLabel("Press Refresh git to inspect repository status.")),
		gitStatus:               widget.NewLabel("Git status has not been loaded."),
		gitDiffText:             gitDiffText,
		gitDiffStatus:           widget.NewLabel("Select a changed file to load a read-only diff."),
		gitDiffMode:             gitDiffModeUnified,
		gitFileBadges:           map[string]string{},
		gitHunkStatus:           widget.NewLabel("No hunk selected."),
		taskResults:             container.NewVBox(widget.NewLabel("Discover workspace tasks to run tests, scripts, or Compose checks.")),
		taskStatus:              widget.NewLabel("No tasks discovered."),
		taskOutput:              taskOutput,
		jobResults:              container.NewVBox(widget.NewLabel("Run a task to create a job record.")),
		jobStatus:               widget.NewLabel("No jobs yet."),
		rollbackResults:         container.NewVBox(widget.NewLabel("Refresh rollback records to inspect undo points.")),
		rollbackStatus:          widget.NewLabel("Rollback records have not been loaded."),
		artifactResults:         container.NewVBox(widget.NewLabel("Refresh artifacts to inspect generated task reports.")),
		artifactStatus:          widget.NewLabel("Artifacts have not been loaded."),
		artifactPreview:         artifactPreview,
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
		diagnosticsStatus: widget.NewLabel(
			"Open a workspace to run diagnostics.",
		),
		diagnosticsDetail: diagnosticsDetail,
		approvalResults:   container.NewVBox(widget.NewLabel("Open a workspace to inspect approval records.")),
		approvalStatus:    widget.NewLabel("Approval records have not been loaded."),
		accessStatus:      widget.NewLabel("Full project access: inactive"),
		diagnosticsProber: llmClient,
	}
	welcomeItem.Content = view.newWelcomePanel()
	view.configureEditorTabs()
	return view
}

func (v *View) Canvas() fyne.CanvasObject {
	rail := v.newRail()
	mainSplit := container.NewHSplit(v.editorTabs, v.newAssistantPanel())
	mainSplit.SetOffset(0.82)
	workbenchTop := container.NewBorder(v.newToolbar(), nil, v.navigator, nil, mainSplit)
	workbench := container.NewVSplit(workbenchTop, v.newBottomPanel())
	workbench.SetOffset(0.68)
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
