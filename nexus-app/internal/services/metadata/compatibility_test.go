package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestImportCompatibilityDataImportsWailsJSONStores(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	chatPath := filepath.Join(t.TempDir(), "chat-history.json")
	chatHistory := map[string][]compatibilityChatMessage{
		compatibilityWorkspaceHistoryKey(root): {
			{Role: "user", Content: "Analyze sales", ContextRelPath: "data/sales.csv", SourcePaths: []string{"tracker.md"}, CreatedAt: created},
			{Role: "assistant", Content: "Sales look healthy.", CreatedAt: created},
		},
		"other-workspace": {
			{Role: "user", Content: "Do not import", CreatedAt: created},
		},
	}
	writeJSON(t, chatPath, chatHistory)
	writeJSON(t, filepath.Join(root, ".nexusdesk", "approvals", "log.json"), []compatibilityApprovalRecord{
		{ID: "approval-1", Action: "file.write", Target: "README.md", Risk: "high", Decision: "applied", Message: "ok", CreatedAt: created},
	})
	writeJSON(t, filepath.Join(root, ".nexusdesk", "tool-runs", "log.json"), []compatibilityToolRunRecord{
		{ID: "tool-1", ToolName: "read_context", Target: "README.md", Risk: "low", Status: "completed", Mode: "execute", Inputs: map[string]string{"relPath": "README.md"}, OutputSummary: "Read README", StartedAt: created, CompletedAt: created},
	})
	artifactPath := filepath.Join(root, ".nexusdesk", "artifacts", "reports", "sales.md")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, []byte("# Sales\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(root, ".nexusdesk", "artifacts", "reports", "sales.meta.json"), compatibilityArtifactMetadata{
		Kind:           "dataset-summary",
		Title:          "Sales Summary",
		Source:         "data/sales.csv",
		SourcePaths:    []string{"data/sales.csv"},
		ContextRelPath: "data",
		CreatedAt:      created,
	})

	report, err := store.ImportCompatibilityData(CompatibilityImportOptions{ChatHistoryPath: chatPath})
	if err != nil {
		t.Fatalf("ImportCompatibilityData returned error: %v", err)
	}
	if report.Chats != 2 || report.Approvals != 1 || report.Artifacts != 1 || report.ToolRuns != 1 || report.Skipped != 0 {
		t.Fatalf("unexpected import report: %#v", report)
	}
	chats, err := store.SearchChatMessages("", 10)
	if err != nil {
		t.Fatal(err)
	}
	var userChat ChatMessageRecord
	for _, chat := range chats {
		if chat.Role == "user" {
			userChat = chat
		}
	}
	if len(chats) != 2 || userChat.Content != "Analyze sales" || userChat.ContextRelPath != "data/sales.csv" || !containsString(userChat.SourcePaths, "tracker.md") {
		t.Fatalf("unexpected imported chats: %#v", chats)
	}
	approvals, err := store.ListApprovalRecords(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(approvals) != 1 || approvals[0].ID != "approval-1" {
		t.Fatalf("unexpected imported approvals: %#v", approvals)
	}
	artifacts, err := store.ListArtifacts("sales", true, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 || artifacts[0].RelPath != ".nexusdesk/artifacts/reports/sales.md" || artifacts[0].MetadataPath != ".nexusdesk/artifacts/reports/sales.meta.json" {
		t.Fatalf("unexpected imported artifacts: %#v", artifacts)
	}
	agentRuns, err := store.ListAgentRuns(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(agentRuns) != 1 || agentRuns[0].ID != compatibilityAgentRunID {
		t.Fatalf("expected compatibility agent run, got %#v", agentRuns)
	}
	toolRuns, err := store.ListToolRuns(compatibilityAgentRunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(toolRuns) != 1 || toolRuns[0].ToolName != "read_context" || toolRuns[0].Args["relPath"] != "README.md" {
		t.Fatalf("unexpected imported tool runs: %#v", toolRuns)
	}
}

func TestImportCompatibilityDataIsIdempotent(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	approvalPath := filepath.Join(root, ".nexusdesk", "approvals", "log.json")
	writeJSON(t, approvalPath, []compatibilityApprovalRecord{
		{ID: "approval-1", Action: "file.write", Risk: "high", Decision: "applied", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)},
	})
	for i := 0; i < 2; i++ {
		if _, err := store.ImportCompatibilityData(CompatibilityImportOptions{}); err != nil {
			t.Fatalf("ImportCompatibilityData pass %d returned error: %v", i+1, err)
		}
	}
	approvals, err := store.ListApprovalRecords(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(approvals) != 1 {
		t.Fatalf("expected idempotent approval import, got %#v", approvals)
	}
}

func TestImportCompatibilityDataWritesCompletionStampAndSkipsSubsequentRuns(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	approvalPath := filepath.Join(root, ".nexusdesk", "approvals", "log.json")
	writeJSON(t, approvalPath, []compatibilityApprovalRecord{
		{ID: "approval-1", Action: "file.write", Risk: "high", Decision: "applied", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)},
	})
	first, err := store.ImportCompatibilityData(CompatibilityImportOptions{})
	if err != nil {
		t.Fatalf("first import returned error: %v", err)
	}
	if first.Approvals != 1 {
		t.Fatalf("expected one imported approval, got %#v", first)
	}
	if _, err := os.Stat(store.compatibilityImportStampPath()); err != nil {
		t.Fatalf("expected compatibility import stamp file to exist: %v", err)
	}
	second, err := store.ImportCompatibilityData(CompatibilityImportOptions{})
	if err != nil {
		t.Fatalf("second import returned error: %v", err)
	}
	if second.Approvals != 0 || !strings.Contains(strings.ToLower(second.Message), "already completed") {
		t.Fatalf("expected second import to be skipped, got %#v", second)
	}
	forced, err := store.ImportCompatibilityData(CompatibilityImportOptions{Force: true})
	if err != nil {
		t.Fatalf("forced import returned error: %v", err)
	}
	if forced.Approvals != 1 {
		t.Fatalf("expected forced import to run again, got %#v", forced)
	}
}

func TestCompatibilityImportPendingReflectsCompletionStamp(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	pending, err := store.CompatibilityImportPending()
	if err != nil {
		t.Fatalf("CompatibilityImportPending returned error: %v", err)
	}
	if !pending {
		t.Fatal("expected compatibility import to be pending before first import")
	}
	if _, err := store.ImportCompatibilityData(CompatibilityImportOptions{}); err != nil {
		t.Fatalf("ImportCompatibilityData returned error: %v", err)
	}
	pending, err = store.CompatibilityImportPending()
	if err != nil {
		t.Fatalf("CompatibilityImportPending returned error after import: %v", err)
	}
	if pending {
		t.Fatal("expected compatibility import to be complete after stamp is written")
	}
}

func TestCompatibilityImportPendingQuarantinesMalformedStamp(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	stampPath := store.compatibilityImportStampPath()
	if err := os.MkdirAll(filepath.Dir(stampPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stampPath, []byte("{ malformed-json "), 0o644); err != nil {
		t.Fatal(err)
	}
	pending, err := store.CompatibilityImportPending()
	if err != nil {
		t.Fatalf("CompatibilityImportPending returned error: %v", err)
	}
	if !pending {
		t.Fatal("expected malformed stamp to be treated as pending import")
	}
	if _, err := os.Stat(stampPath); !os.IsNotExist(err) {
		t.Fatalf("expected malformed stamp to be moved away, stat err=%v", err)
	}
	matches, err := filepath.Glob(stampPath + ".corrupt.*")
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected quarantined malformed stamp file")
	}
}

func TestImportCompatibilityDataContextReturnsCanceled(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.ImportCompatibilityDataContext(ctx, CompatibilityImportOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}

func TestImportCompatibilityDataMigratesLegacySQLiteDatasetTables(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	createLegacyDatasetSQLite(t, store.Path(), root)

	report, err := store.ImportCompatibilityData(CompatibilityImportOptions{})
	if err != nil {
		t.Fatalf("ImportCompatibilityData returned error: %v", err)
	}
	if report.SQLRuns != 1 || report.DatasetDependencies != 1 || report.Skipped != 0 {
		t.Fatalf("unexpected import report: %#v", report)
	}
	runs, err := store.ListSQLRuns(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].ID != "legacy-sql-1" || runs[0].Status != "success" || runs[0].ShownRows != 5 || runs[0].ArtifactPath != ".nexusdesk/artifacts/sql/sales.md" {
		t.Fatalf("unexpected imported SQL runs: %#v", runs)
	}
	dependencies, err := store.ListDatasetDependencies("data/sales.csv", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(dependencies) != 1 || dependencies[0].DependentKind != "sql-snippet" || dependencies[0].Relation != "saves" || dependencies[0].Metadata["query"] == "" {
		t.Fatalf("unexpected imported dependencies: %#v", dependencies)
	}
	db, err := sql.Open("sqlite", store.Path())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if exists, err := tableExists(db, "legacy_wails_sql_runs"); err != nil || !exists {
		t.Fatalf("expected legacy SQL backup table, exists=%v err=%v", exists, err)
	}
	if exists, err := tableExists(db, "legacy_wails_dataset_dependencies"); err != nil || !exists {
		t.Fatalf("expected legacy dependency backup table, exists=%v err=%v", exists, err)
	}
	if exists, err := tableExists(db, "legacy_wails_artifacts"); err != nil || !exists {
		t.Fatalf("expected legacy artifact backup table, exists=%v err=%v", exists, err)
	}
	if exists, err := tableExists(db, "legacy_wails_tool_runs"); err != nil || !exists {
		t.Fatalf("expected legacy tool-run backup table, exists=%v err=%v", exists, err)
	}
}

func TestFindCompatibilityArtifactFileRejectsMissingArtifact(t *testing.T) {
	sidecar := filepath.Join(t.TempDir(), "missing.meta.json")
	if err := os.WriteFile(sidecar, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := findCompatibilityArtifactFile(sidecar); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing artifact error, got %v", err)
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func createLegacyDatasetSQLite(t *testing.T, path string, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE sql_runs (
		id TEXT PRIMARY KEY,
		workspace_root TEXT NOT NULL,
		rel_path TEXT NOT NULL,
		sql_text TEXT NOT NULL,
		engine TEXT NOT NULL,
		rows_returned INTEGER,
		artifact TEXT,
		status TEXT NOT NULL,
		message TEXT,
		created_at TEXT NOT NULL
	);
	CREATE TABLE dataset_dependencies (
		id TEXT PRIMARY KEY,
		workspace_root TEXT NOT NULL,
		rel_path TEXT NOT NULL,
		kind TEXT NOT NULL,
		target TEXT,
		query TEXT,
		artifact TEXT,
		created_at TEXT NOT NULL,
		last_refresh TEXT
	);
	CREATE TABLE artifacts (
		id TEXT PRIMARY KEY,
		workspace_root TEXT NOT NULL,
		rel_path TEXT NOT NULL,
		kind TEXT NOT NULL,
		title TEXT,
		source TEXT,
		context_rel_path TEXT,
		metadata_json TEXT,
		created_at TEXT NOT NULL
	);
	CREATE TABLE tool_runs (
		id TEXT PRIMARY KEY,
		workspace_root TEXT NOT NULL,
		tool_name TEXT NOT NULL,
		target TEXT,
		risk TEXT NOT NULL,
		status TEXT NOT NULL,
		mode TEXT NOT NULL,
		approval_id TEXT,
		inputs_json TEXT,
		output_summary TEXT,
		error TEXT,
		started_at TEXT NOT NULL,
		completed_at TEXT,
		duration_ms INTEGER
	);`)
	if err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO sql_runs (id, workspace_root, rel_path, sql_text, engine, rows_returned, artifact, status, message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"legacy-sql-1", root, "data/sales.csv", "select * from dataset", "duckdb-compatible-dataset", 5, ".nexusdesk/artifacts/sql/sales.md", "completed", "ok", created); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO dataset_dependencies (id, workspace_root, rel_path, kind, target, query, artifact, created_at, last_refresh)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"legacy-dep-1", root, "data/sales.csv", "sql-snippet", "sales-query", "select * from dataset", "", created, created); err != nil {
		t.Fatal(err)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
