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
		"Planned tools are roadmap contracts",
	} {
		if !strings.Contains(result.Observation, expected) {
			t.Fatalf("catalog observation missing %q:\n%s", expected, result.Observation)
		}
	}
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
