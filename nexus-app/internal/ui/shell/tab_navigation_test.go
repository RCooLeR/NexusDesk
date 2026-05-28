package shell

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestDirtyTabCloseMessageUsesTitleFallback(t *testing.T) {
	if got := dirtyTabCloseMessage("README.md"); got != "Discard unsaved changes in README.md?" {
		t.Fatalf("unexpected dirty close message: %q", got)
	}
	if got := dirtyTabCloseMessage(""); got != "Discard unsaved changes in this tab?" {
		t.Fatalf("unexpected dirty close fallback: %q", got)
	}
}

func TestDirtyTabCloseCancelKeepsTabOpen(t *testing.T) {
	app := fynetest.NewTempApp(t)
	window := app.NewWindow("dirty-close-cancel")
	defer window.Close()
	view := New(window)
	tab := view.editorSession.OpenFileWithSource("README.md", "README.md", "# Hello\n")
	view.editorSession.UpdateDraft(tab.ID, "# Draft\n")
	item := container.NewTabItem("README.md", widget.NewLabel("README.md"))
	view.openTabs[tab.ID] = item
	view.tabIDs[item] = tab.ID
	view.editorTabs.Append(item)
	view.editorTabs.Select(item)

	view.handleDirtyTabCloseDecision(item, tab.ID, tab.Title, false)

	if _, ok := view.editorSession.Tab(tab.ID); !ok {
		t.Fatal("expected dirty tab to remain open after cancel")
	}
	if view.editorTabs.Selected() != item {
		t.Fatal("expected dirty tab to remain selected after cancel")
	}
	if !containsActivityLine(view.recentActivityLines(5), "Kept modified tab README.md open.") {
		t.Fatalf("expected cancel activity message, got %#v", view.recentActivityLines(5))
	}
}

func TestDirtyTabCloseConfirmDiscardsTab(t *testing.T) {
	app := fynetest.NewTempApp(t)
	window := app.NewWindow("dirty-close-confirm")
	defer window.Close()
	view := New(window)
	tab := view.editorSession.OpenFileWithSource("README.md", "README.md", "# Hello\n")
	view.editorSession.UpdateDraft(tab.ID, "# Draft\n")
	item := container.NewTabItem("README.md", widget.NewLabel("README.md"))
	view.openTabs[tab.ID] = item
	view.tabIDs[item] = tab.ID
	view.editorTabs.Append(item)
	view.editorTabs.Select(item)

	view.handleDirtyTabCloseDecision(item, tab.ID, tab.Title, true)

	if _, ok := view.editorSession.Tab(tab.ID); ok {
		t.Fatal("expected dirty tab to close after confirm")
	}
	if view.openTabs[tab.ID] != nil || view.tabIDs[item] != "" {
		t.Fatalf("expected tab maps to be cleaned: open=%#v id=%q", view.openTabs[tab.ID], view.tabIDs[item])
	}
	for _, line := range view.recentActivityLines(5) {
		if strings.Contains(line, "Close blocked") {
			t.Fatalf("unexpected blocked close activity: %#v", view.recentActivityLines(5))
		}
	}
}
