package workspace

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestScanProblemsFindsMarkersAndJSONErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "main.go"), "package main\n\n// TODO: tighten policy\n// FIXME: handle failure\n")
	writeFile(t, filepath.Join(root, "config", "app.json"), "{\n  \"name\": \"nexus\",\n")

	summary, err := New().ScanProblems(root, 20)
	if err != nil {
		t.Fatalf("ScanProblems returned error: %v", err)
	}
	assertProblem(t, summary.Problems, "config/app.json", "error", "json")
	assertProblem(t, summary.Problems, "src/main.go", "warning", "marker")
	assertProblem(t, summary.Problems, "src/main.go", "info", "marker")
	if summary.Message == "" || summary.GeneratedAt.IsZero() {
		t.Fatalf("expected message and timestamp, got %#v", summary)
	}
}

func TestScanProblemsFindsLanguageDiagnostics(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "broken.go"), "package main\n\nfunc broken( {\n")
	writeFile(t, filepath.Join(root, "config", "bad.yaml"), "services:\n  app: [\n")
	writeFile(t, filepath.Join(root, "config", "bad.toml"), "[package\nname = \"nexus\"\n")

	summary, err := New().ScanProblems(root, 20)
	if err != nil {
		t.Fatalf("ScanProblems returned error: %v", err)
	}
	assertProblem(t, summary.Problems, "src/broken.go", "error", "go")
	assertProblem(t, summary.Problems, "config/bad.yaml", "error", "yaml")
	assertProblem(t, summary.Problems, "config/bad.toml", "error", "toml")
}

func TestScanProblemsIgnoresValidLanguageFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "main.go"), "package main\n\nfunc main() {}\n")
	writeFile(t, filepath.Join(root, "config", "app.yaml"), "services:\n  app:\n    image: nexus\n")
	writeFile(t, filepath.Join(root, "config", "app.toml"), "[package]\nname = \"nexus\"\n")

	summary, err := New().ScanProblems(root, 20)
	if err != nil {
		t.Fatalf("ScanProblems returned error: %v", err)
	}
	if len(summary.Problems) != 0 {
		t.Fatalf("expected no problems for valid language files, got %#v", summary.Problems)
	}
}

func TestScanProblemsFindsMergeConflictMarkers(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "main.go"), "<<<<<<< HEAD\nleft\n=======\nright\n>>>>>>> branch\n")

	summary, err := New().ScanProblems(root, 20)
	if err != nil {
		t.Fatalf("ScanProblems returned error: %v", err)
	}
	assertProblem(t, summary.Problems, "src/main.go", "error", "merge-conflict")
}

func TestScanProblemsSkipsIgnoredFolders(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "node_modules", "pkg", "index.js"), "// TODO: ignored\n")
	writeFile(t, filepath.Join(root, ".nexusdesk", "state.json"), "{\n")

	summary, err := New().ScanProblems(root, 20)
	if err != nil {
		t.Fatalf("ScanProblems returned error: %v", err)
	}
	if len(summary.Problems) != 0 {
		t.Fatalf("expected ignored folders to be skipped, got %#v", summary.Problems)
	}
}

func TestScanProblemsCapsResults(t *testing.T) {
	root := t.TempDir()
	var builder strings.Builder
	for index := 0; index < 20; index++ {
		builder.WriteString("// TODO: item\n")
	}
	writeFile(t, filepath.Join(root, "src", "main.go"), builder.String())

	summary, err := New().ScanProblems(root, 3)
	if err != nil {
		t.Fatalf("ScanProblems returned error: %v", err)
	}
	if len(summary.Problems) != 3 || !summary.Truncated {
		t.Fatalf("expected capped truncated problems, got %#v", summary)
	}
}

func assertProblem(t *testing.T, problems []WorkspaceProblem, relPath string, severity string, source string) {
	t.Helper()
	for _, problem := range problems {
		if problem.RelPath == relPath && problem.Severity == severity && problem.Source == source {
			return
		}
	}
	t.Fatalf("expected problem %s/%s/%s in %#v", relPath, severity, source, problems)
}
