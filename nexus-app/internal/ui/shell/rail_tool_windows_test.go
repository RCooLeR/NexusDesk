package shell

import (
	"reflect"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestLeftRailToolWindowsCoverProductionTargets(t *testing.T) {
	tools := leftRailToolWindows()
	gotLabels := make([]string, 0, len(tools))
	gotTargets := map[string]bool{}
	for _, tool := range tools {
		gotLabels = append(gotLabels, tool.Label)
		if tool.ShortcutKey == "" {
			t.Fatalf("expected shortcut key for %s", tool.Label)
		}
		if tool.TargetTab != "" {
			gotTargets[tool.TargetTab] = true
		}
	}

	wantLabels := []string{
		"Project",
		"Search",
		"Problems",
		"Git",
		"Tasks",
		"Jobs",
		"Data",
		"Artifacts",
		"Operations",
		"Diagnostics",
	}
	if !reflect.DeepEqual(gotLabels, wantLabels) {
		t.Fatalf("unexpected rail tools: got %#v want %#v", gotLabels, wantLabels)
	}
	for _, target := range []string{"Search", "Problems", "Git", "Tasks", "Jobs", "Data", "Artifacts", "Operations", "Diagnostics"} {
		if !gotTargets[target] {
			t.Fatalf("expected rail target %q", target)
		}
	}
}

func TestOpenLeftRailToolWindowSelectsNestedPanel(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	workbenchTabs := container.NewAppTabs(
		container.NewTabItem("Activity", widget.NewLabel("activity")),
		container.NewTabItem("Search", widget.NewLabel("search")),
		container.NewTabItem("Problems", widget.NewLabel("problems")),
	)
	dataTabs := container.NewAppTabs(
		container.NewTabItem("Data", widget.NewLabel("data")),
		container.NewTabItem("Operations", widget.NewLabel("operations")),
	)
	view := &View{
		bottomTabs: container.NewAppTabs(
			container.NewTabItem("Workbench", workbenchTabs),
			container.NewTabItem("Data Studio", dataTabs),
		),
		activityLog:   widget.NewRichTextFromMarkdown("Ready."),
		activityText:  "Ready.",
		activityLines: []string{"Ready."},
	}

	view.openLeftRailToolWindow(leftRailToolWindow{Label: "Operations", TargetTab: "Operations", Activity: "Operations selected."})

	if got := view.bottomTabs.Selected().Text; got != "Data Studio" {
		t.Fatalf("expected Data Studio group, got %q", got)
	}
	if got := dataTabs.Selected().Text; got != "Operations" {
		t.Fatalf("expected Operations tab, got %q", got)
	}
	if !containsActivityLine(view.recentActivityLines(3), "Operations selected.") {
		t.Fatalf("expected rail activity, got %#v", view.recentActivityLines(3))
	}
}

func TestLeftRailButtonLabelIncludesShortcut(t *testing.T) {
	if got := (leftRailToolWindow{Label: "Search", Shortcut: "Alt+2"}).ButtonLabel(); got != "Alt+2  Search" {
		t.Fatalf("unexpected button label %q", got)
	}
	if got := (leftRailToolWindow{Label: "Search"}).ButtonLabel(); got != "Search" {
		t.Fatalf("unexpected button label without shortcut %q", got)
	}
}

func TestLeftRailShortcutUsesAltModifier(t *testing.T) {
	shortcut, ok := shortcutLeftRailTool(leftRailToolWindow{ShortcutKey: fyne.Key2}).(*desktop.CustomShortcut)
	if !ok {
		t.Fatalf("expected desktop shortcut, got %#v", shortcut)
	}
	if shortcut.KeyName != fyne.Key2 || shortcut.Modifier != fyne.KeyModifierAlt {
		t.Fatalf("unexpected shortcut: %#v", shortcut)
	}
	if shortcutLeftRailTool(leftRailToolWindow{}) != nil {
		t.Fatal("expected missing shortcut key to return nil")
	}
}
