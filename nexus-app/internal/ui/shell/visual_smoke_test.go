package shell

import (
	"os"
	"path/filepath"
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

func TestVisualSmokeWorkspaceAndEditorStates(t *testing.T) {
	window, view := newVisualSmokeWindow(t)
	window.Resize(fyne.NewSize(1280, 820))
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Smoke\n\nhello visual smoke\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	workspace, err := view.workspaceService.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	view.state.SetWorkspace(workspace)
	view.refreshNavigator()
	view.refreshStatusBar()
	_ = fynetest.RenderToMarkup(window.Canvas())
	if view.navigatorTree == nil {
		t.Fatal("expected workspace visual smoke to initialize the navigator tree")
	}
	if !strings.Contains(view.status.Text, "Workspace: ") || !strings.Contains(view.status.Text, filepath.Base(root)) {
		t.Fatalf("expected workspace status for %q, got %q", filepath.Base(root), view.status.Text)
	}

	view.openWorkspaceRelFile("README.md")
	_ = fynetest.RenderToMarkup(window.Canvas())
	active, ok := view.editorSession.Tab(view.editorSession.ActiveID())
	if !ok || active.RelPath != "README.md" || !strings.Contains(active.DraftText, "hello visual smoke") {
		t.Fatalf("expected README editor smoke tab, got %#v ok=%v", active, ok)
	}
}

func TestVisualSmokeCoreToolStates(t *testing.T) {
	window, view := newVisualSmokeWindow(t)
	window.Resize(fyne.NewSize(1280, 820))

	if view.assistant != nil && view.assistant.runStatus != nil {
		view.assistant.runStatus.SetText("Assistant streaming smoke: route Main coding model.")
		view.assistant.runStatus.Refresh()
	}
	assistantMarkup := fynetest.RenderToMarkup(window.Canvas())
	if !strings.Contains(assistantMarkup, "Assistant streaming smoke") && (view.assistant == nil || view.assistant.runStatus == nil || !strings.Contains(view.assistant.runStatus.Text, "Assistant streaming smoke")) {
		t.Fatal("expected assistant streaming smoke status to render or remain in header state")
	}

	openVisualSmokeTool(t, view, window, "data", "Data")
	openVisualSmokeTool(t, view, window, "artifacts", "Artifacts")
	openVisualSmokeTool(t, view, window, "diagnostics", "Diagnostics")
	openVisualSmokeTool(t, view, window, "approvals", "Approvals")

	view.openSettingsTab()
	settingsMarkup := fynetest.RenderToMarkup(window.Canvas())
	assertVisualSmokeContains(t, settingsMarkup, "Settings")
	assertVisualSmokeContains(t, settingsMarkup, "Diagnostics & Tests")
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

func openVisualSmokeTool(t *testing.T, view *View, window fyne.Window, id string, expected string) {
	t.Helper()
	tool, ok := defaultToolWindowRegistry().Lookup(id)
	if !ok {
		t.Fatalf("expected tool %q to be registered", id)
	}
	view.openToolWindow(tool)
	_ = fynetest.RenderToMarkup(window.Canvas())
	if !view.isBottomTabSelected(expected) {
		t.Fatalf("expected visual smoke tool %q to select %q", id, expected)
	}
}
