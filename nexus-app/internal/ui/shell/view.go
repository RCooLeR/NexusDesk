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
	perfSvc "nexusdesk/internal/services/perf"
	recentWorkspacesSvc "nexusdesk/internal/services/recentworkspaces"
	settingsSvc "nexusdesk/internal/services/settings"
	startupSvc "nexusdesk/internal/services/startup"
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
	assistantProfileStore    *assistantSvc.ProfileStore
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
	gitStatusSnapshot        gitSvc.Status
	toolbarWorkspaceStatus   *widget.Label
	toolbarBranchStatus      *widget.Label
	toolbarProviderStatus    *widget.Label
	leftRailButtons          map[string]*widget.Button
	rightRailButtons         map[string]*widget.Button
	activeLeftRailTool       string
	activeRightRailTool      string
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
	editorSplitEnabled       bool
	editorSecondaryRelPath   string
	navigatorClipboard       navigatorClipboard
	activityLog              *widget.RichText
	activityText             string
	activityLines            []string
	search                   *searchController
	problemResults           *fyne.Container
	problemStatus            *widget.Label
	dataProfileStatus        *widget.Label
	dataProfileDetail        *widget.Entry
	dataRowsDetail           *widget.Entry
	dataRowsStatus           *widget.Label
	dataRowsContainer        *fyne.Container
	dataRowsTable            *widget.Table
	dataRowsColumnWidths     []float32
	dataRowsRenderPolicy     dataGridRenderPolicy
	dataRowsColumns          []string
	dataRowsValues           [][]string
	dataRowsSelectedRow      int
	dataRowsSelectedCol      int
	dataRowsSampledRows      int
	dataRowsOriginalRows     int
	dataRowsClippedColumns   int
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
	jobs                     *jobsController
	rollbacks                *rollbackController
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
	diagnostics              *diagnosticsController
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
	assistantProfile         assistantSvc.Profile
	assistantProfileSelect   *widget.Select
	assistantModelRoute      *widget.Select
	assistantRunStatus       *widget.Label
	assistantMemory          *widget.Entry
	assistantLastPrompt      string
	assistantLastResult      assistantSvc.Result
	diagnosticsProber        diagnosticsProber
	startupStatus            startupSvc.Status
	performanceRecorder      *perfSvc.Recorder
}

type artifactsCompareSelection struct {
	RelPath string
	Kind    string
	Title   string
}

func New(window fyne.Window) *View {
	return NewWithStartupStatus(window, startupSvc.Status{})
}

func NewWithStartupStatus(window fyne.Window, startupStatus startupSvc.Status) *View {
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
	workspaceService := workspaceSvc.New()
	llmClient := llmSvc.NewClient()
	assistantService := assistantSvc.New(settingsStore, workspaceService, llmClient)
	assistantProfileStore, err := assistantSvc.NewDefaultProfileStore()
	if err != nil {
		assistantProfileStore = assistantSvc.NewProfileStore("nexus-assistant-profile.json")
	}
	assistantService.SetProfileStore(assistantProfileStore)
	gitService := gitSvc.New()
	taskService := tasksSvc.New()
	jobService := jobsSvc.New()
	toolDispatcher := toolsSvc.NewDefaultDispatcher(toolsSvc.Dependencies{
		Workspace: workspaceService,
		Git:       gitService,
		Tasks:     taskService,
		Jobs:      jobService,
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
	dataRowsStatus := widget.NewLabel("Rows: run a query to load grid results.")
	dataRowsStatus.Wrapping = fyne.TextWrapWord
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
		jobService:            jobService,
		approvalService:       approvalsSvc.New(),
		assistantService:      assistantService,
		assistantProfileStore: assistantProfileStore,
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
		problemResults:        container.NewVBox(widget.NewLabel("Run a scan to inspect lightweight workspace problems.")),
		problemStatus:         widget.NewLabel("No problem scan yet."),
		dataProfileStatus: widget.NewLabel(
			"Select a CSV, TSV, or JSON file, then profile or query it.",
		),
		dataProfileDetail:       dataProfileDetail,
		dataRowsDetail:          dataRowsDetail,
		dataRowsStatus:          dataRowsStatus,
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
		agentAuditStatus:    widget.NewLabel("Agent audit has not been loaded."),
		agentAuditDetail:    agentAuditDetail,
		approvalResults:     container.NewVBox(widget.NewLabel("Open a workspace to inspect approval records.")),
		approvalStatus:      widget.NewLabel("Approval records have not been loaded."),
		accessStatus:        widget.NewLabel("Full project access: inactive"),
		diagnosticsProber:   llmClient,
		startupStatus:       startupStatus,
		performanceRecorder: perfSvc.NewRecorder(64),
	}
	view.search = newSearchController(view)
	view.jobs = newJobsController(view)
	view.rollbacks = newRollbackController(view)
	view.diagnostics = newDiagnosticsController(view)
	welcomeItem.Content = view.newWelcomePanel()
	view.configureEditorTabs()
	view.refreshStatusBar()
	return view
}

func (v *View) Canvas() fyne.CanvasObject {
	rail := v.newRail()
	rightWorkbench := container.NewBorder(nil, nil, nil, v.newRightRail(), v.newAssistantPanel())
	mainSplit := container.NewHSplit(v.editorTabs, rightWorkbench)
	mainSplit.SetOffset(0.82)
	workbenchTop := container.NewBorder(v.newToolbar(), nil, v.navigator, nil, mainSplit)
	workbench := container.NewVSplit(workbenchTop, v.newBottomPanel())
	workbench.SetOffset(0.68)
	return container.NewBorder(nil, v.newStatusBar(), rail, nil, workbench)
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
