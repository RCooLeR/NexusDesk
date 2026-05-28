package editor

import (
	"strings"
	"testing"
)

func TestAnalyzeDraftDiagnosticsFindsMarkersAndJSON(t *testing.T) {
	diagnostics := AnalyzeDraftDiagnostics("config/app.json", "{\n  \"name\": true,,\n  \"note\": \"TODO wire this\"\n}\n")

	assertDraftDiagnostic(t, diagnostics, "error", "json")
	assertDraftDiagnostic(t, diagnostics, "info", "marker")
	if diagnostics[0].Severity != "error" {
		t.Fatalf("expected errors to sort before markers, got %#v", diagnostics)
	}
}

func TestAnalyzeDraftDiagnosticsFindsLanguageDiagnostics(t *testing.T) {
	cases := []struct {
		fileName string
		content  string
		source   string
	}{
		{fileName: "src/broken.go", content: "package main\nfunc main( {\n", source: "go"},
		{fileName: "config/bad.yaml", content: "root:\n  - a\n bad\n", source: "yaml"},
		{fileName: "config/bad.toml", content: "name = \"nexus\"\n[bad\n", source: "toml"},
		{fileName: "docs/bad.xml", content: "<root>\n  <child>\n</root>\n", source: "xml"},
	}
	for _, tc := range cases {
		t.Run(tc.fileName, func(t *testing.T) {
			diagnostics := AnalyzeDraftDiagnostics(tc.fileName, tc.content)
			assertDraftDiagnostic(t, diagnostics, "error", tc.source)
		})
	}
}

func TestAnalyzeDraftDiagnosticsFindsMergeConflictMarker(t *testing.T) {
	diagnostics := AnalyzeDraftDiagnostics("src/main.go", "package main\n<<<<<<< HEAD\n")

	assertDraftDiagnostic(t, diagnostics, "error", "merge-conflict")
}

func TestAnalyzeDraftDiagnosticsIgnoresValidDrafts(t *testing.T) {
	cases := map[string]string{
		"src/main.go":       "package main\nfunc main() {}\n",
		"config/app.json":   "{\"name\": true}\n",
		"config/app.yaml":   "name: nexus\n",
		"config/app.toml":   "name = \"nexus\"\n",
		"docs/example.xml":  "<root><child /></root>\n",
		"docs/example.txt":  "plain text\n",
		"docs/example.md":   "# Notes\n",
		"scripts/build.ps1": "Write-Host nexus\n",
	}
	for fileName, content := range cases {
		if diagnostics := AnalyzeDraftDiagnostics(fileName, content); len(diagnostics) != 0 {
			t.Fatalf("expected no diagnostics for %s, got %#v", fileName, diagnostics)
		}
	}
}

func assertDraftDiagnostic(t *testing.T, diagnostics []DraftDiagnostic, severity string, source string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == severity && diagnostic.Source == source {
			if diagnostic.Line <= 0 || strings.TrimSpace(diagnostic.Message) == "" {
				t.Fatalf("diagnostic should include line and message: %#v", diagnostic)
			}
			return
		}
	}
	t.Fatalf("expected diagnostic %s/%s in %#v", severity, source, diagnostics)
}
