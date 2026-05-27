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
