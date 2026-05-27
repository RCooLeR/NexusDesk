package agent

import (
	"fmt"
	"strings"
)

func systemPrompt() string {
	return strings.Join([]string{
		"You are Nexus Agent, the local-first assistant inside Nexus Augentic Studio.",
		"Use a ReAct loop: concise Thought, exactly one Action when a tool is needed, Observation from the tool, then Final Answer when done.",
		"Use update_plan for multi-step work and keep exactly one step in_progress.",
		"Do not claim a file or artifact was created, saved, written, or modified unless a tool observation confirmed the mutation.",
		"High-impact writes, shell, deletes, moves, and Docker actions require explicit approval from Nexus before they can run.",
	}, "\n")
}

func runtimePrompt(request Request, state runState) string {
	var builder strings.Builder
	builder.WriteString(systemPrompt())
	builder.WriteString("\n\nAvailable built-in tool:\n")
	builder.WriteString("- update_plan: Replace visible plan steps. Risk=low Inputs=steps.\n")
	builder.WriteString("\nAdditional deterministic tools are provided by the native tool executor. If a needed tool is unavailable or approval is missing, explain the limitation in the final answer.\n")
	builder.WriteString("\nOutput format:\n")
	builder.WriteString("Thought: ...\nAction: tool_name({\"key\":\"value\"})\n")
	builder.WriteString("or\nFinal Answer: ...\n")
	if request.WorkspaceRoot != "" {
		builder.WriteString("\nWorkspace root is available to approved tools.\n")
	}
	if request.ContextContent != "" {
		builder.WriteString("\nQuoted workspace context: ")
		builder.WriteString(strings.TrimSpace(request.ContextRelPath))
		builder.WriteString("\nBEGIN_NEXUS_AGENT_CONTEXT\n")
		builder.WriteString(sanitizeContext(request.ContextContent))
		builder.WriteString("\nEND_NEXUS_AGENT_CONTEXT\n")
	}
	builder.WriteString("\nUser request:\n")
	builder.WriteString(strings.TrimSpace(request.Prompt))
	if len(state.plan) > 0 {
		builder.WriteString("\n\nCurrent plan:\n")
		for _, step := range state.plan {
			builder.WriteString(fmt.Sprintf("- [%s] %s\n", step.Status, step.Step))
		}
	}
	if len(state.history) > 0 {
		builder.WriteString("\nRecent observations:\n")
		for _, item := range state.history {
			builder.WriteString(item)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func finalizationPrompt(request Request, state runState) string {
	var builder strings.Builder
	builder.WriteString(systemPrompt())
	builder.WriteString("\n\nThe backend guard is stopping further tool calls. Do not request more tools. Produce a concise final answer from the completed observations.\n")
	builder.WriteString("\nUser request:\n")
	builder.WriteString(strings.TrimSpace(request.Prompt))
	if len(state.toolCalls) > 0 {
		builder.WriteString("\n\nCompleted tool calls:\n")
		for _, call := range state.toolCalls {
			builder.WriteString("- ")
			builder.WriteString(call.Name)
			if call.Error != "" {
				builder.WriteString(" error: ")
				builder.WriteString(call.Error)
			} else {
				builder.WriteString(": ")
				builder.WriteString(limitText(call.Observation, 500))
			}
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\nFinal Answer:")
	return builder.String()
}

func sanitizeContext(content string) string {
	replacer := strings.NewReplacer(
		"BEGIN_NEXUS_AGENT_CONTEXT", "BEGIN_NEXUS_AGENT_CONTEXT_ESCAPED",
		"END_NEXUS_AGENT_CONTEXT", "END_NEXUS_AGENT_CONTEXT_ESCAPED",
		"```", "'''",
	)
	return replacer.Replace(content)
}
