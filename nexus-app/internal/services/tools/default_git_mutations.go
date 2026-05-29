package tools

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"nexusdesk/internal/services/agent"
	gitSvc "nexusdesk/internal/services/git"
	workspacesvc "nexusdesk/internal/services/workspace"
)

func (h defaultHandlers) stageFile(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	return h.applyGitFileAction(ctx, call, request, gitSvc.FileActionStage)
}

func (h defaultHandlers) unstageFile(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	return h.applyGitFileAction(ctx, call, request, gitSvc.FileActionUnstage)
}

func (h defaultHandlers) stageHunk(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	return h.applyGitHunkAction(ctx, call, request, gitSvc.DiffKindUnstaged, gitSvc.HunkActionStage)
}

func (h defaultHandlers) unstageHunk(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	return h.applyGitHunkAction(ctx, call, request, gitSvc.DiffKindStaged, gitSvc.HunkActionUnstage)
}

func (h defaultHandlers) commitChanges(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	if !request.ApproveWrites {
		err := errors.New("approval is required before creating Git commits")
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: err.Error(), Error: err.Error()}, err
	}
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "high", err), err
	}
	result, err := h.deps.Git.CommitChanges(root, firstArg(call, "message", "subject", "title"), firstArg(call, "body", "description"))
	if err != nil {
		return toolError(call, "high", err), err
	}
	if result.Hash == "" {
		err := errors.New(result.Message)
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{
		Name:        call.Name,
		Args:        call.Args,
		Risk:        "high",
		Mutated:     true,
		Observation: formatGitCommitObservation(result),
	}, nil
}

func (h defaultHandlers) createBranch(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	if !request.ApproveWrites {
		err := errors.New("approval is required before creating Git branches")
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: err.Error(), Error: err.Error()}, err
	}
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "high", err), err
	}
	result, err := h.deps.Git.CreateBranch(root, firstArg(call, "branchName", "name", "branch"), firstArg(call, "startPoint", "from"), boolArg(call, "checkout"))
	if err != nil {
		return toolError(call, "high", err), err
	}
	if result.StartPointSHA == "" {
		err := errors.New(result.Message)
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{
		Name:        call.Name,
		Args:        call.Args,
		Risk:        "high",
		Mutated:     true,
		Observation: formatGitBranchObservation(result),
	}, nil
}

func (h defaultHandlers) revertChanges(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	if !request.ApproveWrites {
		err := errors.New("approval is required before reverting workspace changes")
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: err.Error(), Error: err.Error()}, err
	}
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "high", err), err
	}
	relPath := firstArg(call, "relPath", "path")
	if relPath == "" {
		err := errors.New("relPath is required")
		return toolError(call, "high", err), err
	}
	plan, err := h.deps.Git.PlanRevertChanges(root, relPath, firstArg(call, "scope"))
	if err != nil {
		return toolError(call, "high", err), err
	}
	switch plan.Action {
	case gitSvc.RevertActionWrite:
		proposal, err := h.deps.Workspace.ApplyFileWrite(root, workspacesvc.FileWriteRequest{RelPath: plan.Path, Content: plan.Content})
		if err != nil {
			return toolError(call, "high", err), err
		}
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: formatGitRevertWriteObservation(plan, proposal), Mutated: true}, nil
	case gitSvc.RevertActionDelete:
		proposal, err := h.deps.Workspace.ApplyFileDelete(root, plan.Path)
		if err != nil {
			return toolError(call, "high", err), err
		}
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: formatGitRevertDeleteObservation(plan, proposal), Mutated: true}, nil
	default:
		err := errors.New(plan.Message)
		return toolError(call, "high", err), err
	}
}

func (h defaultHandlers) revertStagedChanges(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	if !request.ApproveWrites {
		err := errors.New("approval is required before reverting staged Git changes")
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: err.Error(), Error: err.Error()}, err
	}
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "high", err), err
	}
	relPath := firstArg(call, "relPath", "path")
	if relPath == "" {
		err := errors.New("relPath is required")
		return toolError(call, "high", err), err
	}
	plan, err := h.deps.Git.PlanRevertStagedChanges(root, relPath, firstArg(call, "scope"))
	if err != nil {
		return toolError(call, "high", err), err
	}
	if plan.Action == "" {
		err := errors.New(plan.Message)
		return toolError(call, "high", err), err
	}
	unstaged, err := h.deps.Git.ApplyFileAction(root, plan.Path, gitSvc.FileActionUnstage)
	if err != nil {
		return toolError(call, "high", err), err
	}
	if !unstaged.Status.Available {
		err := errors.New(unstaged.Message)
		return toolError(call, "high", err), err
	}
	switch plan.Action {
	case gitSvc.RevertActionWrite:
		proposal, err := h.deps.Workspace.ApplyFileWrite(root, workspacesvc.FileWriteRequest{RelPath: plan.Path, Content: plan.Content})
		if err != nil {
			return toolError(call, "high", err), err
		}
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: formatGitStagedRevertWriteObservation(plan, proposal), Mutated: true}, nil
	case gitSvc.RevertActionDelete:
		proposal, err := h.deps.Workspace.ApplyFileDelete(root, plan.Path)
		if err != nil {
			return toolError(call, "high", err), err
		}
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: formatGitStagedRevertDeleteObservation(plan, proposal), Mutated: true}, nil
	default:
		err := errors.New(plan.Message)
		return toolError(call, "high", err), err
	}
}

func (h defaultHandlers) applyGitFileAction(ctx context.Context, call agent.ToolCall, request agent.Request, action gitSvc.FileAction) (agent.ToolResult, error) {
	_ = ctx
	if !request.ApproveWrites {
		err := errors.New("approval is required before changing the Git index")
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: err.Error(), Error: err.Error()}, err
	}
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "high", err), err
	}
	relPath := firstArg(call, "relPath", "path")
	if relPath == "" {
		err := errors.New("relPath is required")
		return toolError(call, "high", err), err
	}
	result, err := h.deps.Git.ApplyFileAction(root, relPath, action)
	if err != nil {
		return toolError(call, "high", err), err
	}
	if !result.Status.Available {
		err := errors.New(result.Message)
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{
		Name:        call.Name,
		Args:        call.Args,
		Risk:        "high",
		Mutated:     true,
		Observation: formatGitFileActionObservation(result),
	}, nil
}

func (h defaultHandlers) applyGitHunkAction(ctx context.Context, call agent.ToolCall, request agent.Request, defaultKind gitSvc.DiffKind, action gitSvc.HunkAction) (agent.ToolResult, error) {
	_ = ctx
	if !request.ApproveWrites {
		err := errors.New("approval is required before changing the Git index")
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: err.Error(), Error: err.Error()}, err
	}
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "high", err), err
	}
	relPath := firstArg(call, "relPath", "path")
	if relPath == "" {
		err := errors.New("relPath is required")
		return toolError(call, "high", err), err
	}
	hunkIndex, err := gitHunkIndexArg(call)
	if err != nil {
		return toolError(call, "high", err), err
	}
	kind, err := gitDiffKindArg(call, defaultKind)
	if err != nil {
		return toolError(call, "high", err), err
	}
	result, err := h.deps.Git.ApplyHunkAction(root, relPath, kind, hunkIndex, action)
	if err != nil {
		return toolError(call, "high", err), err
	}
	if strings.TrimSpace(result.Message) == "" {
		err := errors.New("selected hunk could not be applied")
		return toolError(call, "high", err), err
	}
	if !result.Status.Available {
		err := errors.New(result.Message)
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{
		Name:        call.Name,
		Args:        call.Args,
		Risk:        "high",
		Mutated:     true,
		Observation: formatGitHunkActionObservation(result),
	}, nil
}

func gitHunkIndexArg(call agent.ToolCall) (int, error) {
	if value := strings.TrimSpace(firstArg(call, "hunkIndex", "index")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			return 0, errors.New("hunkIndex must be zero or greater")
		}
		return parsed, nil
	}
	if value := strings.TrimSpace(firstArg(call, "hunkId", "hunkNumber")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return 0, errors.New("hunkId must be one or greater")
		}
		return parsed - 1, nil
	}
	return 0, nil
}

func gitDiffKindArg(call agent.ToolCall, fallback gitSvc.DiffKind) (gitSvc.DiffKind, error) {
	value := strings.ToLower(strings.TrimSpace(firstArg(call, "diffKind", "kind")))
	if value == "" {
		return fallback, nil
	}
	switch value {
	case string(gitSvc.DiffKindStaged):
		return gitSvc.DiffKindStaged, nil
	case string(gitSvc.DiffKindUnstaged):
		return gitSvc.DiffKindUnstaged, nil
	default:
		return "", fmt.Errorf("unsupported diffKind %q", value)
	}
}

func formatGitBranchObservation(result gitSvc.BranchResult) string {
	lines := []string{
		result.Message,
		fmt.Sprintf("Branch: %s", result.BranchName),
		fmt.Sprintf("Start point: %s", result.StartPoint),
		fmt.Sprintf("Start SHA: %s", result.StartPointSHA),
		fmt.Sprintf("Checked out: %t", result.CheckedOut),
	}
	lines = append(lines, formatGitMutationStatus(result.Status)...)
	return strings.Join(lines, "\n")
}

func formatGitCommitObservation(result gitSvc.CommitResult) string {
	lines := []string{
		result.Message,
		fmt.Sprintf("Commit: %s", result.ShortHash),
		fmt.Sprintf("Subject: %s", result.Subject),
		"Staged changes committed:",
		result.StagedStat,
	}
	lines = append(lines, formatGitMutationStatus(result.Status)...)
	return strings.Join(lines, "\n")
}

func formatGitFileActionObservation(result gitSvc.FileActionResult) string {
	lines := []string{
		result.Message,
		fmt.Sprintf("Path: %s", result.Path),
		fmt.Sprintf("Action: %s", result.Action),
	}
	lines = append(lines, formatGitMutationStatus(result.Status)...)
	return strings.Join(lines, "\n")
}

func formatGitHunkActionObservation(result gitSvc.HunkActionResult) string {
	lines := []string{
		result.Message,
		fmt.Sprintf("Path: %s", result.Path),
		fmt.Sprintf("Action: %s", result.Action),
		fmt.Sprintf("Diff kind: %s", result.DiffKind),
		fmt.Sprintf("Hunk index: %d", result.HunkIndex),
	}
	lines = append(lines, formatGitMutationStatus(result.Status)...)
	return strings.Join(lines, "\n")
}

func formatGitRevertWriteObservation(plan gitSvc.RevertPlan, proposal workspacesvc.FileWriteProposal) string {
	lines := []string{
		plan.Message,
		fmt.Sprintf("Path: %s", plan.Path),
		fmt.Sprintf("Scope: %s", plan.Scope),
		fmt.Sprintf("Action: %s", plan.Action),
		"Rollback: " + proposal.RollbackID,
	}
	if proposal.Diff != "" {
		lines = append(lines, "Diff:\n"+proposal.Diff)
	}
	return strings.Join(lines, "\n")
}

func formatGitRevertDeleteObservation(plan gitSvc.RevertPlan, proposal workspacesvc.FileOperationProposal) string {
	lines := []string{
		plan.Message,
		fmt.Sprintf("Path: %s", plan.Path),
		fmt.Sprintf("Scope: %s", plan.Scope),
		fmt.Sprintf("Action: %s", plan.Action),
		"Rollback: " + proposal.RollbackID,
	}
	return strings.Join(lines, "\n")
}

func formatGitStagedRevertWriteObservation(plan gitSvc.RevertPlan, proposal workspacesvc.FileWriteProposal) string {
	lines := []string{
		plan.Message,
		fmt.Sprintf("Path: %s", plan.Path),
		fmt.Sprintf("Scope: %s", plan.Scope),
		fmt.Sprintf("Action: %s", plan.Action),
		"Rollback: " + proposal.RollbackID,
	}
	if plan.Diff != "" {
		lines = append(lines, "Discarded staged diff preview:\n"+plan.Diff)
	}
	if proposal.Diff != "" {
		lines = append(lines, "Applied workspace diff:\n"+proposal.Diff)
	}
	return strings.Join(lines, "\n")
}

func formatGitStagedRevertDeleteObservation(plan gitSvc.RevertPlan, proposal workspacesvc.FileOperationProposal) string {
	lines := []string{
		plan.Message,
		fmt.Sprintf("Path: %s", plan.Path),
		fmt.Sprintf("Scope: %s", plan.Scope),
		fmt.Sprintf("Action: %s", plan.Action),
		"Rollback: " + proposal.RollbackID,
	}
	if plan.Diff != "" {
		lines = append(lines, "Discarded staged diff preview:\n"+plan.Diff)
	}
	return strings.Join(lines, "\n")
}

func formatGitMutationStatus(status gitSvc.Status) []string {
	if !status.Available {
		return []string{"Status: " + status.Message}
	}
	return []string{
		fmt.Sprintf("Branch: %s @ %s", status.Branch, status.Head),
		fmt.Sprintf("Changed: %d staged=%d unstaged=%d", len(status.ChangedFiles), len(status.StagedFiles), len(status.UnstagedFiles)),
	}
}
