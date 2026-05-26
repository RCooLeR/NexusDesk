package workspace

import "testing"

func TestSearchIncludesSymbolsWhenRequested(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/service.ts", "export function buildContextPack() {\n  return true;\n}\n")

	results, err := Search(root, "buildContext", SearchOptions{Symbols: true})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	assertSearchContains(t, results, "src/service.ts", "symbol")
}

func TestBuildSymbolsSupportsGoAndMarkdown(t *testing.T) {
	symbols := BuildSymbols("README.md", "README.md", "# Product\n\n## Setup\n")
	if len(symbols) != 2 || symbols[0].Name != "Product" || symbols[1].Name != "Setup" {
		t.Fatalf("expected markdown headings, got %#v", symbols)
	}

	symbols = BuildSymbols("app/service.go", "service.go", "package app\n\ntype Runner struct{}\n\nfunc Execute() {}\n")
	if len(symbols) != 2 || symbols[0].Name != "Runner" || symbols[1].Name != "Execute" {
		t.Fatalf("expected Go symbols, got %#v", symbols)
	}
}
