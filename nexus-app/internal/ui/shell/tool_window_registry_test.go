package shell

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

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

func TestRailIconButtonIsIconFirst(t *testing.T) {
	search, ok := defaultToolWindowRegistry().Lookup("search")
	if !ok {
		t.Fatal("expected search tool registration")
	}
	button := newRailIconButton(search, nil, nil, nil)
	if button.Text != "" {
		t.Fatalf("expected icon-first rail button without visible text, got %q", button.Text)
	}
	if button.Icon == nil {
		t.Fatal("expected rail button icon")
	}
}

func TestRailIconButtonShowsTooltipOnHover(t *testing.T) {
	search, ok := defaultToolWindowRegistry().Lookup("search")
	if !ok {
		t.Fatal("expected search tool registration")
	}
	hovered := ""
	left := false
	button := newRailIconButton(search, nil, func(text string) {
		hovered = text
	}, func() {
		left = true
	})

	button.MouseIn(nil)
	if hovered != "Search (Alt+2)" {
		t.Fatalf("expected rail tooltip text on hover, got %q", hovered)
	}
	button.MouseOut()
	if !left {
		t.Fatal("expected hover leave callback")
	}
}

func TestToolWindowRegistryShortcutRoutingUsesAltModifier(t *testing.T) {
	registry := defaultToolWindowRegistry()
	tools := registry.ShortcutTools()
	if len(tools) == 0 {
		t.Fatal("expected registered shortcut tools")
	}
	search, ok := registry.Lookup("search")
	if !ok {
		t.Fatal("expected search tool registration")
	}
	shortcut, ok := shortcutToolWindow(search).(*desktop.CustomShortcut)
	if !ok {
		t.Fatalf("expected desktop shortcut, got %#v", shortcutToolWindow(search))
	}
	if shortcut.KeyName != fyne.Key2 || shortcut.Modifier != fyne.KeyModifierAlt {
		t.Fatalf("expected Alt+2 search shortcut, got key=%s modifier=%v", shortcut.KeyName, shortcut.Modifier)
	}
}

func TestToolWindowRegistryCoversRailKeyboardNavigation(t *testing.T) {
	registry := defaultToolWindowRegistry()
	shortcutTools := map[string]bool{}
	for _, tool := range registry.ShortcutTools() {
		shortcutTools[tool.ID] = true
	}
	for _, tool := range append(leftRailToolWindows(), rightRailToolWindows()...) {
		if !shortcutTools[tool.ID] {
			t.Fatalf("expected rail tool %q to be reachable by keyboard shortcut", tool.ID)
		}
	}
}
