package approval

import "testing"

func TestAppendAndListApprovalRecords(t *testing.T) {
	root := t.TempDir()
	if _, err := Append(root, Record{Action: "write", Target: "notes.md"}); err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	items, err := List(root)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 approval record, got %d", len(items))
	}
	if items[0].Decision != "applied" || items[0].Risk != "medium" {
		t.Fatalf("unexpected approval defaults: %#v", items[0])
	}
}
