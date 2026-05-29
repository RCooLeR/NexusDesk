package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateProjectMemoryCreatesAndUpdatesRecordWithSources(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "README.md", "# Project\n")
	writeTestFile(t, root, "docs/architecture.md", "Keep services framework-free.\n")

	service := New()
	created, err := service.UpdateProjectMemory(root, ProjectMemoryUpdateRequest{
		Key:            "architecture.boundaries",
		Content:        "Services stay framework-free; Fyne remains in UI/app/theme packages.",
		SourceRelPaths: []string{"README.md", "docs/architecture.md", "README.md"},
	})
	if err != nil {
		t.Fatalf("UpdateProjectMemory create returned error: %v", err)
	}
	if !created.Created || created.Count != 1 || created.Record.Key != "architecture.boundaries" {
		t.Fatalf("unexpected create result: %#v", created)
	}
	assertStringSliceEqual(t, created.Record.SourceRelPaths, []string{"README.md", "docs/architecture.md"})
	if created.Record.SourceSHA256 == "" {
		t.Fatal("expected source fingerprint")
	}

	updated, err := service.UpdateProjectMemory(root, ProjectMemoryUpdateRequest{
		Key:     "architecture.boundaries",
		Content: "Domain and service packages stay framework-free.",
	})
	if err != nil {
		t.Fatalf("UpdateProjectMemory update returned error: %v", err)
	}
	if updated.Created || updated.Count != 1 || updated.Record.Content != "Domain and service packages stay framework-free." {
		t.Fatalf("unexpected update result: %#v", updated)
	}
	if updated.Record.CreatedAt.IsZero() || updated.Record.UpdatedAt.Before(updated.Record.CreatedAt) {
		t.Fatalf("unexpected timestamps: %#v", updated.Record)
	}

	data, err := os.ReadFile(filepath.Join(root, ".nexusdesk", "project-memory", "memory.json"))
	if err != nil {
		t.Fatalf("expected project memory file: %v", err)
	}
	var stored projectMemoryFile
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("stored memory should be valid JSON: %v", err)
	}
	if stored.Version != projectMemoryVersion || len(stored.Records) != 1 {
		t.Fatalf("unexpected stored memory: %#v", stored)
	}
}

func TestUpdateProjectMemoryRejectsUnsafeInputs(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "README.md", "# Project\n")
	service := New()

	tests := []ProjectMemoryUpdateRequest{
		{Key: "../bad", Content: "fact"},
		{Key: "valid", Content: ""},
		{Key: "valid", Content: strings.Repeat("x", projectMemoryMaxContentLen+1)},
		{Key: "valid", Content: "fact", SourceRelPaths: []string{"missing.md"}},
		{Key: "valid", Content: "fact", SourceRelPaths: []string{".nexusdesk/project-memory/memory.json"}},
	}
	for _, request := range tests {
		if _, err := service.UpdateProjectMemory(root, request); err == nil {
			t.Fatalf("expected rejection for %#v", request)
		}
	}
}
