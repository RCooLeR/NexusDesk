package operations

import (
	"context"
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
	if inspection.Topology.Summary == "" || len(inspection.Topology.Edges) != 1 || inspection.Topology.Edges[0].From != "api" || inspection.Topology.Edges[0].To != "db" {
		t.Fatalf("topology was not built: %#v", inspection.Topology)
	}
}

func TestInspectRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := New().Inspect(root, "../Dockerfile"); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestScanContextCanceled(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New().ScanContext(ctx, root); err == nil {
		t.Fatal("expected canceled context to fail scan")
	}
}

func TestInspectContextCanceled(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Dockerfile", "FROM alpine\n")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New().InspectContext(ctx, root, "Dockerfile"); err == nil {
		t.Fatal("expected canceled context to fail inspect")
	}
}

func TestBuildComposeTopologySummarizesDependenciesPortsAndVolumes(t *testing.T) {
	topology := BuildComposeTopology([]ComposeService{
		{Name: "api", Image: "example/api", Ports: []string{"8080:80"}, Volumes: []string{"./src:/app", "app-cache:/cache"}, DependsOn: []string{"db", "missing"}},
		{Name: "db", Image: "postgres", Volumes: []string{"pgdata:/var/lib/postgresql/data"}},
	})
	if topology.Summary != "2 service(s), 2 dependency edge(s), 1 exposed port(s), 2 named volume(s)." {
		t.Fatalf("unexpected summary: %q", topology.Summary)
	}
	if len(topology.ExposedPorts) != 1 || topology.ExposedPorts[0].Service != "api" || topology.ExposedPorts[0].Port != "8080:80" {
		t.Fatalf("unexpected port exposures: %#v", topology.ExposedPorts)
	}
	if strings.Join(topology.NamedVolumes, ",") != "app-cache,pgdata" {
		t.Fatalf("unexpected named volumes: %#v", topology.NamedVolumes)
	}
	if len(topology.Warnings) != 1 || !strings.Contains(topology.Warnings[0], "missing") {
		t.Fatalf("expected missing dependency warning, got %#v", topology.Warnings)
	}
}

func TestParseComposeServicesSupportsDependsOnMapping(t *testing.T) {
	services := ParseComposeServices(strings.Join([]string{
		"services:",
		"  api:",
		"    image: app",
		"    depends_on:",
		"      db:",
		"        condition: service_healthy",
		"  db:",
		"    image: postgres",
	}, "\n"))
	if len(services) != 2 {
		t.Fatalf("services = %d, want 2: %#v", len(services), services)
	}
	if len(services[0].DependsOn) != 1 || services[0].DependsOn[0] != "db" {
		t.Fatalf("depends_on mapping was not parsed: %#v", services[0])
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
