package workspace

import "testing"

func TestScanProblemsFindsMarkersAndJSONErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/main.go", "package main\n\n// TODO: tighten policy\n// FIXME: handle failure\n")
	writeFile(t, root, "config/app.json", "{\n  \"name\": \"nexus\",\n")

	summary, err := ScanProblems(root, 20)
	if err != nil {
		t.Fatalf("ScanProblems returned error: %v", err)
	}

	assertProblem(t, summary.Problems, "config/app.json", "error", "json")
	assertProblem(t, summary.Problems, "src/main.go", "warning", "marker")
	assertProblem(t, summary.Problems, "src/main.go", "info", "marker")
	if summary.Message == "" || summary.GeneratedAt == "" {
		t.Fatalf("expected message and timestamp, got %#v", summary)
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
