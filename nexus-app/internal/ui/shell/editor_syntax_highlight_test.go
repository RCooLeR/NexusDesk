package shell

import (
	"strings"
	"testing"

	editorSvc "nexusdesk/internal/services/editor"
)

func TestSyntaxHighlightGridStylesTokens(t *testing.T) {
	grid := newSyntaxHighlightGrid("main.go", "package main\n// boot\nconst count = 42\n")

	if got := grid.RowText(0); got != "package main" {
		t.Fatalf("unexpected first row: %q", got)
	}
	keyword := grid.Row(0).Cells[0].Style
	comment := grid.Row(1).Cells[0].Style
	number := grid.Row(2).Cells[14].Style
	if keyword != syntaxStyleForKind("keyword") {
		t.Fatalf("expected package keyword style, got %#v", keyword)
	}
	if comment != syntaxStyleForKind("comment") {
		t.Fatalf("expected comment style, got %#v", comment)
	}
	if number != syntaxStyleForKind("number") {
		t.Fatalf("expected number style, got %#v", number)
	}
}

func TestSyntaxHighlightGridKeepsPlainTextUnstyled(t *testing.T) {
	grid := newSyntaxHighlightGrid("main.go", "package main\n")

	if grid.Row(0).Cells[8].Style != nil {
		t.Fatalf("expected plain identifier to stay unstyled, got %#v", grid.Row(0).Cells[8].Style)
	}
}

func TestSyntaxHighlightGridMarksActiveCursorLine(t *testing.T) {
	grid := newSyntaxHighlightGrid("main.go", "package main\nfunc main() {}\n")
	analysis := editorSvc.AnalyzeSyntax("main.go", "package main\nfunc main() {}\n")

	applySyntaxHighlightGridWithCursor(grid, "package main\nfunc main() {}\n", analysis, 1)

	if grid.Row(1).Style != syntaxStyleForKind("active-line") {
		t.Fatalf("expected active line row style, got %#v", grid.Row(1).Style)
	}
	if grid.Row(1).Cells[0].Style != syntaxStyleForKind("keyword") {
		t.Fatalf("expected token style to remain after active row style, got %#v", grid.Row(1).Cells[0].Style)
	}
}

func TestNormalizeSyntaxHighlightTextUsesLF(t *testing.T) {
	if got := normalizeSyntaxHighlightText("a\r\nb"); got != "a\nb" {
		t.Fatalf("unexpected normalized text: %q", got)
	}
}

func TestFormatLanguageActionPlanShowsStatuses(t *testing.T) {
	text := formatLanguageActionPlan(editorSvc.BuildLanguageActionPlan("main.go", "package main\n\nfunc main() {}\n"))
	for _, expected := range []string{"Language Actions", "Summary:", "Native Parity Beta: accepted-for-native-parity-beta", "Syntax mirror", "Syntax highlighting [available]", "Definition and references [fallback]", "External LSP [planned]", "Native Parity Beta strategy [available]"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("language action plan missing %q:\n%s", expected, text)
		}
	}
}
