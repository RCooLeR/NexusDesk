package shell

import (
	"strings"
	"testing"

	workspaceSvc "nexusdesk/internal/services/workspace"
)

func TestEditorReferenceCandidatesFromSearchKeepsContentMatches(t *testing.T) {
	results := []workspaceSvc.SearchResult{
		{RelPath: "cmd/main.go", Kind: "file", MatchType: "path", Line: 0, Snippet: "cmd/main.go"},
		{RelPath: "cmd/main.go", Kind: "file", MatchType: "content", Line: 3, Snippet: "func main() { Start() }"},
		{RelPath: "internal/app/app.go", Kind: "file", MatchType: "content", Line: 5, Snippet: "func Start() {}"},
		{RelPath: "internal/app", Kind: "directory", MatchType: "path", Snippet: "internal/app"},
	}

	candidates := editorReferenceCandidatesFromSearch("Start", results)

	if len(candidates) != 2 {
		t.Fatalf("expected two reference candidates, got %#v", candidates)
	}
	if candidates[0].RelPath != "cmd/main.go" || candidates[0].Line != 3 {
		t.Fatalf("unexpected first candidate: %#v", candidates[0])
	}
}

func TestEditorReferenceCandidatesFromSearchDeduplicates(t *testing.T) {
	results := []workspaceSvc.SearchResult{
		{RelPath: "cmd/main.go", Kind: "file", MatchType: "content", Line: 3, Snippet: "Start()"},
		{RelPath: "cmd/main.go", Kind: "file", MatchType: "content", Line: 3, Snippet: "Start()"},
	}

	candidates := editorReferenceCandidatesFromSearch("Start", results)

	if len(candidates) != 1 {
		t.Fatalf("expected duplicate references to collapse, got %#v", candidates)
	}
}

func TestEditorReferenceLabelAndStatus(t *testing.T) {
	label := editorReferenceLabel(editorReferenceCandidate{RelPath: "cmd/main.go", Line: 3, Snippet: "Start()"})
	if !strings.Contains(label, "cmd/main.go:3") || !strings.Contains(label, "Start()") {
		t.Fatalf("unexpected reference label: %q", label)
	}
	if got := editorReferencesStatus("Start", 0); got != "No references found for Start." {
		t.Fatalf("unexpected empty references status: %q", got)
	}
	if got := editorReferencesStatus("Start", 2); got != "2 reference(s) for Start. Select one to open and jump." {
		t.Fatalf("unexpected references status: %q", got)
	}
}
