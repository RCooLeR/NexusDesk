package dataset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndListSavedQueries(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "leads.csv"), []byte("channel,revenue\nsearch,10\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	saved, err := SaveQuery(root, "leads.csv", "channel=search", "Search leads")
	if err != nil {
		t.Fatalf("SaveQuery returned error: %v", err)
	}
	if saved.Label != "Search leads" || saved.Query != "channel=search" {
		t.Fatalf("unexpected saved query: %#v", saved)
	}

	queries, err := ListSavedQueries(root, "leads.csv")
	if err != nil {
		t.Fatalf("ListSavedQueries returned error: %v", err)
	}
	if len(queries) != 1 || queries[0].Query != "channel=search" {
		t.Fatalf("unexpected saved queries: %#v", queries)
	}
}

func TestSaveQueryRejectsTraversal(t *testing.T) {
	if _, err := SaveQuery(t.TempDir(), "../leads.csv", "x", "bad"); err == nil {
		t.Fatal("expected traversal path to fail")
	}
}
