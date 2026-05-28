package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	fynetest "fyne.io/fyne/v2/test"
)

func TestCompatibilityImportJobLabel(t *testing.T) {
	if got := compatibilityImportJobLabel(); got != "Compatibility metadata import" {
		t.Fatalf("unexpected compatibility import job label: %q", got)
	}
}

func TestCompatibilityImportDedupGuardsByWorkspace(t *testing.T) {
	view := &View{}
	root := "E:/workspace"
	if !view.beginCompatibilityImport(root) {
		t.Fatal("expected first import begin to succeed")
	}
	if view.beginCompatibilityImport(root) {
		t.Fatal("expected duplicate import begin to be blocked")
	}
	if !view.beginCompatibilityImport("E:/workspace-2") {
		t.Fatal("expected different workspace import begin to succeed")
	}
	view.endCompatibilityImport(root)
	if !view.beginCompatibilityImport(root) {
		t.Fatal("expected import begin to succeed after end")
	}
}

func TestOpenWorkspaceHandlesMetadataStoreUnavailable(t *testing.T) {
	root := t.TempDir()
	metadataPath := filepath.Join(root, ".nexusdesk")
	if err := os.WriteFile(metadataPath, []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	app := fynetest.NewTempApp(t)
	window := app.NewWindow("workspace-open-test")
	defer window.Close()
	view := New(window)
	t.Cleanup(func() {
		if view.metadataStore != nil {
			_ = view.metadataStore.Close()
		}
	})
	view.compatibilityImportByWS[root] = true

	view.openWorkspace(root)

	if got := view.state.Workspace().Root; got == "" {
		t.Fatal("expected workspace root to be set even when metadata store is unavailable")
	}
	if view.metadataStore != nil {
		t.Fatal("expected metadata store to remain nil on metadata initialization failure")
	}
	if !containsActivityLine(view.recentActivityLines(20), "Metadata store unavailable:") {
		t.Fatalf("expected metadata unavailable activity message, got %#v", view.recentActivityLines(20))
	}
	if !containsActivityLine(view.recentActivityLines(20), "Opened workspace "+root) {
		t.Fatalf("expected workspace-open activity message, got %#v", view.recentActivityLines(20))
	}
}

func TestOpenWorkspaceLoadsMetadataStoreWhenAvailable(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# test"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	app := fynetest.NewTempApp(t)
	window := app.NewWindow("workspace-open-success")
	defer window.Close()
	view := New(window)
	t.Cleanup(func() {
		if view.metadataStore != nil {
			_ = view.metadataStore.Close()
		}
	})
	view.compatibilityImportByWS[root] = true

	view.openWorkspace(root)

	if got := view.state.Workspace().Root; got == "" {
		t.Fatal("expected workspace root to be set")
	}
	if view.metadataStore == nil {
		t.Fatal("expected metadata store to be initialized")
	}
	if !containsActivityLine(view.recentActivityLines(20), "SQLite metadata store is active.") {
		t.Fatalf("expected metadata-active activity message, got %#v", view.recentActivityLines(20))
	}
	if !containsActivityLine(view.recentActivityLines(20), "Opened workspace "+root) {
		t.Fatalf("expected workspace-open activity message, got %#v", view.recentActivityLines(20))
	}
	for _, tab := range view.editorSession.Tabs() {
		if tab.Kind == "welcome" {
			t.Fatalf("expected welcome tab to close after workspace open, got %#v", view.editorSession.Tabs())
		}
	}
}

func TestOpenSingleFileOpensParentWorkspaceAndPreview(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "notes.md")
	if err := os.WriteFile(filePath, []byte("# Notes\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	app := fynetest.NewTempApp(t)
	window := app.NewWindow("single-file-open")
	defer window.Close()
	view := New(window)
	t.Cleanup(func() {
		if view.metadataStore != nil {
			_ = view.metadataStore.Close()
		}
	})
	view.compatibilityImportByWS[root] = true

	view.openSingleFile(filePath)

	if got := view.state.Workspace().Root; got != root {
		t.Fatalf("expected parent workspace %q, got %q", root, got)
	}
	tab, ok := view.editorSession.Tab(view.editorSession.ActiveID())
	if !ok {
		t.Fatal("expected active editor tab")
	}
	if tab.RelPath != "notes.md" || tab.SourceText != "# Notes\n" {
		t.Fatalf("unexpected opened tab: %#v", tab)
	}
}

func containsActivityLine(lines []string, part string) bool {
	part = strings.TrimSpace(part)
	if part == "" {
		return false
	}
	for _, line := range lines {
		if strings.Contains(line, part) {
			return true
		}
	}
	return false
}
