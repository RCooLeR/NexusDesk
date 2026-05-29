package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"nexusdesk/internal/services/agent"
	editorSvc "nexusdesk/internal/services/editor"
	workspacesvc "nexusdesk/internal/services/workspace"
)

type symbolIndexCandidate struct {
	RelPath string
	Line    int
	Kind    string
	Label   string
	Level   int
}

func (h defaultHandlers) readSymbolIndex(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	query := strings.ToLower(strings.TrimSpace(firstArg(call, "query", "symbol")))
	maxFiles := intArg(call, "maxFiles", 80)
	maxSymbols := intArg(call, "maxSymbols", 160)
	if maxSymbols > 240 {
		maxSymbols = 240
	}
	sourceFiles, err := h.deps.Workspace.SourceFiles(root, workspacesvc.SourceFileOptions{
		RelPath:  firstArg(call, "relPath", "path"),
		MaxFiles: maxFiles,
	})
	if err != nil {
		return toolError(call, "low", err), err
	}
	symbols := make([]symbolIndexCandidate, 0)
	filesRead := 0
	filesSkipped := sourceFiles.FilesSkipped
	truncated := sourceFiles.Truncated
	for _, relPath := range sourceFiles.Files {
		read, err := h.deps.Workspace.ReadTextFile(root, relPath)
		if err != nil {
			filesSkipped++
			continue
		}
		filesRead++
		for _, item := range editorSvc.BuildOutline(read.RelPath, read.Content) {
			candidate := symbolIndexCandidate{
				RelPath: read.RelPath,
				Line:    item.Line,
				Kind:    item.Kind,
				Label:   item.Label,
				Level:   item.Level,
			}
			if query != "" && !symbolIndexCandidateMatches(candidate, query) {
				continue
			}
			symbols = append(symbols, candidate)
			if len(symbols) >= maxSymbols {
				truncated = true
				break
			}
		}
		if len(symbols) >= maxSymbols {
			break
		}
	}
	sort.SliceStable(symbols, func(left int, right int) bool {
		if symbols[left].RelPath == symbols[right].RelPath {
			return symbols[left].Line < symbols[right].Line
		}
		return symbols[left].RelPath < symbols[right].RelPath
	})
	return toolOK(call, "low", formatSymbolIndexObservation(sourceFiles.RootRelPath, query, filesRead, filesSkipped, truncated, symbols)), nil
}

func symbolIndexCandidateMatches(candidate symbolIndexCandidate, query string) bool {
	return strings.Contains(strings.ToLower(candidate.Label), query) ||
		strings.Contains(strings.ToLower(candidate.Kind), query) ||
		strings.Contains(strings.ToLower(candidate.RelPath), query)
}

func formatSymbolIndexObservation(scope string, query string, filesRead int, filesSkipped int, truncated bool, symbols []symbolIndexCandidate) string {
	lines := []string{
		"Native symbol index.",
		"Scope: " + firstNonEmptySymbolIndexString(scope, "."),
		fmt.Sprintf("Files scanned: %d", filesRead),
		fmt.Sprintf("Files skipped: %d", filesSkipped),
		fmt.Sprintf("Symbols: %d", len(symbols)),
	}
	if query != "" {
		lines = append(lines, "Query: "+query)
	}
	if truncated {
		lines = append(lines, "Truncated: true")
	}
	if len(symbols) == 0 {
		lines = append(lines, "No supported outline symbols found.")
		return strings.Join(lines, "\n")
	}
	for index, symbol := range symbols {
		if index >= 120 {
			lines = append(lines, "[symbols truncated]")
			break
		}
		level := ""
		if symbol.Level > 0 {
			level = fmt.Sprintf(" level=%d", symbol.Level)
		}
		lines = append(lines, fmt.Sprintf("- %s:%d [%s%s] %s", symbol.RelPath, symbol.Line, symbol.Kind, level, symbol.Label))
	}
	return strings.Join(lines, "\n")
}

func firstNonEmptySymbolIndexString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
