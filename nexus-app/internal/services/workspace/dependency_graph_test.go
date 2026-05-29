package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDependencyGraphScansBoundedCodeImports(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/app\n")
	writeTestFile(t, root, "cmd/main.go", `package main

import (
	"fmt"
	"example.com/app/internal/service"
)

func main() {
	_ = fmt.Sprintf("%s", service.Name)
}
`)
	writeTestFile(t, root, "internal/service/service.go", "package service\n\nconst Name = \"svc\"\n")
	writeTestFile(t, root, "web/app.ts", `import { widget } from "./widget"
const lazy = import("./lazy")
const fs = require("fs")
console.log(widget, lazy, fs)
`)
	writeTestFile(t, root, "web/widget.ts", "export const widget = 1\n")
	writeTestFile(t, root, "web/lazy.ts", "export const lazy = 2\n")
	writeTestFile(t, root, "scripts/report.py", "import os, json as js\nfrom .helpers import build\n")
	writeTestFile(t, root, "scripts/helpers.py", "def build(): pass\n")

	graph, err := New().DependencyGraph(root, DependencyGraphOptions{MaxFiles: 20, MaxEdges: 20})
	if err != nil {
		t.Fatalf("DependencyGraph returned error: %v", err)
	}
	if graph.FilesScanned != 7 {
		t.Fatalf("expected 7 scanned code files, got %d (%#v)", graph.FilesScanned, graph)
	}
	if graph.Truncated {
		t.Fatalf("expected graph not to truncate: %#v", graph)
	}
	assertDependencyEdge(t, graph, "cmd/main.go", "internal/service", "example.com/app/internal/service", true)
	assertDependencyEdge(t, graph, "cmd/main.go", "external:fmt", "fmt", false)
	assertDependencyEdge(t, graph, "web/app.ts", "web/widget.ts", "./widget", true)
	assertDependencyEdge(t, graph, "web/app.ts", "web/lazy.ts", "./lazy", true)
	assertDependencyEdge(t, graph, "web/app.ts", "external:fs", "fs", false)
	assertDependencyEdge(t, graph, "scripts/report.py", "scripts/helpers.py", ".helpers", true)
	assertDependencyEdge(t, graph, "scripts/report.py", "external:os", "os", false)
}

func TestDependencyGraphFocusAndCaps(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "a/main.ts", `import "./one"
import "./two"
`)
	writeTestFile(t, root, "a/one.ts", "export const one = 1\n")
	writeTestFile(t, root, "a/two.ts", "export const two = 2\n")
	writeTestFile(t, root, "b/ignored.ts", `import "../a/one"`)

	graph, err := New().DependencyGraph(root, DependencyGraphOptions{RelPath: "a/main.ts", MaxFiles: 10, MaxEdges: 1})
	if err != nil {
		t.Fatalf("DependencyGraph focused file returned error: %v", err)
	}
	if graph.FilesScanned != 1 || len(graph.Edges) != 1 || !graph.Truncated {
		t.Fatalf("expected focused capped graph, got %#v", graph)
	}
	if graph.Edges[0].From != "a/main.ts" || graph.Edges[0].To != "a/one.ts" {
		t.Fatalf("unexpected focused edge: %#v", graph.Edges[0])
	}
}

func TestSourceFilesReturnsBoundedSupportedFiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "src/a.go", "package src\n")
	writeTestFile(t, root, "src/b.ts", "export const b = 1\n")
	writeTestFile(t, root, "src/readme.md", "# ignored\n")
	writeTestFile(t, root, "other/c.py", "def c(): pass\n")

	list, err := New().SourceFiles(root, SourceFileOptions{RelPath: "src", MaxFiles: 10})
	if err != nil {
		t.Fatalf("SourceFiles returned error: %v", err)
	}
	if list.RootRelPath != "src" || list.Truncated {
		t.Fatalf("unexpected source file list metadata: %#v", list)
	}
	assertStringSliceEqual(t, list.Files, []string{"src/a.go", "src/b.ts"})

	capped, err := New().SourceFiles(root, SourceFileOptions{MaxFiles: 2})
	if err != nil {
		t.Fatalf("SourceFiles capped returned error: %v", err)
	}
	if len(capped.Files) != 2 || !capped.Truncated {
		t.Fatalf("expected capped source file list, got %#v", capped)
	}
}

func assertDependencyEdge(t *testing.T, graph DependencyGraph, from string, to string, spec string, resolved bool) {
	t.Helper()
	for _, edge := range graph.Edges {
		if edge.From == from && edge.To == to && edge.Spec == spec && edge.Resolved == resolved {
			return
		}
	}
	t.Fatalf("missing edge from=%s to=%s spec=%s resolved=%v in %#v", from, to, spec, resolved, graph.Edges)
}

func assertStringSliceEqual(t *testing.T, actual []string, expected []string) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("expected %#v, got %#v", expected, actual)
	}
	for index := range expected {
		if actual[index] != expected[index] {
			t.Fatalf("expected %#v, got %#v", expected, actual)
		}
	}
}

func writeTestFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
