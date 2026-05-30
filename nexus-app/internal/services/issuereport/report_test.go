package issuereport

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExportRedactsSecretsAndExcludesWorkspaceContentsByDefault(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".nexusdesk", "metadata"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "secret.txt"), []byte("workspace content should not be included"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".nexusdesk", "metadata", "schema.sql"), []byte("create table jobs(id text);"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".nexusdesk", "jobs", "job-0001"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".nexusdesk", "jobs", "job-0001", "job.log"), []byte("started\nAuthorization: Bearer job-secret\nsafe line"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := Export(Options{
		WorkspaceRoot:     root,
		DiagnosticsReport: "Workspace " + root + "\nAuthorization: Bearer abc123\napi_key=sk-test123",
		ActivityTail:      []string{"password=hunter2", "Opened " + root},
		Now:               time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	if result.Path == "" || result.SizeBytes <= 0 {
		t.Fatalf("unexpected result: %#v", result)
	}

	entries := readZipEntries(t, result.Path)
	for _, required := range []string{"issue-report.json", "diagnostics.md", "activity-tail.txt", "environment.json", "workspace-summary.json"} {
		if _, ok := entries[required]; !ok {
			t.Fatalf("missing %s in issue report entries %#v", required, keys(entries))
		}
	}
	jobLog, ok := entries["job-logs/job-0001/job.log"]
	if !ok {
		t.Fatalf("missing redacted job log in issue report entries %#v", keys(entries))
	}
	if !strings.Contains(jobLog, "safe line") || strings.Contains(jobLog, "job-secret") || !strings.Contains(jobLog, "Authorization: [redacted]") {
		t.Fatalf("job log was not redacted correctly:\n%s", jobLog)
	}
	for name, content := range entries {
		if strings.Contains(content, root) {
			t.Fatalf("%s leaked workspace root in content:\n%s", name, content)
		}
		if strings.Contains(content, "abc123") || strings.Contains(content, "sk-test123") || strings.Contains(content, "hunter2") {
			t.Fatalf("%s leaked secret content:\n%s", name, content)
		}
		if strings.Contains(content, "workspace content should not be included") {
			t.Fatalf("%s included workspace file content by default", name)
		}
	}
	if _, ok := entries["explicit-workspace-files/secret.txt"]; ok {
		t.Fatal("workspace file was included without explicit opt-in")
	}

	var m manifest
	if err := json.Unmarshal([]byte(entries["issue-report.json"]), &m); err != nil {
		t.Fatalf("manifest JSON did not parse: %v", err)
	}
	if m.IncludesWorkspaceContents {
		t.Fatalf("expected default bundle to exclude workspace contents: %#v", m)
	}
	if m.WorkspacePathHash == "" || m.WorkspacePath != "[workspace-root]" {
		t.Fatalf("unexpected manifest workspace fields: %#v", m)
	}
}

func TestExportIncludesOnlyExplicitWorkspaceFilesWhenRequested(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte("token=abc123\nsafe note"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := Export(Options{
		WorkspaceRoot:            root,
		IncludeWorkspaceContents: true,
		WorkspaceContentRelPaths: []string{"notes.md"},
		Now:                      time.Date(2026, 5, 28, 12, 1, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	entries := readZipEntries(t, result.Path)
	content, ok := entries["explicit-workspace-files/notes.md"]
	if !ok {
		t.Fatalf("expected explicit file entry, got %#v", keys(entries))
	}
	if !strings.Contains(content, "safe note") {
		t.Fatalf("expected included file content, got %q", content)
	}
	if strings.Contains(content, "abc123") {
		t.Fatalf("explicit file content was not redacted: %q", content)
	}
}

func TestExportRejectsUnsafeExplicitWorkspacePaths(t *testing.T) {
	root := t.TempDir()
	_, err := Export(Options{
		WorkspaceRoot:            root,
		IncludeWorkspaceContents: true,
		WorkspaceContentRelPaths: []string{"../outside.txt"},
	})
	if err == nil || !strings.Contains(err.Error(), "escapes workspace") {
		t.Fatalf("expected traversal rejection, got %v", err)
	}
}

func readZipEntries(t *testing.T, path string) map[string]string {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("OpenReader failed: %v", err)
	}
	defer reader.Close()
	result := map[string]string{}
	for _, file := range reader.File {
		handle, err := file.Open()
		if err != nil {
			t.Fatalf("opening %s failed: %v", file.Name, err)
		}
		data, err := io.ReadAll(handle)
		_ = handle.Close()
		if err != nil {
			t.Fatalf("reading %s failed: %v", file.Name, err)
		}
		result[file.Name] = string(data)
	}
	return result
}

func keys(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	return out
}
