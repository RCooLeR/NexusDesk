package shell

import (
	"testing"

	"fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
)

func TestApplyRailButtonImportanceMarksOnlyActiveButton(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	buttons := map[string]*railToolButton{
		"Search":   newRailIconButton(toolWindowRegistration{Label: "Search"}, nil, nil, nil),
		"Problems": newRailIconButton(toolWindowRegistration{Label: "Problems"}, nil, nil, nil),
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
		leftRailButtons: map[string]*railToolButton{
			"Artifacts": newRailIconButton(toolWindowRegistration{Label: "Artifacts"}, nil, nil, nil),
			"Jobs":      newRailIconButton(toolWindowRegistration{Label: "Jobs"}, nil, nil, nil),
		},
		rightRailButtons: map[string]*railToolButton{
			"Sources":   newRailIconButton(toolWindowRegistration{Label: "Sources"}, nil, nil, nil),
			"Monitor":   newRailIconButton(toolWindowRegistration{Label: "Monitor"}, nil, nil, nil),
			"Inspector": newRailIconButton(toolWindowRegistration{Label: "Inspector"}, nil, nil, nil),
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
		leftRailButtons: map[string]*railToolButton{
			"Data":      newRailIconButton(toolWindowRegistration{Label: "Data"}, nil, nil, nil),
			"Artifacts": newRailIconButton(toolWindowRegistration{Label: "Artifacts"}, nil, nil, nil),
		},
		rightRailButtons: map[string]*railToolButton{
			"Sources": newRailIconButton(toolWindowRegistration{Label: "Sources"}, nil, nil, nil),
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
		leftRailButtons:      map[string]*railToolButton{"Search": newRailIconButton(toolWindowRegistration{Label: "Search"}, nil, nil, nil)},
		rightRailButtons:     map[string]*railToolButton{},
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

func TestRailActiveStateRestoresPerWorkspace(t *testing.T) {
	view := &View{
		state:                NewState(),
		railStateByWorkspace: map[string]railWorkspaceState{},
		leftRailButtons:      map[string]*railToolButton{},
		rightRailButtons:     map[string]*railToolButton{},
	}
	view.state.SetWorkspace(domain.Workspace{Root: "C:/one"})
	view.setLeftRailActive("Git")
	view.setRightRailActive("Monitor")

	view.state.SetWorkspace(domain.Workspace{Root: "C:/two"})
	view.restoreActiveRailTools("C:/two")
	if view.activeLeftRailTool != defaultLeftRailTool || view.activeRightRailTool != defaultRightRailTool {
		t.Fatalf("expected defaults for unseen workspace, got left=%q right=%q", view.activeLeftRailTool, view.activeRightRailTool)
	}

	view.restoreActiveRailTools("C:/one")
	if view.activeLeftRailTool != "Git" || view.activeRightRailTool != "Monitor" {
		t.Fatalf("expected remembered workspace rail state, got left=%q right=%q", view.activeLeftRailTool, view.activeRightRailTool)
	}
}
