package metadata

import (
	"encoding/json"
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
	if len(chats) != 2 || userChat.Content != "Analyze sales" || !containsString(userChat.SourcePaths, "data/sales.csv") {
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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
