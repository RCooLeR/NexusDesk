package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"NexusAugenticStudio/internal/agenttools"
	"NexusAugenticStudio/internal/analytics"
	"NexusAugenticStudio/internal/appmeta"
	"NexusAugenticStudio/internal/approval"
	"NexusAugenticStudio/internal/artifact"
	"NexusAugenticStudio/internal/artifactsvc"
	"NexusAugenticStudio/internal/dataset"
	"NexusAugenticStudio/internal/dbconnector"
	"NexusAugenticStudio/internal/gitservice"
	"NexusAugenticStudio/internal/llm"
	"NexusAugenticStudio/internal/storage"
	"NexusAugenticStudio/internal/webfetch"
	"NexusAugenticStudio/internal/workspace"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Capability struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type WorkspaceItem struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Meta string `json:"meta"`
}

type ToolEvent struct {
	Time   string `json:"time"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type StartupState struct {
	ProductName    string          `json:"productName"`
	Tagline        string          `json:"tagline"`
	BuildStage     string          `json:"buildStage"`
	Capabilities   []Capability    `json:"capabilities"`
	WorkspaceItems []WorkspaceItem `json:"workspaceItems"`
	ToolEvents     []ToolEvent     `json:"toolEvents"`
}

type WorkspaceOpenResult struct {
	Selected bool                        `json:"selected"`
	Snapshot workspace.WorkspaceSnapshot `json:"snapshot"`
}

type ChatStreamEvent struct {
	RequestID      string   `json:"requestId"`
	Type           string   `json:"type"`
	Delta          string   `json:"delta"`
	Message        string   `json:"message"`
	Model          string   `json:"model"`
	Endpoint       string   `json:"endpoint"`
	ContextRelPath string   `json:"contextRelPath"`
	SourcePaths    []string `json:"sourcePaths"`
}

type LineageNode struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Label   string `json:"label"`
	RelPath string `json:"relPath"`
}

type LineageEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
}

type ArtifactLineage struct {
	Nodes              []LineageNode  `json:"nodes"`
	Edges              []LineageEdge  `json:"edges"`
	RelationshipCounts map[string]int `json:"relationshipCounts"`
	Message            string         `json:"message"`
}

type StaleContextRefresh struct {
	Preview        workspace.ContextPreview `json:"preview"`
	AffectedChats  int                      `json:"affectedChats"`
	StaleArtifacts []string                 `json:"staleArtifacts"`
	StaleDatasets  []string                 `json:"staleDatasets"`
	Message        string                   `json:"message"`
}

type ArtifactLineageImport struct {
	Lineage ArtifactLineage `json:"lineage"`
	Message string          `json:"message"`
}

const chatContextFallbackMaxBytes = 16 * 1024
const chatCSVContextMaxRows = 20
const chatContextFallbackMaxFiles = 32
const chatContextMinBudgetBytes = 16 * 1024
const chatContextMaxBudgetBytes = 4 * 1024 * 1024
const chatContextTokenByteEstimate = 4
const chatContextOverheadTokens = 2048
const chatContextMaxFilesCap = 256
const chatHistoryMaxMessages = 10
const chatHistoryMinBudgetBytes = 4 * 1024
const chatHistoryMaxBudgetBytes = 64 * 1024
const chatStreamEventName = "nexus:chat-stream"
const agentRunEventName = "nexus:agent-run"

var emitChatStreamEventFn = func(ctx context.Context, name string, event any) {
	runtime.EventsEmit(ctx, name, event)
}

type App struct {
	ctx                     context.Context
	llmClient               *llm.Client
	chatStore               *storage.ChatHistoryStore
	connectorStore          *storage.ConnectorProfileStore
	profileStore            *storage.AssistantProfileStore
	llmStore                *storage.LLMSettingsStore
	recentStore             *storage.RecentWorkspaceStore
	workspaceSvc            *WorkspaceService
	artifactSvc             *artifactsvc.Service
	datasetSvc              *DatasetService
	connectorQueryCancels   map[string]context.CancelFunc
	connectorQueryCancelsMu sync.Mutex
}

func NewApp() *App {
	chatStore := storage.NewDefaultChatHistoryStore()
	recentStore := storage.NewDefaultRecentWorkspaceStore()
	app := &App{
		llmClient:             llm.NewClient(),
		chatStore:             chatStore,
		connectorStore:        storage.NewDefaultConnectorProfileStore(),
		profileStore:          storage.NewDefaultAssistantProfileStore(),
		llmStore:              storage.NewDefaultLLMSettingsStore(),
		recentStore:           recentStore,
		connectorQueryCancels: map[string]context.CancelFunc{},
	}
	app.workspaceSvc = NewWorkspaceService(recentStore, chatStore, app.recordApproval)
	app.artifactSvc = artifactsvc.New(app.getWorkspaceRoot, app.mirrorMetadataStore, app.listArtifactsFromMetadata, app.persistArtifactMetadata, app.recordApproval)
	app.datasetSvc = NewDatasetService(app.getWorkspaceRoot, app.mirrorMetadataStore, app.persistArtifactMetadata, app.recordDatasetDependency, app.recordSQLRun, app.recordApproval)
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetStartupState() StartupState {
	return StartupState{
		ProductName: "Nexus Augentic Studio",
		Tagline:     "Agentic work. Augmented by context.",
		BuildStage:  "Studio MVP",
		Capabilities: []Capability{
			{
				Title:       "Project IDE",
				Description: "Open local folders, inspect files, keep tabs, and stay inside approved roots.",
				Status:      "planned",
			},
			{
				Title:       "Data & analytics studio",
				Description: "Profile datasets, query rows, summarize sources, and prepare report artifacts.",
				Status:      "planned",
			},
			{
				Title:       "Artifact workflow",
				Description: "Save reports, summaries, and file edits with provenance and visible approvals.",
				Status:      "planned",
			},
		},
		WorkspaceItems: []WorkspaceItem{
			{Name: "app", Kind: "folder", Meta: "Wails desktop studio shell"},
			{Name: "docs", Kind: "folder", Meta: "Product and studio architecture docs"},
			{Name: "services", Kind: "folder", Meta: "Development helper services"},
		},
		ToolEvents: []ToolEvent{
			{Time: "now", Title: "Studio shell ready", Detail: "React + TypeScript frontend bound to Go backend."},
			{Time: "next", Title: "Workspace indexing", Detail: "Open projects, preview files, and build context packs."},
			{Time: "then", Title: "Model and artifact flows", Detail: "Ground local models in selected context and save outputs."},
		},
	}
}

func (a *App) SelectWorkspace() (WorkspaceOpenResult, error) {
	if a.ctx == nil {
		return WorkspaceOpenResult{}, errors.New("application is not ready")
	}

	root, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open Nexus Workspace",
	})
	if err != nil {
		return WorkspaceOpenResult{}, err
	}

	if root == "" {
		return WorkspaceOpenResult{Selected: false}, nil
	}

	return a.workspaceSvc.Open(root)
}

func (a *App) OpenWorkspace(root string) (WorkspaceOpenResult, error) {
	if root == "" {
		return WorkspaceOpenResult{Selected: false}, nil
	}

	return a.workspaceSvc.Open(root)
}

func (a *App) RefreshWorkspace() (WorkspaceOpenResult, error) {
	return a.workspaceSvc.Refresh()
}

func (a *App) SearchWorkspace(query string) ([]workspace.SearchResult, error) {
	return a.workspaceSvc.Search(query)
}

func (a *App) ReadWorkspaceFile(relPath string) (workspace.FilePreview, error) {
	return a.workspaceSvc.ReadFile(relPath)
}

func (a *App) PreviewFileWrite(request workspace.FileWriteRequest) (workspace.FileWriteProposal, error) {
	return a.workspaceSvc.PreviewFileWrite(request)
}

func (a *App) ApplyFileWrite(request workspace.FileWriteRequest) (workspace.FileWriteProposal, error) {
	return a.workspaceSvc.ApplyFileWrite(request)
}

func (a *App) PreviewFileDelete(relPath string) (workspace.FileDeleteProposal, error) {
	return a.workspaceSvc.PreviewFileDelete(relPath)
}

func (a *App) ApplyFileDelete(relPath string) (workspace.FileDeleteProposal, error) {
	return a.workspaceSvc.ApplyFileDelete(relPath)
}

func (a *App) PreviewFileMove(request workspace.FileMoveRequest) (workspace.FileMoveProposal, error) {
	return a.workspaceSvc.PreviewFileMove(request)
}

func (a *App) ApplyFileMove(request workspace.FileMoveRequest) (workspace.FileMoveProposal, error) {
	return a.workspaceSvc.ApplyFileMove(request)
}

func (a *App) PreviewFileCopy(request workspace.FileCopyRequest) (workspace.FileCopyProposal, error) {
	return a.workspaceSvc.PreviewFileCopy(request)
}

func (a *App) ApplyFileCopy(request workspace.FileCopyRequest) (workspace.FileCopyProposal, error) {
	return a.workspaceSvc.ApplyFileCopy(request)
}

func (a *App) ListWorkspaceRollbacks() ([]workspace.RollbackRecord, error) {
	return a.workspaceSvc.ListRollbacks()
}

func (a *App) ApplyWorkspaceRollback(id string) (workspace.RollbackApplyResult, error) {
	return a.workspaceSvc.ApplyRollback(id)
}

func (a *App) CreateMarkdownReport(relPath string) (artifact.MarkdownReport, error) {
	return a.artifactSvc.CreateMarkdownReport(relPath)
}

func (a *App) CreateScanReportArtifact() (artifact.MarkdownReport, error) {
	return a.artifactSvc.CreateScanReport()
}

func (a *App) CreateChatMarkdownArtifact(request artifact.MarkdownArtifactRequest) (artifact.MarkdownReport, error) {
	return a.artifactSvc.CreateGeneratedMarkdown(request)
}

func (a *App) ListArtifacts() ([]artifact.WorkspaceArtifact, error) {
	return a.artifactSvc.List()
}

func (a *App) GetArtifactMetadata(relPath string) (artifact.ArtifactMetadata, error) {
	return a.artifactSvc.Metadata(relPath)
}

func (a *App) ArchiveArtifact(relPath string) (artifact.MarkdownReport, error) {
	return a.artifactSvc.Archive(relPath)
}

func (a *App) DeleteArtifact(relPath string) (artifact.MarkdownReport, error) {
	return a.artifactSvc.Delete(relPath)
}

func (a *App) CompareArtifacts(leftRelPath string, rightRelPath string) (artifact.ArtifactComparison, error) {
	return a.artifactSvc.Compare(leftRelPath, rightRelPath)
}

func (a *App) ProfileDataset(relPath string) (dataset.Profile, error) {
	return a.datasetSvc.Profile(relPath)
}

func (a *App) ListDatasetProfiles() ([]dataset.Profile, error) {
	return a.datasetSvc.ListProfiles()
}

func (a *App) QueryDataset(relPath string, query string) (workspace.DatasetQueryResult, error) {
	return a.datasetSvc.Query(relPath, query)
}

func (a *App) QueryDatasetSQL(request analytics.SQLQueryRequest) (analytics.SQLQueryResult, error) {
	return a.datasetSvc.QuerySQL(request)
}

func (a *App) ListAgentTools() []agenttools.Descriptor {
	return agenttools.Registry()
}

func (a *App) PreviewAgentTool(request agenttools.RunRequest) (agenttools.RunRecord, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return agenttools.RunRecord{}, errors.New("open a workspace before planning tools")
	}
	record, err := a.runAgentTool(root, request, "dry-run")
	if appendErr := a.appendToolRun(root, record); appendErr != nil && err == nil {
		err = appendErr
	}
	return record, err
}

func (a *App) ExecuteAgentTool(request agenttools.RunRequest) (agenttools.RunRecord, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return agenttools.RunRecord{}, errors.New("open a workspace before executing tools")
	}
	record, err := a.runAgentTool(root, request, "execute")
	if appendErr := a.appendToolRun(root, record); appendErr != nil && err == nil {
		err = appendErr
	}
	return record, err
}

func (a *App) ListAgentToolRuns() ([]agenttools.RunRecord, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return []agenttools.RunRecord{}, nil
	}
	if appmeta.Exists(root) {
		if _, err := a.mirrorMetadataStore(root, false); err == nil {
			if items, readErr := appmeta.ListToolRuns(root); readErr == nil {
				return toolRunsFromMirror(items), nil
			}
		}
	}
	return agenttools.List(root)
}

func (a *App) CheckWorkspaceFreshness() (workspace.FreshnessStatus, error) {
	return a.workspaceSvc.CheckFreshness()
}

func (a *App) EnsureSQLiteMetadataStore() (appmeta.SQLiteStatus, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return appmeta.SQLiteStatus{}, errors.New("open a workspace before preparing metadata storage")
	}
	status, err := a.mirrorMetadataStore(root, true)
	if err != nil {
		return appmeta.SQLiteStatus{}, err
	}
	a.recordApproval("metadata.sqlite.prepare", ".nexusdesk/metadata", "low", status.Message)
	return status, nil
}

func (a *App) InspectMetadataStore() (appmeta.MetadataBrowser, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return appmeta.MetadataBrowser{}, errors.New("open a workspace before inspecting metadata storage")
	}
	if _, err := a.mirrorMetadataStore(root, true); err != nil {
		return appmeta.MetadataBrowser{}, err
	}
	return appmeta.Inspect(root, a.datasetViews(root))
}

func (a *App) GetArtifactLineage() (ArtifactLineage, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return ArtifactLineage{}, errors.New("open a workspace before building artifact lineage")
	}
	nodes := map[string]LineageNode{}
	edges := []LineageEdge{}
	relationshipCounts := map[string]int{}
	addNode := func(node LineageNode) {
		if node.ID == "" {
			return
		}
		if _, ok := nodes[node.ID]; !ok {
			nodes[node.ID] = node
		}
	}
	addEdge := func(from string, to string, label string) {
		if from == "" || to == "" {
			return
		}
		edges = append(edges, LineageEdge{From: from, To: to, Label: label})
		relationshipCounts[label]++
	}

	items, err := artifact.List(root)
	if err != nil {
		return ArtifactLineage{}, err
	}
	for _, item := range items {
		artifactID := "artifact:" + item.RelPath
		addNode(LineageNode{ID: artifactID, Kind: "artifact", Label: item.Name, RelPath: item.RelPath})
		metadata, err := artifact.Metadata(root, item.RelPath)
		if err != nil {
			continue
		}
		for _, sourcePath := range metadata.SourcePaths {
			sourceID := "source:" + sourcePath
			addNode(LineageNode{ID: sourceID, Kind: "source", Label: filepath.Base(sourcePath), RelPath: sourcePath})
			addEdge(sourceID, artifactID, "source")
		}
		if metadata.ContextRelPath != "" {
			contextID := "source:" + metadata.ContextRelPath
			addNode(LineageNode{ID: contextID, Kind: "source", Label: filepath.Base(metadata.ContextRelPath), RelPath: metadata.ContextRelPath})
			addEdge(contextID, artifactID, "context")
		}
		if metadata.Prompt != "" {
			promptID := "chat:" + item.RelPath
			addNode(LineageNode{ID: promptID, Kind: "chat", Label: "Prompt", RelPath: metadata.ContextRelPath})
			addEdge(promptID, artifactID, "generated")
		}
	}

	toolRuns, _ := agenttools.List(root)
	for _, run := range toolRuns {
		runID := "tool:" + run.ID
		addNode(LineageNode{ID: runID, Kind: "tool", Label: run.Title, RelPath: run.Target})
		if run.Target != "" {
			targetID := "source:" + run.Target
			if isArtifactRelPath(run.Target) {
				targetID = "artifact:" + run.Target
			}
			addNode(LineageNode{ID: targetID, Kind: targetKind(run.Target), Label: filepath.Base(run.Target), RelPath: run.Target})
			addEdge(targetID, runID, run.Mode)
		}
	}

	chats, _ := a.GetChatHistory()
	for index, message := range chats {
		if message.Role != "assistant" || len(message.SourcePaths) == 0 {
			continue
		}
		chatID := fmt.Sprintf("chat:assistant:%d", index)
		addNode(LineageNode{ID: chatID, Kind: "chat", Label: "Assistant answer", RelPath: message.ContextRelPath})
		for _, sourcePath := range message.SourcePaths {
			sourceID := "source:" + sourcePath
			addNode(LineageNode{ID: sourceID, Kind: "source", Label: filepath.Base(sourcePath), RelPath: sourcePath})
			addEdge(sourceID, chatID, "cited")
		}
	}

	nodeList := make([]LineageNode, 0, len(nodes))
	for _, node := range nodes {
		nodeList = append(nodeList, node)
	}
	return ArtifactLineage{
		Nodes:              nodeList,
		Edges:              edges,
		RelationshipCounts: relationshipCounts,
		Message:            fmt.Sprintf("%d lineage nodes and %d relationships.", len(nodeList), len(edges)),
	}, nil
}

func (a *App) ExportArtifactLineageJSON() (artifact.MarkdownReport, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return artifact.MarkdownReport{}, errors.New("open a workspace before exporting artifact lineage")
	}
	lineage, err := a.GetArtifactLineage()
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	payload, err := json.MarshalIndent(lineage, "", "  ")
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	report, err := artifact.CreateJSONArtifact(root, artifact.JSONArtifactRequest{
		Name:        "artifact-lineage",
		Title:       "Artifact Lineage Graph",
		Content:     string(payload),
		Source:      "artifact lineage",
		SourcePaths: lineageSourcePaths(lineage),
		Prompt:      "Export current Nexus artifact lineage graph.",
	}, time.Now())
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	a.persistArtifactMetadata(root, report.RelPath)
	a.recordApproval("artifact.lineage.export", report.RelPath, "low", report.Message)
	return report, nil
}

func (a *App) ImportArtifactLineageJSON(relPath string) (ArtifactLineageImport, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return ArtifactLineageImport{}, errors.New("open a workspace before importing artifact lineage")
	}
	preview, err := workspace.Preview(root, relPath, workspace.PreviewOptions{MaxBytes: 512 * 1024})
	if err != nil {
		return ArtifactLineageImport{}, err
	}
	var lineage ArtifactLineage
	if err := json.Unmarshal([]byte(preview.Content), &lineage); err != nil {
		return ArtifactLineageImport{}, err
	}
	return ArtifactLineageImport{
		Lineage: lineage,
		Message: fmt.Sprintf("Imported %d lineage nodes and %d relationships from %s.", len(lineage.Nodes), len(lineage.Edges), preview.RelPath),
	}, nil
}

func (a *App) SaveDatasetQuery(relPath string, query string, label string) (dataset.SavedQuery, error) {
	return a.datasetSvc.SaveQuery(relPath, query, label)
}

func (a *App) ListDatasetQueries(relPath string) ([]dataset.SavedQuery, error) {
	return a.datasetSvc.ListQueries(relPath)
}

func (a *App) SaveDatasetSQLQuery(relPath string, query string, label string) (dataset.SavedQuery, error) {
	return a.datasetSvc.SaveSQLQuery(relPath, query, label)
}

func (a *App) SaveSQLiteConnectorQuery(relPath string, query string, label string) (dataset.SavedQuery, error) {
	return a.datasetSvc.SaveSQLiteConnectorQuery(relPath, query, label)
}

func (a *App) ListDatasetDependencies(relPath string) ([]appmeta.DatasetDependency, error) {
	return a.datasetSvc.ListDependencies(relPath)
}

func (a *App) ListDatasetSQLRuns(relPath string) ([]appmeta.SQLRun, error) {
	return a.datasetSvc.ListSQLRuns(relPath)
}

func (a *App) SearchMetadata(query string) ([]appmeta.MetadataSearchResult, error) {
	return a.datasetSvc.SearchMetadata(query)
}

func (a *App) QueryWorkspaceSQLite(request dbconnector.SQLiteQueryRequest) (dbconnector.SQLiteQueryResult, error) {
	return a.datasetSvc.QueryWorkspaceSQLite(request)
}

func (a *App) CancelWorkspaceSQLiteQuery(requestID string) bool {
	return a.datasetSvc.CancelWorkspaceSQLiteQuery(requestID)
}

func (a *App) InspectWorkspaceSQLite(relPath string) (dbconnector.ConnectorMetadata, error) {
	return a.datasetSvc.InspectWorkspaceSQLite(relPath)
}

func (a *App) ListDatasetSQLQueries(relPath string) ([]dataset.SavedQuery, error) {
	return a.datasetSvc.ListSQLQueries(relPath)
}

func (a *App) SaveDatasetSQLNotebook(request dataset.NotebookSaveRequest) (dataset.Notebook, error) {
	return a.datasetSvc.SaveSQLNotebook(request)
}

func (a *App) ListDatasetSQLNotebooks(relPath string) ([]dataset.Notebook, error) {
	return a.datasetSvc.ListSQLNotebooks(relPath)
}

func (a *App) ListSQLiteConnectorQueries(relPath string) ([]dataset.SavedQuery, error) {
	return a.datasetSvc.ListSQLiteConnectorQueries(relPath)
}

func (a *App) PreviewDatasetChart(request workspace.DatasetChartRequest) (workspace.DatasetChartResult, error) {
	return a.datasetSvc.PreviewChart(request)
}

func (a *App) CreateDatasetChartArtifact(request workspace.DatasetChartRequest) (artifact.MarkdownReport, error) {
	return a.datasetSvc.CreateChartArtifact(request)
}

func (a *App) CreateDatasetQueryArtifact(relPath string, query string) (artifact.MarkdownReport, error) {
	return a.datasetSvc.CreateQueryArtifact(relPath, query)
}

func (a *App) CreateDatasetSQLArtifact(request analytics.SQLQueryRequest) (artifact.MarkdownReport, error) {
	return a.datasetSvc.CreateSQLArtifact(request)
}

func (a *App) CreateSQLiteQueryCSVArtifact(request dbconnector.SQLiteQueryRequest) (artifact.MarkdownReport, error) {
	return a.datasetSvc.CreateSQLiteQueryCSVArtifact(request)
}

func (a *App) CreateSQLiteQueryMarkdownArtifact(request dbconnector.SQLiteQueryRequest) (artifact.MarkdownReport, error) {
	return a.datasetSvc.CreateSQLiteQueryMarkdownArtifact(request)
}

func (a *App) CreateDatasetSummaryArtifact(relPath string) (artifact.MarkdownReport, error) {
	return a.datasetSvc.CreateSummaryArtifact(relPath)
}

func (a *App) RebuildDatasetDependency(id string) (artifact.MarkdownReport, error) {
	return a.datasetSvc.RebuildDependency(id)
}

func (a *App) ListApprovals() ([]approval.Record, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return []approval.Record{}, nil
	}
	if appmeta.Exists(root) {
		if _, err := a.mirrorMetadataStore(root, false); err == nil {
			if items, readErr := appmeta.ListApprovals(root); readErr == nil {
				return approvalsFromMirror(items), nil
			}
		}
	}
	return approval.List(root)
}

func (a *App) GetRecentWorkspaces() ([]storage.RecentWorkspace, error) {
	return a.recentStore.List()
}

func (a *App) RemoveRecentWorkspace(path string) ([]storage.RecentWorkspace, error) {
	return a.recentStore.Remove(path)
}

func (a *App) ClearRecentWorkspaces() ([]storage.RecentWorkspace, error) {
	return a.recentStore.Clear()
}

func (a *App) GetLLMSettings() (storage.LLMSettings, error) {
	return a.llmStore.Get()
}

func (a *App) SaveLLMSettings(settings storage.LLMSettings) (storage.LLMSettings, error) {
	return a.llmStore.Save(settings)
}

func (a *App) ListConnectorProfiles() ([]storage.ConnectorProfile, error) {
	return a.connectorStore.List()
}

func (a *App) SaveConnectorProfile(profile storage.ConnectorProfile) (storage.ConnectorProfile, error) {
	return a.connectorStore.Save(profile)
}

func (a *App) DeleteConnectorProfile(id string) error {
	return a.connectorStore.Delete(id)
}

func (a *App) TestConnectorProfile(id string) (dbconnector.ConnectorProfileStatus, error) {
	profile, err := a.connectorStore.ResolveByIDForUse(id)
	if err != nil {
		return dbconnector.ConnectorProfileStatus{}, err
	}
	switch profile.Kind {
	case "postgres":
		return dbconnector.TestPostgresProfile(profile)
	case "mysql", "mariadb":
		return dbconnector.TestMySQLProfile(profile)
	case "sqlserver":
		return dbconnector.TestSQLServerProfile(profile)
	case "duckdb":
		return dbconnector.TestDuckDBProfile(profile)
	default:
		return dbconnector.ConnectorProfileStatus{}, fmt.Errorf("connector kind %q is not runnable yet", profile.Kind)
	}
}

func (a *App) InspectConnectorProfile(id string) (dbconnector.ConnectorMetadata, error) {
	profile, err := a.connectorStore.ResolveByIDForUse(id)
	if err != nil {
		return dbconnector.ConnectorMetadata{}, err
	}
	switch profile.Kind {
	case "postgres":
		return dbconnector.InspectPostgresProfile(profile)
	case "mysql", "mariadb":
		return dbconnector.InspectMySQLProfile(profile)
	case "sqlserver":
		return dbconnector.InspectSQLServerProfile(profile)
	case "duckdb":
		return dbconnector.InspectDuckDBProfile(profile)
	default:
		return dbconnector.ConnectorMetadata{}, fmt.Errorf("connector kind %q is not inspectable yet", profile.Kind)
	}
}

func (a *App) QueryConnectorProfile(request dbconnector.ConnectorQueryRequest) (dbconnector.ConnectorQueryResult, error) {
	request = dbconnector.NormalizeConnectorQueryRequest(request)
	profile, err := a.connectorStore.ResolveByIDForUse(request.ProfileID)
	if err != nil {
		return dbconnector.ConnectorQueryResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	if request.RequestID != "" {
		a.registerConnectorQueryCancel(request.RequestID, cancel)
		defer a.unregisterConnectorQueryCancel(request.RequestID)
	}
	defer cancel()
	switch profile.Kind {
	case "postgres":
		return dbconnector.QueryPostgresProfileContext(ctx, profile, request)
	case "mysql", "mariadb":
		return dbconnector.QueryMySQLProfileContext(ctx, profile, request)
	case "sqlserver":
		return dbconnector.QuerySQLServerProfileContext(ctx, profile, request)
	case "duckdb":
		return dbconnector.QueryDuckDBProfileContext(ctx, profile, request)
	default:
		return dbconnector.ConnectorQueryResult{}, fmt.Errorf("connector kind %q is not queryable yet", profile.Kind)
	}
}

func (a *App) CancelConnectorProfileQuery(requestID string) bool {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return false
	}
	a.connectorQueryCancelsMu.Lock()
	cancel := a.connectorQueryCancels[requestID]
	a.connectorQueryCancelsMu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (a *App) registerConnectorQueryCancel(requestID string, cancel context.CancelFunc) {
	a.connectorQueryCancelsMu.Lock()
	defer a.connectorQueryCancelsMu.Unlock()
	if a.connectorQueryCancels == nil {
		a.connectorQueryCancels = map[string]context.CancelFunc{}
	}
	a.connectorQueryCancels[requestID] = cancel
}

func (a *App) unregisterConnectorQueryCancel(requestID string) {
	a.connectorQueryCancelsMu.Lock()
	defer a.connectorQueryCancelsMu.Unlock()
	delete(a.connectorQueryCancels, requestID)
}

func (a *App) GetAssistantProfile() (storage.AssistantProfile, error) {
	return a.profileStore.Get()
}

func (a *App) SaveAssistantProfile(profile storage.AssistantProfile) (storage.AssistantProfile, error) {
	return a.profileStore.Save(profile)
}

func (a *App) TestLLMConnection(settings storage.LLMSettings) (llm.ProbeResult, error) {
	resolvedSettings, err := a.llmStore.ResolveForUse(settings)
	if err != nil {
		return llm.ProbeResult{}, err
	}

	return a.llmClient.Probe(context.Background(), resolvedSettings)
}

func (a *App) GetChatHistory() ([]storage.ChatMessage, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return []storage.ChatMessage{}, nil
	}
	if appmeta.Exists(root) {
		if _, err := a.mirrorMetadataStore(root, false); err == nil {
			if items, readErr := appmeta.ListChats(root); readErr == nil {
				return chatsFromMirror(items), nil
			}
		}
	}

	return a.chatStore.List(root)
}

func (a *App) ClearChatHistory() ([]storage.ChatMessage, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return []storage.ChatMessage{}, nil
	}

	items, err := a.chatStore.Clear(root)
	if err == nil {
		if appmeta.Exists(root) {
			_ = appmeta.ClearChats(root)
		}
	}
	return items, err
}

func (a *App) AskLLM(prompt string, relPath string) (llm.ChatResult, error) {
	chatRequest, settings, err := a.prepareChat(prompt, []string{relPath})
	if err != nil {
		return llm.ChatResult{}, errors.New(sanitizeProviderMessageWithAudit(err.Error(), "ask_llm_prepare"))
	}

	result, err := a.llmClient.Chat(context.Background(), settings, chatRequest)
	if err != nil {
		return llm.ChatResult{}, errors.New(sanitizeProviderMessageWithAudit(err.Error(), "ask_llm_chat"))
	}

	result = chatResultWithSourceCitations(result)
	if err := a.persistChatPair(prompt, chatRequest, result); err != nil {
		return llm.ChatResult{}, errors.New(sanitizeProviderMessageWithAudit(err.Error(), "ask_llm_persist"))
	}

	return result, nil
}

func (a *App) AskLLMStream(prompt string, relPath string, requestID string) (llm.ChatResult, error) {
	chatRequest, settings, err := a.prepareChat(prompt, []string{relPath})
	if err != nil {
		sanitized := errors.New(sanitizeProviderMessageWithAudit(err.Error(), "ask_llm_stream_prepare"))
		a.emitChatStreamEvent(ChatStreamEvent{RequestID: requestID, Type: "error", Message: sanitized.Error()})
		return llm.ChatResult{}, sanitized
	}

	result, err := a.llmClient.ChatStream(context.Background(), settings, chatRequest, func(delta string) error {
		a.emitChatStreamEvent(ChatStreamEvent{
			RequestID:      requestID,
			Type:           "delta",
			Delta:          delta,
			ContextRelPath: chatRequest.ContextRelPath,
		})
		return nil
	})
	if err != nil {
		sanitized := errors.New(sanitizeProviderMessageWithAudit(err.Error(), "ask_llm_stream"))
		a.emitChatStreamEvent(ChatStreamEvent{RequestID: requestID, Type: "error", Message: sanitized.Error()})
		return llm.ChatResult{}, sanitized
	}

	result = chatResultWithSourceCitations(result)
	if err := a.persistChatPair(prompt, chatRequest, result); err != nil {
		sanitized := errors.New(sanitizeProviderMessageWithAudit(err.Error(), "ask_llm_stream_persist"))
		a.emitChatStreamEvent(ChatStreamEvent{RequestID: requestID, Type: "error", Message: sanitized.Error()})
		return llm.ChatResult{}, sanitized
	}

	a.emitChatStreamEvent(ChatStreamEvent{
		RequestID:      requestID,
		Type:           "done",
		Message:        result.Message,
		Model:          result.Model,
		Endpoint:       result.Endpoint,
		ContextRelPath: result.ContextRelPath,
		SourcePaths:    result.SourcePaths,
	})

	return result, nil
}

func (a *App) AskLLMContextPack(prompt string, relPaths []string) (llm.ChatResult, error) {
	chatRequest, settings, err := a.prepareChat(prompt, relPaths)
	if err != nil {
		return llm.ChatResult{}, errors.New(sanitizeProviderMessageWithAudit(err.Error(), "ask_llm_context_pack_prepare"))
	}

	result, err := a.llmClient.Chat(context.Background(), settings, chatRequest)
	if err != nil {
		return llm.ChatResult{}, errors.New(sanitizeProviderMessageWithAudit(err.Error(), "ask_llm_context_pack"))
	}
	result = chatResultWithSourceCitations(result)
	if err := a.persistChatPair(prompt, chatRequest, result); err != nil {
		return llm.ChatResult{}, errors.New(sanitizeProviderMessageWithAudit(err.Error(), "ask_llm_context_pack_persist"))
	}
	return result, nil
}

func (a *App) AskLLMStreamContextPack(prompt string, relPaths []string, requestID string) (llm.ChatResult, error) {
	chatRequest, settings, err := a.prepareChat(prompt, relPaths)
	if err != nil {
		sanitized := errors.New(sanitizeProviderMessageWithAudit(err.Error(), "ask_llm_stream_context_prepare"))
		a.emitChatStreamEvent(ChatStreamEvent{RequestID: requestID, Type: "error", Message: sanitized.Error()})
		return llm.ChatResult{}, sanitized
	}

	result, err := a.llmClient.ChatStream(context.Background(), settings, chatRequest, func(delta string) error {
		a.emitChatStreamEvent(ChatStreamEvent{
			RequestID:      requestID,
			Type:           "delta",
			Delta:          delta,
			ContextRelPath: chatRequest.ContextRelPath,
		})
		return nil
	})
	if err != nil {
		sanitized := errors.New(sanitizeProviderMessageWithAudit(err.Error(), "ask_llm_stream_context"))
		a.emitChatStreamEvent(ChatStreamEvent{RequestID: requestID, Type: "error", Message: sanitized.Error()})
		return llm.ChatResult{}, sanitized
	}
	result = chatResultWithSourceCitations(result)
	if err := a.persistChatPair(prompt, chatRequest, result); err != nil {
		sanitized := errors.New(sanitizeProviderMessageWithAudit(err.Error(), "ask_llm_stream_context_persist"))
		a.emitChatStreamEvent(ChatStreamEvent{RequestID: requestID, Type: "error", Message: sanitized.Error()})
		return llm.ChatResult{}, sanitized
	}
	a.emitChatStreamEvent(ChatStreamEvent{
		RequestID:      requestID,
		Type:           "done",
		Message:        result.Message,
		Model:          result.Model,
		Endpoint:       result.Endpoint,
		ContextRelPath: result.ContextRelPath,
		SourcePaths:    result.SourcePaths,
	})
	return result, nil
}

func (a *App) PreviewChatContextPack(relPaths []string) (workspace.ContextPreview, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.ContextPreview{}, errors.New("open a workspace before previewing context packs")
	}

	settings, err := a.llmStore.Get()
	if err != nil {
		return workspace.ContextPreview{}, err
	}
	return workspace.PreviewContextFiles(root, relPaths, workspace.ContextCollectOptions{MaxFiles: chatContextMaxFiles(settings)})
}

func (a *App) RefreshStaleContext(relPaths []string) (StaleContextRefresh, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return StaleContextRefresh{}, errors.New("open a workspace before refreshing stale context")
	}
	paths := cleanContextPaths(relPaths)
	if len(paths) == 0 {
		return StaleContextRefresh{}, errors.New("choose changed files before refreshing stale context")
	}
	settings, settingsErr := a.llmStore.Get()
	if settingsErr != nil {
		return StaleContextRefresh{}, settingsErr
	}
	preview, err := workspace.PreviewContextFiles(root, paths, workspace.ContextCollectOptions{MaxFiles: chatContextMaxFiles(settings)})
	if err != nil {
		return StaleContextRefresh{}, err
	}
	chats, _ := a.GetChatHistory()
	affectedChats := 0
	for _, message := range chats {
		for _, sourcePath := range message.SourcePaths {
			if containsPath(paths, sourcePath) {
				affectedChats++
				break
			}
		}
	}
	changes := make([]workspace.FileChange, 0, len(paths))
	for _, relPath := range paths {
		changes = append(changes, workspace.FileChange{RelPath: relPath, Kind: "refreshed", Message: relPath + " context was refreshed."})
	}
	result := StaleContextRefresh{
		Preview:        preview,
		AffectedChats:  affectedChats,
		StaleArtifacts: a.staleArtifactsForChanges(root, changes),
		StaleDatasets:  staleDatasetsForChanges(changes),
		Message:        fmt.Sprintf("Refreshed context preview for %d changed roots.", len(paths)),
	}
	a.recordApproval("context.refresh", strings.Join(paths, ", "), "low", result.Message)
	return result, nil
}

func (a *App) prepareChat(prompt string, relPaths []string) (llm.ChatRequest, storage.LLMSettings, error) {
	settings, err := a.llmStore.Get()
	if err != nil {
		return llm.ChatRequest{}, storage.LLMSettings{}, err
	}

	resolvedSettings, err := a.llmStore.ResolveForUse(settings)
	if err != nil {
		return llm.ChatRequest{}, storage.LLMSettings{}, err
	}

	chatRequest := llm.ChatRequest{
		Prompt: prompt,
	}
	if profile, err := a.profileStore.Get(); err == nil {
		chatRequest.Prompt = applyAssistantProfileToPrompt(prompt, profile)
	}
	if root := a.getWorkspaceRoot(); root != "" {
		if history, err := a.GetChatHistory(); err == nil {
			chatRequest.Conversation = buildChatConversationHistory(history, chatHistoryBudgetBytes(resolvedSettings))
		}
	}

	contextPaths := cleanContextPaths(relPaths)
	if len(contextPaths) == 1 && !a.contextPathRequiresPack(contextPaths[0]) {
		contextPreview, err := a.previewChatContext(contextPaths[0], chatContextBudgetBytes(resolvedSettings))
		if err != nil {
			return llm.ChatRequest{}, storage.LLMSettings{}, err
		}
		chatRequest.ContextRelPath = contextPreview.RelPath
		chatRequest.ContextContent = contextPreview.Content
		chatRequest.SourcePaths = []string{contextPreview.RelPath}
	} else if len(contextPaths) > 0 {
		contextRelPath, contextContent, sourcePaths, err := a.buildContextPack(contextPaths, resolvedSettings)
		if err != nil {
			return llm.ChatRequest{}, storage.LLMSettings{}, err
		}
		chatRequest.ContextRelPath = contextRelPath
		chatRequest.ContextContent = contextContent
		chatRequest.SourcePaths = sourcePaths
	}

	return chatRequest, resolvedSettings, nil
}

func (a *App) persistChatPair(prompt string, chatRequest llm.ChatRequest, result llm.ChatResult) error {
	root := a.getWorkspaceRoot()
	if root != "" {
		messages, err := a.chatStore.AppendPair(root, storage.ChatMessage{
			Role:           "user",
			Content:        prompt,
			ContextRelPath: chatRequest.ContextRelPath,
			SourcePaths:    chatRequest.SourcePaths,
		}, storage.ChatMessage{
			Role:           "assistant",
			Content:        appendSourceCitations(result.Message, result.SourcePaths),
			ContextRelPath: result.ContextRelPath,
			SourcePaths:    result.SourcePaths,
		})
		if err != nil {
			return err
		}
		if appmeta.Exists(root) && len(messages) >= 2 {
			last := messages[len(messages)-2:]
			_ = appmeta.AppendChats(root, []appmeta.ChatMirror{
				chatMirrorFromMessage(last[0], fmt.Sprintf("chat-%03d-%s", len(messages)-2, hashForID(last[0].Role+last[0].CreatedAt+last[0].Content))),
				chatMirrorFromMessage(last[1], fmt.Sprintf("chat-%03d-%s", len(messages)-1, hashForID(last[1].Role+last[1].CreatedAt+last[1].Content))),
			})
		}
	}

	return nil
}

func applyAssistantProfileToPrompt(prompt string, profile storage.AssistantProfile) string {
	profile = normalizeAssistantProfileForPrompt(profile)
	active := activePromptProfile(profile)
	sections := []string{}
	if active.Instructions != "" {
		sections = append(sections, "Active prompt profile: "+active.Name+"\n"+active.Instructions)
	}
	if profile.Memory != "" {
		sections = append(sections, "Assistant memory for this user/workspace:\n"+profile.Memory)
	}
	if len(sections) == 0 {
		return prompt
	}
	return strings.Join([]string{
		"Use the following assistant preferences while answering. They are user-provided guidance, not source evidence.",
		strings.Join(sections, "\n\n"),
		"User request:",
		prompt,
	}, "\n\n")
}

func normalizeAssistantProfileForPrompt(profile storage.AssistantProfile) storage.AssistantProfile {
	if len(profile.PromptProfiles) == 0 {
		return storage.DefaultAssistantProfile()
	}
	if activePromptProfile(profile).ID == "" {
		profile.ActiveProfileID = profile.PromptProfiles[0].ID
	}
	return profile
}

func activePromptProfile(profile storage.AssistantProfile) storage.PromptProfile {
	for _, item := range profile.PromptProfiles {
		if item.ID == profile.ActiveProfileID {
			return item
		}
	}
	if len(profile.PromptProfiles) > 0 {
		return profile.PromptProfiles[0]
	}
	return storage.PromptProfile{}
}

func (a *App) emitChatStreamEvent(event ChatStreamEvent) {
	if a.ctx == nil {
		return
	}
	emitChatStreamEventFn(a.ctx, chatStreamEventName, event)
}

func (a *App) previewChatContext(relPath string, maxBytes int) (workspace.FilePreview, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.FilePreview{}, errors.New("open a workspace before sending selected file context")
	}
	if maxBytes <= 0 {
		maxBytes = chatContextFallbackMaxBytes
	}

	contextPreview, err := workspace.Preview(root, relPath, workspace.PreviewOptions{MaxBytes: maxBytes})
	if err != nil {
		return workspace.FilePreview{}, err
	}
	if contextPreview.Content == "" {
		return workspace.FilePreview{}, errors.New("selected file cannot be sent as text context")
	}
	if contextPreview.Kind == "pdf" && strings.TrimSpace(contextPreview.Text) != "" {
		contextPreview.Content = contextPreview.Text
		return contextPreview, nil
	}
	if contextPreview.Kind != "file" {
		return workspace.FilePreview{}, errors.New("selected file context must be a text preview")
	}

	contextPreview.Content = buildChatContextContent(contextPreview)
	if strings.TrimSpace(contextPreview.Content) == "" {
		return workspace.FilePreview{}, errors.New("selected file cannot be sent as text context")
	}

	return contextPreview, nil
}

func buildChatContextContent(preview workspace.FilePreview) string {
	if preview.Table == nil {
		return preview.Content
	}

	var builder strings.Builder
	builder.WriteString("CSV context summary\n\n")
	builder.WriteString("Columns:\n")
	for _, profile := range preview.Table.Profiles {
		builder.WriteString("- ")
		builder.WriteString(profile.Name)
		builder.WriteString(": ")
		builder.WriteString(profile.Type)
		builder.WriteString(fmt.Sprintf(", distinct=%d, missing=%d", profile.Distinct, profile.Missing))
		if profile.Min != "" || profile.Max != "" {
			builder.WriteString(", range=")
			builder.WriteString(profile.Min)
			builder.WriteString("..")
			builder.WriteString(profile.Max)
		}
		builder.WriteString("\n")
	}

	builder.WriteString("\nSample rows:\n")
	csvWriter := csv.NewWriter(&builder)
	_ = csvWriter.Write(preview.Table.Columns)
	for index, row := range preview.Table.Rows {
		if index >= chatCSVContextMaxRows {
			break
		}
		_ = csvWriter.Write(row)
	}
	csvWriter.Flush()
	if preview.Table.Truncated || len(preview.Table.Rows) > chatCSVContextMaxRows {
		builder.WriteString("\nCSV context sample was truncated.\n")
	}

	return builder.String()
}

func (a *App) buildContextPack(relPaths []string, settings storage.LLMSettings) (string, string, []string, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return "", "", nil, errors.New("open a workspace before sending context packs")
	}

	contextBudgetBytes := chatContextBudgetBytes(settings)
	collection, err := workspace.CollectContextFiles(root, relPaths, workspace.ContextCollectOptions{MaxFiles: chatContextMaxFiles(settings)})
	if err != nil {
		return "", "", nil, err
	}

	var builder strings.Builder
	usedPaths := []string{}
	builder.WriteString("Workspace context pack\n")
	builder.WriteString("Requested roots: ")
	builder.WriteString(strings.Join(collection.Roots, ", "))
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("Included files: %d", len(collection.Files)))
	if collection.Truncated {
		builder.WriteString(" (truncated)")
	}
	builder.WriteString("\n")

	for _, file := range collection.Files {
		remaining := contextBudgetBytes - builder.Len()
		if remaining <= 0 {
			break
		}
		preview, err := a.previewChatContext(file.RelPath, remaining)
		if err != nil {
			if file.Required {
				return "", "", nil, err
			}
			continue
		}

		entry := "\n\n# Workspace context: " + preview.RelPath + "\n\n" + preview.Content
		truncated := len(entry) > remaining
		if truncated {
			entry = truncateContextString(entry, remaining)
		}

		builder.WriteString(entry)
		usedPaths = append(usedPaths, preview.RelPath)
		if truncated {
			builder.WriteString("\n\n_Context pack truncated._\n")
			break
		}
	}
	if len(usedPaths) == 0 {
		return "", "", nil, errors.New("context pack did not include usable text")
	}

	contextLabel := buildContextLabel(collection.Roots, usedPaths)
	return contextLabel, strings.TrimSpace(builder.String()), usedPaths, nil
}

func (a *App) contextPathRequiresPack(relPath string) bool {
	trimmed := strings.TrimSpace(relPath)
	if trimmed == "" || trimmed == "." || trimmed == "/" {
		return true
	}

	root := a.getWorkspaceRoot()
	if root == "" {
		return false
	}

	target := filepath.Join(root, filepath.FromSlash(trimmed))
	info, err := os.Lstat(target)
	return err == nil && info.IsDir()
}

func buildContextLabel(roots []string, usedPaths []string) string {
	if len(roots) == 1 {
		if roots[0] == "." {
			return fmt.Sprintf("project: %d files", len(usedPaths))
		}
		if len(usedPaths) == 1 && usedPaths[0] == roots[0] {
			return usedPaths[0]
		}
		return fmt.Sprintf("dir: %s (%d files)", roots[0], len(usedPaths))
	}

	return fmt.Sprintf("pack: %d roots, %d files", len(roots), len(usedPaths))
}

func truncateContextString(content string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(content) <= maxBytes {
		return content
	}

	truncated := content[:maxBytes]
	for !utf8.ValidString(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}

func buildChatConversationHistory(messages []storage.ChatMessage, maxBytes int) []llm.ChatTurn {
	if maxBytes <= 0 || len(messages) == 0 {
		return []llm.ChatTurn{}
	}

	turns := []llm.ChatTurn{}
	usedBytes := 0
	for index := len(messages) - 1; index >= 0 && len(turns) < chatHistoryMaxMessages; index-- {
		role := normalizeConversationRole(messages[index].Role)
		content := strings.TrimSpace(messages[index].Content)
		if role == "" || content == "" {
			continue
		}
		remaining := maxBytes - usedBytes
		if remaining <= 0 {
			break
		}
		if len(content) > remaining {
			content = truncateContextString(content, remaining)
		}
		if strings.TrimSpace(content) == "" {
			break
		}
		turns = append([]llm.ChatTurn{{Role: role, Content: content}}, turns...)
		usedBytes += len(content)
	}
	return turns
}

func normalizeConversationRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "user":
		return "user"
	case "assistant":
		return "assistant"
	default:
		return ""
	}
}

func chatHistoryBudgetBytes(settings storage.LLMSettings) int {
	budget := chatContextBudgetBytes(settings) / 8
	if budget < chatHistoryMinBudgetBytes {
		return chatHistoryMinBudgetBytes
	}
	if budget > chatHistoryMaxBudgetBytes {
		return chatHistoryMaxBudgetBytes
	}
	return budget
}

func chatContextBudgetBytes(settings storage.LLMSettings) int {
	maxTokens := settings.MaxContextTokens
	if maxTokens <= 0 {
		maxTokens = storage.DefaultLLMSettings().MaxContextTokens
	}
	reserveTokens := settings.ResponseReserveTokens
	if reserveTokens <= 0 {
		reserveTokens = storage.DefaultLLMSettings().ResponseReserveTokens
	}
	availableTokens := maxTokens - reserveTokens - chatContextOverheadTokens
	if availableTokens <= 0 {
		availableTokens = maxTokens / 2
	}
	budget := availableTokens * chatContextTokenByteEstimate
	if budget < chatContextMinBudgetBytes {
		return chatContextMinBudgetBytes
	}
	if budget > chatContextMaxBudgetBytes {
		return chatContextMaxBudgetBytes
	}
	return budget
}

func chatContextMaxFiles(settings storage.LLMSettings) int {
	budget := chatContextBudgetBytes(settings)
	files := budget / (8 * 1024)
	if files < chatContextFallbackMaxFiles {
		return chatContextFallbackMaxFiles
	}
	if files > chatContextMaxFilesCap {
		return chatContextMaxFilesCap
	}
	return files
}

func cleanContextPaths(relPaths []string) []string {
	seen := map[string]bool{}
	cleaned := []string{}
	for _, relPath := range relPaths {
		relPath = strings.TrimSpace(relPath)
		if relPath == "" || seen[relPath] {
			continue
		}
		seen[relPath] = true
		cleaned = append(cleaned, relPath)
	}
	return cleaned
}

func trimAppSnippet(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n"))
	if len(value) <= 180 {
		return value
	}
	return value[:177] + "..."
}

func (a *App) runAgentTool(root string, request agenttools.RunRequest, mode string) (agenttools.RunRecord, error) {
	descriptor, err := agenttools.RequireDescriptor(request.ToolName)
	startedAt := time.Now()
	if err != nil {
		return agenttools.RunRecord{}, err
	}
	record := agenttools.NewRecord(request, descriptor, mode, "planned", startedAt)
	if mode == "execute" && descriptor.RequiresApproval && !request.Approved {
		err := errors.New("tool execution requires approval")
		return agenttools.FinishRecord(record, "blocked", "", err, time.Now()), err
	}

	summary, runErr := a.agentToolSummary(root, request, descriptor, mode)
	if mode == "execute" && runErr == nil && descriptor.RequiresApproval {
		record.ApprovalID = a.recordApproval("agenttool."+descriptor.Name, record.Target, descriptor.Risk, summary)
	}
	status := "dry-run"
	if mode == "execute" {
		status = "executed"
	}
	finished := agenttools.FinishRecord(record, status, summary, runErr, time.Now())
	return finished, runErr
}

func (a *App) agentToolSummary(root string, request agenttools.RunRequest, descriptor agenttools.Descriptor, mode string) (string, error) {
	target := strings.TrimSpace(request.Target)
	switch descriptor.Name {
	case "workspace.preview":
		if mode == "dry-run" {
			return "Ready to preview " + target + " inside the active workspace.", nil
		}
		preview, err := workspace.Preview(root, target, workspace.PreviewOptions{MaxBytes: chatContextFallbackMaxBytes})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Previewed %s as %s (%d bytes).", preview.RelPath, preview.Kind, preview.Size), nil
	case "workspace.context":
		paths := parseAgentRelPaths(firstNonEmpty(request.Inputs["relPaths"], request.Inputs["paths"], target))
		if len(paths) == 0 {
			paths = []string{"."}
		}
		settings, err := a.llmStore.Get()
		if err != nil {
			return "", err
		}
		settings, err = a.llmStore.ResolveForUse(settings)
		if err != nil {
			return "", err
		}
		if mode == "dry-run" {
			preview, err := workspace.PreviewContextFiles(root, paths, workspace.ContextCollectOptions{MaxFiles: chatContextMaxFiles(settings)})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s Sources: %d.", preview.Message, preview.FileCount), nil
		}
		label, _, sourcePaths, err := a.buildContextPack(paths, settings)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Built context pack %s with %d source file(s).", label, len(sourcePaths)), nil
	case "git.diff":
		service := gitservice.New(a.getWorkspaceRoot)
		relPath := firstNonEmpty(request.Inputs["relPath"], request.Inputs["path"], target)
		if strings.TrimSpace(relPath) == "" {
			status, err := service.Status()
			if err != nil {
				return "", err
			}
			if mode == "dry-run" {
				return "Ready to read Git status and bounded staged/unstaged diffs.", nil
			}
			return fmt.Sprintf("Read Git diff context: %s, %d changed file(s).", status.Message, len(status.ChangedFiles)), nil
		}
		diff, err := service.FileDiff(relPath)
		if err != nil {
			return "", err
		}
		if mode == "dry-run" {
			return "Ready to read bounded Git diff for " + cleanAgentRelPath(relPath) + ".", nil
		}
		return diff.Message, nil
	case "git.changedFiles":
		service := gitservice.New(a.getWorkspaceRoot)
		status, err := service.Status()
		if err != nil {
			return "", err
		}
		if mode == "dry-run" {
			return "Ready to read changed-file context from the current Git working tree.", nil
		}
		return fmt.Sprintf("Read changed-file context for %d file(s).", len(status.ChangedFiles)), nil
	case "git.history":
		service := gitservice.New(a.getWorkspaceRoot)
		relPath := firstNonEmpty(request.Inputs["relPath"], request.Inputs["path"], target)
		limit := parseAgentInt(request.Inputs["limit"], gitservice.DefaultHistoryLimit)
		if mode == "dry-run" {
			if strings.TrimSpace(relPath) == "" {
				return "Ready to read bounded Git commit history for the repository.", nil
			}
			return "Ready to read bounded Git commit history for " + cleanAgentRelPath(relPath) + ".", nil
		}
		result, err := service.History(GitHistoryRequest{Path: relPath, Limit: limit})
		if err != nil {
			return "", err
		}
		return result.Message, nil
	case "git.blame":
		service := gitservice.New(a.getWorkspaceRoot)
		relPath := firstNonEmpty(request.Inputs["relPath"], request.Inputs["path"], target)
		startLine := parseAgentInt(firstNonEmpty(request.Inputs["startLine"], request.Inputs["start"]), 0)
		endLine := parseAgentInt(firstNonEmpty(request.Inputs["endLine"], request.Inputs["end"]), 0)
		if mode == "dry-run" {
			return "Ready to read bounded Git blame for " + cleanAgentRelPath(relPath) + ".", nil
		}
		result, err := service.Blame(GitBlameRequest{Path: relPath, StartLine: startLine, EndLine: endLine})
		if err != nil {
			return "", err
		}
		return result.Message, nil
	case "workspace.problems":
		maxResults := parseAgentInt(request.Inputs["maxResults"], 40)
		if maxResults <= 0 || maxResults > 120 {
			maxResults = 40
		}
		if mode == "dry-run" {
			return fmt.Sprintf("Ready to scan up to %d lightweight workspace problems.", maxResults), nil
		}
		summary, err := workspace.ScanProblems(root, maxResults)
		if err != nil {
			return "", err
		}
		return summary.Message, nil
	case "workspace.tasks":
		if mode == "dry-run" {
			return "Ready to list discovered workspace tasks.", nil
		}
		summary, err := discoverWorkspaceTasks(root)
		if err != nil {
			return "", err
		}
		return summary.Message, nil
	case "workspace.task.run":
		taskID := firstNonEmpty(request.Inputs["taskId"], request.Inputs["id"], target)
		if mode == "dry-run" {
			return "Ready to run discovered workspace task " + taskID + " after approval.", nil
		}
		result, err := a.RunWorkspaceTask(WorkspaceTaskRunRequest{TaskID: taskID})
		if err != nil {
			return "", err
		}
		return result.Message, nil
	case "workspace.write":
		if mode == "dry-run" {
			return "Ready to write text/code content to " + target + " through diff preview.", nil
		}
		rollback, err := workspace.PrepareRollback(root, "workspace.write", target, []string{target})
		if err != nil {
			return "", err
		}
		proposal, err := workspace.ApplyFileWrite(root, workspace.FileWriteRequest{
			RelPath:  target,
			Content:  request.Inputs["content"],
			Encoding: strings.TrimSpace(request.Inputs["encoding"]),
		})
		if err != nil {
			return "", err
		}
		rollback, err = workspace.CommitRollback(root, rollback)
		if err != nil {
			return "", err
		}
		return proposal.Message + " Rollback: " + rollback.ID, nil
	case "workspace.writeBinary":
		if mode == "dry-run" {
			return "Ready to write binary content to " + target + " through size/hash preview.", nil
		}
		rollback, err := workspace.PrepareRollback(root, "workspace.writeBinary", target, []string{target})
		if err != nil {
			return "", err
		}
		proposal, err := workspace.ApplyBinaryFileWrite(root, workspace.BinaryFileWriteRequest{
			RelPath:       target,
			Base64Content: firstNonEmpty(request.Inputs["base64Content"], request.Inputs["contentBase64"], request.Inputs["base64"]),
			ContentType:   strings.TrimSpace(request.Inputs["contentType"]),
		})
		if err != nil {
			return "", err
		}
		rollback, err = workspace.CommitRollback(root, rollback)
		if err != nil {
			return "", err
		}
		return proposal.Message + " Rollback: " + rollback.ID, nil
	case "workspace.patch":
		patch := firstNonEmpty(request.Inputs["patch"], request.Inputs["unifiedDiff"], request.Inputs["diff"])
		if mode == "dry-run" {
			proposal, err := workspace.PreviewUnifiedPatch(root, workspace.UnifiedPatchRequest{Patch: patch})
			if err != nil {
				return "", err
			}
			return unifiedPatchProposalSummary(proposal, false), nil
		}
		preview, err := workspace.PreviewUnifiedPatch(root, workspace.UnifiedPatchRequest{Patch: patch})
		if err != nil {
			return "", err
		}
		rollback, err := workspace.PrepareRollback(root, "workspace.patch", "workspace", unifiedPatchRollbackPaths(preview))
		if err != nil {
			return "", err
		}
		proposal, err := workspace.ApplyUnifiedPatch(root, workspace.UnifiedPatchRequest{Patch: patch})
		if err != nil {
			return "", err
		}
		rollback, err = workspace.CommitRollback(root, rollback)
		if err != nil {
			return "", err
		}
		return proposal.Message + " Rollback: " + rollback.ID, nil
	case "workspace.copy":
		request := workspace.FileCopyRequest{
			SourceRelPath: firstNonEmpty(request.Inputs["sourceRelPath"], request.Inputs["source"], target),
			TargetRelPath: firstNonEmpty(request.Inputs["targetRelPath"], request.Inputs["target"], request.Inputs["to"]),
		}
		if mode == "dry-run" {
			proposal, err := workspace.PreviewFileCopy(root, request)
			if err != nil {
				return "", err
			}
			return proposal.Message, nil
		}
		rollback, err := workspace.PrepareRollback(root, "workspace.copy", request.TargetRelPath, []string{request.TargetRelPath})
		if err != nil {
			return "", err
		}
		proposal, err := workspace.ApplyFileCopy(root, request)
		if err != nil {
			return "", err
		}
		rollback, err = workspace.CommitRollback(root, rollback)
		if err != nil {
			return "", err
		}
		return proposal.Message + " Rollback: " + rollback.ID, nil
	case "workspace.move":
		request := workspace.FileMoveRequest{
			SourceRelPath: firstNonEmpty(request.Inputs["sourceRelPath"], request.Inputs["source"], target),
			TargetRelPath: firstNonEmpty(request.Inputs["targetRelPath"], request.Inputs["target"], request.Inputs["to"]),
		}
		if mode == "dry-run" {
			proposal, err := workspace.PreviewFileMove(root, request)
			if err != nil {
				return "", err
			}
			return proposal.Message, nil
		}
		rollback, err := workspace.PrepareRollback(root, "workspace.move", request.TargetRelPath, []string{request.SourceRelPath, request.TargetRelPath})
		if err != nil {
			return "", err
		}
		proposal, err := workspace.ApplyFileMove(root, request)
		if err != nil {
			return "", err
		}
		rollback, err = workspace.CommitRollback(root, rollback)
		if err != nil {
			return "", err
		}
		return proposal.Message + " Rollback: " + rollback.ID, nil
	case "workspace.delete":
		if mode == "dry-run" {
			proposal, err := workspace.PreviewFileDelete(root, target)
			if err != nil {
				return "", err
			}
			return proposal.Message, nil
		}
		rollback, err := workspace.PrepareRollback(root, "workspace.delete", target, []string{target})
		if err != nil {
			return "", err
		}
		proposal, err := workspace.ApplyFileDelete(root, target)
		if err != nil {
			return "", err
		}
		rollback, err = workspace.CommitRollback(root, rollback)
		if err != nil {
			return "", err
		}
		return proposal.Message + " Rollback: " + rollback.ID, nil
	case "workspace.rollback.list":
		items, err := workspace.ListRollbacks(root)
		if err != nil {
			return "", err
		}
		if mode == "dry-run" {
			return fmt.Sprintf("Ready to list %d rollback snapshot(s).", len(items)), nil
		}
		return fmt.Sprintf("Listed %d rollback snapshot(s).", len(items)), nil
	case "workspace.rollback.apply":
		id := firstNonEmpty(request.Inputs["id"], request.Inputs["rollbackId"], target)
		if mode == "dry-run" {
			return "Ready to apply rollback " + strings.TrimSpace(id) + " after approval.", nil
		}
		result, err := workspace.ApplyRollback(root, id)
		if err != nil {
			return "", err
		}
		return result.Message, nil
	case "dataset.query":
		query := request.Inputs["query"]
		if mode == "dry-run" {
			return fmt.Sprintf("Ready to query %s with %q.", target, fallbackInput(query, "first rows")), nil
		}
		result, err := workspace.QueryCSV(root, target, query)
		if err != nil {
			return "", err
		}
		return result.Message, nil
	case "dataset.profile":
		if mode == "dry-run" {
			return "Ready to profile dataset " + target + ".", nil
		}
		profile, err := a.datasetSvc.Profile(target)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s Rows: %d. Columns: %d.", profile.Message, profile.Rows, profile.Columns), nil
	case "dataset.sql":
		sqlText := firstNonEmpty(request.Inputs["sql"], request.Inputs["query"])
		if mode == "dry-run" {
			return "Ready to run read-only dataset SQL for " + target + ".", nil
		}
		result, err := a.datasetSvc.QuerySQL(analytics.SQLQueryRequest{RelPath: target, SQL: sqlText})
		if err != nil {
			return "", err
		}
		return result.Message, nil
	case "sqlite.inspect":
		if mode == "dry-run" {
			return "Ready to inspect SQLite schema for " + target + " in read-only mode.", nil
		}
		metadata, err := a.datasetSvc.InspectWorkspaceSQLite(target)
		if err != nil {
			return "", err
		}
		return metadata.Message, nil
	case "sqlite.query":
		sqlText := firstNonEmpty(request.Inputs["sql"], request.Inputs["query"])
		sqliteRequest := dbconnector.SQLiteQueryRequest{
			RelPath:        target,
			SQL:            sqlText,
			ResultLimit:    parseAgentInt(request.Inputs["resultLimit"], 100),
			TimeoutSeconds: parseAgentInt(request.Inputs["timeoutSeconds"], 30),
		}
		if mode == "dry-run" {
			return "Ready to run bounded read-only SQLite query for " + target + ".", nil
		}
		result, err := a.datasetSvc.QueryWorkspaceSQLite(sqliteRequest)
		if err != nil {
			return "", err
		}
		return result.Message, nil
	case "document.set":
		paths := parseAgentRelPaths(firstNonEmpty(request.Inputs["relPaths"], request.Inputs["paths"], target))
		if len(paths) == 0 {
			paths = []string{"."}
		}
		maxFiles := parseAgentInt(request.Inputs["maxFiles"], 16)
		if maxFiles <= 0 || maxFiles > 48 {
			maxFiles = 16
		}
		docPaths, truncated, err := collectAgentDocumentPaths(root, paths, maxFiles)
		if err != nil {
			return "", err
		}
		if mode == "dry-run" {
			return fmt.Sprintf("Ready to read up to %d document file(s) from %s.", maxFiles, strings.Join(paths, ", ")), nil
		}
		message := fmt.Sprintf("Read document-set context for %d file(s).", len(docPaths))
		if truncated {
			message += " Some documents were skipped by caps."
		}
		return message, nil
	case "artifact.create":
		if mode == "dry-run" {
			return "Ready to create a Markdown report artifact from " + target + ".", nil
		}
		report, err := a.CreateMarkdownReport(target)
		if err != nil {
			return "", err
		}
		return report.Message + " " + report.RelPath, nil
	case "artifact.list":
		if mode == "dry-run" {
			return "Ready to list generated artifacts.", nil
		}
		items, err := a.artifactSvc.List()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Listed %d artifact(s).", len(items)), nil
	case "artifact.read":
		if mode == "dry-run" {
			return "Ready to read artifact " + target + " with metadata.", nil
		}
		metadata, err := a.artifactSvc.Metadata(target)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Read artifact %s (%s, %s).", target, metadata.Kind, metadata.Title), nil
	case "artifact.lineage":
		if mode == "dry-run" {
			return "Ready to read artifact lineage graph context.", nil
		}
		lineage, err := a.GetArtifactLineage()
		if err != nil {
			return "", err
		}
		return lineage.Message, nil
	case "web.fetch":
		targetURL := firstNonEmpty(request.Inputs["url"], target)
		if mode == "dry-run" {
			return "Ready to fetch approved web URL " + targetURL + " with text/content safety caps.", nil
		}
		result, err := webfetch.Fetch(context.Background(), webfetch.Request{
			URL:            targetURL,
			AllowedDomains: parseAgentList(request.Inputs["allowedDomains"]),
			AllowLocal:     parseAgentBool(request.Inputs["allowLocal"]),
			MaxBytes:       parseAgentInt(request.Inputs["maxBytes"], 128*1024),
		})
		if err != nil {
			return "", err
		}
		return result.Message, nil
	case "artifact.archive":
		if mode == "dry-run" {
			return "Ready to archive " + target + " with metadata sidecar.", nil
		}
		report, err := artifact.Archive(root, target)
		if err != nil {
			return "", err
		}
		return report.Message + " " + report.RelPath, nil
	case "operations.inspect":
		if mode == "dry-run" {
			return "Ready to inspect operations context " + target + " without mutating Docker state.", nil
		}
		preview, err := workspace.Preview(root, target, workspace.PreviewOptions{MaxBytes: chatContextFallbackMaxBytes})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Inspected %s (%s, %d bytes) read-only.", preview.RelPath, preview.FileType, preview.Size), nil
	default:
		return "", errors.New("agent tool execution is not implemented for " + descriptor.Name)
	}
}

func (a *App) appendToolRun(root string, record agenttools.RunRecord) error {
	if record.ToolName == "" {
		return nil
	}
	_, err := agenttools.Append(root, record)
	if err == nil && appmeta.Exists(root) {
		inputs, _ := json.Marshal(record.Inputs)
		_ = appmeta.AppendToolRun(root, appmeta.ToolRunMirror{
			ID:            record.ID,
			ToolName:      record.ToolName,
			Target:        record.Target,
			Risk:          record.Risk,
			Status:        record.Status,
			Mode:          record.Mode,
			ApprovalID:    record.ApprovalID,
			Inputs:        inputs,
			OutputSummary: record.OutputSummary,
			Error:         record.Error,
			StartedAt:     record.StartedAt,
			CompletedAt:   record.CompletedAt,
			DurationMs:    record.DurationMs,
		})
	}
	return err
}

func fallbackInput(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func (a *App) recordApproval(action string, target string, risk string, message string) string {
	root := a.getWorkspaceRoot()
	if root == "" {
		return ""
	}
	items, _ := approval.Append(root, approval.Record{
		Action:   action,
		Target:   target,
		Risk:     risk,
		Decision: "applied",
		Message:  message,
	})
	if len(items) == 0 {
		return ""
	}
	if appmeta.Exists(root) {
		_ = appmeta.AppendApproval(root, appmeta.ApprovalMirror{
			ID:        items[0].ID,
			Action:    items[0].Action,
			Target:    items[0].Target,
			Risk:      items[0].Risk,
			Decision:  items[0].Decision,
			Message:   items[0].Message,
			CreatedAt: items[0].CreatedAt,
		})
	}
	return items[0].ID
}

func (a *App) openWorkspace(root string) (WorkspaceOpenResult, error) {
	return a.workspaceSvc.Open(root)
}

func (a *App) setWorkspaceRoot(root string) {
	a.workspaceSvc.SetRoot(root)
}

func (a *App) getWorkspaceRoot() string {
	return a.workspaceSvc.Root()
}

func (a *App) resetWorkspaceFreshness(root string) {
	a.workspaceSvc.ResetFreshness(root)
}

func (a *App) staleArtifactsForChanges(root string, changes []workspace.FileChange) []string {
	return a.workspaceSvc.StaleArtifactsForChanges(root, changes)
}

func staleDatasetsForChanges(changes []workspace.FileChange) []string {
	stale := []string{}
	seen := map[string]bool{}
	for _, change := range changes {
		relPath := filepath.ToSlash(change.RelPath)
		ext := strings.ToLower(filepath.Ext(relPath))
		if ext != ".csv" && ext != ".tsv" && ext != ".xlsx" && ext != ".xls" {
			continue
		}
		if seen[relPath] {
			continue
		}
		seen[relPath] = true
		stale = append(stale, relPath)
	}
	return stale
}

func containsPath(paths []string, relPath string) bool {
	relPath = filepath.ToSlash(strings.TrimSpace(relPath))
	for _, path := range paths {
		if filepath.ToSlash(strings.TrimSpace(path)) == relPath {
			return true
		}
	}
	return false
}

func lineageSourcePaths(lineage ArtifactLineage) []string {
	paths := []string{}
	seen := map[string]bool{}
	for _, node := range lineage.Nodes {
		if node.RelPath == "" || seen[node.RelPath] {
			continue
		}
		seen[node.RelPath] = true
		paths = append(paths, node.RelPath)
	}
	return paths
}

func appendSourceCitations(message string, sourcePaths []string) string {
	sourcePaths = compactStrings(sourcePaths)
	if strings.TrimSpace(message) == "" || len(sourcePaths) == 0 || strings.Contains(message, "\n\nSources:") {
		return message
	}
	var builder strings.Builder
	builder.WriteString(strings.TrimRight(message, "\n"))
	builder.WriteString("\n\nSources:\n")
	for _, sourcePath := range sourcePaths {
		builder.WriteString("- ")
		builder.WriteString(sourcePath)
		builder.WriteString("\n")
	}
	return builder.String()
}

func chatResultWithSourceCitations(result llm.ChatResult) llm.ChatResult {
	result.Message = appendSourceCitations(result.Message, result.SourcePaths)
	return result
}

func compactStrings(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func isArtifactRelPath(relPath string) bool {
	normalized := strings.ToLower(filepath.ToSlash(relPath))
	return strings.HasPrefix(normalized, ".nexusdesk/artifacts/")
}

func targetKind(relPath string) string {
	if isArtifactRelPath(relPath) {
		return "artifact"
	}
	return "source"
}
