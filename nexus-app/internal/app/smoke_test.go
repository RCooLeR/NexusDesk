package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunSmokeCheckExercisesCorePackagedFlows(t *testing.T) {
	root := t.TempDir()
	var output bytes.Buffer
	if err := RunSmokeCheck(root, &output); err != nil {
		t.Fatalf("RunSmokeCheck returned error: %v", err)
	}

	var report smokeReport
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatalf("smoke report was not JSON: %v\n%s", err, output.String())
	}
	if report.Workspace == "" || len(report.Checks) != 8 {
		t.Fatalf("unexpected smoke report: %#v", report)
	}
	checks := map[string]bool{}
	for _, check := range report.Checks {
		if check.Status != "ok" {
			t.Fatalf("unexpected smoke check status: %#v", check)
		}
		checks[check.Name] = true
	}
	for _, name := range []string{
		"workspace-open",
		"file-preview",
		"workspace-search",
		"edit-save-revert",
		"assistant-settings",
		"dataset-profile",
		"artifact-write-read",
		"diagnostics-export",
	} {
		if !checks[name] {
			t.Fatalf("missing smoke check %q in %#v", name, checks)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "notes", "smoke-edit.txt")); !os.IsNotExist(err) {
		t.Fatalf("smoke edit should have been rolled back, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".nexusdesk", "smoke", "settings.json")); err != nil {
		t.Fatalf("smoke settings were not written: %v", err)
	}
}

func TestRunWithArgsSmokeCheckValidatesWorkspaceArgument(t *testing.T) {
	if got := RunWithArgs([]string{"--smoke-check"}); got != 2 {
		t.Fatalf("RunWithArgs missing smoke workspace exit=%d, want 2", got)
	}
}
