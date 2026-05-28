package shell

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

func TestQuickOpenShortcutUsesControlP(t *testing.T) {
	shortcut, ok := shortcutQuickOpen().(*desktop.CustomShortcut)
	if !ok {
		t.Fatalf("unexpected quick-open shortcut type: %#v", shortcutQuickOpen())
	}
	if shortcut.KeyName != fyne.KeyP || shortcut.Modifier != fyne.KeyModifierShortcutDefault {
		t.Fatalf("unexpected quick-open shortcut: %#v", shortcut)
	}
}

func TestQuickOpenStatusText(t *testing.T) {
	if text := quickOpenStatusText(0, "readme"); !strings.Contains(text, "No matches") {
		t.Fatalf("unexpected empty status: %q", text)
	}
	if text := quickOpenStatusText(3, ""); !strings.Contains(text, "Type to filter") {
		t.Fatalf("unexpected initial status: %q", text)
	}
	if text := quickOpenStatusText(2, "main"); !strings.Contains(text, "Enter opens") {
		t.Fatalf("unexpected match status: %q", text)
	}
}
