package shell

import (
	"fmt"
	"sort"
	"strings"
	"time"

	metadataSvc "nexusdesk/internal/services/metadata"
)

func formatAgentAuditDetail(run metadataSvc.AgentRunRecord, tools []metadataSvc.ToolRunRecord) string {
	var builder strings.Builder
	builder.WriteString("Agent Run\n")
	writeAuditLine(&builder, "ID", run.ID)
	writeAuditLine(&builder, "Job", run.JobID)
	writeAuditLine(&builder, "Status", run.Status)
	writeAuditLine(&builder, "Iterations", fmt.Sprintf("%d", run.Iterations))
	writeAuditLine(&builder, "Stop reason", run.StopReason)
	writeAuditLine(&builder, "Started", formatAuditTime(run.StartedAt))
	writeAuditLine(&builder, "Completed", formatAuditTime(run.CompletedAt))
	writeAuditLine(&builder, "Duration", fmt.Sprintf("%d ms", run.DurationMs))
	writeAuditLine(&builder, "Sources", strings.Join(run.SourcePaths, ", "))
	builder.WriteString("\nPrompt\n")
	builder.WriteString(strings.TrimSpace(run.Prompt))
	builder.WriteString("\n\nFinal Message\n")
	builder.WriteString(strings.TrimSpace(run.Message))
	builder.WriteString("\n")
	if len(run.Plan) > 0 {
		builder.WriteString("\nPlan\n")
		for _, step := range run.Plan {
			builder.WriteString("- [")
			builder.WriteString(firstNonEmpty(step.Status, "unknown"))
			builder.WriteString("] ")
			builder.WriteString(step.Step)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\nTool Runs\n")
	if len(tools) == 0 {
		builder.WriteString("(none)\n")
		return builder.String()
	}
	sort.SliceStable(tools, func(i, j int) bool {
		return tools[i].Sequence < tools[j].Sequence
	})
	for _, tool := range tools {
		builder.WriteString("\n")
		builder.WriteString(formatToolAuditDetail(tool))
	}
	return builder.String()
}

func formatToolAuditDetail(tool metadataSvc.ToolRunRecord) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("#%d %s", tool.Sequence, tool.ToolName))
	if tool.Risk != "" {
		builder.WriteString(" risk=")
		builder.WriteString(tool.Risk)
	}
	if tool.Mutated {
		builder.WriteString(" mutated=true")
	}
	builder.WriteString("\n")
	if len(tool.Args) > 0 {
		builder.WriteString("Args: ")
		builder.WriteString(formatAuditArgs(tool.Args))
		builder.WriteString("\n")
	}
	if tool.Error != "" {
		builder.WriteString("Error: ")
		builder.WriteString(tool.Error)
		builder.WriteString("\n")
	}
	if tool.Observation != "" {
		builder.WriteString("Observation: ")
		builder.WriteString(strings.TrimSpace(tool.Observation))
		builder.WriteString("\n")
	}
	return builder.String()
}

func agentAuditTitle(run metadataSvc.AgentRunRecord) string {
	return firstNonEmpty(run.ID, "Agent run") + " - " + compactAgentAuditMessage(run.Prompt)
}

func agentAuditMeta(run metadataSvc.AgentRunRecord) string {
	return fmt.Sprintf("%s - %d iteration(s) - %s", firstNonEmpty(run.Status, "unknown"), run.Iterations, formatAuditTime(run.StartedAt))
}

func compactAgentAuditMessage(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "(empty)"
	}
	if len(value) > 100 {
		return value[:97] + "..."
	}
	return value
}

func writeAuditLine(builder *strings.Builder, key string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "-"
	}
	builder.WriteString(key)
	builder.WriteString(": ")
	builder.WriteString(value)
	builder.WriteString("\n")
}

func formatAuditArgs(args map[string]string) string {
	if len(args) == 0 {
		return ""
	}
	keys := make([]string, 0, len(args))
	for key := range args {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+args[key])
	}
	return strings.Join(parts, ", ")
}

func formatAuditTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.UTC().Format(time.RFC3339)
}
