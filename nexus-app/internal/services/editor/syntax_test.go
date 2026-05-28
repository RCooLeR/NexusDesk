package editor

import (
	"strings"
	"testing"
)

func TestDetectSyntaxLanguageUsesWailsLanguageCoverage(t *testing.T) {
	cases := map[string]string{
		"main.go":               "go",
		"app.tsx":               "typescript",
		"config.code-workspace": "json",
		"Dockerfile":            "dockerfile",
		"query.sql":             "sql",
		"readme.md":             "markdown",
		"script.ps1":            "powershell",
	}
	for fileName, want := range cases {
		if got := DetectSyntaxLanguage(fileName).ID; got != want {
			t.Fatalf("DetectSyntaxLanguage(%q) = %q, want %q", fileName, got, want)
		}
	}
}

func TestAnalyzeSyntaxClassifiesNativeTokens(t *testing.T) {
	analysis := AnalyzeSyntax("main.go", "package main\n\n// boot\nfunc main() {\n\tprintln(\"hi\", 42)\n}\n")
	if analysis.Language.ID != "go" {
		t.Fatalf("unexpected language: %#v", analysis.Language)
	}
	for _, kind := range []string{"keyword", "comment", "string", "number"} {
		if analysis.Counts[kind] == 0 {
			t.Fatalf("expected %s token count in %#v", kind, analysis.Counts)
		}
	}
	if len(analysis.Tokens) == 0 || analysis.Tokens[0].Line != 1 {
		t.Fatalf("unexpected tokens: %#v", analysis.Tokens)
	}
}

func TestAnalyzeSyntaxCapsLargeFiles(t *testing.T) {
	content := strings.Repeat("func main() { println(123) }\n", syntaxMaxTokens+20)
	analysis := AnalyzeSyntax("main.go", content)
	if len(analysis.Tokens) != syntaxMaxTokens || !analysis.Truncated {
		t.Fatalf("expected capped truncated analysis, got tokens=%d truncated=%t", len(analysis.Tokens), analysis.Truncated)
	}
}
