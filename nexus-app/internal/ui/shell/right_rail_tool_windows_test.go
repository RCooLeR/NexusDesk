package shell

import (
	"reflect"
	"testing"

	"fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestRightRailToolWindowsCoverAssistantTargets(t *testing.T) {
	tools := rightRailToolWindows()
	gotLabels := make([]string, 0, len(tools))
	gotTargets := map[string]bool{}
	for _, tool := range tools {
		gotLabels = append(gotLabels, tool.Label)
		if tool.TargetTab != "" {
			gotTargets[tool.TargetTab] = true
		}
	}

	wantLabels := []string{"Assistant", "Sources", "Lineage", "Monitor", "Inspector"}
	if !reflect.DeepEqual(gotLabels, wantLabels) {
		t.Fatalf("unexpected right rail tools: got %#v want %#v", gotLabels, wantLabels)
	}
	for _, target := range []string{"Artifacts", "Jobs", "Diagnostics"} {
		if !gotTargets[target] {
			t.Fatalf("expected right rail target %q", target)
		}
	}
	if !tools[0].FocusAssistant {
		t.Fatal("expected Assistant rail item to focus assistant")
	}
}

func TestOpenRightRailToolWindowSelectsNestedPanel(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	dataTabs := container.NewAppTabs(
		container.NewTabItem("Data", widget.NewLabel("data")),
		container.NewTabItem("Artifacts", widget.NewLabel("artifacts")),
	)
	systemTabs := container.NewAppTabs(
		container.NewTabItem("Jobs", widget.NewLabel("jobs")),
		container.NewTabItem("Diagnostics", widget.NewLabel("diagnostics")),
	)
	view := &View{
		bottomTabs: container.NewAppTabs(
			container.NewTabItem("Data Studio", dataTabs),
			container.NewTabItem("System", systemTabs),
		),
		activityLog:   widget.NewRichTextFromMarkdown("Ready."),
		activityText:  "Ready.",
		activityLines: []string{"Ready."},
	}

	view.openRightRailToolWindow(rightRailToolWindow{Label: "Inspector", TargetTab: "Diagnostics", Activity: "Inspector diagnostics selected."})

	if got := view.bottomTabs.Selected().Text; got != "System" {
		t.Fatalf("expected System group, got %q", got)
	}
	if got := systemTabs.Selected().Text; got != "Diagnostics" {
		t.Fatalf("expected Diagnostics tab, got %q", got)
	}
	if !containsActivityLine(view.recentActivityLines(3), "Inspector diagnostics selected.") {
		t.Fatalf("expected rail activity, got %#v", view.recentActivityLines(3))
	}
}

func TestRightRailAssistantFocusRecordsActivity(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	view := &View{
		assistantPrompt: widget.NewMultiLineEntry(),
		activityLog:     widget.NewRichTextFromMarkdown("Ready."),
		activityText:    "Ready.",
		activityLines:   []string{"Ready."},
	}

	view.openRightRailToolWindow(rightRailToolWindow{Label: "Assistant", Activity: "Assistant selected.", FocusAssistant: true})

	if !containsActivityLine(view.recentActivityLines(3), "Assistant selected.") {
		t.Fatalf("expected assistant activity, got %#v", view.recentActivityLines(3))
	}
}

func TestRightRailButtonLabelIncludesShortcut(t *testing.T) {
	if got := (rightRailToolWindow{Label: "Assistant", Shortcut: "A"}).ButtonLabel(); got != "A  Assistant" {
		t.Fatalf("unexpected button label %q", got)
	}
	if got := (rightRailToolWindow{Label: "Assistant"}).ButtonLabel(); got != "Assistant" {
		t.Fatalf("unexpected button label without shortcut %q", got)
	}
}
