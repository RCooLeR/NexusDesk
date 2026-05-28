package editor

import "testing"

func TestResolveDefinitionFindsGoSymbolReference(t *testing.T) {
	content := "package main\n\nfunc main() {\n\tStart()\n}\n\nfunc Start() {}\n"

	result, ok := ResolveDefinition("main.go", content, 3, 3)

	if !ok {
		t.Fatal("expected local definition to resolve")
	}
	if result.Query != "Start" || result.Item.Label != "Start" || result.Item.Line != 7 {
		t.Fatalf("unexpected definition result: %#v", result)
	}
	if result.RelPath != "main.go" {
		t.Fatalf("expected local definition rel path, got %q", result.RelPath)
	}
}

func TestResolveDefinitionFindsQualifiedMethodReference(t *testing.T) {
	content := "package main\n\ntype Server struct{}\n\nfunc main() {\n\tserver.Run()\n}\n\nfunc (s *Server) Run() {}\n"

	result, ok := ResolveDefinition("main.go", content, 5, 10)

	if !ok {
		t.Fatal("expected qualified method reference to resolve")
	}
	if result.Query != "server.Run" || result.Item.Label != "Run" || result.Item.Line != 9 {
		t.Fatalf("unexpected definition result: %#v", result)
	}
}

func TestResolveDefinitionFindsCSSSelectorReference(t *testing.T) {
	content := ".app-shell {\n  color: red;\n}\n\n.preview .app-shell {\n}\n"

	result, ok := ResolveDefinition("style.css", content, 4, 10)

	if !ok {
		t.Fatal("expected selector reference to resolve")
	}
	if result.Query != ".app-shell" || result.Item.Label != ".app-shell" || result.Item.Line != 1 {
		t.Fatalf("unexpected definition result: %#v", result)
	}
}

func TestResolveDefinitionReportsMissingIdentifier(t *testing.T) {
	if result, ok := ResolveDefinition("main.go", "package main\n", 0, 0); ok || result.Query != "package" {
		t.Fatalf("expected unresolved query, got %#v / %v", result, ok)
	}
}

func TestIdentifierAtCursorUsesPreviousRuneAtTokenBoundary(t *testing.T) {
	got := identifierAtCursor("callThing()\n", 0, len("callThing"))

	if got != "callThing" {
		t.Fatalf("unexpected identifier: %q", got)
	}
}

func TestSymbolAtCursorExportsIdentifierLookup(t *testing.T) {
	if got := SymbolAtCursor("server.Start()\n", 0, 9); got != "server.Start" {
		t.Fatalf("unexpected exported cursor symbol: %q", got)
	}
}

func TestResolveWorkspaceDefinitionFindsOtherFileFirst(t *testing.T) {
	result, ok := ResolveWorkspaceDefinition("Start", "cmd/main.go", []DefinitionFile{
		{RelPath: "cmd/main.go", Content: "package main\n\nfunc main() { Start() }\n"},
		{RelPath: "internal/app/app.go", Content: "package app\n\nfunc Start() {}\n"},
	})

	if !ok {
		t.Fatal("expected workspace definition to resolve")
	}
	if result.RelPath != "internal/app/app.go" || result.Item.Label != "Start" || result.Item.Line != 3 {
		t.Fatalf("unexpected workspace definition: %#v", result)
	}
}

func TestResolveWorkspaceDefinitionFallsBackToCurrentFile(t *testing.T) {
	result, ok := ResolveWorkspaceDefinition("Start", "cmd/main.go", []DefinitionFile{
		{RelPath: "cmd/main.go", Content: "package main\n\nfunc Start() {}\n"},
	})

	if !ok || result.RelPath != "cmd/main.go" || result.Item.Line != 3 {
		t.Fatalf("unexpected fallback definition: %#v ok=%t", result, ok)
	}
}

func TestResolveWorkspaceDefinitionFindsQualifiedSymbol(t *testing.T) {
	result, ok := ResolveWorkspaceDefinition("app.Start", "cmd/main.go", []DefinitionFile{
		{RelPath: "internal/app/app.go", Content: "package app\n\nfunc Start() {}\n"},
	})

	if !ok || result.Item.Label != "Start" {
		t.Fatalf("unexpected qualified workspace definition: %#v ok=%t", result, ok)
	}
}
