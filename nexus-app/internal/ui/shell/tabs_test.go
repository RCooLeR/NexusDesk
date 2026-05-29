package shell

import (
	"testing"

	"fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
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
