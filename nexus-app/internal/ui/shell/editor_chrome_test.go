package shell

import (
	"strings"
	"testing"

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
