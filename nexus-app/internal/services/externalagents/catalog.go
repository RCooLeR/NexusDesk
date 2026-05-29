// Package externalagents detects optional coding-agent CLIs without executing them.
package externalagents

import (
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	jobsSvc "nexusdesk/internal/services/jobs"
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

type InvocationRequest struct {
	ToolID        string
	WorkspaceRoot string
	Prompt        string
	LookupPath    func(string) (string, error)
}

type InvocationPlan struct {
	ToolID            string
	Label             string
	Command           string
	CommandPath       string
	Args              []string
	WorkingDirectory  string
	PromptBytes       int
	PromptDelivery    string
	JobKind           string
	RequiresApproval  bool
	RequiresAudit     bool
	Cancellable       bool
	ExecutionPolicy   string
	OutputCaptureHint string
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

func PlanInvocation(request InvocationRequest) (InvocationPlan, error) {
	toolID := normalizeToolID(request.ToolID)
	if toolID == "" {
		return InvocationPlan{}, errors.New("external agent tool id is required")
	}
	workspaceRoot := strings.TrimSpace(request.WorkspaceRoot)
	if workspaceRoot == "" {
		return InvocationPlan{}, errors.New("workspace root is required before planning an external agent run")
	}
	prompt := strings.TrimSpace(request.Prompt)
	if prompt == "" {
		return InvocationPlan{}, errors.New("prompt is required before planning an external agent run")
	}
	status, ok := FindStatus(Probe(Options{LookupPath: request.LookupPath}), toolID)
	if !ok {
		return InvocationPlan{}, fmt.Errorf("external agent tool %q is not in the catalog", request.ToolID)
	}
	if !status.Available {
		return InvocationPlan{}, fmt.Errorf("%s is not available on PATH", status.Label)
	}
	spec, _ := jobsSvc.SlowWorkflowSpec(jobsSvc.KindExternalAgentRun)
	return InvocationPlan{
		ToolID:            status.ID,
		Label:             status.Label,
		Command:           status.Command,
		CommandPath:       status.Path,
		Args:              []string{},
		WorkingDirectory:  workspaceRoot,
		PromptBytes:       len([]byte(prompt)),
		PromptDelivery:    "stdin",
		JobKind:           jobsSvc.KindExternalAgentRun,
		RequiresApproval:  true,
		RequiresAudit:     spec.AuditRequired,
		Cancellable:       spec.Cancellable,
		ExecutionPolicy:   "Plan only. Do not execute until the user approves an external-agent job and NexusDesk can capture output, cancellation, audit, and rollback/artifact metadata.",
		OutputCaptureHint: ".nexusdesk/jobs/<job-id>/external-agent-output.txt",
	}, nil
}

func FindStatus(statuses []ToolStatus, toolID string) (ToolStatus, bool) {
	toolID = normalizeToolID(toolID)
	for _, status := range statuses {
		if normalizeToolID(status.ID) == toolID {
			return status, true
		}
		for _, command := range status.Commands {
			if normalizeToolID(command) == toolID {
				return status, true
			}
		}
		if normalizeToolID(status.Label) == toolID {
			return status, true
		}
	}
	return ToolStatus{}, false
}

func FormatInvocationPlan(plan InvocationPlan) string {
	lines := []string{
		fmt.Sprintf("External agent plan: %s (%s)", plan.Label, plan.ToolID),
		"Job kind: " + plan.JobKind,
		"Command: " + firstNonEmpty(plan.CommandPath, plan.Command),
		"Working directory: " + plan.WorkingDirectory,
		fmt.Sprintf("Prompt delivery: %s (%d bytes)", plan.PromptDelivery, plan.PromptBytes),
		fmt.Sprintf("Requires approval: %t", plan.RequiresApproval),
		fmt.Sprintf("Requires audit: %t", plan.RequiresAudit),
		fmt.Sprintf("Cancellable: %t", plan.Cancellable),
		"Output capture: " + plan.OutputCaptureHint,
		"Policy: " + plan.ExecutionPolicy,
	}
	return strings.Join(lines, "\n")
}

func normalizeToolID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "_", "-")
	if value == "claude" {
		return "claude-code"
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
