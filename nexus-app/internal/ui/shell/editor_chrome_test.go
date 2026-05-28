package shell

import (
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
