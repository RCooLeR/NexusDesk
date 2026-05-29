package tools

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"nexusdesk/internal/services/agent"
	editorSvc "nexusdesk/internal/services/editor"
	workspacesvc "nexusdesk/internal/services/workspace"
)

func (h defaultHandlers) formatFile(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	if !request.ApproveWrites {
		err := errors.New("approval is required before formatting workspace files")
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
	formatter := strings.ToLower(strings.TrimSpace(firstArg(call, "formatter")))
	if formatter != "" && formatter != "native" {
		err := fmt.Errorf("unsupported formatter %q; only native formatting is available", formatter)
		return toolError(call, "high", err), err
	}
	read, err := h.deps.Workspace.ReadTextFile(root, relPath)
	if err != nil {
		return toolError(call, "high", err), err
	}
	result, err := editorSvc.FormatDocument(read.RelPath, read.Content)
	if err != nil {
		return toolError(call, "high", err), err
	}
	if !result.Changed {
		return agent.ToolResult{
			Name:        call.Name,
			Args:        call.Args,
			Risk:        "high",
			Observation: fmt.Sprintf("Document is already formatted.\nPath: %s\nFormatter: native\nEncoding: %s\nSize: %d", read.RelPath, read.Encoding, read.Size),
			Mutated:     false,
		}, nil
	}
	proposal, err := h.deps.Workspace.ApplyFileWrite(root, workspacesvc.FileWriteRequest{RelPath: read.RelPath, Content: result.Content, Encoding: read.Encoding})
	if err != nil {
		return toolError(call, "high", err), err
	}
	return agent.ToolResult{
		Name:        call.Name,
		Args:        call.Args,
		Risk:        "high",
		Observation: formatFileObservation(read, proposal),
		Mutated:     true,
	}, nil
}

func (h defaultHandlers) lintFile(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "medium", err), err
	}
	relPath := firstArg(call, "relPath", "path")
	if relPath == "" {
		err := errors.New("relPath is required")
		return toolError(call, "medium", err), err
	}
	linter := strings.ToLower(strings.TrimSpace(firstArg(call, "linter")))
	if linter != "" && linter != "native" {
		err := fmt.Errorf("unsupported linter %q; only native diagnostics are available", linter)
		return toolError(call, "medium", err), err
	}
	read, err := h.deps.Workspace.ReadTextFile(root, relPath)
	if err != nil {
		return toolError(call, "medium", err), err
	}
	diagnostics := editorSvc.AnalyzeDraftDiagnostics(read.RelPath, read.Content)
	return toolOK(call, "medium", formatLintFileObservation(read, diagnostics)), nil
}

func (h defaultHandlers) gotoDefinition(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	relPath := firstArg(call, "relPath", "path")
	if relPath == "" {
		err := errors.New("relPath is required")
		return toolError(call, "low", err), err
	}
	read, err := h.deps.Workspace.ReadTextFile(root, relPath)
	if err != nil {
		return toolError(call, "low", err), err
	}
	query := strings.TrimSpace(firstArg(call, "query", "symbol"))
	if query == "" {
		row, column, err := cursorArgs(call)
		if err != nil {
			return toolError(call, "low", err), err
		}
		local, ok := editorSvc.ResolveDefinition(read.RelPath, read.Content, row, column)
		if ok {
			return toolOK(call, "low", formatDefinitionObservation(local, true, false)), nil
		}
		query = local.Query
	}
	result, ok, err := h.workspaceDefinition(root, read.RelPath, read.Content, query)
	if err != nil {
		return toolError(call, "low", err), err
	}
	return toolOK(call, "low", formatDefinitionObservation(result, ok, true)), nil
}

func (h defaultHandlers) findReferences(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	query := strings.TrimSpace(firstArg(call, "query", "symbol"))
	if query == "" {
		relPath := firstArg(call, "relPath", "path")
		if relPath == "" {
			err := errors.New("relPath is required when query is not provided")
			return toolError(call, "low", err), err
		}
		read, err := h.deps.Workspace.ReadTextFile(root, relPath)
		if err != nil {
			return toolError(call, "low", err), err
		}
		row, column, err := cursorArgs(call)
		if err != nil {
			return toolError(call, "low", err), err
		}
		query = editorSvc.SymbolAtCursor(read.Content, row, column)
	}
	if strings.TrimSpace(query) == "" {
		err := errors.New("query or cursor symbol is required")
		return toolError(call, "low", err), err
	}
	results, err := h.deps.Workspace.Search(root, query, workspacesvc.SearchOptions{MaxResults: 120})
	if err != nil {
		return toolError(call, "low", err), err
	}
	return toolOK(call, "low", formatReferencesObservation(query, referenceCandidatesFromSearch(query, results))), nil
}

func formatFileObservation(read workspacesvc.TextFileRead, proposal workspacesvc.FileWriteProposal) string {
	lines := []string{
		"Formatted workspace file.",
		"Path: " + read.RelPath,
		"Formatter: native",
		"Encoding: " + proposal.Encoding,
		fmt.Sprintf("Size: %d", proposal.Size),
		"Rollback: " + proposal.RollbackID,
	}
	if proposal.Diff != "" {
		lines = append(lines, "Diff:\n"+proposal.Diff)
	}
	return strings.Join(lines, "\n")
}

func (h defaultHandlers) workspaceDefinition(root string, currentRelPath string, currentContent string, query string) (editorSvc.DefinitionResult, bool, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return editorSvc.DefinitionResult{}, false, nil
	}
	files := []editorSvc.DefinitionFile{{RelPath: currentRelPath, Content: currentContent}}
	seen := map[string]bool{strings.Trim(strings.ReplaceAll(currentRelPath, "\\", "/"), "/"): true}
	results, err := h.deps.Workspace.Search(root, query, workspacesvc.SearchOptions{MaxResults: 60})
	if err != nil {
		return editorSvc.DefinitionResult{}, false, err
	}
	for _, result := range results {
		relPath := strings.Trim(strings.ReplaceAll(result.RelPath, "\\", "/"), "/")
		if relPath == "" || seen[relPath] || result.Kind == "directory" {
			continue
		}
		seen[relPath] = true
		read, err := h.deps.Workspace.ReadTextFile(root, relPath)
		if err != nil || strings.TrimSpace(read.Content) == "" {
			continue
		}
		files = append(files, editorSvc.DefinitionFile{RelPath: read.RelPath, Content: read.Content})
	}
	definition, ok := editorSvc.ResolveWorkspaceDefinition(query, currentRelPath, files)
	return definition, ok, nil
}

func cursorArgs(call agent.ToolCall) (int, int, error) {
	line, err := positiveIntArg(firstArg(call, "line", "startLine"))
	if err != nil {
		return 0, 0, fmt.Errorf("line must be one or greater")
	}
	column, err := positiveIntArg(firstArg(call, "column", "col", "character"))
	if err != nil {
		return 0, 0, fmt.Errorf("column must be one or greater")
	}
	return line - 1, column - 1, nil
}

func positiveIntArg(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("missing integer")
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, errors.New("invalid integer")
	}
	return parsed, nil
}

func formatDefinitionObservation(result editorSvc.DefinitionResult, resolved bool, workspaceFallback bool) string {
	query := strings.TrimSpace(result.Query)
	if query == "" {
		return "No symbol query was available for definition lookup."
	}
	scope := "local"
	if workspaceFallback {
		scope = "workspace"
	}
	lines := []string{
		"Native definition lookup.",
		"Query: " + query,
		"Scope: " + scope,
		fmt.Sprintf("Resolved: %t", resolved),
	}
	if !resolved {
		lines = append(lines, "No definition found.")
		return strings.Join(lines, "\n")
	}
	lines = append(lines,
		"Path: "+result.RelPath,
		fmt.Sprintf("Line: %d", result.Item.Line),
		"Kind: "+result.Item.Kind,
		"Label: "+result.Item.Label,
	)
	return strings.Join(lines, "\n")
}

type referenceCandidate struct {
	RelPath string
	Line    int
	Snippet string
}

func referenceCandidatesFromSearch(query string, results []workspacesvc.SearchResult) []referenceCandidate {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	candidates := make([]referenceCandidate, 0, len(results))
	seen := map[string]bool{}
	for _, result := range results {
		if result.Kind == "directory" || result.Line <= 0 || !strings.HasPrefix(result.MatchType, "content") {
			continue
		}
		if !strings.Contains(strings.ToLower(result.Snippet), strings.ToLower(query)) {
			continue
		}
		key := fmt.Sprintf("%s:%d:%s", result.RelPath, result.Line, result.Snippet)
		if seen[key] {
			continue
		}
		seen[key] = true
		candidates = append(candidates, referenceCandidate{
			RelPath: result.RelPath,
			Line:    result.Line,
			Snippet: result.Snippet,
		})
	}
	return candidates
}

func formatReferencesObservation(query string, candidates []referenceCandidate) string {
	lines := []string{
		"Native reference lookup.",
		"Query: " + strings.TrimSpace(query),
		fmt.Sprintf("References: %d", len(candidates)),
	}
	if len(candidates) == 0 {
		lines = append(lines, "No references found in previewable workspace files.")
		return strings.Join(lines, "\n")
	}
	for index, candidate := range candidates {
		if index >= 60 {
			lines = append(lines, "[references truncated]")
			break
		}
		lines = append(lines, fmt.Sprintf("- %s:%d %s", candidate.RelPath, candidate.Line, strings.TrimSpace(candidate.Snippet)))
	}
	return strings.Join(lines, "\n")
}

func formatLintFileObservation(read workspacesvc.TextFileRead, diagnostics []editorSvc.DraftDiagnostic) string {
	lines := []string{
		"Native lint diagnostics.",
		"Path: " + read.RelPath,
		"Linter: native",
		"Encoding: " + read.Encoding,
		fmt.Sprintf("Size: %d", read.Size),
		fmt.Sprintf("Diagnostics: %d", len(diagnostics)),
	}
	if len(diagnostics) == 0 {
		lines = append(lines, "No lint diagnostics found.")
		return strings.Join(lines, "\n")
	}
	for index, diagnostic := range diagnostics {
		if index >= 30 {
			lines = append(lines, "[diagnostics truncated]")
			break
		}
		lines = append(lines, fmt.Sprintf("- %s/%s L%d %s", diagnostic.Severity, diagnostic.Source, diagnostic.Line, diagnostic.Message))
	}
	return strings.Join(lines, "\n")
}
