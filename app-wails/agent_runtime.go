package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"NexusAugenticStudio/internal/agent"
	"NexusAugenticStudio/internal/agenttools"
	"NexusAugenticStudio/internal/analytics"
	"NexusAugenticStudio/internal/dataset"
	"NexusAugenticStudio/internal/dbconnector"
	"NexusAugenticStudio/internal/gitservice"
	"NexusAugenticStudio/internal/webfetch"
	"NexusAugenticStudio/internal/workspace"
)

const (
	maxAgentShellOutputBytes  = 12000
	maxAgentContextOutputSize = 24000
)

func (a *App) RunAgent(request agent.RunRequest) (agent.RunResult, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return agent.RunResult{}, errors.New("open a workspace before running the agent")
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	runner := agent.New(a.llmClient, a.llmStore)
	return runner.Run(ctx, request, func(ctx context.Context, call agent.ToolCall, request agent.RunRequest) (agent.ToolCall, error) {
		return a.executeAgentRuntimeTool(ctx, root, call, request)
	}, func(event agent.RunEvent) {
		emitChatStreamEventFn(ctx, agentRunEventName, event)
	})
}

func (a *App) AgentSystemPrompt() string {
	return agent.SystemPrompt()
}

func (a *App) executeAgentRuntimeTool(ctx context.Context, root string, call agent.ToolCall, request agent.RunRequest) (agent.ToolCall, error) {
	call.Name = strings.TrimSpace(call.Name)
	if call.Arguments == nil {
		call.Arguments = map[string]string{}
	}

	if descriptor, ok := agenttools.Find(call.Name); ok {
		call.Risk = descriptor.Risk
		record, err := a.runAgentTool(root, agenttools.RunRequest{
			ToolName: call.Name,
			Target:   agentToolTarget(call),
			Inputs:   call.Arguments,
			Approved: request.ApproveHighImpact,
		}, "execute")
		call.Observation = record.OutputSummary
		call.Error = record.Error
		if appendErr := a.appendToolRun(root, record); appendErr != nil && err == nil {
			err = appendErr
			call.Error = appendErr.Error()
		}
		return call, err
	}

	switch call.Name {
	case "list_directory":
		call.Risk = "low"
		return a.agentListDirectory(root, call)
	case "read_file":
		call.Risk = "low"
		return a.agentReadFile(root, call)
	case "search_files":
		call.Risk = "low"
		return a.agentSearchFiles(root, call)
	case "read_context":
		call.Risk = "low"
		return a.agentReadContext(root, call)
	case "read_git_diff":
		call.Risk = "low"
		return a.agentReadGitDiff(call)
	case "read_changed_files":
		call.Risk = "low"
		return a.agentReadChangedFiles(root, call)
	case "read_git_history":
		call.Risk = "low"
		return a.agentReadGitHistory(call)
	case "read_git_blame":
		call.Risk = "low"
		return a.agentReadGitBlame(call)
	case "read_problems":
		call.Risk = "low"
		return a.agentReadProblems(root, call)
	case "list_tasks":
		call.Risk = "low"
		return a.agentListTasks(root, call)
	case "list_artifacts":
		call.Risk = "low"
		return a.agentListArtifacts(call)
	case "read_artifact":
		call.Risk = "low"
		return a.agentReadArtifact(root, call)
	case "read_artifact_lineage":
		call.Risk = "low"
		return a.agentReadArtifactLineage(call)
	case "web_fetch":
		call.Risk = "medium"
		return a.agentWebFetch(ctx, call, request.ApproveHighImpact)
	case "list_datasets":
		call.Risk = "low"
		return a.agentListDatasets(call)
	case "profile_dataset":
		call.Risk = "low"
		return a.agentProfileDataset(call)
	case "query_dataset":
		call.Risk = "low"
		return a.agentQueryDataset(call)
	case "query_dataset_sql":
		call.Risk = "low"
		return a.agentQueryDatasetSQL(call)
	case "inspect_sqlite":
		call.Risk = "low"
		return a.agentInspectSQLite(call)
	case "query_sqlite":
		call.Risk = "low"
		return a.agentQuerySQLite(call)
	case "inspect_operations":
		call.Risk = "low"
		return a.agentInspectOperations(root, call)
	case "read_document_set":
		call.Risk = "low"
		return a.agentReadDocumentSet(root, call)
	case "write_file":
		call.Risk = "high"
		return a.agentWriteFile(root, call, request.ApproveHighImpact)
	case "write_binary_file":
		call.Risk = "high"
		return a.agentWriteBinaryFile(root, call, request.ApproveHighImpact)
	case "apply_patch":
		call.Risk = "high"
		return a.agentApplyPatch(root, call, request.ApproveHighImpact)
	case "append_file":
		call.Risk = "high"
		return a.agentAppendFile(root, call, request.ApproveHighImpact)
	case "copy_file":
		call.Risk = "high"
		return a.agentCopyFile(root, call, request.ApproveHighImpact)
	case "move_file":
		call.Risk = "high"
		return a.agentMoveFile(root, call, request.ApproveHighImpact)
	case "delete_file":
		call.Risk = "high"
		return a.agentDeleteFile(root, call, request.ApproveHighImpact)
	case "list_rollbacks":
		call.Risk = "low"
		return a.agentListRollbacks(root, call)
	case "rollback_file_mutation":
		call.Risk = "high"
		return a.agentRollbackFileMutation(root, call, request.ApproveHighImpact)
	case "run_task":
		call.Risk = "high"
		return a.agentRunTask(call, request.ApproveHighImpact)
	case "execute_shell_command":
		call.Risk = "high"
		return a.agentExecuteShell(ctx, root, call, request)
	case "analyze_csv_excel":
		call.Risk = "low"
		return a.agentAnalyzeDataset(root, call)
	case "generate_artifact":
		call.Risk = "low"
		return a.agentGenerateArtifact(call)
	case "update_plan":
		call.Risk = "low"
		call.Observation = "Plan updated."
		return call, nil
	default:
		call.Error = "agent tool is not registered"
		return call, errors.New(call.Error)
	}
}

func (a *App) agentListDirectory(root string, call agent.ToolCall) (agent.ToolCall, error) {
	relPath := cleanAgentRelPath(call.Arguments["relPath"])
	recursive := parseAgentBool(call.Arguments["recursive"])
	maxDepth := parseAgentInt(call.Arguments["maxDepth"], 1)
	if recursive && maxDepth < 2 {
		maxDepth = 3
	}
	snapshot, err := workspace.Scan(root, workspace.ScanOptions{MaxDepth: 10, MaxEntries: 800})
	if err != nil {
		call.Error = err.Error()
		return call, err
	}

	prefix := strings.Trim(relPath, "/")
	lines := []string{}
	for _, node := range snapshot.Nodes {
		nodeRel := filepath.ToSlash(node.RelPath)
		if prefix != "" {
			if nodeRel == prefix {
				continue
			}
			if !strings.HasPrefix(nodeRel, prefix+"/") {
				continue
			}
		}
		remainder := strings.TrimPrefix(nodeRel, prefix)
		remainder = strings.TrimPrefix(remainder, "/")
		depth := strings.Count(remainder, "/") + 1
		if remainder == "" || (!recursive && depth > 1) || (recursive && maxDepth > 0 && depth > maxDepth) {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s\t%s\t%s", node.Kind, nodeRel, node.Meta))
		if len(lines) >= 120 {
			lines = append(lines, "[truncated]")
			break
		}
	}
	if len(lines) == 0 {
		call.Observation = "No indexed entries found."
		return call, nil
	}
	call.Observation = strings.Join(lines, "\n")
	return call, nil
}

func (a *App) agentReadFile(root string, call agent.ToolCall) (agent.ToolCall, error) {
	relPath := cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"]))
	preview, err := workspace.Preview(root, relPath, workspace.PreviewOptions{MaxBytes: chatContextFallbackMaxBytes})
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	content := preview.Content
	if content == "" {
		content = preview.Text
	}
	call.Observation = fmt.Sprintf("File: %s\nKind: %s\nType: %s\nEncoding: %s\nSize: %d\nMessage: %s\n\n%s", preview.RelPath, preview.Kind, preview.FileType, preview.Encoding, preview.Size, preview.Message, content)
	return call, nil
}

func (a *App) agentSearchFiles(root string, call agent.ToolCall) (agent.ToolCall, error) {
	query := strings.TrimSpace(firstNonEmpty(call.Arguments["query"], call.Arguments["pattern"]))
	if query == "" {
		call.Error = "search query is required"
		return call, errors.New(call.Error)
	}
	results, err := workspace.Search(root, query, workspace.SearchOptions{MaxResults: 50})
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	lines := make([]string, 0, len(results))
	for _, result := range results {
		lines = append(lines, fmt.Sprintf("%s:%d [%s] %s", result.RelPath, result.Line, result.MatchType, result.Snippet))
	}
	call.Observation = firstNonEmpty(strings.Join(lines, "\n"), "No matches.")
	return call, nil
}

func (a *App) agentReadContext(_ string, call agent.ToolCall) (agent.ToolCall, error) {
	paths := parseAgentRelPaths(
		firstNonEmpty(call.Arguments["relPaths"], call.Arguments["paths"], call.Arguments["path"], call.Arguments["relPath"]),
	)
	if len(paths) == 0 {
		paths = []string{"."}
	}
	settings, err := a.llmStore.Get()
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	settings, err = a.llmStore.ResolveForUse(settings)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	label, content, sourcePaths, err := a.buildContextPack(paths, settings)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = fmt.Sprintf("Context pack: %s\nSources: %s\n\n%s", label, strings.Join(sourcePaths, ", "), content)
	return call, nil
}

func (a *App) agentReadGitDiff(call agent.ToolCall) (agent.ToolCall, error) {
	service := gitservice.New(a.getWorkspaceRoot)
	relPath := cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"], call.Arguments["target"]))
	if relPath != "" {
		diff, err := service.FileDiff(relPath)
		if err != nil {
			call.Error = err.Error()
			return call, err
		}
		call.Observation = limitAgentOutput(formatGitFileDiffObservation(diff), maxAgentContextOutputSize)
		return call, nil
	}

	status, err := service.Status()
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = limitAgentOutput(formatGitStatusObservation(status), maxAgentContextOutputSize)
	return call, nil
}

func (a *App) agentReadChangedFiles(root string, call agent.ToolCall) (agent.ToolCall, error) {
	maxFiles := parseAgentInt(call.Arguments["maxFiles"], 12)
	if maxFiles <= 0 || maxFiles > 40 {
		maxFiles = 12
	}
	includeContent := parseAgentBool(firstNonEmpty(call.Arguments["includeContent"], "true"))
	status, err := gitservice.New(a.getWorkspaceRoot).Status()
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	if !status.Available {
		call.Observation = status.Message
		return call, nil
	}
	lines := []string{
		fmt.Sprintf("Changed files: %d (%d staged, %d unstaged)", len(status.ChangedFiles), len(status.StagedFiles), len(status.UnstagedFiles)),
		status.Message,
	}
	for index, change := range status.ChangedFiles {
		if index >= maxFiles {
			lines = append(lines, fmt.Sprintf("Skipped %d additional changed file(s) by maxFiles cap.", len(status.ChangedFiles)-index))
			break
		}
		lines = append(lines, fmt.Sprintf("\n## %s\nStatus: %s [%s%s]", change.Path, change.Summary, change.Index, change.Worktree))
		if change.OldPath != "" {
			lines = append(lines, "Old path: "+change.OldPath)
		}
		if !includeContent || gitChangeLooksDeleted(change) {
			continue
		}
		preview, previewErr := workspace.Preview(root, change.Path, workspace.PreviewOptions{MaxBytes: 4 * 1024})
		if previewErr != nil {
			lines = append(lines, "Preview unavailable: "+previewErr.Error())
			continue
		}
		content := firstNonEmpty(preview.Text, preview.Content)
		if strings.TrimSpace(content) == "" {
			lines = append(lines, fmt.Sprintf("Preview: %s (%s, %d bytes)", preview.Message, preview.Kind, preview.Size))
			continue
		}
		lines = append(lines, fmt.Sprintf("Preview: %s (%s, %s, %d bytes)\n%s", preview.Message, preview.Kind, preview.Encoding, preview.Size, content))
	}
	call.Observation = limitAgentOutput(strings.Join(lines, "\n"), maxAgentContextOutputSize)
	return call, nil
}

func (a *App) agentReadGitHistory(call agent.ToolCall) (agent.ToolCall, error) {
	request := GitHistoryRequest{
		Path:  cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"], call.Arguments["target"])),
		Limit: parseAgentInt(call.Arguments["limit"], gitservice.DefaultHistoryLimit),
	}
	result, err := gitservice.New(a.getWorkspaceRoot).History(request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = limitAgentOutput(formatGitHistoryObservation(result), maxAgentContextOutputSize)
	return call, nil
}

func (a *App) agentReadGitBlame(call agent.ToolCall) (agent.ToolCall, error) {
	request := GitBlameRequest{
		Path:      cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"], call.Arguments["target"])),
		StartLine: parseAgentInt(firstNonEmpty(call.Arguments["startLine"], call.Arguments["start"]), 0),
		EndLine:   parseAgentInt(firstNonEmpty(call.Arguments["endLine"], call.Arguments["end"]), 0),
	}
	result, err := gitservice.New(a.getWorkspaceRoot).Blame(request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = limitAgentOutput(formatGitBlameObservation(result), maxAgentContextOutputSize)
	return call, nil
}

func (a *App) agentReadProblems(root string, call agent.ToolCall) (agent.ToolCall, error) {
	maxResults := parseAgentInt(call.Arguments["maxResults"], 40)
	if maxResults <= 0 || maxResults > 120 {
		maxResults = 40
	}
	summary, err := workspace.ScanProblems(root, maxResults)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = formatProblemSummaryObservation(summary)
	return call, nil
}

func (a *App) agentListTasks(root string, call agent.ToolCall) (agent.ToolCall, error) {
	summary, err := discoverWorkspaceTasks(root)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = formatTaskSummaryObservation(summary)
	return call, nil
}

func (a *App) agentRunTask(call agent.ToolCall, approved bool) (agent.ToolCall, error) {
	taskID := strings.TrimSpace(firstNonEmpty(call.Arguments["taskId"], call.Arguments["id"]))
	if taskID == "" {
		call.Error = "task id is required"
		return call, errors.New(call.Error)
	}
	if !approved {
		call.Observation = "Approval required before running discovered workspace task: " + taskID
		call.Error = "approval required"
		return call, errors.New(call.Error)
	}
	result, err := a.RunWorkspaceTask(WorkspaceTaskRunRequest{TaskID: taskID})
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = formatTaskRunObservation(result)
	return call, nil
}

func (a *App) agentListArtifacts(call agent.ToolCall) (agent.ToolCall, error) {
	items, err := a.artifactSvc.List()
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = formatArtifactListObservation(items)
	return call, nil
}

func (a *App) agentReadArtifact(root string, call agent.ToolCall) (agent.ToolCall, error) {
	relPath := cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"], call.Arguments["target"]))
	if relPath == "" {
		call.Error = "artifact path is required"
		return call, errors.New(call.Error)
	}
	metadata, err := a.artifactSvc.Metadata(relPath)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	preview, err := workspace.Preview(root, relPath, workspace.PreviewOptions{MaxBytes: maxAgentContextOutputSize})
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = formatArtifactObservation(relPath, metadata, preview)
	return call, nil
}

func (a *App) agentReadArtifactLineage(call agent.ToolCall) (agent.ToolCall, error) {
	lineage, err := a.GetArtifactLineage()
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = formatArtifactLineageObservation(lineage)
	return call, nil
}

func (a *App) agentWebFetch(ctx context.Context, call agent.ToolCall, approved bool) (agent.ToolCall, error) {
	targetURL := strings.TrimSpace(firstNonEmpty(call.Arguments["url"], call.Arguments["href"], call.Arguments["target"]))
	if targetURL == "" {
		call.Error = "URL is required"
		return call, errors.New(call.Error)
	}
	allowedDomains := parseAgentList(call.Arguments["allowedDomains"])
	allowLocal := parseAgentBool(call.Arguments["allowLocal"])
	maxBytes := parseAgentInt(call.Arguments["maxBytes"], 128*1024)
	if !approved {
		call.Observation = "Approval required before fetching external web content: " + targetURL
		call.Error = "approval required"
		return call, errors.New(call.Error)
	}
	result, err := webfetch.Fetch(ctx, webfetch.Request{
		URL:            targetURL,
		AllowedDomains: allowedDomains,
		AllowLocal:     allowLocal,
		MaxBytes:       maxBytes,
	})
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	a.recordApproval("agent.web_fetch", result.FinalURL, "medium", result.Message)
	call.Observation = formatWebFetchObservation(result)
	return call, nil
}

func (a *App) agentListDatasets(call agent.ToolCall) (agent.ToolCall, error) {
	profiles, err := a.datasetSvc.ListProfiles()
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = formatDatasetProfilesObservation(profiles)
	return call, nil
}

func (a *App) agentProfileDataset(call agent.ToolCall) (agent.ToolCall, error) {
	relPath := cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"], call.Arguments["target"]))
	if relPath == "" {
		call.Error = "dataset path is required"
		return call, errors.New(call.Error)
	}
	profile, err := a.datasetSvc.Profile(relPath)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = formatDatasetProfileObservation(profile)
	return call, nil
}

func (a *App) agentQueryDataset(call agent.ToolCall) (agent.ToolCall, error) {
	relPath := cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"], call.Arguments["target"]))
	query := strings.TrimSpace(call.Arguments["query"])
	if relPath == "" {
		call.Error = "dataset path is required"
		return call, errors.New(call.Error)
	}
	result, err := a.datasetSvc.Query(relPath, query)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = formatDatasetQueryObservation(result)
	return call, nil
}

func (a *App) agentQueryDatasetSQL(call agent.ToolCall) (agent.ToolCall, error) {
	request := analytics.SQLQueryRequest{
		RelPath: cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"], call.Arguments["target"])),
		SQL:     strings.TrimSpace(firstNonEmpty(call.Arguments["sql"], call.Arguments["query"])),
	}
	if request.RelPath == "" {
		call.Error = "dataset path is required"
		return call, errors.New(call.Error)
	}
	result, err := a.datasetSvc.QuerySQL(request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = formatSQLQueryObservation(result)
	return call, nil
}

func (a *App) agentInspectSQLite(call agent.ToolCall) (agent.ToolCall, error) {
	relPath := cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"], call.Arguments["target"]))
	if relPath == "" {
		call.Error = "SQLite database path is required"
		return call, errors.New(call.Error)
	}
	metadata, err := a.datasetSvc.InspectWorkspaceSQLite(relPath)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = formatSQLiteMetadataObservation(metadata)
	return call, nil
}

func (a *App) agentQuerySQLite(call agent.ToolCall) (agent.ToolCall, error) {
	request := dbconnector.SQLiteQueryRequest{
		RelPath:        cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"], call.Arguments["target"])),
		SQL:            strings.TrimSpace(firstNonEmpty(call.Arguments["sql"], call.Arguments["query"])),
		ResultLimit:    parseAgentInt(call.Arguments["resultLimit"], 100),
		TimeoutSeconds: parseAgentInt(call.Arguments["timeoutSeconds"], 30),
	}
	if request.RelPath == "" {
		call.Error = "SQLite database path is required"
		return call, errors.New(call.Error)
	}
	result, err := a.datasetSvc.QueryWorkspaceSQLite(request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = formatSQLiteQueryObservation(result)
	return call, nil
}

func (a *App) agentInspectOperations(root string, call agent.ToolCall) (agent.ToolCall, error) {
	relPath := cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"], call.Arguments["target"]))
	if relPath == "" {
		call.Error = "operations file path is required"
		return call, errors.New(call.Error)
	}
	preview, err := workspace.Preview(root, relPath, workspace.PreviewOptions{MaxBytes: maxAgentContextOutputSize})
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	content := firstNonEmpty(preview.Text, preview.Content)
	if isEnvironmentLikePath(preview.RelPath, preview.Name) {
		content = redactEnvironmentContent(content)
	}
	lines := []string{
		"Operations file: " + preview.RelPath,
		"Type: " + preview.FileType,
		"Encoding: " + preview.Encoding,
		fmt.Sprintf("Size: %d truncated=%t", preview.Size, preview.Truncated),
		"Message: " + preview.Message,
	}
	if content == "" {
		lines = append(lines, "Content is not previewable as text.")
	} else {
		lines = append(lines, "\nContent:\n"+content)
	}
	call.Observation = limitAgentOutput(strings.Join(lines, "\n"), maxAgentContextOutputSize)
	return call, nil
}

func (a *App) agentReadDocumentSet(root string, call agent.ToolCall) (agent.ToolCall, error) {
	paths := parseAgentRelPaths(firstNonEmpty(call.Arguments["relPaths"], call.Arguments["paths"], call.Arguments["path"], call.Arguments["relPath"]))
	if len(paths) == 0 {
		paths = []string{"."}
	}
	maxFiles := parseAgentInt(call.Arguments["maxFiles"], 16)
	if maxFiles <= 0 || maxFiles > 48 {
		maxFiles = 16
	}
	docPaths, truncated, err := collectAgentDocumentPaths(root, paths, maxFiles)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	if len(docPaths) == 0 {
		call.Observation = "No previewable document files found in requested paths."
		return call, nil
	}
	lines := []string{fmt.Sprintf("Document set: %d file(s). Roots: %s", len(docPaths), strings.Join(paths, ", "))}
	if truncated {
		lines = append(lines, "Document set was truncated by maxFiles or scan caps.")
	}
	for _, relPath := range docPaths {
		preview, previewErr := workspace.Preview(root, relPath, workspace.PreviewOptions{MaxBytes: 8 * 1024})
		lines = append(lines, "\n## "+relPath)
		if previewErr != nil {
			lines = append(lines, "Preview unavailable: "+previewErr.Error())
			continue
		}
		content := firstNonEmpty(preview.Text, preview.Content)
		lines = append(lines, fmt.Sprintf("Kind: %s Type: %s Encoding: %s Size: %d Truncated: %t", preview.Kind, preview.FileType, preview.Encoding, preview.Size, preview.Truncated))
		lines = append(lines, "Message: "+preview.Message)
		if strings.TrimSpace(content) == "" {
			lines = append(lines, "No extractable text content.")
			continue
		}
		lines = append(lines, "\n"+content)
	}
	call.Observation = limitAgentOutput(strings.Join(lines, "\n"), maxAgentContextOutputSize)
	return call, nil
}

func (a *App) agentAnalyzeDataset(root string, call agent.ToolCall) (agent.ToolCall, error) {
	relPath := cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"]))
	query := strings.TrimSpace(call.Arguments["query"])
	if query != "" {
		result, err := workspace.QueryCSV(root, relPath, query)
		if err != nil {
			call.Error = err.Error()
			return call, err
		}
		call.Observation = fmt.Sprintf("%s\nRows: %d matched of %d\nColumns: %s", result.Message, result.MatchedRows, result.TotalRows, strings.Join(result.Columns, ", "))
		return call, nil
	}
	profile, err := dataset.Build(root, relPath)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = fmt.Sprintf("%s\nRows: %d\nColumns: %d\nSheets: %s\nFormulas: %d\nTables: %d\nNamed ranges: %d\nPivots: %d",
		profile.Message,
		profile.Rows,
		profile.Columns,
		strings.Join(profile.Sheets, ", "),
		profile.Workbook.FormulaCount,
		len(profile.Workbook.TableRanges),
		len(profile.Workbook.NamedRanges),
		len(profile.Workbook.PivotTables),
	)
	return call, nil
}

func (a *App) agentGenerateArtifact(call agent.ToolCall) (agent.ToolCall, error) {
	sourcePath := cleanAgentRelPath(firstNonEmpty(call.Arguments["sourcePath"], call.Arguments["relPath"]))
	report, err := a.CreateMarkdownReport(sourcePath)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = report.Message + " " + report.RelPath
	return call, nil
}

func agentToolTarget(call agent.ToolCall) string {
	for _, key := range []string{"relPath", "sourcePath", "path", "target"} {
		if value := strings.TrimSpace(call.Arguments[key]); value != "" {
			return cleanAgentRelPath(value)
		}
	}
	return ""
}

func cleanAgentRelPath(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	value = filepath.ToSlash(value)
	value = strings.TrimPrefix(value, "/")
	if value == "." {
		return ""
	}
	return value
}

func parseAgentBool(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "true" || value == "1" || value == "yes"
}

func parseAgentInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseAgentRelPaths(value string) []string {
	return parseAgentList(value)
}

func parseAgentList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	value = strings.Trim(value, "[]")
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t'
	})
	paths := []string{}
	seen := map[string]bool{}
	for _, part := range parts {
		path := cleanAgentRelPath(part)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	return paths
}

func gitChangeLooksDeleted(change GitFileChange) bool {
	return strings.Contains(change.Index, "D") || strings.Contains(change.Worktree, "D") || strings.Contains(strings.ToLower(change.Summary), "deleted")
}

func collectAgentDocumentPaths(root string, paths []string, maxFiles int) ([]string, bool, error) {
	snapshot, err := workspace.Scan(root, workspace.ScanOptions{MaxDepth: 10, MaxEntries: 1200})
	if err != nil {
		return nil, false, err
	}
	seen := map[string]bool{}
	docPaths := []string{}
	truncated := snapshot.Truncated
	appendDoc := func(relPath string) {
		relPath = cleanAgentRelPath(relPath)
		if relPath == "" || seen[relPath] || !isDocumentContextPath(relPath) || len(docPaths) >= maxFiles {
			return
		}
		seen[relPath] = true
		docPaths = append(docPaths, relPath)
	}
	for _, rawPath := range paths {
		cleanPath := cleanAgentRelPath(rawPath)
		if cleanPath == "" {
			cleanPath = "."
		}
		if cleanPath != "." && isDocumentContextPath(cleanPath) {
			appendDoc(cleanPath)
			continue
		}
		before := len(docPaths)
		for _, node := range snapshot.Nodes {
			if node.Kind != "file" || !pathIsInsideAgentRoot(node.RelPath, cleanPath) || !isDocumentContextPath(node.RelPath) {
				continue
			}
			if len(docPaths) >= maxFiles {
				truncated = true
				break
			}
			appendDoc(node.RelPath)
		}
		if len(docPaths) == before && cleanPath != "." && !isDocumentContextPath(cleanPath) {
			continue
		}
		if len(docPaths) >= maxFiles {
			break
		}
	}
	return docPaths, truncated, nil
}

func pathIsInsideAgentRoot(relPath string, rootRelPath string) bool {
	if rootRelPath == "." || rootRelPath == "" {
		return true
	}
	return relPath == rootRelPath || strings.HasPrefix(relPath, strings.TrimSuffix(rootRelPath, "/")+"/")
}

func isDocumentContextPath(relPath string) bool {
	ext := strings.ToLower(filepath.Ext(relPath))
	switch ext {
	case ".md", ".markdown", ".txt", ".rtf", ".pdf", ".docx", ".xml", ".html", ".htm":
		return true
	default:
		return false
	}
}

func isEnvironmentLikePath(relPath string, name string) bool {
	lowerPath := strings.ToLower(filepath.ToSlash(relPath))
	lowerName := strings.ToLower(name)
	return lowerName == ".env" ||
		strings.HasPrefix(lowerName, ".env.") ||
		strings.HasSuffix(lowerName, ".env") ||
		strings.Contains(lowerPath, "/.env") ||
		strings.HasSuffix(lowerName, ".env.local") ||
		strings.HasSuffix(lowerName, ".env.production") ||
		strings.HasSuffix(lowerName, ".env.development")
}

func redactEnvironmentContent(content string) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if shouldRedactEnvironmentKey(key) && strings.TrimSpace(value) != "" {
			lines[index] = key + "=<redacted>"
		}
	}
	return strings.Join(lines, "\n")
}

func shouldRedactEnvironmentKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	for _, marker := range []string{"secret", "token", "password", "passwd", "apikey", "api_key", "private", "credential", "dsn", "connection"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func limitAgentOutput(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes] + "\n[truncated]"
}
