package shell

import "testing"

func TestDefaultToolWindowRegistryRegistersCoreTools(t *testing.T) {
	registry := defaultToolWindowRegistry()
	for _, id := range []string{
		"project",
		"search",
		"problems",
		"git",
		"data",
		"artifacts",
		"operations",
		"tasks",
		"jobs",
		"history",
		"approvals",
		"diagnostics",
		"activity",
	} {
		if _, ok := registry.Lookup(id); !ok {
			t.Fatalf("expected tool window %q to be registered", id)
		}
	}
}

func TestRailToolWindowsReadFromRegistry(t *testing.T) {
	left := leftRailToolWindows()
	right := rightRailToolWindows()

	if len(left) == 0 || left[0].ID != "project" || !left[0].OpenProject {
		t.Fatalf("expected project to be first left tool, got %#v", left)
	}
	if len(right) == 0 || right[0].ID != "assistant" || !right[0].FocusAssistant {
		t.Fatalf("expected assistant to be first right tool, got %#v", right)
	}
	if left[1].ButtonLabel() != "Alt+2  Search" {
		t.Fatalf("expected shortcut label from shared registration, got %q", left[1].ButtonLabel())
	}
}
