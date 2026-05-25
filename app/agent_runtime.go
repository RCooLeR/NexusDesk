package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"NexusAugenticStudio/internal/agent"
	"NexusAugenticStudio/internal/agenttools"
	"NexusAugenticStudio/internal/dataset"
	"NexusAugenticStudio/internal/workspace"
)

const maxAgentShellOutputBytes = 12000

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
	case "write_file":
		call.Risk = "high"
		return a.agentWriteFile(root, call, request.ApproveHighImpact)
	case "append_file":
		call.Risk = "high"
		return a.agentAppendFile(root, call, request.ApproveHighImpact)
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

func (a *App) agentWriteFile(root string, call agent.ToolCall, approved bool) (agent.ToolCall, error) {
	request := workspace.FileWriteRequest{
		RelPath: cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"])),
		Content: call.Arguments["content"],
	}
	proposal, err := workspace.PreviewFileWrite(root, request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	if !approved {
		call.Observation = "Approval required before writing file. Proposed diff:\n" + proposal.Diff
		call.Error = "approval required"
		return call, errors.New(call.Error)
	}
	applied, err := workspace.ApplyFileWrite(root, request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	a.recordApproval("agent.write_file", applied.RelPath, "high", applied.Message)
	call.Observation = applied.Message + "\n" + proposal.Diff
	return call, nil
}

func (a *App) agentAppendFile(root string, call agent.ToolCall, approved bool) (agent.ToolCall, error) {
	relPath := cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"]))
	preview, err := workspace.Preview(root, relPath, workspace.PreviewOptions{MaxBytes: 512 * 1024})
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	content := preview.Content
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	call.Arguments["content"] = content + call.Arguments["content"]
	call.Arguments["relPath"] = relPath
	return a.agentWriteFile(root, call, approved)
}

func (a *App) agentExecuteShell(ctx context.Context, root string, call agent.ToolCall, request agent.RunRequest) (agent.ToolCall, error) {
	command := strings.TrimSpace(call.Arguments["command"])
	if command == "" {
		call.Error = "shell command is required"
		return call, errors.New(call.Error)
	}
	if !request.AllowShellCommands || !request.ApproveHighImpact {
		call.Observation = "Approval required before running shell command inside workspace: " + command
		call.Error = "approval required"
		return call, errors.New(call.Error)
	}
	if !isAgentShellCommandAllowed(command) {
		call.Observation = "Shell command blocked by workspace sandbox policy: " + command
		call.Error = "command escapes workspace sandbox"
		return call, errors.New(call.Error)
	}

	shellCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(shellCtx, "cmd", "/c", command)
	} else {
		cmd = exec.CommandContext(shellCtx, "sh", "-c", command)
	}
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	call.Observation = limitAgentOutput(string(output), maxAgentShellOutputBytes)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	a.recordApproval("agent.shell", command, "high", "Shell command executed inside workspace.")
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
	call.Observation = fmt.Sprintf("%s\nRows: %d\nColumns: %d\nSheets: %s", profile.Message, profile.Rows, profile.Columns, strings.Join(profile.Sheets, ", "))
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

func limitAgentOutput(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes] + "\n[truncated]"
}

func isAgentShellCommandAllowed(command string) bool {
	normalized := strings.ToLower(strings.TrimSpace(command))
	if normalized == "" {
		return false
	}
	forbidden := []string{
		"..",
		"~",
		"%userprofile%",
		"%home%",
		"$home",
	}
	for _, token := range forbidden {
		if strings.Contains(normalized, token) {
			return false
		}
	}
	absolutePathPattern := regexp.MustCompile(`(?i)(^|\s)([a-z]:\\|\\\\|/)`)
	return !absolutePathPattern.MatchString(command)
}
