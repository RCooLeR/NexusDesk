package operations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanFindsOperationsFilesWithoutReadingIgnoredDirs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Dockerfile", "FROM alpine\n")
	writeFile(t, root, "compose.yml", "services:\n  api:\n    image: app\n")
	writeFile(t, root, ".env", "API_TOKEN=secret\n")
	writeFile(t, root, "config/app.toml", "debug=true\n")
	writeFile(t, root, "logs/app.log", "started\n")
	writeFile(t, root, "scripts/run.sh", "echo run\n")
	writeFile(t, root, "node_modules/pkg/Dockerfile", "FROM ignored\n")

	result, err := New().Scan(root)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if result.Summary.Files != 6 {
		t.Fatalf("files = %d, want 6: %#v", result.Summary.Files, result.Files)
	}
	if result.Summary.Compose != 1 || result.Summary.Dockerfiles != 1 || result.Summary.Env != 1 || result.Summary.Config != 1 || result.Summary.Logs != 1 || result.Summary.Scripts != 1 {
		t.Fatalf("unexpected summary: %#v", result.Summary)
	}
}

func TestInspectRedactsSecretsAndParsesComposeServices(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docker-compose.yml", strings.Join([]string{
		"services:",
		"  api:",
		"    image: example/api",
		"    ports:",
		"      - \"8080:80\"",
		"    depends_on: [db]",
		"    environment:",
		"      API_TOKEN: super-secret",
		"  db:",
		"    image: postgres",
	}, "\n"))

	inspection, err := New().Inspect(root, "docker-compose.yml")
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if strings.Contains(inspection.Text, "super-secret") {
		t.Fatalf("secret was not redacted:\n%s", inspection.Text)
	}
	if len(inspection.Services) != 2 {
		t.Fatalf("services = %d, want 2: %#v", len(inspection.Services), inspection.Services)
	}
	if inspection.Services[0].Name != "api" || inspection.Services[0].Image != "example/api" {
		t.Fatalf("unexpected first service: %#v", inspection.Services[0])
	}
	if len(inspection.Services[0].DependsOn) != 1 || inspection.Services[0].DependsOn[0] != "db" {
		t.Fatalf("depends_on was not parsed: %#v", inspection.Services[0])
	}
}

func TestInspectRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := New().Inspect(root, "../Dockerfile"); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func writeFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
