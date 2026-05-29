package shell

import (
	editorSvc "nexusdesk/internal/services/editor"
)

func (v *View) formatActiveEditorDraft() {
	tabID, editor, ok := v.activeTextEditor()
	if !ok {
		v.addActivity("Select a text editor tab before formatting.")
		return
	}
	tab, ok := v.editorSession.Tab(tabID)
	if !ok {
		v.addActivity("Editor tab is no longer available.")
		return
	}
	result, err := editorSvc.FormatDocument(tab.RelPath, editor.source.Text)
	if err != nil {
		editor.status.SetText(err.Error())
		return
	}
	if result.Changed {
		editor.source.SetText(result.Content)
	}
	editor.status.SetText(result.Message)
}

func (v *View) openActiveEditorSymbols() {
	tabID, _, ok := v.activeTextEditor()
	if !ok {
		v.addActivity("Select a text editor tab before opening symbols.")
		return
	}
	v.openEditorSymbolDialog(tabID)
}

func (v *View) openActiveEditorReferences() {
	tabID, _, ok := v.activeTextEditor()
	if !ok {
		v.addActivity("Select a text editor tab before finding references.")
		return
	}
	v.openEditorReferencesDialog(tabID)
}
