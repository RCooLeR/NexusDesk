package shell

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
)

func TestVisualSmokeSupportedWindowSizes(t *testing.T) {
	for _, spec := range []struct {
		name string
		size fyne.Size
	}{
		{name: "default", size: fyne.NewSize(1280, 820)},
		{name: "minimum", size: fyne.NewSize(1024, 640)},
		{name: "desktop", size: fyne.NewSize(1600, 900)},
	} {
		t.Run(spec.name, func(t *testing.T) {
			window, view := newVisualSmokeWindow(t)

			window.Resize(spec.size)
			markup := fynetest.RenderToMarkup(window.Canvas())

			assertVisualSmokeContains(t, markup, "Open Workspace")
			assertVisualSmokeContains(t, markup, "Assistant")
			if view.status == nil || !strings.Contains(view.status.Text, "Workspace: none") {
				t.Fatalf("expected status bar to report no workspace, got %#v", view.status)
			}
			if view.workbenchSplit == nil || !view.workbenchSplit.Horizontal {
				t.Fatal("expected visual smoke shell to use horizontal tool/editor dock split")
			}
			if view.mainSplit == nil || !view.mainSplit.Horizontal {
				t.Fatal("expected visual smoke shell to use horizontal editor/assistant split")
			}
		})
	}
}

func TestVisualSmokeFirstLaunchNoWorkspace(t *testing.T) {
	window, view := newVisualSmokeWindow(t)
	window.Resize(fyne.NewSize(1280, 820))

	markup := fynetest.RenderToMarkup(window.Canvas())

	assertVisualSmokeContains(t, markup, "Open Workspace")
	assertVisualSmokeContains(t, markup, "Welcome")
	assertVisualSmokeContains(t, markup, "Model:")
	if view.status == nil || !strings.Contains(view.status.Text, "Workspace: none") {
		t.Fatalf("expected first launch status to report no workspace, got %#v", view.status)
	}
	if view.state.Workspace().Root != "" {
		t.Fatalf("expected first launch smoke to have no workspace, got %q", view.state.Workspace().Root)
	}
}

func newVisualSmokeWindow(t *testing.T) (fyne.Window, *View) {
	t.Helper()
	app := fynetest.NewTempApp(t)
	window := app.NewWindow("visual smoke")
	view := New(window)
	window.SetContent(view.Canvas())
	t.Cleanup(func() {
		window.Close()
	})
	return window, view
}

func assertVisualSmokeContains(t *testing.T, markup string, expected string) {
	t.Helper()
	if !strings.Contains(markup, expected) {
		t.Fatalf("expected rendered shell markup to contain %q", expected)
	}
}
