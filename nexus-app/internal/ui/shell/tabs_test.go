package shell

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	readinessSvc "nexusdesk/internal/services/readiness"
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
	for _, line := range strings.Split(text, "\n") {
		if len(line) > 260 {
			t.Fatalf("home readiness line is too wide for resizable layout (%d chars): %s", len(line), line)
		}
	}
}

type readinessTestError string

func (e readinessTestError) Error() string {
	return string(e)
}
