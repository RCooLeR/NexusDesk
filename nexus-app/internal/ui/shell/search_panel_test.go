package shell

import (
	"testing"

	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestSearchControllerInitialState(t *testing.T) {
	_ = fynetest.NewTempApp(t)

	view := &View{}
	controller := newSearchController(view)

	if controller.status.Text != "No search yet." {
		t.Fatalf("expected initial status, got %q", controller.status.Text)
	}
	if len(controller.results.Objects) != 1 {
		t.Fatalf("expected one placeholder result, got %d", len(controller.results.Objects))
	}
}

func TestSearchControllerRequiresWorkspace(t *testing.T) {
	_ = fynetest.NewTempApp(t)

	view := &View{
		state:         NewState(),
		activityLog:   widget.NewRichTextFromMarkdown("Ready."),
		activityText:  "Ready.",
		activityLines: []string{"Ready."},
	}
	controller := newSearchController(view)
	view.search = controller

	controller.Search("needle")

	if controller.status.Text != "Open a workspace before searching." {
		t.Fatalf("expected missing workspace status, got %q", controller.status.Text)
	}
	if got := view.activityLines[len(view.activityLines)-1]; got != "Open a workspace before searching." {
		t.Fatalf("expected missing workspace activity, got %q", got)
	}
}
