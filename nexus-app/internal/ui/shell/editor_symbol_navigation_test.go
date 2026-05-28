package shell

import "testing"

func TestEditorSymbolCandidatesUseOutlineLabels(t *testing.T) {
	items := editorSymbolCandidates("main.go", "package main\n\ntype Runner struct{}\n\nfunc Start() {}\n")

	if len(items) != 2 {
		t.Fatalf("expected two symbol candidates, got %#v", items)
	}
	if items[0].Label != "type  Runner  L3" || items[0].Line != 3 {
		t.Fatalf("unexpected first candidate: %#v", items[0])
	}
	if items[1].Label != "func  Start  L5" || items[1].Line != 5 {
		t.Fatalf("unexpected second candidate: %#v", items[1])
	}
}

func TestFilterEditorSymbolCandidatesMatchesTermsAndLimit(t *testing.T) {
	items := []editorSymbolCandidate{
		{Label: "func  Start  L5", SearchText: "func start line 5", Line: 5},
		{Label: "func  Stop  L9", SearchText: "func stop line 9", Line: 9},
		{Label: "type  Runner  L12", SearchText: "type runner line 12", Line: 12},
	}

	got := filterEditorSymbolCandidates(items, "func st", 1)

	if len(got) != 1 || got[0].Label != "func  Start  L5" {
		t.Fatalf("unexpected filtered symbols: %#v", got)
	}
}

func TestEditorSymbolStatus(t *testing.T) {
	if got := editorSymbolStatus(3, ""); got != "3 symbol(s). Select one or type to filter." {
		t.Fatalf("unexpected empty-query status: %q", got)
	}
	if got := editorSymbolStatus(1, "start"); got != "1 matching symbol(s). Press Enter to jump to the first match." {
		t.Fatalf("unexpected filtered status: %q", got)
	}
}
