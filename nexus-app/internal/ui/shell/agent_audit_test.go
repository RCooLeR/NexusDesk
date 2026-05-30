package shell

import (
	"strings"
	"testing"
	"time"

	agentSvc "nexusdesk/internal/services/agent"
	metadataSvc "nexusdesk/internal/services/metadata"
)

func TestAgentAuditControllerOwnsView(t *testing.T) {
	view := &View{}
	controller := newAgentAuditController(view)
	if controller.view != view {
		t.Fatalf("expected agent audit controller to retain owning view")
	}
}

func TestAgentJobLabelCompactsLongPrompt(t *testing.T) {
	label := agentJobLabel(strings.Repeat("word ", 30))
	if len(label) > 80 || strings.Contains(label, "\n") {
		t.Fatalf("unexpected compact label: %q", label)
	}
	if got := agentJobLabel("   "); got != "Agent run" {
		t.Fatalf("unexpected empty label: %q", got)
	}
}

func TestFormatAgentAuditDetailIncludesRunAndToolData(t *testing.T) {
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	text := formatAgentAuditDetail(metadataSvc.AgentRunRecord{
		ID:         "agent-1",
		JobID:      "job-1",
		Prompt:     "Review project",
		Status:     "success",
		Message:    "Done",
		Model:      "qwen3-coder:30b",
		ModelRoute: "Main coding model",
		Iterations: 2,
		Plan:       []metadataSvc.AgentPlanStep{{Step: "Inspect", Status: "completed"}},
		StartedAt:  started,
		DurationMs: 1200,
	}, []metadataSvc.ToolRunRecord{{
		Sequence:    1,
		ToolName:    "read_context",
		Risk:        "low",
		Args:        map[string]string{"relPath": "README.md"},
		Observation: "ok",
	}})
	for _, expected := range []string{"Agent Run", "agent-1", "qwen3-coder:30b", "Main coding model", "Review project", "Tool Runs", "#1 read_context", "relPath=README.md"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("audit detail missing %q:\n%s", expected, text)
		}
	}
}

func TestAgentAuditRowsReturnEmptyState(t *testing.T) {
	rows := agentAuditRows(nil, func(metadataSvc.AgentRunRecord) {})
	if len(rows) != 1 {
		t.Fatalf("expected one empty-state row, got %d", len(rows))
	}
}

func TestFormatAuditArgsSortsKeys(t *testing.T) {
	text := formatAuditArgs(map[string]string{"z": "last", "a": "first"})
	if text != "a=first, z=last" {
		t.Fatalf("unexpected sorted args: %q", text)
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
