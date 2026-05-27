package artifacts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteOperationsRunbookCreatesMarkdownArtifact(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "compose.yml"), []byte("services:\n  api:\n    image: app\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	artifact, err := store.WriteOperationsRunbook(OperationsRunbookReport{
		SourcePath: "compose.yml",
		Kind:       "compose",
		Size:       42,
		Content:    "services:\n  api:\n    image: app",
		Services: []OperationsServiceSummary{
			{Name: "api", Image: "app", Ports: []string{"8080:80"}, DependsOn: []string{"db"}},
		},
		TopologySummary: "1 service(s), 1 dependency edge(s), 1 exposed port(s), 0 named volume(s).",
		TopologyEdges:   []OperationsTopologyEdge{{From: "api", To: "db", Relation: "depends_on", Missing: true}},
		ExposedPorts:    []OperationsPortExposure{{Service: "api", Port: "8080:80"}},
		Warnings:        []string{"Read-only inspection only."},
	})
	if err != nil {
		t.Fatalf("WriteOperationsRunbook() error = %v", err)
	}
	if artifact.Kind != "operations-runbook" || artifact.MetadataPath == "" || len(artifact.SourcePaths) != 1 {
		t.Fatalf("unexpected operations artifact: %#v", artifact)
	}
	if !strings.HasPrefix(artifact.RelPath, ".nexusdesk/artifacts/operations-runbooks/") {
		t.Fatalf("unexpected artifact path: %q", artifact.RelPath)
	}
	text, err := store.ReadArtifactText(artifact.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText() error = %v", err)
	}
	for _, expected := range []string{"# Operations Runbook", "compose.yml", "Safety Notes", "api", "Compose Topology", "api -> db", "api exposes 8080:80", "Operator Checklist", "Source Evidence"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected runbook to contain %q, got:\n%s", expected, text)
		}
	}
	matches, err := store.ListArtifacts(ListOptions{Query: "kind:operations-runbook"})
	if err != nil {
		t.Fatalf("ListArtifacts() error = %v", err)
	}
	if len(matches) != 1 || matches[0].RelPath != artifact.RelPath {
		t.Fatalf("expected searchable operations runbook artifact, got %#v", matches)
	}
}

func TestWriteOperationsRunbookRequiresSourceAndEvidence(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if _, err := store.WriteOperationsRunbook(OperationsRunbookReport{Content: "evidence"}); err == nil {
		t.Fatal("expected missing source to be rejected")
	}
	if _, err := store.WriteOperationsRunbook(OperationsRunbookReport{SourcePath: "compose.yml"}); err == nil {
		t.Fatal("expected missing evidence to be rejected")
	}
}
