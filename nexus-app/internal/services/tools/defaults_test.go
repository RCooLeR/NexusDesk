package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nexusdesk/internal/services/agent"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

func TestDefaultDispatcherReadAndSearchTools(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Hello\n\nTODO: wire tools\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	request := agent.Request{WorkspaceRoot: root}

	read, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "read_file", Args: map[string]string{"relPath": "README.md"}}, request)
	if err != nil {
		t.Fatalf("read_file returned error: %v", err)
	}
	if !strings.Contains(read.Observation, "TODO: wire tools") {
		t.Fatalf("unexpected read observation:\n%s", read.Observation)
	}

	search, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "search_workspace", Args: map[string]string{"query": "wire tools"}}, request)
	if err != nil {
		t.Fatalf("search_workspace returned error: %v", err)
	}
	if !strings.Contains(search.Observation, "README.md") {
		t.Fatalf("unexpected search observation:\n%s", search.Observation)
	}
}

func TestDefaultDispatcherContextAndProblemsTools(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte("FIXME: check dispatcher\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	request := agent.Request{WorkspaceRoot: root}

	contextResult, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "read_context", Args: map[string]string{"relPath": "."}}, request)
	if err != nil {
		t.Fatalf("read_context returned error: %v", err)
	}
	if !strings.Contains(contextResult.Observation, "notes.md") {
		t.Fatalf("unexpected context observation:\n%s", contextResult.Observation)
	}

	problems, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "read_problems"}, request)
	if err != nil {
		t.Fatalf("read_problems returned error: %v", err)
	}
	if !strings.Contains(problems.Observation, "FIXME") {
		t.Fatalf("unexpected problems observation:\n%s", problems.Observation)
	}
}

func TestRunTaskRequiresApproval(t *testing.T) {
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "run_task", Args: map[string]string{"taskId": "go-test-root"}}, agent.Request{WorkspaceRoot: t.TempDir()})
	if err == nil || result.Risk != "high" || !strings.Contains(result.Observation, "approval") {
		t.Fatalf("expected approval error, got result=%#v err=%v", result, err)
	}
}

func TestDefaultDispatcherWriteToolsRequireApprovalAndCreateRollback(t *testing.T) {
	root := t.TempDir()
	workspace := workspaceSvc.New()
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspace})
	call := agent.ToolCall{Name: "write_file", Args: map[string]string{"relPath": "docs/report.md", "content": "# Report\n"}}

	blocked, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root})
	if err == nil || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected approval block, got result=%#v err=%v", blocked, err)
	}

	written, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("write_file returned error: %v", err)
	}
	if !written.Mutated || !strings.Contains(written.Observation, "Rollback:") || !strings.Contains(written.Observation, "docs/report.md") {
		t.Fatalf("unexpected write result: %#v", written)
	}
	data, err := os.ReadFile(filepath.Join(root, "docs", "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# Report\n" {
		t.Fatalf("unexpected written content: %q", data)
	}
	rollbacks, err := workspace.ListRollbacks(root)
	if err != nil {
		t.Fatalf("ListRollbacks returned error: %v", err)
	}
	if len(rollbacks) != 1 {
		t.Fatalf("expected rollback record, got %#v", rollbacks)
	}
}

func TestDefaultDispatcherAppendToolUsesSafeAppend(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "notes.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})

	appended, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "append_file", Args: map[string]string{"relPath": "docs/notes.txt", "content": "two\n"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("append_file returned error: %v", err)
	}
	if !appended.Mutated || !strings.Contains(appended.Observation, "Append applied") {
		t.Fatalf("unexpected append result: %#v", appended)
	}
	data, err := os.ReadFile(filepath.Join(root, "docs", "notes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "one\ntwo\n" {
		t.Fatalf("unexpected appended content: %q", data)
	}
}

func TestDefaultDispatcherFileOperationToolsRequireApprovalAndRollback(t *testing.T) {
	root := t.TempDir()
	workspace := workspaceSvc.New()
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspace})
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "source.txt"), []byte("source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	copyCall := agent.ToolCall{Name: "copy_file", Args: map[string]string{"sourceRelPath": "docs/source.txt", "targetRelPath": "docs/copy.txt"}}

	blocked, err := dispatcher.ExecuteTool(context.Background(), copyCall, agent.Request{WorkspaceRoot: root})
	if err == nil || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected approval block, got result=%#v err=%v", blocked, err)
	}
	copied, err := dispatcher.ExecuteTool(context.Background(), copyCall, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("copy_file returned error: %v", err)
	}
	if !copied.Mutated || !strings.Contains(copied.Observation, "Rollback:") || !strings.Contains(copied.Observation, "docs/copy.txt") {
		t.Fatalf("unexpected copy result: %#v", copied)
	}

	moved, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "move_file", Args: map[string]string{"sourceRelPath": "docs/copy.txt", "targetRelPath": "docs/moved.txt"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("move_file returned error: %v", err)
	}
	if !moved.Mutated || !strings.Contains(moved.Observation, "docs/moved.txt") {
		t.Fatalf("unexpected move result: %#v", moved)
	}

	deleted, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "delete_file", Args: map[string]string{"relPath": "docs/moved.txt"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("delete_file returned error: %v", err)
	}
	if !deleted.Mutated || !strings.Contains(deleted.Observation, "delete") {
		t.Fatalf("unexpected delete result: %#v", deleted)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "moved.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected moved file to be deleted, got err=%v", err)
	}
	rollbacks, err := workspace.ListRollbacks(root)
	if err != nil {
		t.Fatalf("ListRollbacks returned error: %v", err)
	}
	if len(rollbacks) != 3 {
		t.Fatalf("expected three rollback records, got %#v", rollbacks)
	}
}

func TestDefaultDispatcherApplyPatchTool(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	call := agent.ToolCall{Name: "apply_patch", Args: map[string]string{"patch": `--- a/notes.txt
+++ b/notes.txt
@@ -1,2 +1,2 @@
 one
-two
+TWO
`}}

	blocked, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root})
	if err == nil || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected approval block, got result=%#v err=%v", blocked, err)
	}
	applied, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("apply_patch returned error: %v", err)
	}
	if !applied.Mutated || !strings.Contains(applied.Observation, "Rollback:") || !strings.Contains(applied.Observation, "notes.txt") {
		t.Fatalf("unexpected patch result: %#v", applied)
	}
	data, err := os.ReadFile(filepath.Join(root, "notes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "one\nTWO\n" {
		t.Fatalf("unexpected patched content: %q", data)
	}
}

func TestDefaultDispatcherRollbackTools(t *testing.T) {
	root := t.TempDir()
	workspace := workspaceSvc.New()
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	applied, err := workspace.ApplyFileWrite(root, workspaceSvc.FileWriteRequest{RelPath: "notes.md", Content: "after\n"})
	if err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspace})
	request := agent.Request{WorkspaceRoot: root}

	listed, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "list_rollbacks"}, request)
	if err != nil {
		t.Fatalf("list_rollbacks returned error: %v", err)
	}
	if !strings.Contains(listed.Observation, applied.RollbackID) {
		t.Fatalf("rollback list missing id %q:\n%s", applied.RollbackID, listed.Observation)
	}

	blocked, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "rollback_file_mutation", Args: map[string]string{"id": applied.RollbackID}}, request)
	if err == nil || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected approval block, got result=%#v err=%v", blocked, err)
	}

	rolledBack, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "rollback_file_mutation", Args: map[string]string{"id": applied.RollbackID}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("rollback_file_mutation returned error: %v", err)
	}
	if !rolledBack.Mutated || !strings.Contains(rolledBack.Observation, "applied") {
		t.Fatalf("unexpected rollback result: %#v", rolledBack)
	}
	data, err := os.ReadFile(filepath.Join(root, "notes.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "before\n" {
		t.Fatalf("rollback did not restore file, got %q", data)
	}
}
