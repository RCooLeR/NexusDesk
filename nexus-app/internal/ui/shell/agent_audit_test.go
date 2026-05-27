package shell

import (
	"strings"
	"testing"

	agentSvc "nexusdesk/internal/services/agent"
	metadataSvc "nexusdesk/internal/services/metadata"
)

func TestAgentJobLabelCompactsLongPrompt(t *testing.T) {
	label := agentJobLabel(strings.Repeat("word ", 30))
	if len(label) > 80 || strings.Contains(label, "\n") {
		t.Fatalf("unexpected compact label: %q", label)
	}
	if got := agentJobLabel("   "); got != "Agent run" {
		t.Fatalf("unexpected empty label: %q", got)
	}
}

func TestToolRunForMetadataCopiesToolResult(t *testing.T) {
	record := metadataSvc.AgentRunRecord{ID: "agent-1", JobID: "job-1"}
	tool := toolRunForMetadata(record, agentSvc.ToolResult{
		Name:        "read_context",
		Risk:        "low",
		Mutated:     true,
		Args:        map[string]string{"relPath": "README.md"},
		Observation: "ok",
		StartedAt:   "2026-05-27T12:00:00Z",
		CompletedAt: "2026-05-27T12:00:01Z",
	}, 2)
	if tool.AgentRunID != "agent-1" || tool.JobID != "job-1" || tool.Sequence != 2 || !tool.Mutated {
		t.Fatalf("unexpected metadata tool run: %#v", tool)
	}
	if tool.Args["relPath"] != "README.md" || tool.StartedAt.IsZero() || tool.CompletedAt.IsZero() {
		t.Fatalf("unexpected metadata conversion: %#v", tool)
	}
}
