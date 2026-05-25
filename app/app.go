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
	"NexusAugenticStudio/internal/dataset"
	"NexusAugenticStudio/internal/dbconnector"
	"NexusAugenticStudio/internal/llm"
	"NexusAugenticStudio/internal/storage"
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
const chatStreamEventName = "nexus:chat-stream"
const agentRunEventName = "nexus:agent-run"

var emitChatStreamEventFn = func(ctx context.Context, name string, event any) {
	runtime.EventsEmit(ctx, name, event)
}

type App struct {
	ctx           context.Context
	llmClient     *llm.Client
	chatStore     *storage.ChatHistoryStore
	llmStore      *storage.LLMSettingsStore
	recentStore   *storage.RecentWorkspaceStore
	workspaceMu   sync.RWMutex
	workspaceRoot string
	watchMu       sync.Mutex
	fingerprints  map[string]workspace.FileFingerprint
}

func NewApp() *App {
	return &App{
		llmClient:   llm.NewClient(),
		chatStore:   storage.NewDefaultChatHistoryStore(),
		llmStore:    storage.NewDefaultLLMSettingsStore(),
		recentStore: storage.NewDefaultRecentWorkspaceStore(),
	}
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

	return a.openWorkspace(root)
}

func (a *App) OpenWorkspace(root string) (WorkspaceOpenResult, error) {
	if root == "" {
		return WorkspaceOpenResult{Selected: false}, nil
	}

	return a.openWorkspace(root)
}

func (a *App) RefreshWorkspace() (WorkspaceOpenResult, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return WorkspaceOpenResult{Selected: false}, nil
	}

	snapshot, err := workspace.Scan(root, workspace.ScanOptions{})
	if err != nil {
		return WorkspaceOpenResult{}, err
	}

	return WorkspaceOpenResult{
		Selected: true,
		Snapshot: snapshot,
	}, nil
}

func (a *App) SearchWorkspace(query string) ([]workspace.SearchResult, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return []workspace.SearchResult{}, errors.New("open a workspace before searching")
	}

	results, err := workspace.Search(root, query, workspace.SearchOptions{MaxResults: 70})
	if err != nil {
		return nil, err
	}

	artifactResults, err := artifact.Search(root, query)
	if err != nil {
		return nil, err
	}
	results = append(results, artifactResults...)

	chatMessages, err := a.chatStore.Search(root, query)
	if err != nil {
		return nil, err
	}
	for _, message := range chatMessages {
		results = append(results, workspace.SearchResult{
			RelPath:   "Chat history",
			Name:      "Chat history",
			Kind:      "chat",
			FileType:  "chat",
			MatchType: message.Role,
			Snippet:   trimAppSnippet(message.Content),
		})
	}
	if len(results) > 100 {
		results = results[:100]
	}
	return results, nil
}

func (a *App) ReadWorkspaceFile(relPath string) (workspace.FilePreview, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.FilePreview{}, errors.New("open a workspace before reading files")
	}

	return workspace.Preview(root, relPath, workspace.PreviewOptions{})
}

func (a *App) PreviewFileWrite(request workspace.FileWriteRequest) (workspace.FileWriteProposal, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.FileWriteProposal{}, errors.New("open a workspace before previewing file writes")
	}

	return workspace.PreviewFileWrite(root, request)
}

func (a *App) ApplyFileWrite(request workspace.FileWriteRequest) (workspace.FileWriteProposal, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.FileWriteProposal{}, errors.New("open a workspace before applying file writes")
	}

	proposal, err := workspace.ApplyFileWrite(root, request)
	if err != nil {
		return workspace.FileWriteProposal{}, err
	}
	a.recordApproval("file.write", proposal.RelPath, "medium", proposal.Message)
	return proposal, nil
}

func (a *App) PreviewFileDelete(relPath string) (workspace.FileDeleteProposal, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.FileDeleteProposal{}, errors.New("open a workspace before previewing file deletes")
	}

	return workspace.PreviewFileDelete(root, relPath)
}

func (a *App) ApplyFileDelete(relPath string) (workspace.FileDeleteProposal, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.FileDeleteProposal{}, errors.New("open a workspace before deleting files")
	}

	proposal, err := workspace.ApplyFileDelete(root, relPath)
	if err != nil {
		return workspace.FileDeleteProposal{}, err
	}
	a.recordApproval("file.delete", proposal.RelPath, "high", proposal.Message)
	return proposal, nil
}

func (a *App) PreviewFileMove(request workspace.FileMoveRequest) (workspace.FileMoveProposal, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.FileMoveProposal{}, errors.New("open a workspace before previewing file moves")
	}

	return workspace.PreviewFileMove(root, request)
}

func (a *App) ApplyFileMove(request workspace.FileMoveRequest) (workspace.FileMoveProposal, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.FileMoveProposal{}, errors.New("open a workspace before moving files")
	}

	proposal, err := workspace.ApplyFileMove(root, request)
	if err != nil {
		return workspace.FileMoveProposal{}, err
	}
	a.recordApproval("file.move", proposal.TargetRelPath, "high", proposal.Message)
	return proposal, nil
}

func (a *App) CreateMarkdownReport(relPath string) (artifact.MarkdownReport, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return artifact.MarkdownReport{}, errors.New("open a workspace before creating reports")
	}

	source := workspace.FilePreview{
		RelPath: relPath,
		Name:    "workspace-report",
	}
	if relPath != "" {
		preview, err := workspace.Preview(root, relPath, workspace.PreviewOptions{MaxBytes: chatContextFallbackMaxBytes})
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		source = preview
	}

	report, err := artifact.CreateMarkdownReport(root, source, time.Now())
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	a.persistArtifactMetadata(root, report.RelPath)
	a.recordApproval("artifact.report", report.RelPath, "low", report.Message)
	return report, nil
}

func (a *App) CreateScanReportArtifact() (artifact.MarkdownReport, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return artifact.MarkdownReport{}, errors.New("open a workspace before creating scan reports")
	}

	snapshot, err := workspace.Scan(root, workspace.ScanOptions{})
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	report, err := artifact.CreateScanReportMarkdown(root, snapshot, time.Now())
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	a.persistArtifactMetadata(root, report.RelPath)
	a.recordApproval("artifact.scan-report", report.RelPath, "low", report.Message)
	return report, nil
}

func (a *App) CreateChatMarkdownArtifact(request artifact.MarkdownArtifactRequest) (artifact.MarkdownReport, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return artifact.MarkdownReport{}, errors.New("open a workspace before creating artifacts")
	}

	report, err := artifact.CreateGeneratedMarkdown(root, request, time.Now())
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	a.persistArtifactMetadata(root, report.RelPath)
	a.recordApproval("artifact.markdown", report.RelPath, "low", report.Message)
	return report, nil
}

func (a *App) ListArtifacts() ([]artifact.WorkspaceArtifact, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return []artifact.WorkspaceArtifact{}, nil
	}
	if appmeta.Exists(root) {
		if _, err := a.mirrorMetadataStore(root, false); err == nil {
			if items, readErr := a.listArtifactsFromMetadata(root); readErr == nil {
				return items, nil
			}
		}
	}
	return artifact.List(root)
}

func (a *App) GetArtifactMetadata(relPath string) (artifact.ArtifactMetadata, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return artifact.ArtifactMetadata{}, errors.New("open a workspace before reading artifact metadata")
	}
	return artifact.Metadata(root, relPath)
}

func (a *App) ArchiveArtifact(relPath string) (artifact.MarkdownReport, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return artifact.MarkdownReport{}, errors.New("open a workspace before archiving artifacts")
	}

	report, err := artifact.Archive(root, relPath)
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	a.recordApproval("artifact.archive", relPath, "medium", report.Message)
	return report, nil
}

func (a *App) DeleteArtifact(relPath string) (artifact.MarkdownReport, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return artifact.MarkdownReport{}, errors.New("open a workspace before deleting artifacts")
	}

	report, err := artifact.Delete(root, relPath)
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	a.recordApproval("artifact.delete", relPath, "high", report.Message)
	return report, nil
}

func (a *App) CompareArtifacts(leftRelPath string, rightRelPath string) (artifact.ArtifactComparison, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return artifact.ArtifactComparison{}, errors.New("open a workspace before comparing artifacts")
	}
	return artifact.Compare(root, leftRelPath, rightRelPath)
}

func (a *App) ProfileDataset(relPath string) (dataset.Profile, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return dataset.Profile{}, errors.New("open a workspace before profiling datasets")
	}
	return dataset.Build(root, relPath)
}

func (a *App) ListDatasetProfiles() ([]dataset.Profile, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return []dataset.Profile{}, nil
	}
	return dataset.List(root)
}

func (a *App) QueryDataset(relPath string, query string) (workspace.DatasetQueryResult, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.DatasetQueryResult{}, errors.New("open a workspace before querying datasets")
	}
	return workspace.QueryCSV(root, relPath, query)
}

func (a *App) QueryDatasetSQL(request analytics.SQLQueryRequest) (analytics.SQLQueryResult, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return analytics.SQLQueryResult{}, errors.New("open a workspace before querying datasets")
	}
	result, err := analytics.QueryCSVSQL(root, request)
	if err != nil {
		a.recordSQLRun(root, request.RelPath, sanitizeQueryForMetadata(request.SQL), "", 0, "", "failed", sanitizeProviderMessage(err.Error()))
		return analytics.SQLQueryResult{}, err
	}
	a.recordSQLRun(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, "", "completed", sanitizeProviderMessage(result.Message))
	a.recordDatasetDependency(root, result.RelPath, "sql-run", sanitizeQueryForMetadata(result.SQL), result.Engine, "")
	return result, nil
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
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.FreshnessStatus{}, errors.New("open a workspace before checking file changes")
	}
	current, err := workspace.SnapshotFingerprints(root)
	if err != nil {
		return workspace.FreshnessStatus{}, err
	}
	a.watchMu.Lock()
	previous := a.fingerprints
	a.fingerprints = current
	a.watchMu.Unlock()
	if previous == nil {
		return workspace.FreshnessStatus{
			Changed:        []workspace.FileChange{},
			StaleArtifacts: []string{},
			StaleDatasets:  []string{},
			Message:        "Workspace watcher baseline captured.",
		}, nil
	}

	changes := workspace.CompareFingerprints(previous, current)
	staleArtifacts := a.staleArtifactsForChanges(root, changes)
	staleDatasets := staleDatasetsForChanges(changes)
	message := "Workspace files are current."
	if len(changes) > 0 {
		message = fmt.Sprintf("%d workspace file changes detected.", len(changes))
	}
	if len(staleArtifacts) > 0 {
		message = fmt.Sprintf("%s %d artifacts may be stale.", message, len(staleArtifacts))
	}
	if len(staleDatasets) > 0 {
		message = fmt.Sprintf("%s %d dataset-derived views need refresh.", message, len(staleDatasets))
	}
	return workspace.FreshnessStatus{
		Changed:        changes,
		StaleArtifacts: staleArtifacts,
		StaleDatasets:  staleDatasets,
		Message:        message,
	}, nil
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
	root := a.getWorkspaceRoot()
	if root == "" {
		return dataset.SavedQuery{}, errors.New("open a workspace before saving dataset queries")
	}
	saved, err := dataset.SaveQuery(root, relPath, query, label)
	if err == nil {
		a.recordDatasetDependency(root, saved.RelPath, "filter-snippet", saved.Query, saved.Label, "")
	}
	return saved, err
}

func (a *App) ListDatasetQueries(relPath string) ([]dataset.SavedQuery, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return []dataset.SavedQuery{}, nil
	}
	return dataset.ListSavedQueries(root, relPath)
}

func (a *App) SaveDatasetSQLQuery(relPath string, query string, label string) (dataset.SavedQuery, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return dataset.SavedQuery{}, errors.New("open a workspace before saving SQL snippets")
	}
	saved, err := dataset.SaveQueryKind(root, relPath, query, label, "sql")
	if err == nil {
		a.recordDatasetDependency(root, saved.RelPath, "sql-snippet", saved.Query, saved.Label, "")
	}
	return saved, err
}

func (a *App) ListDatasetDependencies(relPath string) ([]appmeta.DatasetDependency, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return []appmeta.DatasetDependency{}, nil
	}
	if !appmeta.Exists(root) {
		return []appmeta.DatasetDependency{}, nil
	}
	return appmeta.ListDatasetDependencies(root, relPath)
}

func (a *App) ListDatasetSQLRuns(relPath string) ([]appmeta.SQLRun, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return []appmeta.SQLRun{}, nil
	}
	if !appmeta.Exists(root) {
		return []appmeta.SQLRun{}, nil
	}
	return appmeta.ListSQLRuns(root, relPath)
}

func (a *App) SearchMetadata(query string) ([]appmeta.MetadataSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []appmeta.MetadataSearchResult{}, nil
	}
	root := a.getWorkspaceRoot()
	if root == "" {
		return []appmeta.MetadataSearchResult{}, nil
	}
	if _, err := a.mirrorMetadataStore(root, true); err != nil {
		return []appmeta.MetadataSearchResult{}, err
	}
	return appmeta.Search(root, query, 40)
}

func (a *App) QueryWorkspaceSQLite(request dbconnector.SQLiteQueryRequest) (dbconnector.SQLiteQueryResult, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return dbconnector.SQLiteQueryResult{}, errors.New("open a workspace before querying SQLite files")
	}
	result, err := dbconnector.QuerySQLite(root, request)
	if err != nil {
		a.recordSQLRun(root, request.RelPath, sanitizeQueryForMetadata(request.SQL), "sqlite-readonly", 0, "", "failed", sanitizeProviderMessage(err.Error()))
		return dbconnector.SQLiteQueryResult{}, err
	}
	a.recordSQLRun(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, "", "completed", sanitizeProviderMessage(result.Message))
	a.recordDatasetDependency(root, result.RelPath, "sqlite-query", sanitizeQueryForMetadata(result.SQL), result.Engine, "")
	return result, nil
}

func (a *App) ListDatasetSQLQueries(relPath string) ([]dataset.SavedQuery, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return []dataset.SavedQuery{}, nil
	}
	return dataset.ListSavedQueriesKind(root, relPath, "sql")
}

func (a *App) PreviewDatasetChart(request workspace.DatasetChartRequest) (workspace.DatasetChartResult, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return workspace.DatasetChartResult{}, errors.New("open a workspace before previewing dataset charts")
	}
	return workspace.BuildCSVChart(root, request)
}

func (a *App) CreateDatasetChartArtifact(request workspace.DatasetChartRequest) (artifact.MarkdownReport, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return artifact.MarkdownReport{}, errors.New("open a workspace before creating dataset charts")
	}

	chart, err := workspace.BuildCSVChart(root, request)
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	report, err := artifact.CreateDatasetChartSVG(root, chart, time.Now())
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	a.persistArtifactMetadata(root, report.RelPath)
	a.recordDatasetDependency(root, chart.RelPath, "chart", chart.CategoryColumn, chart.ValueColumn, report.RelPath)
	a.recordApproval("artifact.chart", report.RelPath, "low", report.Message)
	return report, nil
}

func (a *App) CreateDatasetQueryArtifact(relPath string, query string) (artifact.MarkdownReport, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return artifact.MarkdownReport{}, errors.New("open a workspace before exporting dataset queries")
	}

	result, err := workspace.QueryCSV(root, relPath, query)
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	report, err := artifact.CreateDatasetQueryCSV(root, result, time.Now())
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	a.persistArtifactMetadata(root, report.RelPath)
	a.recordDatasetDependency(root, result.RelPath, "filter-export", result.Query, "", report.RelPath)
	a.recordApproval("artifact.query", report.RelPath, "low", report.Message)
	return report, nil
}

func (a *App) CreateDatasetSQLArtifact(request analytics.SQLQueryRequest) (artifact.MarkdownReport, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return artifact.MarkdownReport{}, errors.New("open a workspace before exporting SQL results")
	}
	result, err := analytics.QueryCSVSQL(root, request)
	if err != nil {
		a.recordSQLRun(root, request.RelPath, sanitizeQueryForMetadata(request.SQL), "", 0, "", "failed", sanitizeProviderMessage(err.Error()))
		return artifact.MarkdownReport{}, err
	}
	report, err := artifact.CreateDatasetSQLMarkdown(root, result, time.Now())
	if err != nil {
		a.recordSQLRun(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, "", "failed", sanitizeProviderMessage(err.Error()))
		return artifact.MarkdownReport{}, err
	}
	a.persistArtifactMetadata(root, report.RelPath)
	a.recordSQLRun(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, report.RelPath, "completed", sanitizeProviderMessage(result.Message))
	a.recordDatasetDependency(root, result.RelPath, "sql-report", sanitizeQueryForMetadata(result.SQL), result.Engine, report.RelPath)
	a.recordApproval("artifact.dataset_sql.create", report.RelPath, "medium", fmt.Sprintf("Created SQL result artifact from %s using %s.", result.RelPath, result.Engine))
	return report, nil
}

func (a *App) CreateDatasetSummaryArtifact(relPath string) (artifact.MarkdownReport, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return artifact.MarkdownReport{}, errors.New("open a workspace before creating dataset summaries")
	}

	preview, err := workspace.Preview(root, relPath, workspace.PreviewOptions{MaxBytes: 1024 * 1024})
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	report, err := artifact.CreateDatasetSummaryMarkdown(root, preview, time.Now())
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	a.persistArtifactMetadata(root, report.RelPath)
	a.recordDatasetDependency(root, relPath, "summary", "dataset summary", "", report.RelPath)
	a.recordApproval("artifact.dataset-summary", report.RelPath, "low", report.Message)
	return report, nil
}

func (a *App) RebuildDatasetDependency(id string) (artifact.MarkdownReport, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return artifact.MarkdownReport{}, errors.New("open a workspace before rebuilding dataset artifacts")
	}

	dependency, err := appmeta.GetDatasetDependency(root, id)
	if err != nil {
		return artifact.MarkdownReport{}, err
	}

	rebuildDependency := func() error {
		if strings.TrimSpace(dependency.Artifact) == "" {
			return nil
		}
		if _, err := artifact.Delete(root, dependency.Artifact); err != nil {
			var pathErr *os.PathError
			if errors.As(err, &pathErr) && errors.Is(pathErr.Err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		return nil
	}

	now := time.Now()
	switch dependency.Kind {
	case "filter-export":
		if err := rebuildDependency(); err != nil {
			return artifact.MarkdownReport{}, err
		}
		query := strings.TrimSpace(dependency.Query)
		if query == "" {
			return artifact.MarkdownReport{}, errors.New("cannot rebuild filter export without query text")
		}
		result, err := workspace.QueryCSV(root, dependency.RelPath, query)
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		report, err := artifact.CreateDatasetQueryCSV(root, result, now)
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		a.persistArtifactMetadata(root, report.RelPath)
		if _, err := appmeta.UpdateDatasetDependencyRefresh(root, dependency.ID, report.RelPath); err != nil {
			return artifact.MarkdownReport{}, err
		}
		a.recordApproval("artifact.dataset_query.rebuild", report.RelPath, "low", fmt.Sprintf("Rebuilt dataset export for %s.", dependency.RelPath))
		return report, nil

	case "sql-report":
		if err := rebuildDependency(); err != nil {
			return artifact.MarkdownReport{}, err
		}
		query := strings.TrimSpace(dependency.Query)
		if query == "" {
			return artifact.MarkdownReport{}, errors.New("cannot rebuild SQL report without SQL text")
		}
		result, err := analytics.QueryCSVSQL(root, analytics.SQLQueryRequest{
			RelPath: dependency.RelPath,
			SQL:     query,
		})
		if err != nil {
			a.recordSQLRun(root, dependency.RelPath, sanitizeQueryForMetadata(query), "unknown", 0, "", "failed", sanitizeProviderMessage(err.Error()))
			return artifact.MarkdownReport{}, err
		}
		report, err := artifact.CreateDatasetSQLMarkdown(root, result, now)
		if err != nil {
			a.recordSQLRun(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, "", "failed", sanitizeProviderMessage(err.Error()))
			return artifact.MarkdownReport{}, err
		}
		a.persistArtifactMetadata(root, report.RelPath)
		a.recordSQLRun(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, report.RelPath, "completed", sanitizeProviderMessage(result.Message))
		if _, err := appmeta.UpdateDatasetDependencyRefresh(root, dependency.ID, report.RelPath); err != nil {
			return artifact.MarkdownReport{}, err
		}
		a.recordApproval("artifact.dataset_sql.rebuild", report.RelPath, "medium", fmt.Sprintf("Rebuilt SQL result artifact for %s using %s.", result.RelPath, result.Engine))
		return report, nil

	case "chart":
		if err := rebuildDependency(); err != nil {
			return artifact.MarkdownReport{}, err
		}
		category := strings.TrimSpace(dependency.Target)
		if category == "" {
			return artifact.MarkdownReport{}, errors.New("cannot rebuild chart without a category column")
		}
		chart, err := workspace.BuildCSVChart(root, workspace.DatasetChartRequest{
			RelPath:        dependency.RelPath,
			ChartType:      "bar",
			CategoryColumn: category,
			ValueColumn:    strings.TrimSpace(dependency.Query),
		})
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		report, err := artifact.CreateDatasetChartSVG(root, chart, now)
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		a.persistArtifactMetadata(root, report.RelPath)
		if _, err := appmeta.UpdateDatasetDependencyRefresh(root, dependency.ID, report.RelPath); err != nil {
			return artifact.MarkdownReport{}, err
		}
		a.recordApproval("artifact.chart.rebuild", report.RelPath, "low", fmt.Sprintf("Rebuilt chart for %s from category %s.", dependency.RelPath, dependency.Target))
		return report, nil

	case "summary":
		if err := rebuildDependency(); err != nil {
			return artifact.MarkdownReport{}, err
		}
		preview, err := workspace.Preview(root, dependency.RelPath, workspace.PreviewOptions{MaxBytes: 1024 * 1024})
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		report, err := artifact.CreateDatasetSummaryMarkdown(root, preview, now)
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		a.persistArtifactMetadata(root, report.RelPath)
		if _, err := appmeta.UpdateDatasetDependencyRefresh(root, dependency.ID, report.RelPath); err != nil {
			return artifact.MarkdownReport{}, err
		}
		a.recordApproval("artifact.dataset-summary.rebuild", report.RelPath, "low", fmt.Sprintf("Rebuilt dataset summary for %s.", dependency.RelPath))
		return report, nil
	}

	return artifact.MarkdownReport{}, fmt.Errorf("cannot rebuild dependency kind %q", dependency.Kind)
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
	case "workspace.write":
		if mode == "dry-run" {
			return "Workspace writes must use the editor diff preview before apply.", nil
		}
		return "", errors.New("agent workspace.write execution is blocked until diff payload execution is implemented")
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
	case "artifact.create":
		if mode == "dry-run" {
			return "Ready to create a Markdown report artifact from " + target + ".", nil
		}
		report, err := a.CreateMarkdownReport(target)
		if err != nil {
			return "", err
		}
		return report.Message + " " + report.RelPath, nil
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
	info, err := os.Stat(root)
	if err != nil {
		return WorkspaceOpenResult{}, err
	}
	if !info.IsDir() {
		return WorkspaceOpenResult{}, errors.New("workspace root must be a directory")
	}

	snapshot, err := workspace.Scan(root, workspace.ScanOptions{})
	if err != nil {
		return WorkspaceOpenResult{}, err
	}

	a.setWorkspaceRoot(snapshot.Root)
	a.resetWorkspaceFreshness(snapshot.Root)
	if _, err := a.recentStore.Add(snapshot.Root); err != nil {
		return WorkspaceOpenResult{}, err
	}

	return WorkspaceOpenResult{
		Selected: true,
		Snapshot: snapshot,
	}, nil
}

func (a *App) setWorkspaceRoot(root string) {
	a.workspaceMu.Lock()
	defer a.workspaceMu.Unlock()
	a.workspaceRoot = root
}

func (a *App) getWorkspaceRoot() string {
	a.workspaceMu.RLock()
	defer a.workspaceMu.RUnlock()
	return a.workspaceRoot
}

func (a *App) resetWorkspaceFreshness(root string) {
	fingerprints, err := workspace.SnapshotFingerprints(root)
	if err != nil {
		return
	}
	a.watchMu.Lock()
	a.fingerprints = fingerprints
	a.watchMu.Unlock()
}

func (a *App) staleArtifactsForChanges(root string, changes []workspace.FileChange) []string {
	if len(changes) == 0 {
		return nil
	}
	changed := map[string]bool{}
	for _, change := range changes {
		changed[filepath.ToSlash(change.RelPath)] = true
	}
	items, err := artifact.List(root)
	if err != nil {
		return nil
	}
	stale := []string{}
	for _, item := range items {
		metadata, err := artifact.Metadata(root, item.RelPath)
		if err != nil {
			continue
		}
		sourcePaths := append([]string{}, metadata.SourcePaths...)
		if metadata.ContextRelPath != "" {
			sourcePaths = append(sourcePaths, metadata.ContextRelPath)
		}
		for _, sourcePath := range sourcePaths {
			if changed[filepath.ToSlash(sourcePath)] {
				stale = append(stale, item.RelPath)
				break
			}
		}
	}
	return stale
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
