package shell

import "testing"

func TestCopyDataCellMenuItemDoesNotInstallGlobalShortcut(t *testing.T) {
	item := copyDataCellMenuItem(func() {})
	if item.Label != "Copy Data Cell" {
		t.Fatalf("unexpected copy menu label: %q", item.Label)
	}
	if item.Shortcut != nil {
		t.Fatalf("copy menu should not reserve Ctrl+C globally: %#v", item.Shortcut)
	}
}
