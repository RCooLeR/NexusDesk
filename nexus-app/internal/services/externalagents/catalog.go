// Package externalagents detects optional coding-agent CLIs without executing them.
package externalagents

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

const ExecutionPolicy = "Detection only. Execution must be routed through an approved job/shell integration with audit records."

type Tool struct {
	ID       string
	Label    string
	Commands []string
	Purpose  string
}

type ToolStatus struct {
	ID              string
	Label           string
	Commands        []string
	Command         string
	Path            string
	Available       bool
	Purpose         string
	ExecutionPolicy string
	Action          string
}

type Options struct {
	LookupPath func(string) (string, error)
}

func DefaultCatalog() []Tool {
	return []Tool{
		{
			ID:       "codex",
			Label:    "Codex CLI",
			Commands: []string{"codex"},
			Purpose:  "OpenAI Codex-class local coding workflows, implementation review, and repository automation.",
		},
		{
			ID:       "claude-code",
			Label:    "Claude Code",
			Commands: []string{"claude"},
			Purpose:  "Anthropic Claude Code workflows for implementation, refactoring, and review assistance.",
		},
		{
			ID:       "opencode",
			Label:    "OpenCode",
			Commands: []string{"opencode"},
			Purpose:  "OpenCode terminal-agent workflows for codebase edits, review, and local automation.",
		},
	}
}

func Probe(options Options) []ToolStatus {
	lookupPath := options.LookupPath
	if lookupPath == nil {
		lookupPath = exec.LookPath
	}
	statuses := make([]ToolStatus, 0, len(DefaultCatalog()))
	for _, tool := range DefaultCatalog() {
		status := ToolStatus{
			ID:              strings.TrimSpace(tool.ID),
			Label:           strings.TrimSpace(tool.Label),
			Commands:        append([]string{}, tool.Commands...),
			Purpose:         strings.TrimSpace(tool.Purpose),
			ExecutionPolicy: ExecutionPolicy,
			Action:          "Install the CLI and ensure its command is on PATH if you want NexusDesk to route future approved workflows to it.",
		}
		for _, command := range tool.Commands {
			path, err := lookupPath(command)
			if err == nil && strings.TrimSpace(path) != "" {
				status.Command = command
				status.Path = strings.TrimSpace(path)
				status.Available = true
				status.Action = ""
				break
			}
		}
		statuses = append(statuses, status)
	}
	sort.SliceStable(statuses, func(left int, right int) bool {
		return statuses[left].Label < statuses[right].Label
	})
	return statuses
}

func Summary(statuses []ToolStatus) string {
	total := len(statuses)
	available := 0
	names := []string{}
	for _, status := range statuses {
		if status.Available {
			available++
			names = append(names, status.Label)
		}
	}
	if total == 0 {
		return "No external coding-agent CLI catalog is configured."
	}
	if available == 0 {
		return fmt.Sprintf("0/%d external coding-agent CLIs detected.", total)
	}
	return fmt.Sprintf("%d/%d external coding-agent CLIs detected: %s.", available, total, strings.Join(names, ", "))
}

func FormatMarkdown(statuses []ToolStatus) string {
	var builder strings.Builder
	builder.WriteString(Summary(statuses))
	for _, status := range statuses {
		builder.WriteString("\n- ")
		builder.WriteString(status.Label)
		builder.WriteString(": ")
		if status.Available {
			builder.WriteString("available")
			builder.WriteString(" via `")
			builder.WriteString(status.Command)
			builder.WriteString("`")
			if status.Path != "" {
				builder.WriteString(" at ")
				builder.WriteString(status.Path)
			}
		} else {
			builder.WriteString("missing")
			if len(status.Commands) > 0 {
				builder.WriteString(" (expected ")
				builder.WriteString(strings.Join(status.Commands, " or "))
				builder.WriteString(")")
			}
		}
		if status.Purpose != "" {
			builder.WriteString(" - ")
			builder.WriteString(status.Purpose)
		}
		builder.WriteString("\n  Policy: ")
		builder.WriteString(status.ExecutionPolicy)
		if status.Action != "" {
			builder.WriteString("\n  Next: ")
			builder.WriteString(status.Action)
		}
	}
	return builder.String()
}

func HasAnyAvailable(statuses []ToolStatus) bool {
	for _, status := range statuses {
		if status.Available {
			return true
		}
	}
	return false
}

func HasMissing(statuses []ToolStatus) bool {
	for _, status := range statuses {
		if !status.Available {
			return true
		}
	}
	return false
}
