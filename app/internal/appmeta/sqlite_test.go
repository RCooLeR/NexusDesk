package appmeta

import (
	"encoding/json"
	"os"
	"testing"
)

func TestEnsureWritesSQLiteSchemaManifest(t *testing.T) {
	status, err := Ensure(t.TempDir())
	if err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}
	if status.SchemaVersion != 1 || status.SchemaHash == "" {
		t.Fatalf("unexpected status: %#v", status)
	}
	if _, err := os.Stat(status.SchemaPath); err != nil {
		t.Fatalf("expected schema file: %v", err)
	}
	for _, table := range status.Tables {
		if !HasSchemaTable(SchemaSQL(), table) {
			t.Fatalf("schema missing table %s", table)
		}
	}
}

func TestMirrorAndInspectSQLiteMetadata(t *testing.T) {
	root := t.TempDir()
	metadata, _ := json.Marshal(map[string]string{"kind": "chat-answer"})
	status, err := Mirror(root, MirrorData{
		Chats: []ChatMirror{{
			Role:           "assistant",
			Content:        "answer",
			ContextRelPath: "README.md",
			SourcePaths:    []string{"README.md"},
			CreatedAt:      "2026-05-14T00:00:00Z",
		}},
		Approvals: []ApprovalMirror{{
			ID:        "approval-1",
			Action:    "artifact.create",
			Target:    ".nexusdesk/artifacts/report.md",
			Risk:      "medium",
			Decision:  "applied",
			CreatedAt: "2026-05-14T00:00:01Z",
		}},
		Artifacts: []ArtifactMirror{{
			RelPath:   ".nexusdesk/artifacts/report.md",
			Kind:      "chat-answer",
			Title:     "Report",
			Metadata:  metadata,
			CreatedAt: "2026-05-14T00:00:02Z",
		}},
		ToolRuns: []ToolRunMirror{{
			ID:        "tool-1",
			ToolName:  "dataset.query",
			Target:    "data.csv",
			Risk:      "low",
			Status:    "dry-run",
			Mode:      "dry-run",
			StartedAt: "2026-05-14T00:00:03Z",
		}},
	})
	if err != nil {
		t.Fatalf("Mirror returned error: %v", err)
	}
	if status.Message == "" {
		t.Fatalf("expected mirror status message")
	}

	browser, err := Inspect(root, []DatasetView{{Name: "data", RelPath: "data.csv", Engine: "duckdb view / csv fallback", Columns: []string{"id"}, Rows: 1}})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if len(browser.Tables) == 0 || len(browser.DatasetViews) != 1 {
		t.Fatalf("unexpected metadata browser: %#v", browser)
	}
	for _, table := range browser.Tables {
		if table.Name == "chats" && table.RowCount != 1 {
			t.Fatalf("expected mirrored chat row, got %#v", table)
		}
	}
}
