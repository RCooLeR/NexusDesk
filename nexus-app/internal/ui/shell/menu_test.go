package shell

import (
	"testing"

	"fyne.io/fyne/v2"
)

func TestCopyDataCellMenuItemDoesNotInstallGlobalShortcut(t *testing.T) {
	item := copyDataCellMenuItem(func() {})
	if item.Label != "Copy Data Cell" {
		t.Fatalf("unexpected copy menu label: %q", item.Label)
	}
	if item.Shortcut != nil {
		t.Fatalf("copy menu should not reserve Ctrl+C globally: %#v", item.Shortcut)
	}
}

func TestMainMenuIncludesIDEChromeGroups(t *testing.T) {
	view := &View{state: NewState()}
	menu := view.mainMenu()
	titles := make([]string, 0, len(menu.Items))
	for _, item := range menu.Items {
		titles = append(titles, item.Label)
	}
	for _, expected := range []string{"File", "Edit", "View", "Navigate", "Code", "Refactor", "Run", "Tools", "Help"} {
		if !containsString(titles, expected) {
			t.Fatalf("expected main menu to include %q, got %#v", expected, titles)
		}
	}
}

func TestMainMenuRunGroupExposesTaskSurfaces(t *testing.T) {
	view := &View{state: NewState()}
	menu := view.mainMenu()
	run := menuByLabel(menu, "Run")
	if run == nil {
		t.Fatal("expected Run menu")
	}
	labels := make([]string, 0, len(run.Items))
	for _, item := range run.Items {
		labels = append(labels, item.Label)
	}
	for _, expected := range []string{"Discover Tasks", "Tasks", "Jobs"} {
		if !containsString(labels, expected) {
			t.Fatalf("expected Run menu to include %q, got %#v", expected, labels)
		}
	}
}

func menuByLabel(menu *fyne.MainMenu, label string) *fyne.Menu {
	for _, item := range menu.Items {
		if item.Label == label {
			return item
		}
	}
	return nil
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
