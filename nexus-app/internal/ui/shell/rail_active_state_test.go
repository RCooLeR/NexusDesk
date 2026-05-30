package shell

import (
	"testing"

	"fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestApplyRailButtonImportanceMarksOnlyActiveButton(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	buttons := map[string]*widget.Button{
		"Search":   widget.NewButton("Search", nil),
		"Problems": widget.NewButton("Problems", nil),
	}

	applyRailButtonImportance(buttons, "Problems")

	if buttons["Problems"].Importance != widget.HighImportance {
		t.Fatalf("expected active button to be high importance, got %v", buttons["Problems"].Importance)
	}
	if buttons["Search"].Importance != widget.LowImportance {
		t.Fatalf("expected inactive button to be low importance, got %v", buttons["Search"].Importance)
	}
}

func TestUpdateRailActiveStateForTabMapsLeftAndRightRails(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	view := &View{
		leftRailButtons: map[string]*widget.Button{
			"Artifacts": widget.NewButton("Artifacts", nil),
			"Jobs":      widget.NewButton("Jobs", nil),
		},
		rightRailButtons: map[string]*widget.Button{
			"Sources":   widget.NewButton("Sources", nil),
			"Monitor":   widget.NewButton("Monitor", nil),
			"Inspector": widget.NewButton("Inspector", nil),
		},
	}

	view.updateRailActiveStateForTab("Artifacts")

	if view.activeLeftRailTool != "Artifacts" {
		t.Fatalf("expected left Artifacts active, got %q", view.activeLeftRailTool)
	}
	if view.activeRightRailTool != "Sources" {
		t.Fatalf("expected right Sources active, got %q", view.activeRightRailTool)
	}
	if view.leftRailButtons["Artifacts"].Importance != widget.HighImportance {
		t.Fatal("expected Artifacts rail button to be highlighted")
	}
	if view.rightRailButtons["Sources"].Importance != widget.HighImportance {
		t.Fatal("expected Sources rail button to be highlighted")
	}
}

func TestSelectBottomTabRefreshesRailActiveState(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	dataTabs := container.NewAppTabs(
		container.NewTabItem("Data", widget.NewLabel("data")),
		container.NewTabItem("Artifacts", widget.NewLabel("artifacts")),
	)
	view := &View{
		bottomTabs: container.NewAppTabs(container.NewTabItem("Data Studio", dataTabs)),
		leftRailButtons: map[string]*widget.Button{
			"Data":      widget.NewButton("Data", nil),
			"Artifacts": widget.NewButton("Artifacts", nil),
		},
		rightRailButtons: map[string]*widget.Button{
			"Sources": widget.NewButton("Sources", nil),
		},
	}

	if !view.selectBottomTab("Artifacts") {
		t.Fatal("expected Artifacts tab to be selectable")
	}

	if view.activeLeftRailTool != "Artifacts" || view.activeRightRailTool != "Sources" {
		t.Fatalf("unexpected rail active state: left=%q right=%q", view.activeLeftRailTool, view.activeRightRailTool)
	}
}

func TestActiveRailClickCollapsesBottomPanel(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	childTabs := container.NewAppTabs(
		container.NewTabItem("Search", widget.NewLabel("search")),
	)
	split := container.NewVSplit(widget.NewLabel("top"), widget.NewLabel("bottom"))
	view := &View{
		bottomTabs:           container.NewAppTabs(container.NewTabItem("Workbench", childTabs)),
		workbenchSplit:       split,
		activeLeftRailTool:   "Search",
		leftRailButtons:      map[string]*widget.Button{"Search": widget.NewButton("", nil)},
		rightRailButtons:     map[string]*widget.Button{},
		activityLog:          widget.NewRichTextFromMarkdown("Ready."),
		activityText:         "Ready.",
		activityLines:        []string{"Ready."},
		bottomPanelCollapsed: false,
	}
	if !view.selectBottomTab("Search") {
		t.Fatal("expected Search tab to be selected")
	}

	view.openLeftRailToolWindow(leftRailToolWindow{Label: "Search", TargetTab: "Search", Activity: "Search selected."})

	if !view.bottomPanelCollapsed {
		t.Fatal("expected active rail click to collapse bottom panel")
	}
	if got := view.activityLines[len(view.activityLines)-1]; got != "Search collapsed." {
		t.Fatalf("expected collapse activity, got %q", got)
	}
}
