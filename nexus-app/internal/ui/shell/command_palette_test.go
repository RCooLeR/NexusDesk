package shell

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

func TestCommandPaletteShortcutUsesShiftControlP(t *testing.T) {
	shortcut, ok := shortcutCommandPalette().(*desktop.CustomShortcut)
	if !ok {
		t.Fatalf("unexpected command palette shortcut type: %#v", shortcutCommandPalette())
	}
	wantModifier := fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift
	if shortcut.KeyName != fyne.KeyP || shortcut.Modifier != wantModifier {
		t.Fatalf("unexpected command palette shortcut: %#v", shortcut)
	}
}

func TestFilterCommandPaletteActionsScoresAndFilters(t *testing.T) {
	commands := []commandPaletteAction{
		{ID: "open", Title: "Open Workspace", Detail: "Choose a folder", Group: "File", Shortcut: "Ctrl+O"},
		{ID: "quick", Title: "Quick Open", Detail: "Search workspace files", Group: "Navigate", Shortcut: "Ctrl+P"},
		{ID: "git", Title: "Show Git", Detail: "Status and diff panel", Group: "Workbench", Disabled: true},
	}
	results := filterCommandPaletteActions(commands, "open")
	if len(results) != 2 || results[0].ID != "open" || results[1].ID != "quick" {
		t.Fatalf("unexpected open results: %#v", results)
	}
	results = filterCommandPaletteActions(commands, "ctrl+p")
	if len(results) != 1 || results[0].ID != "quick" {
		t.Fatalf("unexpected shortcut results: %#v", results)
	}
	results = filterCommandPaletteActions(commands, "workbench")
	if len(results) != 1 || results[0].ID != "git" {
		t.Fatalf("unexpected group results: %#v", results)
	}
}

func TestFilterCommandPaletteActionsKeepsDisabledAfterEnabled(t *testing.T) {
	commands := []commandPaletteAction{
		{ID: "disabled", Title: "Refresh Workspace", Disabled: true},
		{ID: "enabled", Title: "Open Workspace"},
	}
	results := filterCommandPaletteActions(commands, "")
	if len(results) != 2 || results[0].ID != "enabled" || results[1].ID != "disabled" {
		t.Fatalf("expected enabled command before disabled command, got %#v", results)
	}
}

func TestCommandPaletteStatusText(t *testing.T) {
	if text := commandPaletteStatusText(0, "missing"); !strings.Contains(text, "No matching commands") {
		t.Fatalf("unexpected empty status: %q", text)
	}
	if text := commandPaletteStatusText(3, ""); !strings.Contains(text, "Type to filter") {
		t.Fatalf("unexpected initial status: %q", text)
	}
	if text := commandPaletteStatusText(2, "git"); !strings.Contains(text, "command match") {
		t.Fatalf("unexpected match status: %q", text)
	}
}

func TestCommandPaletteTitleMarksUnavailableCommands(t *testing.T) {
	title := commandPaletteTitle(commandPaletteAction{
		Title:    "Refresh Workspace",
		Group:    "Workspace",
		Shortcut: "Ctrl+R",
		Disabled: true,
	})
	for _, expected := range []string{"Refresh Workspace", "unavailable", "Workspace", "Ctrl+R"} {
		if !strings.Contains(title, expected) {
			t.Fatalf("expected title to contain %q, got %q", expected, title)
		}
	}
}
