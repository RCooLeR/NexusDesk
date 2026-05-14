package agenttools

import (
	"testing"
	"time"
)

func TestAppendAndListRunRecords(t *testing.T) {
	root := t.TempDir()
	descriptor, err := RequireDescriptor("workspace.preview")
	if err != nil {
		t.Fatalf("RequireDescriptor returned error: %v", err)
	}
	record := NewRecord(RunRequest{
		ToolName: "workspace.preview",
		Target:   "README.md",
		Inputs:   map[string]string{"unused": ""},
	}, descriptor, "dry-run", "planned", testNow())
	record = FinishRecord(record, "dry-run", "Ready to preview README.md.", nil, testNow())

	if _, err := Append(root, record); err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	items, err := List(root)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 || items[0].ToolName != "workspace.preview" {
		t.Fatalf("unexpected records: %#v", items)
	}
}

func TestRequireDescriptorRejectsUnknownTool(t *testing.T) {
	if _, err := RequireDescriptor("missing.tool"); err == nil {
		t.Fatal("expected unknown descriptor to fail")
	}
}

func testNow() time.Time {
	return time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
}
