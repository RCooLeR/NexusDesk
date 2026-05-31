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

func TestStoreMarksMissingRecentPathsAndRemovesThem(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "recent.json"))
	existing := t.TempDir()
	missing := filepath.Join(t.TempDir(), "missing-workspace")

	if _, err := store.Add(existing); err != nil {
		t.Fatalf("Add existing failed: %v", err)
	}
	if _, err := store.Add(missing); err != nil {
		t.Fatalf("Add missing failed: %v", err)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	byPath := map[string]Workspace{}
	for _, item := range items {
		byPath[item.Path] = item
	}
	if !byPath[existing].Exists || byPath[existing].Missing {
		t.Fatalf("expected existing workspace to be marked present, got %#v", byPath[existing])
	}
	if byPath[missing].Exists || !byPath[missing].Missing {
		t.Fatalf("expected missing workspace to be marked missing, got %#v", byPath[missing])
	}

	items, err = store.RemoveMissing()
	if err != nil {
		t.Fatalf("RemoveMissing failed: %v", err)
	}
	if len(items) != 1 || items[0].Path != existing || !items[0].Exists || items[0].Missing {
		t.Fatalf("expected only existing workspace after RemoveMissing, got %#v", items)
	}
}
