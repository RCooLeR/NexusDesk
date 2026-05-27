package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"nexusdesk/internal/services/agent"
	gitsvc "nexusdesk/internal/services/git"
	taskssvc "nexusdesk/internal/services/tasks"
	workspacesvc "nexusdesk/internal/services/workspace"
)

type Dependencies struct {
	Workspace *workspacesvc.Service
	Git       *gitsvc.Service
	Tasks     *taskssvc.Service
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
		Tool{Descriptor: agent.ToolDescriptor{Name: "list_tasks", Description: "List safe discovered workspace tasks.", Risk: "low", Inputs: ""}, Handler: handlers.listTasks},
		Tool{Descriptor: agent.ToolDescriptor{Name: "run_task", Description: "Run a discovered safe workspace task when shell approval is granted.", Risk: "high", Inputs: "taskId"}, Handler: handlers.runTask},
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
