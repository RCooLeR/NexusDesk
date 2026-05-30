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

func TestCommandPaletteIncludesSafeAgentGuide(t *testing.T) {
	view := &View{state: NewState()}
	commands := view.commandPaletteActions()
	foundBetaFeedback := false
	foundProviderSetup := false
	foundSafeAgent := false
	foundSampleWorkflow := false
	foundKnownLimitations := false
	foundSmokeChecklist := false
	foundAppDataCleanup := false
	foundReleaseHygiene := false
	foundCheckUpdates := false
	foundPackageOwnership := false
	foundContributor := false
	for _, command := range commands {
		switch command.ID {
		case "help.provider_setup":
			if command.Title != "Provider Setup Wizard" || command.Group != "Help" || command.Run == nil {
				t.Fatalf("unexpected provider-setup command: %#v", command)
			}
			foundProviderSetup = true
		case "help.safe_agent":
			if command.Title != "Safe Agent Guide" || command.Group != "Help" || command.Run == nil {
				t.Fatalf("unexpected safe-agent command: %#v", command)
			}
			foundSafeAgent = true
		case "help.sample_workflow":
			if command.Title != "Sample Workflow Guide" || command.Group != "Help" || command.Run == nil {
				t.Fatalf("unexpected sample-workflow command: %#v", command)
			}
			foundSampleWorkflow = true
		case "help.beta_feedback":
			if command.Title != "Beta Feedback & Release Notes" || command.Group != "Help" || command.Run == nil {
				t.Fatalf("unexpected beta-feedback command: %#v", command)
			}
			foundBetaFeedback = true
		case "help.known_limitations":
			if command.Title != "Known Limitations" || command.Group != "Help" || command.Run == nil {
				t.Fatalf("unexpected known-limitations command: %#v", command)
			}
			foundKnownLimitations = true
		case "help.smoke_checklist":
			if command.Title != "Clean-Machine Smoke Checklist" || command.Group != "Help" || command.Run == nil {
				t.Fatalf("unexpected smoke-checklist command: %#v", command)
			}
			foundSmokeChecklist = true
		case "help.app_data_cleanup":
			if command.Title != "App Data & Uninstall Cleanup" || command.Group != "Help" || command.Run == nil {
				t.Fatalf("unexpected app-data-cleanup command: %#v", command)
			}
			foundAppDataCleanup = true
		case "help.release_hygiene":
			if command.Title != "Release Hygiene & Antivirus Notes" || command.Group != "Help" || command.Run == nil {
				t.Fatalf("unexpected release-hygiene command: %#v", command)
			}
			foundReleaseHygiene = true
		case "help.check_updates":
			if command.Title != "Check for Updates" || command.Group != "Help" || command.Run == nil {
				t.Fatalf("unexpected check-updates command: %#v", command)
			}
			foundCheckUpdates = true
		case "help.package_ownership":
			if command.Title != "Internal Package Ownership" || command.Group != "Help" || command.Run == nil {
				t.Fatalf("unexpected package-ownership command: %#v", command)
			}
			foundPackageOwnership = true
		case "help.contributor":
			if command.Title != "Contributor Setup & Standards" || command.Group != "Help" || command.Run == nil {
				t.Fatalf("unexpected contributor command: %#v", command)
			}
			foundContributor = true
		}
	}
	if !foundProviderSetup || !foundSafeAgent || !foundSampleWorkflow || !foundBetaFeedback || !foundKnownLimitations || !foundSmokeChecklist || !foundAppDataCleanup || !foundReleaseHygiene || !foundCheckUpdates || !foundPackageOwnership || !foundContributor {
		t.Fatalf("missing help commands: provider_setup=%t safe_agent=%t sample_workflow=%t beta_feedback=%t known_limitations=%t smoke_checklist=%t app_data_cleanup=%t release_hygiene=%t check_updates=%t package_ownership=%t contributor=%t in %#v", foundProviderSetup, foundSafeAgent, foundSampleWorkflow, foundBetaFeedback, foundKnownLimitations, foundSmokeChecklist, foundAppDataCleanup, foundReleaseHygiene, foundCheckUpdates, foundPackageOwnership, foundContributor, commands)
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
