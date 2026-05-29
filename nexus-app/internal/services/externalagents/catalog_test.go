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

func fixedLookup(paths map[string]string) func(string) (string, error) {
	return func(command string) (string, error) {
		if path := strings.TrimSpace(paths[command]); path != "" {
			return path, nil
		}
		return "", errors.New("not found")
	}
}
