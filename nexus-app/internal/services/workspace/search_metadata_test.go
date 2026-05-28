package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchWithMetadataSummarizesSearchWithoutSourceSnippets(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "docs", "needle-guide.md"), "plain docs\n")
	writeFile(t, filepath.Join(root, "src", "main.go"), "package main\nconst marker = \"needle secret workspace content\"\n")
	writeFile(t, filepath.Join(root, "node_modules", "pkg", "ignored.js"), "needle")

	results, metadata, err := New().SearchWithMetadata(root, "needle", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchWithMetadata returned error: %v", err)
	}
	if metadata.WorkspaceName != filepath.Base(root) {
		t.Fatalf("unexpected workspace name %q", metadata.WorkspaceName)
	}
	if metadata.ResultCount != len(results) {
		t.Fatalf("metadata count %d does not match result count %d", metadata.ResultCount, len(results))
	}
	if metadata.PathMatches == 0 || metadata.ContentMatches == 0 {
		t.Fatalf("expected path and content counts, got %#v", metadata)
	}
	if metadata.FilesScanned == 0 || metadata.FilesWithContentMatches == 0 {
		t.Fatalf("expected file scan counts, got %#v", metadata)
	}
	if metadata.DirectoriesSkipped == 0 {
		t.Fatalf("expected ignored directory to be counted, got %#v", metadata)
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Marshal metadata failed: %v", err)
	}
	if strings.Contains(string(data), "secret workspace content") {
		t.Fatalf("metadata should not persist source snippets: %s", data)
	}
}

func TestWriteReadSearchMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "notes.txt"), "needle\n")
	_, metadata, err := New().SearchWithMetadata(root, "needle", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchWithMetadata returned error: %v", err)
	}

	service := New()
	export, err := service.WriteSearchMetadata(root, metadata)
	if err != nil {
		t.Fatalf("WriteSearchMetadata returned error: %v", err)
	}
	if export.RelPath != searchMetadataRelPath {
		t.Fatalf("unexpected metadata rel path %q", export.RelPath)
	}
	if _, err := os.Stat(export.AbsPath); err != nil {
		t.Fatalf("metadata file was not written: %v", err)
	}
	read, err := service.ReadSearchMetadata(root)
	if err != nil {
		t.Fatalf("ReadSearchMetadata returned error: %v", err)
	}
	if read.Query != "needle" || read.ResultCount != metadata.ResultCount {
		t.Fatalf("unexpected persisted metadata: %#v", read)
	}
}

func TestWriteSearchMetadataQuarantinesCorruptManifest(t *testing.T) {
	root := t.TempDir()
	corruptPath := filepath.Join(root, filepath.FromSlash(searchMetadataRelPath))
	if err := os.MkdirAll(filepath.Dir(corruptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(corruptPath, []byte("{not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	service := New()
	export, err := service.WriteSearchMetadata(root, SearchMetadata{
		Version:     searchMetadataVersion,
		Query:       "needle",
		GeneratedAt: nowUTC(),
	})
	if err != nil {
		t.Fatalf("WriteSearchMetadata returned error: %v", err)
	}
	if !export.Recovered {
		t.Fatalf("expected corrupt manifest recovery, got %#v", export)
	}
	if export.RecoveredRelPath == "" || !strings.HasPrefix(export.RecoveredRelPath, searchMetadataRecoveryDir+"/") {
		t.Fatalf("unexpected recovery path %#v", export)
	}
	if _, err := os.Stat(export.RecoveredAbsPath); err != nil {
		t.Fatalf("recovered corrupt manifest was not archived: %v", err)
	}
	read, err := service.ReadSearchMetadata(root)
	if err != nil {
		t.Fatalf("ReadSearchMetadata returned error after recovery: %v", err)
	}
	if read.Query != "needle" {
		t.Fatalf("unexpected recovered metadata: %#v", read)
	}
}
