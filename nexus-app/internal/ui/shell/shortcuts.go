package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

func (v *View) installShortcuts() {
	canvas := v.window.Canvas()
	bindShortcut(canvas, shortcutOpenWorkspace(), v.openWorkspaceDialog)
	bindShortcut(canvas, shortcutRefreshWorkspace(), v.refreshWorkspace)
	bindShortcut(canvas, shortcutCloseTab(), v.closeSelectedTab)
	bindShortcut(canvas, shortcutNextTab(), v.selectNextTab)
	bindShortcut(canvas, shortcutPreviousTab(), v.selectPreviousTab)
	bindShortcut(canvas, shortcutSettings(), func() {
		v.addPlaceholderTab("Settings", "Provider, access policy, model, and connector settings will live here.")
	})
}

func bindShortcut(canvas fyne.Canvas, shortcut fyne.Shortcut, action func()) {
	canvas.AddShortcut(shortcut, func(fyne.Shortcut) {
		action()
	})
}

func shortcutOpenWorkspace() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: fyne.KeyModifierShortcutDefault}
}

func shortcutRefreshWorkspace() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: fyne.KeyModifierShortcutDefault}
}

func shortcutCloseTab() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: fyne.KeyModifierShortcutDefault}
}

func shortcutNextTab() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyTab, Modifier: fyne.KeyModifierControl}
}

func shortcutPreviousTab() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyTab, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}
}

func shortcutSettings() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyComma, Modifier: fyne.KeyModifierShortcutDefault}
}
