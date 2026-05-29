package tools

import (
	"context"
	"strings"
	"testing"

	"nexusdesk/internal/services/agent"
)

func TestDefaultToolCatalogIncludesAllRegisteredTools(t *testing.T) {
	dispatcher := NewDefaultDispatcher(Dependencies{})
	catalog := DefaultToolCatalog()
	implemented := map[string]bool{}
	for _, entry := range catalog {
		if entry.Status == ToolStatusImplemented {
			implemented[entry.Descriptor.Name] = true
		}
	}
	for _, descriptor := range dispatcher.ToolDescriptors() {
		if !implemented[descriptor.Name] {
			t.Fatalf("registered tool %q is missing from implemented catalog", descriptor.Name)
		}
	}
	for _, entry := range catalog {
		if entry.Status == ToolStatusPlanned && implemented[entry.Descriptor.Name] {
			t.Fatalf("planned tool %q duplicates an implemented tool", entry.Descriptor.Name)
		}
	}
}

func TestDefaultDispatcherListsToolCatalog(t *testing.T) {
	dispatcher := NewDefaultDispatcher(Dependencies{})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "list_tool_catalog"}, agent.Request{})
	if err != nil {
		t.Fatalf("list_tool_catalog returned error: %v", err)
	}
	for _, expected := range []string{
		"NexusDesk native tool catalog",
		"[implemented] run_terminal_command",
		"[planned] browser_navigate",
		"[planned] call_mcp_tool",
		"Controls=rooted-scope,approval,audit",
		"Planned tools are roadmap contracts",
	} {
		if !strings.Contains(result.Observation, expected) {
			t.Fatalf("catalog observation missing %q:\n%s", expected, result.Observation)
		}
	}
}

func TestDefaultToolCatalogAnnotatesRiskControls(t *testing.T) {
	catalog := DefaultToolCatalog()
	for _, entry := range catalog {
		if entry.Descriptor.Risk == "high" && !containsToolControl(entry.Controls, "approval") {
			t.Fatalf("high-risk catalog entry %q missing approval control: %#v", entry.Descriptor.Name, entry.Controls)
		}
		if entry.Descriptor.Risk == "high" && !containsToolControl(entry.Controls, "rollback-or-mitigation") {
			t.Fatalf("high-risk catalog entry %q missing rollback/mitigation control: %#v", entry.Descriptor.Name, entry.Controls)
		}
	}
}

func TestValidateDefaultToolCatalogEnforcesPlannedToolGate(t *testing.T) {
	health := ValidateDefaultToolCatalog()
	if !health.OK() {
		t.Fatalf("expected default tool catalog to be healthy, got %#v", health.Violations)
	}
	if health.ImplementedCount == 0 || health.PlannedCount == 0 {
		t.Fatalf("expected implemented and planned counts, got %#v", health)
	}
}

func TestValidateToolCatalogRejectsExecutablePlannedTool(t *testing.T) {
	descriptor := agent.ToolDescriptor{Name: "browser_navigate", Risk: "medium"}
	health := ValidateToolCatalog([]ToolCatalogEntry{{
		Descriptor: descriptor,
		Category:   "browser",
		Status:     ToolStatusPlanned,
		Controls:   []string{"rooted-scope", "approval", "audit", "timeout", "cancellation", "output-cap", "redaction"},
	}}, []agent.ToolDescriptor{descriptor})
	if health.OK() {
		t.Fatalf("expected planned executable violation")
	}
	joined := strings.Join(health.Violations, "\n")
	if !strings.Contains(joined, "planned tool \"browser_navigate\" is registered as executable") {
		t.Fatalf("expected planned executable violation, got:\n%s", joined)
	}
}

func TestValidateToolCatalogRejectsMissingRiskControls(t *testing.T) {
	descriptor := agent.ToolDescriptor{Name: "dangerous_tool", Risk: "high"}
	health := ValidateToolCatalog([]ToolCatalogEntry{{
		Descriptor: descriptor,
		Category:   "test",
		Status:     ToolStatusImplemented,
		Controls:   []string{"approval"},
	}}, []agent.ToolDescriptor{descriptor})
	if health.OK() {
		t.Fatalf("expected missing control violations")
	}
	joined := strings.Join(health.Violations, "\n")
	for _, expected := range []string{
		"missing control \"audit\"",
		"missing control \"rollback-or-mitigation\"",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %q in violations:\n%s", expected, joined)
		}
	}
}

func containsToolControl(controls []string, expected string) bool {
	for _, control := range controls {
		if control == expected {
			return true
		}
	}
	return false
}

func TestDefaultDispatcherFiltersToolCatalog(t *testing.T) {
	dispatcher := NewDefaultDispatcher(Dependencies{})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "list_tool_catalog", Args: map[string]string{"status": "planned", "category": "browser"}}, agent.Request{})
	if err != nil {
		t.Fatalf("filtered list_tool_catalog returned error: %v", err)
	}
	if !strings.Contains(result.Observation, "[planned] browser_navigate") {
		t.Fatalf("expected planned browser tools, got:\n%s", result.Observation)
	}
	if strings.Contains(result.Observation, "[implemented] web_fetch") {
		t.Fatalf("did not expect implemented browser tools when status=planned:\n%s", result.Observation)
	}
}
