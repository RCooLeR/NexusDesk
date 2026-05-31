package shell

import (
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	readinessSvc "nexusdesk/internal/services/readiness"
	recentWorkspacesSvc "nexusdesk/internal/services/recentworkspaces"
	settingsSvc "nexusdesk/internal/services/settings"
	startupSvc "nexusdesk/internal/services/startup"
)

func TestEditorControllerInitializesTabState(t *testing.T) {
	_ = fynetest.NewTempApp(t)

	welcome := container.NewTabItem("Welcome", widget.NewLabel("Ready."))
	view := &View{}
	controller := newEditorController(view, "welcome-1", welcome)

	if controller.view != view {
		t.Fatal("expected controller to retain parent view")
	}
	if controller.tabs == nil || len(controller.tabs.Items) != 1 {
		t.Fatalf("expected one initial editor tab, got %#v", controller.tabs)
	}
	if controller.openTabs["welcome-1"] != controller.tabs.Items[0] {
		t.Fatalf("expected open tab map to point at initial tab")
	}
	if controller.tabIDs[controller.tabs.Items[0]] != "welcome-1" {
		t.Fatalf("expected reverse tab map to point at initial ID")
	}
	if controller.previews == nil || controller.textEditors == nil {
		t.Fatal("expected editor preview and text binding maps")
	}
}

func TestFormatWelcomeReadinessMarkdownKeepsHomeSummaryCompact(t *testing.T) {
	snapshot := readinessSvc.Collect(readinessSvc.Options{
		Settings:        settingsSvc.Defaults(),
		StartupRecovery: startupSvc.Status{},
		LookupPath: func(string) (string, error) {
			return "", readinessTestError("missing")
		},
		ExternalAgentLookupPath: func(string) (string, error) {
			return "", readinessTestError("missing")
		},
	})
	text := formatWelcomeReadinessMarkdown(snapshot)
	if !strings.Contains(text, "Production failure gates") {
		t.Fatalf("expected readiness summary to include failure gates:\n%s", text)
	}
	if strings.Contains(text, "workspace/readiness/jobs") || strings.Contains(text, "internal/services/readiness:") {
		t.Fatalf("home readiness should not include full failure matrix details:\n%s", text)
	}
	if strings.Contains(strings.ToLower(text), "cockpit") || strings.Contains(strings.ToLower(text), "dashboard") {
		t.Fatalf("home readiness should avoid dashboard/cockpit framing:\n%s", text)
	}
	for _, line := range strings.Split(text, "\n") {
		if len(line) > 260 {
			t.Fatalf("home readiness line is too wide for resizable layout (%d chars): %s", len(line), line)
		}
	}
}

func TestFormatWelcomeOnboardingMarkdownShowsFirstRunFlow(t *testing.T) {
	text := formatWelcomeOnboardingMarkdown(domain.Workspace{}, settingsSvc.Settings{}, "")
	for _, expected := range []string{
		"Provider setup",
		"Open Model Settings",
		"Test connection",
		"Workspace",
		"Open a trusted sample workspace",
		"Sample workflow",
		"Diagnostics",
		"redacted issue report",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected onboarding text to contain %q:\n%s", expected, text)
		}
	}
	ready := formatWelcomeOnboardingMarkdown(
		domain.Workspace{Name: "repo", Root: "C:/repo"},
		settingsSvc.Settings{Provider: "ollama", BaseURL: "http://localhost:11434/v1", Model: "qwen3:8b"},
		"",
	)
	for _, expected := range []string{"**[OK] Provider setup:** ollama/qwen3:8b configured", "**[OK] Workspace:** repo is open"} {
		if !strings.Contains(ready, expected) {
			t.Fatalf("expected ready onboarding text to contain %q:\n%s", expected, ready)
		}
	}
}

func TestFormatWelcomeCrashRecoveryMarkdownIncludesActions(t *testing.T) {
	text := formatWelcomeCrashRecoveryMarkdown(startupSvc.Status{
		PreviousUnclean:   true,
		Message:           "Previous NexusDesk run did not record a clean exit.",
		PreviousStartedAt: time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC),
	})
	for _, expected := range []string{
		"Previous session",
		"did not record a clean exit",
		"Previous start",
		"Diagnostics",
		"Jobs",
		"Agent Audit",
		"History",
		"redacted issue report",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected crash recovery text to contain %q:\n%s", expected, text)
		}
	}
}

func TestShowEditorEmptyWelcomeOnlyForFirstLaunch(t *testing.T) {
	if !showEditorEmptyWelcome(domain.Workspace{}, nil, nil) {
		t.Fatal("expected empty welcome when there is no workspace and no recents")
	}
	if showEditorEmptyWelcome(domain.Workspace{Root: "C:/repo"}, nil, nil) {
		t.Fatal("did not expect empty welcome with an active workspace")
	}
	if showEditorEmptyWelcome(domain.Workspace{}, []recentWorkspacesSvc.Workspace{{Name: "repo", Path: "C:/repo"}}, nil) {
		t.Fatal("did not expect first-launch empty welcome when recents exist")
	}
	if showEditorEmptyWelcome(domain.Workspace{}, nil, readinessTestError("recent store unavailable")) {
		t.Fatal("did not expect first-launch empty welcome when recents cannot be loaded")
	}
}

func TestRecentWorkspaceLabelsMarkMissingPaths(t *testing.T) {
	item := recentWorkspacesSvc.Workspace{Name: "missing-repo", Path: "C:/missing-repo", Missing: true}
	if got := recentWorkspaceButtonLabel(item); got != "missing-repo (missing)" {
		t.Fatalf("unexpected missing button label: %q", got)
	}
	if got := recentWorkspacePathLabel(item); got != "Missing: C:/missing-repo" {
		t.Fatalf("unexpected missing path label: %q", got)
	}

	present := recentWorkspacesSvc.Workspace{Name: "repo", Path: "C:/repo", Exists: true}
	if got := recentWorkspaceButtonLabel(present); got != "repo" {
		t.Fatalf("unexpected present button label: %q", got)
	}
	if got := recentWorkspacePathLabel(present); got != "C:/repo" {
		t.Fatalf("unexpected present path label: %q", got)
	}
}

func TestWelcomeEmptyCommandsStayEditorLike(t *testing.T) {
	commands := welcomeEmptyCommands()
	if len(commands) != 6 {
		t.Fatalf("unexpected command count: %d", len(commands))
	}
	joined := strings.ToLower(formatWelcomeEmptyCommands(commands))
	for _, forbidden := range []string{"dashboard", "cockpit", "setup card"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("empty welcome commands should stay editor-like, got %q", joined)
		}
	}
	for _, expected := range []string{"Project View", "Go to File", "Drop files here"} {
		if !strings.Contains(formatWelcomeEmptyCommands(commands), expected) {
			t.Fatalf("empty welcome commands missing %q: %#v", expected, commands)
		}
	}
}

func formatWelcomeEmptyCommands(commands []welcomeEmptyCommand) string {
	parts := make([]string, 0, len(commands))
	for _, command := range commands {
		parts = append(parts, command.Label+" "+command.Shortcut)
	}
	return strings.Join(parts, "\n")
}

type readinessTestError string

func (e readinessTestError) Error() string {
	return string(e)
}
