package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecentWorkspaceStoreAddsAndMovesExistingPathToTop(t *testing.T) {
	store := NewRecentWorkspaceStore(filepath.Join(t.TempDir(), "recent.json"))
	first := filepath.Join(t.TempDir(), "first")
	second := filepath.Join(t.TempDir(), "second")

	mkdir(t, first)
	mkdir(t, second)

	if _, err := store.Add(first); err != nil {
		t.Fatalf("Add first failed: %v", err)
	}
	if _, err := store.Add(second); err != nil {
		t.Fatalf("Add second failed: %v", err)
	}
	items, err := store.Add(first)
	if err != nil {
		t.Fatalf("Add first again failed: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected two recent items, got %d", len(items))
	}
	if items[0].Path != first {
		t.Fatalf("expected first path to move to top, got %s", items[0].Path)
	}
}

func TestRecentWorkspaceStoreLimitsItems(t *testing.T) {
	store := NewRecentWorkspaceStore(filepath.Join(t.TempDir(), "recent.json"))

	var items []RecentWorkspace
	var err error
	for index := 0; index < 14; index++ {
		path := filepath.Join(t.TempDir(), "workspace-"+string(rune('a'+index)))
		mkdir(t, path)
		items, err = store.Add(path)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	if len(items) != recentWorkspaceLimit {
		t.Fatalf("expected %d items, got %d", recentWorkspaceLimit, len(items))
	}
}

func TestRecentWorkspaceStoreReadsMissingFileAsEmpty(t *testing.T) {
	store := NewRecentWorkspaceStore(filepath.Join(t.TempDir(), "missing.json"))

	items, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no items, got %d", len(items))
	}
}

func TestRecentWorkspaceStoreRemovesPath(t *testing.T) {
	store := NewRecentWorkspaceStore(filepath.Join(t.TempDir(), "recent.json"))
	first := filepath.Join(t.TempDir(), "first")
	second := filepath.Join(t.TempDir(), "second")

	mkdir(t, first)
	mkdir(t, second)

	if _, err := store.Add(first); err != nil {
		t.Fatalf("Add first failed: %v", err)
	}
	if _, err := store.Add(second); err != nil {
		t.Fatalf("Add second failed: %v", err)
	}

	items, err := store.Remove(first)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected one recent item, got %d", len(items))
	}
	if items[0].Path != second {
		t.Fatalf("expected second path to remain, got %s", items[0].Path)
	}
}

func TestRecentWorkspaceStoreClearsItems(t *testing.T) {
	store := NewRecentWorkspaceStore(filepath.Join(t.TempDir(), "recent.json"))
	workspace := filepath.Join(t.TempDir(), "workspace")

	mkdir(t, workspace)

	if _, err := store.Add(workspace); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	items, err := store.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected cleared items, got %d", len(items))
	}

	items, err = store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected persisted clear, got %d", len(items))
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
}
