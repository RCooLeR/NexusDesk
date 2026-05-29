package tools

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"nexusdesk/internal/services/agent"
	artifactsSvc "nexusdesk/internal/services/artifacts"
	externalagentsSvc "nexusdesk/internal/services/externalagents"
	gitsvc "nexusdesk/internal/services/git"
	taskssvc "nexusdesk/internal/services/tasks"
	webfetchSvc "nexusdesk/internal/services/webfetch"
	workspacesvc "nexusdesk/internal/services/workspace"
)

type Dependencies struct {
	Workspace *workspacesvc.Service
	Git       *gitsvc.Service
	Tasks     *taskssvc.Service

	ExternalAgentLookupPath func(string) (string, error)
}

func NewDefaultDispatcher(deps Dependencies) *Dispatcher {
	if deps.Workspace == nil {
		deps.Workspace = workspacesvc.New()
	}
	if deps.Git == nil {
		deps.Git = gitsvc.New()
	}
	if deps.Tasks == nil {
		deps.Tasks = taskssvc.New()
	}
	handlers := defaultHandlers{deps: deps}
	return NewDispatcher(
		Tool{Descriptor: agent.ToolDescriptor{Name: "read_context", Description: "Build a bounded context pack for a file, directory, or project root.", Risk: "low", Inputs: "relPath"}, Handler: handlers.readContext},
		Tool{Descriptor: agent.ToolDescriptor{Name: "read_file", Description: "Read a bounded preview of one workspace file.", Risk: "low", Inputs: "relPath"}, Handler: handlers.readFile},
		Tool{Descriptor: agent.ToolDescriptor{Name: "search_workspace", Description: "Search workspace paths and previewable text content.", Risk: "low", Inputs: "query, regex(optional)"}, Handler: handlers.searchWorkspace},
		Tool{Descriptor: agent.ToolDescriptor{Name: "read_problems", Description: "Run lightweight TODO/FIXME/HACK/BUG, merge-conflict, and JSON diagnostics.", Risk: "low", Inputs: "maxResults(optional)"}, Handler: handlers.readProblems},
		Tool{Descriptor: agent.ToolDescriptor{Name: "read_git_status", Description: "Read manual Git status for the active workspace.", Risk: "low", Inputs: ""}, Handler: handlers.readGitStatus},
		Tool{Descriptor: agent.ToolDescriptor{Name: "read_git_diff", Description: "Read a bounded staged/unstaged diff for one changed file.", Risk: "low", Inputs: "relPath"}, Handler: handlers.readGitDiff},
		Tool{Descriptor: agent.ToolDescriptor{Name: "read_git_history", Description: "Read bounded Git commit history for the repository or one file.", Risk: "low", Inputs: "relPath(optional), limit(optional)"}, Handler: handlers.readGitHistory},
		Tool{Descriptor: agent.ToolDescriptor{Name: "read_git_blame", Description: "Read bounded Git blame lines for one file.", Risk: "low", Inputs: "relPath, startLine(optional), endLine(optional)"}, Handler: handlers.readGitBlame},
		Tool{Descriptor: agent.ToolDescriptor{Name: "list_external_agent_tools", Description: "List optional external coding-agent CLIs detected on PATH, such as Codex, Claude Code, and OpenCode. This is detection-only; execution requires a future approved job/shell integration.", Risk: "low", Inputs: ""}, Handler: handlers.listExternalAgentTools},
		Tool{Descriptor: agent.ToolDescriptor{Name: "plan_external_agent_run", Description: "Plan a future approved external coding-agent CLI run without executing it. Produces job kind, working directory, prompt delivery, audit, approval, cancellation, and output-capture requirements.", Risk: "low", Inputs: "toolID, prompt"}, Handler: handlers.planExternalAgentRun},
		Tool{Descriptor: agent.ToolDescriptor{Name: "read_artifact_lineage", Description: "Read the workspace artifact lineage graph with generated artifacts, sources, jobs, and task relationships.", Risk: "low", Inputs: "query(optional), includeArchived(optional)"}, Handler: handlers.readArtifactLineage},
		Tool{Descriptor: agent.ToolDescriptor{Name: "regenerate_artifact", Description: "Regenerate one supported native artifact from saved source/dependency metadata into a new artifact file.", Risk: "high", Inputs: "relPath"}, Handler: handlers.regenerateArtifact},
		Tool{Descriptor: agent.ToolDescriptor{Name: "web_fetch", Description: "Fetch one approved HTTP(S) text-like URL with redirect, size, content-type, local-network, and optional domain allow-list guards.", Risk: "medium", Inputs: "url, allowedDomains(optional), allowLocal(optional), maxBytes(optional)"}, Handler: handlers.webFetch},
		Tool{Descriptor: agent.ToolDescriptor{Name: "list_tasks", Description: "List safe discovered workspace tasks.", Risk: "low", Inputs: ""}, Handler: handlers.listTasks},
		Tool{Descriptor: agent.ToolDescriptor{Name: "run_task", Description: "Run a discovered safe workspace task when shell approval is granted.", Risk: "high", Inputs: "taskId"}, Handler: handlers.runTask},
		Tool{Descriptor: agent.ToolDescriptor{Name: "run_terminal_command", Description: "Run one approved terminal command by executable name plus explicit JSON args, rooted inside the workspace, with timeout and output caps. Shell interpreters and command paths are blocked.", Risk: "high", Inputs: "command, argsJson(optional), cwd(optional), timeoutSeconds(optional)"}, Handler: handlers.runTerminalCommand},
		Tool{Descriptor: agent.ToolDescriptor{Name: "write_file", Description: "Create or replace a text/code file inside the workspace through safe write validation and rollback.", Risk: "high", Inputs: "relPath, content, encoding(optional)"}, Handler: handlers.writeFile},
		Tool{Descriptor: agent.ToolDescriptor{Name: "append_file", Description: "Append text to a workspace file through safe append validation and rollback.", Risk: "high", Inputs: "relPath, content, encoding(optional)"}, Handler: handlers.appendFile},
		Tool{Descriptor: agent.ToolDescriptor{Name: "copy_file", Description: "Copy one workspace file to a new path through safe path validation and rollback.", Risk: "high", Inputs: "sourceRelPath, targetRelPath"}, Handler: handlers.copyFile},
		Tool{Descriptor: agent.ToolDescriptor{Name: "move_file", Description: "Move or rename one workspace file through safe path validation and rollback.", Risk: "high", Inputs: "sourceRelPath, targetRelPath"}, Handler: handlers.moveFile},
		Tool{Descriptor: agent.ToolDescriptor{Name: "delete_file", Description: "Delete one workspace file through safe path validation and rollback.", Risk: "high", Inputs: "relPath"}, Handler: handlers.deleteFile},
		Tool{Descriptor: agent.ToolDescriptor{Name: "apply_patch", Description: "Apply an exact-match unified diff to one or more safe text files with one rollback snapshot.", Risk: "high", Inputs: "patch"}, Handler: handlers.applyPatch},
		Tool{Descriptor: agent.ToolDescriptor{Name: "list_rollbacks", Description: "List rollback snapshots for approved workspace file mutations.", Risk: "low", Inputs: ""}, Handler: handlers.listRollbacks},
		Tool{Descriptor: agent.ToolDescriptor{Name: "rollback_file_mutation", Description: "Restore or remove files from one rollback snapshot when write approval is granted.", Risk: "high", Inputs: "id"}, Handler: handlers.rollbackFileMutation},
	)
}

type defaultHandlers struct {
	deps Dependencies
}

func (h defaultHandlers) readContext(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	relPath := firstArg(call, "relPath", "path")
	if relPath == "" {
		relPath = "."
	}
	pack, err := h.deps.Workspace.BuildContextPack(root, []string{relPath}, workspacesvc.ContextPackOptions{MaxBytes: 96 * 1024})
	if err != nil {
		return toolError(call, "low", err), err
	}
	return toolOK(call, "low", fmt.Sprintf("%s\nSources: %s\n\n%s", pack.Message, strings.Join(pack.SourcePaths, ", "), pack.Content)), nil
}

func (h defaultHandlers) readFile(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	relPath := firstArg(call, "relPath", "path")
	if relPath == "" {
		err := errors.New("relPath is required")
		return toolError(call, "low", err), err
	}
	preview, err := h.deps.Workspace.PreviewFile(root, relPath)
	if err != nil {
		return toolError(call, "low", err), err
	}
	text := preview.Text
	if strings.TrimSpace(text) == "" {
		text = fmt.Sprintf("No inline text available for %s preview.", preview.Kind)
	}
	return toolOK(call, "low", fmt.Sprintf("File: %s\nKind: %s\nMedia: %s\nEncoding: %s\nSize: %d\n\n%s", preview.RelPath, preview.Kind, preview.MediaType, preview.Encoding, preview.Size, text)), nil
}

func (h defaultHandlers) searchWorkspace(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	query := firstArg(call, "query", "q")
	if query == "" {
		err := errors.New("query is required")
		return toolError(call, "low", err), err
	}
	results, err := h.deps.Workspace.Search(root, query, workspacesvc.SearchOptions{Regex: boolArg(call, "regex")})
	if err != nil {
		return toolError(call, "low", err), err
	}
	lines := []string{fmt.Sprintf("%d result(s) for %q.", len(results), query)}
	for index, result := range results {
		if index >= 20 {
			lines = append(lines, "[results truncated]")
			break
		}
		lines = append(lines, fmt.Sprintf("- %s:%d [%s] %s", result.RelPath, result.Line, result.MatchType, result.Snippet))
	}
	return toolOK(call, "low", strings.Join(lines, "\n")), nil
}

func (h defaultHandlers) readProblems(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	summary, err := h.deps.Workspace.ScanProblems(root, intArg(call, "maxResults", 80))
	if err != nil {
		return toolError(call, "low", err), err
	}
	lines := []string{summary.Message}
	for _, problem := range summary.Problems {
		lines = append(lines, fmt.Sprintf("- %s:%d [%s/%s] %s", problem.RelPath, problem.Line, problem.Severity, problem.Source, problem.Message))
	}
	return toolOK(call, "low", strings.Join(lines, "\n")), nil
}

func (h defaultHandlers) readGitStatus(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	status, err := h.deps.Git.Status(root)
	if err != nil {
		return toolError(call, "low", err), err
	}
	lines := []string{status.Message}
	if status.Available {
		lines = append(lines, fmt.Sprintf("Branch: %s @ %s", status.Branch, status.Head))
		lines = append(lines, fmt.Sprintf("Changed: %d staged=%d unstaged=%d", len(status.ChangedFiles), len(status.StagedFiles), len(status.UnstagedFiles)))
		for _, change := range status.ChangedFiles {
			lines = append(lines, fmt.Sprintf("- %s %s", change.Summary, change.Path))
		}
	}
	return toolOK(call, "low", strings.Join(lines, "\n")), nil
}

func (h defaultHandlers) readGitDiff(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	relPath := firstArg(call, "relPath", "path")
	if relPath == "" {
		err := errors.New("relPath is required")
		return toolError(call, "low", err), err
	}
	diff, err := h.deps.Git.FileDiff(root, relPath)
	if err != nil {
		return toolError(call, "low", err), err
	}
	observation := diff.Message + "\n"
	if diff.StagedDiff != "" {
		observation += "\nStaged diff:\n" + diff.StagedDiff
	}
	if diff.UnstagedDiff != "" {
		observation += "\nUnstaged diff:\n" + diff.UnstagedDiff
	}
	return toolOK(call, "low", observation), nil
}

func (h defaultHandlers) readGitHistory(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	result, err := h.deps.Git.History(root, firstArg(call, "relPath", "path", "target"), intArg(call, "limit", gitsvc.DefaultHistoryLimit))
	if err != nil {
		return toolError(call, "low", err), err
	}
	return toolOK(call, "low", formatGitHistoryObservation(result)), nil
}

func (h defaultHandlers) readGitBlame(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	relPath := firstArg(call, "relPath", "path", "target")
	if relPath == "" {
		err := errors.New("relPath is required")
		return toolError(call, "low", err), err
	}
	result, err := h.deps.Git.Blame(root, relPath, intArg(call, "startLine", 1), intArg(call, "endLine", gitsvc.DefaultHistoryLimit))
	if err != nil {
		return toolError(call, "low", err), err
	}
	return toolOK(call, "low", formatGitBlameObservation(result)), nil
}

func formatGitHistoryObservation(result gitsvc.HistoryResult) string {
	if !result.Available {
		return result.Message
	}
	label := "repository"
	if result.Path != "" {
		label = result.Path
	}
	lines := []string{
		result.Message,
		fmt.Sprintf("History target: %s limit=%d", label, result.Limit),
	}
	for _, entry := range result.Entries {
		lines = append(lines, fmt.Sprintf("- %s %s %s <%s> %s", entry.ShortHash, entry.Date, entry.Author, entry.Email, entry.Subject))
	}
	if result.Truncated {
		lines = append(lines, "History output was truncated.")
	}
	return strings.Join(lines, "\n")
}

func formatGitBlameObservation(result gitsvc.BlameResult) string {
	if !result.Available {
		return result.Message
	}
	lines := []string{result.Message}
	if result.StartLine > 0 {
		lines = append(lines, fmt.Sprintf("Requested lines: %d-%d", result.StartLine, result.EndLine))
	}
	for _, line := range result.Lines {
		lines = append(lines, fmt.Sprintf("%d %s %s %s | %s", line.Line, line.ShortHash, line.Author, line.Date, line.Content))
		if line.Summary != "" {
			lines = append(lines, "  summary: "+line.Summary)
		}
	}
	if result.Truncated {
		lines = append(lines, "Blame output was truncated.")
	}
	return strings.Join(lines, "\n")
}

func (h defaultHandlers) listExternalAgentTools(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	statuses := externalagentsSvc.Probe(externalagentsSvc.Options{LookupPath: h.deps.ExternalAgentLookupPath})
	return toolOK(call, "low", externalagentsSvc.FormatMarkdown(statuses)), nil
}

func (h defaultHandlers) planExternalAgentRun(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	prompt := firstArg(call, "prompt", "task", "instructions")
	if strings.TrimSpace(prompt) == "" {
		prompt = request.Prompt
	}
	plan, err := externalagentsSvc.PlanInvocation(externalagentsSvc.InvocationRequest{
		ToolID:        firstArg(call, "toolID", "tool", "id"),
		WorkspaceRoot: request.WorkspaceRoot,
		Prompt:        prompt,
		LookupPath:    h.deps.ExternalAgentLookupPath,
	})
	if err != nil {
		return toolError(call, "low", err), err
	}
	return toolOK(call, "low", externalagentsSvc.FormatInvocationPlan(plan)), nil
}

func (h defaultHandlers) readArtifactLineage(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		return toolError(call, "low", err), err
	}
	lineage, err := store.LineageGraph(artifactsSvc.ListOptions{Query: firstArg(call, "query", "q"), IncludeArchived: boolArg(call, "includeArchived")})
	if err != nil {
		return toolError(call, "low", err), err
	}
	return toolOK(call, "low", formatArtifactLineageObservation(lineage)), nil
}

func formatArtifactLineageObservation(lineage artifactsSvc.Lineage) string {
	if len(lineage.Nodes) == 0 {
		return "No artifact lineage metadata is available yet."
	}
	lines := []string{lineage.Message}
	if len(lineage.RelationshipCounts) > 0 {
		counts := make([]string, 0, len(lineage.RelationshipCounts))
		for label, count := range lineage.RelationshipCounts {
			counts = append(counts, fmt.Sprintf("%s=%d", label, count))
		}
		sort.Strings(counts)
		lines = append(lines, "Relationship counts: "+strings.Join(counts, ", "))
	}
	lines = append(lines, "Nodes:")
	for index, node := range lineage.Nodes {
		if index >= 40 {
			lines = append(lines, "[nodes truncated]")
			break
		}
		relPath := node.RelPath
		if relPath == "" {
			relPath = "-"
		}
		lines = append(lines, fmt.Sprintf("- %s [%s] %s path=%s", node.ID, node.Kind, node.Label, relPath))
	}
	if len(lineage.Edges) > 0 {
		lines = append(lines, "Relationships:")
		for index, edge := range lineage.Edges {
			if index >= 80 {
				lines = append(lines, "[relationships truncated]")
				break
			}
			lines = append(lines, fmt.Sprintf("- %s --%s--> %s", edge.From, edge.Label, edge.To))
		}
	}
	return strings.Join(lines, "\n")
}

func (h defaultHandlers) webFetch(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	targetURL := firstArg(call, "url", "href", "target")
	if targetURL == "" {
		err := errors.New("URL is required")
		return toolError(call, "medium", err), err
	}
	result, err := webfetchSvc.Fetch(ctx, webfetchSvc.Request{
		URL:            targetURL,
		AllowedDomains: listArg(call, "allowedDomains"),
		AllowLocal:     boolArg(call, "allowLocal"),
		MaxBytes:       intArg(call, "maxBytes", 128*1024),
	})
	if err != nil {
		return toolError(call, "medium", err), err
	}
	return toolOK(call, "medium", formatWebFetchObservation(result)), nil
}

func formatWebFetchObservation(result webfetchSvc.Result) string {
	lines := []string{
		result.Message,
		"URL: " + result.URL,
		"Final URL: " + result.FinalURL,
		fmt.Sprintf("Status: %d", result.Status),
		"Content-Type: " + result.ContentType,
		fmt.Sprintf("Redirects: %d", result.Redirects),
		fmt.Sprintf("Truncated: %t", result.Truncated),
	}
	if result.Title != "" {
		lines = append(lines, "Title: "+result.Title)
	}
	if strings.TrimSpace(result.Text) != "" {
		lines = append(lines, "\nContent:\n"+result.Text)
	}
	return strings.Join(lines, "\n")
}

func (h defaultHandlers) listTasks(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	summary, err := h.deps.Tasks.Discover(root)
	if err != nil {
		return toolError(call, "low", err), err
	}
	lines := []string{summary.Message}
	for _, task := range summary.Tasks {
		lines = append(lines, fmt.Sprintf("- %s [%s] %s cwd=%s source=%s", task.ID, task.Kind, task.Command, task.Cwd, task.Source))
	}
	return toolOK(call, "low", strings.Join(lines, "\n")), nil
}

func (h defaultHandlers) runTask(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	if !request.ApproveShell {
		err := errors.New("approval is required before running workspace tasks")
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: err.Error(), Error: err.Error()}, err
	}
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "high", err), err
	}
	taskID := firstArg(call, "taskId", "id")
	if taskID == "" {
		err := errors.New("taskId is required")
		return toolError(call, "high", err), err
	}
	result, err := h.deps.Tasks.RunContext(ctx, root, taskID)
	if err != nil {
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{
		Name:        call.Name,
		Args:        call.Args,
		Risk:        "high",
		Observation: fmt.Sprintf("%s\nStatus: %s\nExit: %d\nStdout:\n%s\nStderr:\n%s", result.Message, result.Status, result.ExitCode, result.Stdout, result.Stderr),
		Mutated:     false,
	}, nil
}

func (h defaultHandlers) runTerminalCommand(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	if !request.ApproveShell {
		err := errors.New("approval is required before running terminal commands")
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: err.Error(), Error: err.Error()}, err
	}
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "high", err), err
	}
	args, err := jsonListArg(call, "argsJson")
	if err != nil {
		err = fmt.Errorf("argsJson must be a JSON string array: %w", err)
		return toolError(call, "high", err), err
	}
	result, err := h.deps.Tasks.RunTerminalCommandContext(ctx, root, taskssvc.TerminalRequest{
		Command:        firstArg(call, "command", "cmd"),
		Args:           args,
		Cwd:            firstArg(call, "cwd", "workingDirectory"),
		TimeoutSeconds: intArg(call, "timeoutSeconds", 30),
	})
	if err != nil {
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{
		Name: call.Name,
		Args: call.Args,
		Risk: "high",
		Observation: fmt.Sprintf(
			"%s\nCommand: %s\nCwd: %s\nStatus: %s\nExit: %d\nStdout:\n%s\nStderr:\n%s",
			result.Message,
			strings.Join(append([]string{result.Command}, result.Args...), " "),
			result.Cwd,
			result.Status,
			result.ExitCode,
			result.Stdout,
			result.Stderr,
		),
		Mutated: true,
	}, nil
}
