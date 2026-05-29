package externalagents

import (
	"errors"
	"strings"
	"testing"
)

func TestProbeDetectsKnownExternalAgentCLIs(t *testing.T) {
	statuses := Probe(Options{LookupPath: fixedLookup(map[string]string{
		"codex":    "/usr/local/bin/codex",
		"opencode": "/usr/local/bin/opencode",
	})})
	if len(statuses) != 3 {
		t.Fatalf("expected catalog of three tools, got %d", len(statuses))
	}
	available := map[string]ToolStatus{}
	for _, status := range statuses {
		if status.Available {
			available[status.ID] = status
		}
	}
	if available["codex"].Command != "codex" || available["opencode"].Command != "opencode" {
		t.Fatalf("expected codex and opencode to be detected, got %#v", available)
	}
	if _, ok := available["claude-code"]; ok {
		t.Fatalf("did not expect claude-code to be available: %#v", available["claude-code"])
	}
}

func TestFormatMarkdownIncludesExecutionPolicy(t *testing.T) {
	statuses := Probe(Options{LookupPath: fixedLookup(map[string]string{"claude": "/opt/bin/claude"})})
	report := FormatMarkdown(statuses)
	if !strings.Contains(report, "Claude Code: available") {
		t.Fatalf("expected Claude Code availability in report:\n%s", report)
	}
	if !strings.Contains(report, ExecutionPolicy) {
		t.Fatalf("expected execution policy in report:\n%s", report)
	}
	if !strings.Contains(Summary(statuses), "1/3 external coding-agent CLIs detected") {
		t.Fatalf("unexpected summary: %s", Summary(statuses))
	}
}

func TestPlanInvocationBuildsApprovalBackedJobContract(t *testing.T) {
	plan, err := PlanInvocation(InvocationRequest{
		ToolID:        "claude",
		WorkspaceRoot: "/work/project",
		Prompt:        "review the repository",
		LookupPath:    fixedLookup(map[string]string{"claude": "/opt/bin/claude"}),
	})
	if err != nil {
		t.Fatalf("PlanInvocation returned error: %v", err)
	}
	if plan.ToolID != "claude-code" || plan.JobKind != "external-agent-run" {
		t.Fatalf("unexpected plan identity: %#v", plan)
	}
	if !plan.RequiresApproval || !plan.RequiresAudit || !plan.Cancellable {
		t.Fatalf("expected production guardrails in plan: %#v", plan)
	}
	if plan.PromptDelivery != "stdin" || len(plan.Args) != 0 {
		t.Fatalf("expected stdin prompt delivery without shell args: %#v", plan)
	}
	report := FormatInvocationPlan(plan)
	for _, expected := range []string{"External agent plan: Claude Code", "Prompt delivery: stdin", "Policy: Plan only"} {
		if !strings.Contains(report, expected) {
			t.Fatalf("expected %q in plan report:\n%s", expected, report)
		}
	}
}

func TestPlanInvocationRejectsUnavailableTool(t *testing.T) {
	_, err := PlanInvocation(InvocationRequest{
		ToolID:        "opencode",
		WorkspaceRoot: "/work/project",
		Prompt:        "implement tests",
		LookupPath:    fixedLookup(nil),
	})
	if err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("expected unavailable tool error, got %v", err)
	}
}

func fixedLookup(paths map[string]string) func(string) (string, error) {
	return func(command string) (string, error) {
		if path := strings.TrimSpace(paths[command]); path != "" {
			return path, nil
		}
		return "", errors.New("not found")
	}
}
