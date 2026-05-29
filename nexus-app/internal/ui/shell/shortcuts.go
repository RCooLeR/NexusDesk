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
	bindShortcut(canvas, shortcutSaveDraft(), v.saveActiveEditorDraft)
	bindShortcut(canvas, shortcutRevertDraft(), v.revertActiveEditorDraft)
	bindShortcut(canvas, shortcutFindReplace(), v.openFindReplaceDialog)
	bindShortcut(canvas, shortcutDataGridUp(), func() { v.navigateDataGridSelection(-1, 0) })
	bindShortcut(canvas, shortcutDataGridDown(), func() { v.navigateDataGridSelection(1, 0) })
	bindShortcut(canvas, shortcutDataGridLeft(), func() { v.navigateDataGridSelection(0, -1) })
	bindShortcut(canvas, shortcutDataGridRight(), func() { v.navigateDataGridSelection(0, 1) })
	bindShortcut(canvas, shortcutDataGridPageUp(), func() { v.navigateDataGridPage(-dataGridPageStep) })
	bindShortcut(canvas, shortcutDataGridPageDown(), func() { v.navigateDataGridPage(dataGridPageStep) })
	bindShortcut(canvas, shortcutDataGridTop(), v.navigateDataGridTop)
	bindShortcut(canvas, shortcutDataGridBottom(), v.navigateDataGridBottom)
	bindShortcut(canvas, shortcutDataGridRowStart(), v.navigateDataGridRowStart)
	bindShortcut(canvas, shortcutDataGridRowEnd(), v.navigateDataGridRowEnd)
	bindShortcut(canvas, shortcutNextTab(), v.selectNextTab)
	bindShortcut(canvas, shortcutPreviousTab(), v.selectPreviousTab)
	bindShortcut(canvas, shortcutSettings(), v.openSettingsTab)
	bindShortcut(canvas, shortcutQuickOpen(), v.openQuickOpenDialog)
	bindShortcut(canvas, shortcutCommandPalette(), v.openCommandPaletteDialog)
	for _, tool := range leftRailToolWindows() {
		tool := tool
		if shortcut := shortcutLeftRailTool(tool); shortcut != nil {
			bindShortcut(canvas, shortcut, func() { v.openLeftRailToolWindow(tool) })
		}
	}
	for _, tool := range rightRailToolWindows() {
		tool := tool
		if shortcut := shortcutRightRailTool(tool); shortcut != nil {
			bindShortcut(canvas, shortcut, func() { v.openRightRailToolWindow(tool) })
		}
	}
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

func shortcutSaveDraft() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierShortcutDefault}
}

func shortcutRevertDraft() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift}
}

func shortcutFindReplace() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierShortcutDefault}
}

func shortcutDataGridUp() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyUp, Modifier: fyne.KeyModifierAlt}
}

func shortcutDataGridDown() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyDown, Modifier: fyne.KeyModifierAlt}
}

func shortcutDataGridLeft() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyLeft, Modifier: fyne.KeyModifierAlt}
}

func shortcutDataGridRight() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyRight, Modifier: fyne.KeyModifierAlt}
}

func shortcutDataGridPageUp() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyPageUp, Modifier: fyne.KeyModifierAlt}
}

func shortcutDataGridPageDown() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyPageDown, Modifier: fyne.KeyModifierAlt}
}

func shortcutDataGridTop() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyHome, Modifier: fyne.KeyModifierAlt}
}

func shortcutDataGridBottom() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyEnd, Modifier: fyne.KeyModifierAlt}
}

func shortcutDataGridRowStart() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyHome, Modifier: fyne.KeyModifierAlt | fyne.KeyModifierShift}
}

func shortcutDataGridRowEnd() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyEnd, Modifier: fyne.KeyModifierAlt | fyne.KeyModifierShift}
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

func shortcutQuickOpen() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyP, Modifier: fyne.KeyModifierShortcutDefault}
}

func shortcutCommandPalette() fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: fyne.KeyP, Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift}
}

func shortcutLeftRailTool(tool leftRailToolWindow) fyne.Shortcut {
	return shortcutRailTool(tool.ShortcutKey)
}

func shortcutRightRailTool(tool rightRailToolWindow) fyne.Shortcut {
	return shortcutRailTool(tool.ShortcutKey)
}

func shortcutRailTool(key fyne.KeyName) fyne.Shortcut {
	if key == "" {
		return nil
	}
	return &desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierAlt}
}
