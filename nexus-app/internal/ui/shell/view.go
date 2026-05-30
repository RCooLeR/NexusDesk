package shell

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

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
	window                  fyne.Window
	state                   *State
	workspaceService        *workspaceSvc.Service
	gitService              *gitSvc.Service
	jobService              *jobsSvc.Service
	approvalService         *approvalsSvc.Service
	assistantService        *assistantSvc.Service
	assistantProfileStore   *assistantSvc.ProfileStore
	agentService            *agentSvc.Service
	datasetService          *datasetsSvc.Service
	dbconnectorService      *dbconnectorSvc.Service
	connectorProfileStore   *dbconnectorSvc.ConnectorProfileStore
	operationsService       *operationsSvc.Service
	metadataStore           *metadataSvc.Store
	recentWorkspaceStore    *recentWorkspacesSvc.Store
	settingsStore           *settingsSvc.Store
	settings                *settingsController
	taskService             *tasksSvc.Service
	editorSession           *editorSvc.Session
	events                  *shellEventBus
	status                  *widget.Label
	gitStatusSnapshot       gitSvc.Status
	toolbarWorkspaceStatus  *widget.Label
	toolbarBranchStatus     *widget.Label
	toolbarProviderStatus   *widget.Label
	leftRailButtons         map[string]*railToolButton
	rightRailButtons        map[string]*railToolButton
	activeLeftRailTool      string
	activeRightRailTool     string
	activeToolPanelKey      string
	toolPanelOffsetByTool   map[string]float64
	railStateByWorkspace    map[string]railWorkspaceState
	navigator               *fyne.Container
	navigatorTree           *widget.Tree
	navigatorStore          *treeStore
	navigatorRefreshSummary func()
	editor                  *editorController
	bottomTabs              *container.AppTabs
	navigatorClipboard      navigatorClipboard
	activityLog             *widget.RichText
	activityList            *fyne.Container
	activityText            string
	activityLines           []string
	mainSplit               *container.Split
	workbenchSplit          *container.Split
	bottomPanelCollapsed    bool
	search                  *searchController
	toolPanelWidgets
	data                    *dataController
	compatibilityImportMu   sync.Mutex
	compatibilityImportByWS map[string]bool
	operations              *operationsController
	git                     *gitController
	gitFileBadges           map[string]string
	jobs                    *jobsController
	rollbacks               *rollbackController
	artifacts               *artifactsController
	agentAudit              *agentAuditController
	diagnostics             *diagnosticsController
	approvals               *approvalsController
	assistant               *assistantController
	diagnosticsProber       diagnosticsProber
	startupStatus           startupSvc.Status
	performanceRecorder     *perfSvc.Recorder
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
	settingsStore, err := settingsSvc.NewStore()
	if err != nil {
		settingsStore = settingsSvc.NewFileStore("nexus-settings.json")
	}
	recentWorkspaceStore, err := recentWorkspacesSvc.NewStore()
	if err != nil {
		recentWorkspaceStore = recentWorkspacesSvc.NewFileStore("nexus-recent-workspaces.json")
	}
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
	view = &View{
		window:                  window,
		state:                   NewState(),
		workspaceService:        workspaceService,
		gitService:              gitService,
		jobService:              jobService,
		approvalService:         approvalsSvc.New(),
		assistantService:        assistantService,
		assistantProfileStore:   assistantProfileStore,
		agentService:            agentService,
		datasetService:          datasetService,
		dbconnectorService:      dbconnectorService,
		connectorProfileStore:   connectorProfileStore,
		operationsService:       operationsSvc.New(),
		settingsStore:           settingsStore,
		recentWorkspaceStore:    recentWorkspaceStore,
		taskService:             taskService,
		editorSession:           editorSession,
		events:                  newShellEventBus(),
		status:                  widget.NewLabel("No workspace open"),
		toolPanelOffsetByTool:   map[string]float64{},
		railStateByWorkspace:    map[string]railWorkspaceState{},
		navigator:               container.NewStack(widget.NewLabel("Open a workspace to browse files.")),
		activityLog:             widget.NewRichTextFromMarkdown("Ready."),
		activityList:            newActivityList([]string{"Ready."}),
		activityText:            "Ready.",
		activityLines:           []string{"Ready."},
		toolPanelWidgets:        newToolPanelWidgets(),
		compatibilityImportByWS: map[string]bool{},
		gitFileBadges:           map[string]string{},
		diagnosticsProber:       llmClient,
		startupStatus:           startupStatus,
		performanceRecorder:     perfSvc.NewRecorder(64),
	}
	view.editor = newEditorController(view, welcome.ID, welcomeItem)
	view.search = newSearchController(view)
	view.data = newDataController(view)
	view.operations = newOperationsController(view)
	view.jobs = newJobsController(view)
	view.rollbacks = newRollbackController(view)
	view.diagnostics = newDiagnosticsController(view)
	view.git = newGitController(view)
	view.artifacts = newArtifactsController(view)
	view.assistant = newAssistantController(view)
	view.settings = newSettingsController(view)
	view.approvals = newApprovalsController(view)
	view.agentAudit = newAgentAuditController(view)
	welcomeItem.Content = view.newWelcomePanel()
	view.configureEditorTabs()
	view.refreshStatusBar()
	return view
}

func (v *View) Canvas() fyne.CanvasObject {
	rail := v.newRail()
	rightWorkbench := container.NewBorder(nil, nil, nil, v.newRightRail(), v.newAssistantPanel())
	mainSplit := v.newEditorPrioritySplit(rightWorkbench)
	workbench := container.NewBorder(v.newToolbar(), nil, nil, nil, v.newToolPanelSplit(mainSplit))
	v.expandBottomPanel()
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
