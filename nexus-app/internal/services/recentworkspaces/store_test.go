package recentworkspaces

import (
	"path/filepath"
	"testing"
)

func TestStoreAddsNewestWorkspaceFirstAndDeduplicates(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "recent.json"))
	first := filepath.Join(t.TempDir(), "first")
	second := filepath.Join(t.TempDir(), "second")

	if _, err := store.Add(first); err != nil {
		t.Fatalf("Add first failed: %v", err)
	}
	items, err := store.Add(second)
	if err != nil {
		t.Fatalf("Add second failed: %v", err)
	}
	if len(items) != 2 || items[0].Path != second || items[1].Path != first {
		t.Fatalf("unexpected recent ordering: %#v", items)
	}
	items, err = store.Add(first)
	if err != nil {
		t.Fatalf("Add duplicate failed: %v", err)
	}
	if len(items) != 2 || items[0].Path != first || items[1].Path != second {
		t.Fatalf("expected duplicate to move to front: %#v", items)
	}
}

func TestStoreRemoveAndClear(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "recent.json"))
	first := filepath.Join(t.TempDir(), "first")
	second := filepath.Join(t.TempDir(), "second")
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
	if len(items) != 1 || items[0].Path != second {
		t.Fatalf("unexpected items after remove: %#v", items)
	}
	items, err = store.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected clear to remove all items: %#v", items)
	}
}

func TestStoreLimitsRecentWorkspaces(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "recent.json"))
	root := t.TempDir()
	for index := 0; index < Limit+3; index++ {
		if _, err := store.Add(filepath.Join(root, "workspace", string(rune('a'+index)))); err != nil {
			t.Fatalf("Add %d failed: %v", index, err)
		}
	}
	items, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != Limit {
		t.Fatalf("expected %d items, got %d", Limit, len(items))
	}
}
