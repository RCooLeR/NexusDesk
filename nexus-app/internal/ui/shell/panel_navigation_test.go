package shell

import (
	"testing"

	"fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestSelectBottomTabFindsNestedTabs(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	workbenchTabs := container.NewAppTabs(
		container.NewTabItem("Activity", widget.NewLabel("activity")),
		container.NewTabItem("Search", widget.NewLabel("search")),
	)
	dataTabs := container.NewAppTabs(
		container.NewTabItem("Data", widget.NewLabel("data")),
		container.NewTabItem("Artifacts", widget.NewLabel("artifacts")),
	)
	view := &View{bottomTabs: container.NewAppTabs(
		container.NewTabItem("Workbench", workbenchTabs),
		container.NewTabItem("Data Studio", dataTabs),
	)}

	if !view.selectBottomTab("Artifacts") {
		t.Fatal("expected nested Artifacts tab to be selectable")
	}
	if got := view.bottomTabs.Selected().Text; got != "Data Studio" {
		t.Fatalf("expected Data Studio group to be selected, got %q", got)
	}
	if got := dataTabs.Selected().Text; got != "Artifacts" {
		t.Fatalf("expected nested Artifacts tab to be selected, got %q", got)
	}
	if !view.isBottomTabSelected("Artifacts") {
		t.Fatal("expected nested Artifacts tab to report selected")
	}
	if view.isBottomTabSelected("Search") {
		t.Fatal("expected unselected nested Search tab to report inactive")
	}
}

func TestSelectBottomTabCanSelectTopLevelGroup(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	view := &View{bottomTabs: container.NewAppTabs(
		container.NewTabItem("Workbench", widget.NewLabel("workbench")),
		container.NewTabItem("System", widget.NewLabel("system")),
	)}

	if !view.selectBottomTab("System") {
		t.Fatal("expected top-level System tab to be selectable")
	}
	if !view.isBottomTabSelected("System") {
		t.Fatal("expected top-level System tab to report selected")
	}
	if view.selectBottomTab("Missing") {
		t.Fatal("expected missing bottom tab selection to fail")
	}
}

func TestEditorPrioritySplitKeepsEditorWide(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	view := &View{editor: &editorController{
		tabs: container.NewDocTabs(container.NewTabItem("README.md", widget.NewLabel("editor"))),
	}}

	split := view.newEditorPrioritySplit(widget.NewLabel("assistant"))

	if split.Offset != editorWidthPriorityOffset {
		t.Fatalf("expected editor priority offset %v, got %v", editorWidthPriorityOffset, split.Offset)
	}
	if view.mainSplit != split {
		t.Fatal("expected view to retain the main editor split")
	}
}

func TestToolPanelSplitUsesLeftDock(t *testing.T) {
	app := fynetest.NewTempApp(t)
	window := app.NewWindow("tool panel")
	defer window.Close()
	view := New(window)

	_ = view.Canvas()
	split := view.workbenchSplit

	if !split.Horizontal {
		t.Fatal("expected tool panel split to dock horizontally beside the editor")
	}
	if split.Offset != workbenchExpandedOffset {
		t.Fatalf("expected dock offset %v, got %v", workbenchExpandedOffset, split.Offset)
	}
	if view.workbenchSplit != split {
		t.Fatal("expected view to retain the tool panel split")
	}
}

func TestToolPanelOffsetIsRememberedPerTool(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	view := &View{
		workbenchSplit:        container.NewHSplit(widget.NewLabel("tools"), widget.NewLabel("editor")),
		activeToolPanelKey:    "Search",
		toolPanelOffsetByTool: map[string]float64{},
	}
	view.workbenchSplit.SetOffset(0.31)

	view.rememberCurrentToolPanelOffset()
	view.expandToolPanelFor("Git")
	if view.workbenchSplit.Offset != workbenchExpandedOffset {
		t.Fatalf("expected default Git width before adjustment, got %v", view.workbenchSplit.Offset)
	}

	view.workbenchSplit.SetOffset(0.19)
	view.rememberCurrentToolPanelOffset()
	view.expandToolPanelFor("Search")
	if view.workbenchSplit.Offset != 0.31 {
		t.Fatalf("expected remembered Search width, got %v", view.workbenchSplit.Offset)
	}

	view.expandToolPanelFor("Git")
	if view.workbenchSplit.Offset != 0.19 {
		t.Fatalf("expected remembered Git width, got %v", view.workbenchSplit.Offset)
	}
}

func TestSelectBottomTabReassertsEditorWidthPriority(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	view := &View{
		bottomTabs: container.NewAppTabs(container.NewTabItem("Diagnostics", widget.NewLabel("diagnostics"))),
		mainSplit:  container.NewHSplit(widget.NewLabel("editor"), widget.NewLabel("assistant")),
	}
	view.mainSplit.SetOffset(0.5)

	if !view.selectBottomTab("Diagnostics") {
		t.Fatal("expected Diagnostics tab to be selectable")
	}
	if view.mainSplit.Offset != editorWidthPriorityOffset {
		t.Fatalf("expected editor width priority to be restored, got %v", view.mainSplit.Offset)
	}
}
