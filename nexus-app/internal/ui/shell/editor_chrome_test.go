package shell

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	editorSvc "nexusdesk/internal/services/editor"
)

func TestSecondaryEditorOptionsExcludeActiveFile(t *testing.T) {
	session := editorSvc.NewSession()
	session.OpenFile("a.go", "a.go")
	session.OpenFile("b.go", "b.go")
	session.OpenFile("docs/c.md", "c.md")
	view := &View{editorSession: session}

	options := view.secondaryEditorOptions("a.go")
	if len(options) != 2 || options[0] != "b.go" || options[1] != "docs/c.md" {
		t.Fatalf("unexpected secondary options: %#v", options)
	}
}

func TestDocumentMapItemText(t *testing.T) {
	text := documentMapItemText(editorSvc.DocumentMapItem{Kind: "todo", Label: "TODO: wire startup", Line: 12})

	if text != "todo  TODO: wire startup  L12" {
		t.Fatalf("unexpected document map text: %q", text)
	}
}

func TestEditorSaveAllowedBlocksTruncatedPreview(t *testing.T) {
	tab := editorSvc.Tab{Dirty: true}
	preview := domain.FilePreview{Kind: domain.PreviewText, Truncated: true}

	if editorSaveAllowed(tab, preview, false, true) {
		t.Fatal("expected save to be blocked for truncated preview")
	}
	if editorSaveAllowed(editorSvc.Tab{}, preview, true, true) {
		t.Fatal("expected encoding-only save to be blocked for truncated preview")
	}
	if !editorSaveAllowed(tab, domain.FilePreview{Kind: domain.PreviewText}, false, true) {
		t.Fatal("expected dirty full preview to be saveable")
	}
}

func TestEditorSaveAllowedBlocksAmbiguousEncodingUntilExplicit(t *testing.T) {
	tab := editorSvc.Tab{Dirty: true}
	preview := domain.FilePreview{Kind: domain.PreviewText, EncodingAmbiguous: true}

	if editorSaveAllowed(tab, preview, false, false) {
		t.Fatal("expected ambiguous encoding save to be blocked until explicit")
	}
	if !editorSaveAllowed(tab, preview, false, true) {
		t.Fatal("expected explicit encoding to allow dirty save")
	}
}

func TestRefreshEditorAfterSaveUpdatesBindingInPlace(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	session := editorSvc.NewSession()
	tab := session.OpenFileWithSource("notes.txt", "notes.txt", "old")
	if !session.UpdateDraft(tab.ID, "new") {
		t.Fatal("expected draft update")
	}
	dirty, ok := session.Tab(tab.ID)
	if !ok {
		t.Fatal("expected dirty tab")
	}
	view := &View{editorSession: session}
	view.editor = &editorController{
		openTabs:    map[string]*container.TabItem{},
		tabIDs:      map[*container.TabItem]string{},
		previews:    map[string]domain.FilePreview{},
		textEditors: map[string]*textEditorBinding{},
	}
	preview := domain.FilePreview{RelPath: "notes.txt", Kind: domain.PreviewText, Encoding: "utf-8", Text: "new"}
	content := view.newTextEditor(dirty, preview, func(editorSvc.Tab, bool, bool) {})
	item := container.NewTabItem("notes.txt", content)
	view.editor.openTabs[tab.ID] = item
	view.editor.tabIDs[item] = tab.ID
	editor, ok := view.textEditor(tab.ID)
	if !ok {
		t.Fatal("expected text editor binding")
	}
	editor.source.CursorRow = 0
	editor.source.CursorColumn = 2
	saved, ok := session.MarkDraftSaved(tab.ID)
	if !ok {
		t.Fatal("expected saved tab")
	}

	view.refreshEditorAfterSave(saved, preview)

	if item.Content != content {
		t.Fatal("expected editor content to be updated in place after save")
	}
	if editor.source.Text != "new" || editor.status.Text != "Draft matches source." {
		t.Fatalf("expected binding state to be refreshed, source=%q status=%q", editor.source.Text, editor.status.Text)
	}
	if editor.source.CursorRow != 0 || editor.source.CursorColumn != 2 {
		t.Fatalf("expected cursor to be preserved, got row=%d column=%d", editor.source.CursorRow, editor.source.CursorColumn)
	}
}

func TestSetEditorSaveStateShowsSavingAndRetryFailure(t *testing.T) {
	_ = fynetest.NewTempApp(t)
	session := editorSvc.NewSession()
	tab := session.OpenFileWithSource("notes.txt", "notes.txt", "old")
	if !session.UpdateDraft(tab.ID, "new") {
		t.Fatal("expected dirty draft")
	}
	dirty, ok := session.Tab(tab.ID)
	if !ok {
		t.Fatal("expected dirty tab")
	}
	status := widget.NewLabel("")
	state := widget.NewLabel("")
	save := widget.NewButton("", nil)
	view := &View{editorSession: session}
	view.editor = &editorController{
		openTabs:    map[string]*container.TabItem{tab.ID: container.NewTabItem("notes.txt", widget.NewLabel(""))},
		tabIDs:      map[*container.TabItem]string{},
		previews:    map[string]domain.FilePreview{tab.ID: {RelPath: "notes.txt", Kind: domain.PreviewText, Encoding: "utf-8", Text: "old"}},
		textEditors: map[string]*textEditorBinding{},
		savingTabs:  map[string]bool{},
	}
	view.editor.textEditors[tab.ID] = &textEditorBinding{
		status:           status,
		tabState:         state,
		saveButton:       save,
		sourceEncoding:   "utf-8",
		saveEncoding:     "utf-8",
		encodingExplicit: true,
	}

	view.setEditorSaveState(tab.ID, true, "Saving draft...")
	if !view.editorSaving(tab.ID) || status.Text != "Saving draft..." || state.Text != "Saving..." {
		t.Fatalf("expected visible saving state, saving=%v status=%q state=%q", view.editorSaving(tab.ID), status.Text, state.Text)
	}

	view.setEditorSaveState(tab.ID, false, "Save failed: disk is full. Retry Save after fixing the problem.")
	if view.editorSaving(tab.ID) || !strings.Contains(status.Text, "Retry Save") || state.Text != editorStateText(dirty) {
		t.Fatalf("expected retry failure state, saving=%v status=%q state=%q", view.editorSaving(tab.ID), status.Text, state.Text)
	}
}

func TestDraftDiagnosticFormatting(t *testing.T) {
	diagnostics := []editorSvc.DraftDiagnostic{
		{RelPath: "config/app.json", Severity: "error", Source: "json", Line: 2, Message: "invalid character"},
		{RelPath: "config/app.json", Severity: "info", Source: "marker", Line: 3, Message: "TODO"},
	}

	status := draftDiagnosticsStatusText(diagnostics)
	if !containsAll(status, []string{"Diagnostics:", "2 draft problem", "error=1", "info=1"}) {
		t.Fatalf("unexpected diagnostics status: %q", status)
	}
	item := draftDiagnosticItemText(diagnostics[0])
	if !containsAll(item, []string{"config/app.json", "error/json", "L2", "invalid character"}) {
		t.Fatalf("unexpected diagnostic item: %q", item)
	}
	if empty := draftDiagnosticsStatusText(nil); empty != "Diagnostics: no draft problems detected." {
		t.Fatalf("unexpected empty diagnostics status: %q", empty)
	}
}

func TestDefinitionStatusText(t *testing.T) {
	resolved := definitionStatusText(editorSvc.DefinitionResult{
		Query: "Start",
		Item:  editorSvc.OutlineItem{Kind: "func", Label: "Start", Line: 7},
	}, true)
	if resolved != "Moved to func Start on line 7." {
		t.Fatalf("unexpected resolved status: %q", resolved)
	}

	missing := definitionStatusText(editorSvc.DefinitionResult{Query: "Missing"}, false)
	if missing != "No local definition found for Missing." {
		t.Fatalf("unexpected missing status: %q", missing)
	}

	empty := definitionStatusText(editorSvc.DefinitionResult{}, false)
	if empty != "Place the cursor on a symbol name before using Definition." {
		t.Fatalf("unexpected empty status: %q", empty)
	}
}

func TestSyntaxStatusAndAnalysisText(t *testing.T) {
	analysis := editorSvc.AnalyzeSyntax("main.go", "package main\n// hello\nfunc main() { println(\"hi\", 42) }\n")

	status := syntaxStatusText(analysis)
	if status == "" || !containsAll(status, []string{"Syntax: Go", "native lightweight tokenizer", "LSP candidate"}) {
		t.Fatalf("unexpected syntax status: %q", status)
	}
	detail := formatSyntaxAnalysis(analysis)
	if !containsAll(detail, []string{"Language: Go", "Token counts:", "keyword", "comment", "string", "number", "Tokens"}) {
		t.Fatalf("unexpected syntax detail:\n%s", detail)
	}
}

func TestSyntaxStatusTextWithCursorIncludesActiveToken(t *testing.T) {
	content := "package main\nfunc main() { println(\"hi\") }\n"
	analysis := editorSvc.AnalyzeSyntax("main.go", content)
	context := editorSvc.SyntaxContextFromAnalysis(content, analysis, 1, 23)

	status := syntaxStatusTextWithCursor(analysis, context)

	if !containsAll(status, []string{"Syntax: Go", "Cursor: L2:C24", "string token", "symbol hi"}) {
		t.Fatalf("unexpected cursor-aware syntax status: %q", status)
	}
}

func containsAll(value string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
