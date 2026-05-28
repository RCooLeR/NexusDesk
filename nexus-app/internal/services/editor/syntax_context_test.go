package editor

import (
	"strings"
	"testing"
)

func TestSyntaxContextAtCursorReportsTokenAndSymbol(t *testing.T) {
	content := "package main\n\nfunc main() {\n\tprintln(\"hi\", 42)\n}\n"

	context := SyntaxContextAtCursor("main.go", content, 3, 10)

	if context.Line != 4 || context.Column != 11 {
		t.Fatalf("unexpected cursor position: %#v", context)
	}
	if context.Symbol != "hi" {
		t.Fatalf("expected cursor symbol hi, got %q", context.Symbol)
	}
	if context.Token.Kind != "string" || context.Token.Text != `"hi"` {
		t.Fatalf("expected string token under cursor, got %#v", context.Token)
	}
	if len(context.LineTokens) < 2 {
		t.Fatalf("expected line tokens, got %#v", context.LineTokens)
	}
	if !strings.Contains(context.Message, "Cursor: L4:C11") || !strings.Contains(context.Message, "symbol hi") {
		t.Fatalf("unexpected context message: %q", context.Message)
	}
}

func TestSyntaxContextFromAnalysisUsesLineTokenFallback(t *testing.T) {
	content := "package main\n"
	analysis := AnalyzeSyntax("main.go", content)

	context := SyntaxContextFromAnalysis(content, analysis, 0, 8)

	if context.Token.Kind != "" {
		t.Fatalf("expected no token under cursor on plain identifier, got %#v", context.Token)
	}
	if len(context.LineTokens) == 0 || !strings.Contains(context.Message, "token(s) on this line") {
		t.Fatalf("expected line-token fallback message, got %#v / %q", context.LineTokens, context.Message)
	}
}

func TestSyntaxContextClampsNegativeCursor(t *testing.T) {
	context := SyntaxContextAtCursor("notes.txt", "hello\n", -2, -5)

	if context.Line != 1 || context.Column != 1 {
		t.Fatalf("expected clamped one-based cursor, got %#v", context)
	}
}
