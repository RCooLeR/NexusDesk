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
