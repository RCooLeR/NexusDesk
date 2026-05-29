package tools

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexusdesk/internal/services/agent"
	approvalsSvc "nexusdesk/internal/services/approvals"
	artifactsSvc "nexusdesk/internal/services/artifacts"
	jobsSvc "nexusdesk/internal/services/jobs"
	workspaceSvc "nexusdesk/internal/services/workspace"

	_ "modernc.org/sqlite"
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

func TestDefaultDispatcherDatasetTools(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sales.csv"), []byte("channel,spend\nsearch,12\nsocial,8\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	request := agent.Request{WorkspaceRoot: root}

	profile, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "profile_dataset", Args: map[string]string{"relPath": "sales.csv"}}, request)
	if err != nil {
		t.Fatalf("profile_dataset returned error: %v", err)
	}
	if !strings.Contains(profile.Observation, "Dataset profile: sales.csv") || !strings.Contains(profile.Observation, "channel") {
		t.Fatalf("unexpected profile observation:\n%s", profile.Observation)
	}

	query, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "query_dataset", Args: map[string]string{"relPath": "sales.csv", "query": "channel=search"}}, request)
	if err != nil {
		t.Fatalf("query_dataset returned error: %v", err)
	}
	if !strings.Contains(query.Observation, "search") || !strings.Contains(query.Observation, "| channel | spend |") {
		t.Fatalf("unexpected query observation:\n%s", query.Observation)
	}

	blocked, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "query_dataset_sql", Args: map[string]string{"relPath": "sales.csv", "sql": "select channel, spend from dataset where channel = 'search'"}}, request)
	if err == nil || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected medium-risk SQL approval, got result=%#v err=%v", blocked, err)
	}

	sql, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "query_dataset_sql", Args: map[string]string{"relPath": "sales.csv", "sql": "select channel, spend from dataset where channel = 'search'"}}, agent.Request{
		WorkspaceRoot: root,
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
			return request.Name == "query_dataset_sql" && request.Risk == "medium"
		},
	})
	if err != nil {
		t.Fatalf("query_dataset_sql returned error: %v", err)
	}
	if !strings.Contains(sql.Observation, "Dataset SQL result: sales.csv") || !strings.Contains(sql.Observation, "Validate SELECT-only") {
		t.Fatalf("unexpected SQL observation:\n%s", sql.Observation)
	}
}

func TestDefaultDispatcherCreateDatasetChartTool(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sales.csv"), []byte("channel,spend\nsearch,12\nsocial,8\nsearch,18\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	call := agent.ToolCall{Name: "create_dataset_chart", Args: map[string]string{"relPath": "sales.csv", "query": "order by spend desc"}}

	blocked, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root})
	if err == nil || blocked.Risk != "high" || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected create_dataset_chart approval block, got result=%#v err=%v", blocked, err)
	}

	generated, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("create_dataset_chart returned error: %v", err)
	}
	if !generated.Mutated || !strings.Contains(generated.Observation, "Generated dataset chart artifact") || !strings.Contains(generated.Observation, "sales.csv") || !strings.Contains(generated.Observation, "Points:") {
		t.Fatalf("unexpected chart result: %#v", generated)
	}
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	matches, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: "kind:chart"})
	if err != nil {
		t.Fatalf("ListArtifacts returned error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one chart artifact, got %d", len(matches))
	}
	text, err := store.ReadArtifactText(matches[0].RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText returned error: %v", err)
	}
	for _, expected := range []string{"<svg", "search", "social"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected chart artifact text to contain %q:\n%s", expected, text)
		}
	}
	metadata, err := store.ReadArtifactMetadata(matches[0].RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactMetadata returned error: %v", err)
	}
	if metadata.Kind != "chart" || len(metadata.SourcePaths) != 1 || metadata.SourcePaths[0] != "sales.csv" {
		t.Fatalf("unexpected chart metadata: %#v", metadata)
	}
}

func TestDefaultDispatcherJobTools(t *testing.T) {
	jobService := jobsSvc.New()
	running, _ := jobService.Start("dataset-query", "Import with api_key=super-secret")
	jobService.AppendLog(running.ID, "Authorization: Bearer secret-token")
	jobService.AppendLog(running.ID, "processed safe rows")
	finished, _ := jobService.Start("task", "completed task")
	jobService.Finish(finished.ID, jobsSvc.StatusSuccess, "done", nil)
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New(), Jobs: jobService})

	listed, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "list_jobs", Args: map[string]string{"status": "running", "limit": "10"}}, agent.Request{})
	if err != nil {
		t.Fatalf("list_jobs returned error: %v", err)
	}
	if !strings.Contains(listed.Observation, running.ID) || strings.Contains(listed.Observation, finished.ID) {
		t.Fatalf("unexpected filtered job list:\n%s", listed.Observation)
	}
	if strings.Contains(listed.Observation, "super-secret") || !strings.Contains(listed.Observation, "api_key=[redacted]") {
		t.Fatalf("job list did not redact sensitive label:\n%s", listed.Observation)
	}

	logs, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "read_job_logs", Args: map[string]string{"jobId": running.ID, "tailLines": "2"}}, agent.Request{})
	if err != nil {
		t.Fatalf("read_job_logs returned error: %v", err)
	}
	if !strings.Contains(logs.Observation, "processed safe rows") || !strings.Contains(logs.Observation, "Authorization: [redacted]") || strings.Contains(logs.Observation, "secret-token") {
		t.Fatalf("unexpected redacted job logs:\n%s", logs.Observation)
	}

	blocked, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "cancel_job", Args: map[string]string{"jobId": running.ID}}, agent.Request{})
	if err == nil || blocked.Risk != "high" || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected cancel_job approval block, got result=%#v err=%v", blocked, err)
	}

	canceled, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "cancel_job", Args: map[string]string{"jobId": running.ID}}, agent.Request{
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
			return request.Name == "cancel_job" && request.Risk == "high"
		},
	})
	if err != nil {
		t.Fatalf("cancel_job returned error: %v", err)
	}
	if !canceled.Mutated || !strings.Contains(canceled.Observation, "Cancel requested") {
		t.Fatalf("unexpected cancel result: %#v", canceled)
	}
	got, ok := jobService.Get(running.ID)
	if !ok || got.Status != jobsSvc.StatusCanceled {
		t.Fatalf("expected job to be canceled, got ok=%t job=%#v", ok, got)
	}
}

func TestDefaultDispatcherSecurityTools(t *testing.T) {
	root := t.TempDir()
	approvalService := approvalsSvc.New()
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New(), Approvals: approvalService})

	redacted, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "redact_text", Args: map[string]string{
		"content": `Authorization: Bearer secret-token api_key=super-key {"access_token":"json-secret"} password: "db-secret"`,
	}}, agent.Request{WorkspaceRoot: root})
	if err != nil {
		t.Fatalf("redact_text returned error: %v", err)
	}
	for _, secret := range []string{"secret-token", "super-key", "json-secret", "db-secret"} {
		if strings.Contains(redacted.Observation, secret) {
			t.Fatalf("redact_text leaked %q:\n%s", secret, redacted.Observation)
		}
	}
	if !strings.Contains(redacted.Observation, "[redacted]") {
		t.Fatalf("redact_text did not include redaction marker:\n%s", redacted.Observation)
	}

	call := agent.ToolCall{Name: "request_approval", Args: map[string]string{
		"action":  "delete_file",
		"risk":    "high",
		"target":  "secrets.env",
		"summary": "Delete file after checking token=delete-token",
	}}
	blocked, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root})
	if err == nil || blocked.Risk != "medium" || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected request_approval per-call approval block, got result=%#v err=%v", blocked, err)
	}
	requested, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{
		WorkspaceRoot: root,
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
			return request.Name == "request_approval" && request.Risk == "medium"
		},
	})
	if err != nil {
		t.Fatalf("request_approval returned error: %v", err)
	}
	if !requested.Mutated || !strings.Contains(requested.Observation, "Approval request recorded") || strings.Contains(requested.Observation, "delete-token") {
		t.Fatalf("unexpected request_approval observation:\n%s", requested.Observation)
	}

	listed, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "list_approvals"}, agent.Request{WorkspaceRoot: root})
	if err != nil {
		t.Fatalf("list_approvals returned error: %v", err)
	}
	if !strings.Contains(listed.Observation, "tool.approval.request.delete_file") || !strings.Contains(listed.Observation, "decision=requested") || strings.Contains(listed.Observation, "delete-token") {
		t.Fatalf("unexpected list_approvals observation:\n%s", listed.Observation)
	}
}

func TestDefaultDispatcherSQLiteTools(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "data", "store.sqlite")
	writeToolSQLiteFixture(t, dbPath)

	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	request := agent.Request{WorkspaceRoot: root}
	relPath := filepath.Join("data", "store.sqlite")

	blocked, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "inspect_sqlite", Args: map[string]string{"relPath": relPath}}, request)
	if err == nil || blocked.Risk != "medium" || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected inspect_sqlite approval block, got result=%#v err=%v", blocked, err)
	}

	approvedRequest := agent.Request{
		WorkspaceRoot: root,
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
			return request.Risk == "medium"
		},
	}
	metadata, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "inspect_sqlite", Args: map[string]string{"relPath": relPath}}, approvedRequest)
	if err != nil {
		t.Fatalf("inspect_sqlite returned error: %v", err)
	}
	for _, expected := range []string{"SQLite metadata: data/store.sqlite", "orders", "customers", "Relationships:", "sample:"} {
		if !strings.Contains(metadata.Observation, expected) {
			t.Fatalf("expected metadata observation to contain %q:\n%s", expected, metadata.Observation)
		}
	}

	query, err := dispatcher.ExecuteTool(
		context.Background(),
		agent.ToolCall{Name: "query_sqlite", Args: map[string]string{"relPath": relPath, "sql": "select id, total from orders order by id", "limit": "1"}},
		approvedRequest,
	)
	if err != nil {
		t.Fatalf("query_sqlite returned error: %v", err)
	}
	if !strings.Contains(query.Observation, "SQLite query result: data/store.sqlite") || !strings.Contains(query.Observation, "| id | total |") || !strings.Contains(query.Observation, "42.5") || !strings.Contains(query.Observation, "truncated=true") {
		t.Fatalf("unexpected SQLite query observation:\n%s", query.Observation)
	}

	mutation, err := dispatcher.ExecuteTool(
		context.Background(),
		agent.ToolCall{Name: "query_sqlite", Args: map[string]string{"relPath": relPath, "sql": "delete from orders"}},
		approvedRequest,
	)
	if err == nil || !strings.Contains(mutation.Observation, "read-only SELECT") {
		t.Fatalf("expected mutating SQLite SQL to be rejected, got result=%#v err=%v", mutation, err)
	}
}

func TestDefaultDispatcherDocumentAndOperationsTools(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "guide.md"), []byte("# Guide\n\nUseful notes for operations.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compose := "services:\n  web:\n    image: nginx\n    ports:\n      - \"8080:80\"\n"
	if err := os.WriteFile(filepath.Join(root, "docker-compose.yml"), []byte(compose), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	request := agent.Request{WorkspaceRoot: root}

	document, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "extract_document", Args: map[string]string{"relPath": "guide.md"}}, request)
	if err != nil {
		t.Fatalf("extract_document returned error: %v", err)
	}
	if !strings.Contains(document.Observation, "Document extract: guide.md") || !strings.Contains(document.Observation, "Useful notes") {
		t.Fatalf("unexpected document observation:\n%s", document.Observation)
	}

	scan, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "inspect_operations_files"}, request)
	if err != nil {
		t.Fatalf("inspect_operations_files scan returned error: %v", err)
	}
	if !strings.Contains(scan.Observation, "operations files found") || !strings.Contains(scan.Observation, "docker-compose.yml") {
		t.Fatalf("unexpected operations scan observation:\n%s", scan.Observation)
	}

	inspection, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "inspect_operations_files", Args: map[string]string{"relPath": "docker-compose.yml"}}, request)
	if err != nil {
		t.Fatalf("inspect_operations_files inspect returned error: %v", err)
	}
	if !strings.Contains(inspection.Observation, "Operations Inspection") || !strings.Contains(inspection.Observation, "nginx") {
		t.Fatalf("unexpected operations inspection observation:\n%s", inspection.Observation)
	}
}

func TestDefaultDispatcherGenerateRunbookTool(t *testing.T) {
	root := t.TempDir()
	compose := `services:
  api:
    image: example/api:latest
    ports:
      - "8080:80"
    environment:
      API_KEY: super-secret-token
`
	if err := os.WriteFile(filepath.Join(root, "compose.yml"), []byte(compose), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	call := agent.ToolCall{Name: "generate_runbook", Args: map[string]string{"relPath": "compose.yml"}}

	blocked, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root})
	if err == nil || blocked.Risk != "high" || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected generate_runbook approval block, got result=%#v err=%v", blocked, err)
	}

	generated, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("generate_runbook returned error: %v", err)
	}
	if !generated.Mutated || !strings.Contains(generated.Observation, "Generated operations runbook artifact") || !strings.Contains(generated.Observation, "compose.yml") {
		t.Fatalf("unexpected runbook result: %#v", generated)
	}
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	matches, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: "kind:operations-runbook"})
	if err != nil {
		t.Fatalf("ListArtifacts returned error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one operations runbook artifact, got %d", len(matches))
	}
	text, err := store.ReadArtifactText(matches[0].RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText returned error: %v", err)
	}
	for _, expected := range []string{"Operations Runbook", "api", "8080:80", "[REDACTED]"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected runbook text to contain %q:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "super-secret-token") {
		t.Fatalf("runbook leaked unredacted secret:\n%s", text)
	}
}

func writeToolSQLiteFixture(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create sqlite fixture dir: %v", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite fixture: %v", err)
	}
	defer db.Close()
	schema := `
create table customers (
	id integer primary key,
	name text not null
);
create table orders (
	id integer primary key,
	customer_id integer not null references customers(id),
	total real not null
);
create index idx_orders_customer_id on orders(customer_id);
insert into customers(id, name) values (1, 'Ada'), (2, 'Linus');
insert into orders(id, customer_id, total) values (10, 1, 42.5), (11, 2, 7.25);
create view order_totals as select c.name, sum(o.total) as total from customers c join orders o on o.customer_id = c.id group by c.name;
`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("seed sqlite fixture: %v", err)
	}
}

func TestDefaultDispatcherListsExternalAgentTools(t *testing.T) {
	dispatcher := NewDefaultDispatcher(Dependencies{
		ExternalAgentLookupPath: func(command string) (string, error) {
			if command == "codex" {
				return "/usr/local/bin/codex", nil
			}
			return "", errors.New("missing")
		},
	})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "list_external_agent_tools"}, agent.Request{})
	if err != nil {
		t.Fatalf("list_external_agent_tools returned error: %v", err)
	}
	if !strings.Contains(result.Observation, "Codex CLI: available") {
		t.Fatalf("expected Codex CLI availability, got:\n%s", result.Observation)
	}
	if !strings.Contains(result.Observation, "Execution must be routed through an approved job/shell integration") {
		t.Fatalf("expected execution policy, got:\n%s", result.Observation)
	}
}

func TestDefaultDispatcherPlansExternalAgentRun(t *testing.T) {
	dispatcher := NewDefaultDispatcher(Dependencies{
		ExternalAgentLookupPath: func(command string) (string, error) {
			if command == "opencode" {
				return "/usr/local/bin/opencode", nil
			}
			return "", errors.New("missing")
		},
	})
	result, err := dispatcher.ExecuteTool(
		context.Background(),
		agent.ToolCall{Name: "plan_external_agent_run", Args: map[string]string{"toolID": "opencode", "prompt": "inspect the branch"}},
		agent.Request{WorkspaceRoot: "/work/project"},
	)
	if err != nil {
		t.Fatalf("plan_external_agent_run returned error: %v", err)
	}
	if !strings.Contains(result.Observation, "External agent plan: OpenCode") {
		t.Fatalf("expected OpenCode plan, got:\n%s", result.Observation)
	}
	if !strings.Contains(result.Observation, "Requires approval: true") {
		t.Fatalf("expected approval requirement, got:\n%s", result.Observation)
	}
}

func TestDefaultDispatcherRunsApprovedTerminalCommand(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go executable is not available")
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	result, err := dispatcher.ExecuteTool(
		context.Background(),
		agent.ToolCall{Name: "run_terminal_command", Args: map[string]string{"command": "go", "argsJson": `["version"]`}},
		agent.Request{
			WorkspaceRoot: t.TempDir(),
			ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
				return request.Name == "run_terminal_command" && request.Risk == "high"
			},
		},
	)
	if err != nil {
		t.Fatalf("run_terminal_command returned error: %v", err)
	}
	if !strings.Contains(result.Observation, "go version") {
		t.Fatalf("expected go version output, got:\n%s", result.Observation)
	}
	if !result.Mutated {
		t.Fatal("expected terminal command tool to be recorded as mutating-capable")
	}
}

func TestDefaultDispatcherBlocksTerminalCommandWithoutApproval(t *testing.T) {
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	result, err := dispatcher.ExecuteTool(
		context.Background(),
		agent.ToolCall{Name: "run_terminal_command", Args: map[string]string{"command": "go", "argsJson": `["version"]`}},
		agent.Request{WorkspaceRoot: t.TempDir()},
	)
	if err == nil || !strings.Contains(result.Error, "per-call approval") {
		t.Fatalf("expected per-call approval denial, result=%#v err=%v", result, err)
	}
}

func TestDefaultDispatcherGitHistoryAndBlameTools(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	runToolTestGit(t, root, "init")
	runToolTestGit(t, root, "config", "user.email", "test@example.com")
	runToolTestGit(t, root, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("line one\nline two\n"), 0o644); err != nil {
		t.Fatalf("write notes file: %v", err)
	}
	runToolTestGit(t, root, "add", "notes.txt")
	runToolTestGit(t, root, "commit", "-m", "initial notes")

	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	request := agent.Request{WorkspaceRoot: root}
	history, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "read_git_history", Args: map[string]string{"relPath": "notes.txt"}}, request)
	if err != nil {
		t.Fatalf("read_git_history returned error: %v", err)
	}
	if !strings.Contains(history.Observation, "initial notes") || !strings.Contains(history.Observation, "History target: notes.txt") {
		t.Fatalf("unexpected history observation:\n%s", history.Observation)
	}

	blame, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "read_git_blame", Args: map[string]string{"relPath": "notes.txt", "startLine": "2", "endLine": "2"}}, request)
	if err != nil {
		t.Fatalf("read_git_blame returned error: %v", err)
	}
	if !strings.Contains(blame.Observation, "line two") || !strings.Contains(blame.Observation, "Requested lines: 2-2") {
		t.Fatalf("unexpected blame observation:\n%s", blame.Observation)
	}
}

func TestDefaultDispatcherGitIndexMutationTools(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	runToolTestGit(t, root, "init")
	runToolTestGit(t, root, "config", "user.email", "test@example.com")
	runToolTestGit(t, root, "config", "user.name", "Test User")
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("line one\nline two\nline three\n"), 0o644); err != nil {
		t.Fatalf("write notes file: %v", err)
	}
	runToolTestGit(t, root, "add", "notes.txt")
	runToolTestGit(t, root, "commit", "-m", "initial notes")
	if err := os.WriteFile(path, []byte("line one\nline two edited\nline three\n"), 0o644); err != nil {
		t.Fatalf("modify notes file: %v", err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})

	blocked, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "stage_file", Args: map[string]string{"relPath": "notes.txt"}}, agent.Request{WorkspaceRoot: root})
	if err == nil || blocked.Risk != "high" || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected stage_file approval block, got result=%#v err=%v", blocked, err)
	}
	notRepo, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "stage_file", Args: map[string]string{"relPath": "notes.txt"}}, agent.Request{WorkspaceRoot: t.TempDir(), ApproveWrites: true})
	if err == nil || notRepo.Mutated || !strings.Contains(notRepo.Observation, "Git repository") {
		t.Fatalf("expected stage_file to reject non-repo workspace without mutation, got result=%#v err=%v", notRepo, err)
	}

	staged, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "stage_file", Args: map[string]string{"relPath": "notes.txt"}}, agent.Request{
		WorkspaceRoot: root,
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
			return request.Name == "stage_file" && request.Risk == "high"
		},
	})
	if err != nil {
		t.Fatalf("stage_file returned error: %v", err)
	}
	if !staged.Mutated || !strings.Contains(staged.Observation, "Staged notes.txt.") || !strings.Contains(staged.Observation, "staged=1 unstaged=0") {
		t.Fatalf("unexpected stage_file observation:\n%s", staged.Observation)
	}

	unstaged, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "unstage_file", Args: map[string]string{"relPath": "notes.txt"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("unstage_file returned error: %v", err)
	}
	if !strings.Contains(unstaged.Observation, "Unstaged notes.txt.") || !strings.Contains(unstaged.Observation, "staged=0 unstaged=1") {
		t.Fatalf("unexpected unstage_file observation:\n%s", unstaged.Observation)
	}

	hunkStaged, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "stage_hunk", Args: map[string]string{"relPath": "notes.txt", "hunkId": "1"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("stage_hunk returned error: %v", err)
	}
	if !strings.Contains(hunkStaged.Observation, "Staged hunk 1 in notes.txt.") || !strings.Contains(hunkStaged.Observation, "Diff kind: unstaged") {
		t.Fatalf("unexpected stage_hunk observation:\n%s", hunkStaged.Observation)
	}

	hunkUnstaged, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "unstage_hunk", Args: map[string]string{"relPath": "notes.txt", "hunkIndex": "0"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("unstage_hunk returned error: %v", err)
	}
	if !strings.Contains(hunkUnstaged.Observation, "Unstaged hunk 1 in notes.txt.") || !strings.Contains(hunkUnstaged.Observation, "Diff kind: staged") {
		t.Fatalf("unexpected unstage_hunk observation:\n%s", hunkUnstaged.Observation)
	}
}

func TestDefaultDispatcherCommitChangesTool(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	runToolTestGit(t, root, "init")
	runToolTestGit(t, root, "config", "user.email", "test@example.com")
	runToolTestGit(t, root, "config", "user.name", "Test User")
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("line one\n"), 0o644); err != nil {
		t.Fatalf("write notes file: %v", err)
	}
	runToolTestGit(t, root, "add", "notes.txt")
	runToolTestGit(t, root, "commit", "-m", "initial notes")
	if err := os.WriteFile(path, []byte("line one\nline two\n"), 0o644); err != nil {
		t.Fatalf("modify notes file: %v", err)
	}
	runToolTestGit(t, root, "add", "notes.txt")
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})

	blocked, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "commit_changes", Args: map[string]string{"message": "Add line two"}}, agent.Request{WorkspaceRoot: root})
	if err == nil || blocked.Risk != "high" || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected commit_changes approval block, got result=%#v err=%v", blocked, err)
	}

	committed, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "commit_changes", Args: map[string]string{"message": "Add line two", "body": "Body text"}}, agent.Request{
		WorkspaceRoot: root,
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
			return request.Name == "commit_changes" && request.Risk == "high"
		},
	})
	if err != nil {
		t.Fatalf("commit_changes returned error: %v", err)
	}
	if !committed.Mutated || !strings.Contains(committed.Observation, "Committed staged changes.") || !strings.Contains(committed.Observation, "Subject: Add line two") || !strings.Contains(committed.Observation, "notes.txt") {
		t.Fatalf("unexpected commit observation:\n%s", committed.Observation)
	}
	log := runToolTestGitOutput(t, root, "log", "-1", "--pretty=%s%n%b")
	if !strings.Contains(log, "Add line two") || !strings.Contains(log, "Body text") {
		t.Fatalf("commit message/body were not persisted:\n%s", log)
	}
}

func TestDefaultDispatcherCreateBranchTool(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	runToolTestGit(t, root, "init")
	runToolTestGit(t, root, "config", "user.email", "test@example.com")
	runToolTestGit(t, root, "config", "user.name", "Test User")
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("line one\n"), 0o644); err != nil {
		t.Fatalf("write notes file: %v", err)
	}
	runToolTestGit(t, root, "add", "notes.txt")
	runToolTestGit(t, root, "commit", "-m", "initial notes")
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})

	blocked, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "create_branch", Args: map[string]string{"branchName": "feature/native-branch"}}, agent.Request{WorkspaceRoot: root})
	if err == nil || blocked.Risk != "high" || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected create_branch approval block, got result=%#v err=%v", blocked, err)
	}

	created, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "create_branch", Args: map[string]string{"branchName": "feature/native-branch"}}, agent.Request{
		WorkspaceRoot: root,
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
			return request.Name == "create_branch" && request.Risk == "high"
		},
	})
	if err != nil {
		t.Fatalf("create_branch returned error: %v", err)
	}
	if !created.Mutated || !strings.Contains(created.Observation, "Created branch feature/native-branch.") || !strings.Contains(created.Observation, "Checked out: false") {
		t.Fatalf("unexpected create_branch observation:\n%s", created.Observation)
	}
	if out := runToolTestGitOutput(t, root, "branch", "--list", "feature/native-branch"); !strings.Contains(out, "feature/native-branch") {
		t.Fatalf("created branch was not listed: %q", out)
	}

	duplicate, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "create_branch", Args: map[string]string{"branchName": "feature/native-branch"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err == nil || duplicate.Mutated || !strings.Contains(duplicate.Observation, "already exists") {
		t.Fatalf("expected duplicate branch rejection, got result=%#v err=%v", duplicate, err)
	}

	checkedOut, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "create_branch", Args: map[string]string{"branchName": "feature/checked-out", "checkout": "true"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("create_branch checkout returned error: %v", err)
	}
	if !strings.Contains(checkedOut.Observation, "Created and switched to branch feature/checked-out.") || !strings.Contains(checkedOut.Observation, "Checked out: true") {
		t.Fatalf("unexpected checked-out branch observation:\n%s", checkedOut.Observation)
	}
	if branch := strings.TrimSpace(runToolTestGitOutput(t, root, "branch", "--show-current")); branch != "feature/checked-out" {
		t.Fatalf("expected checked-out branch, got %q", branch)
	}
}

func TestDefaultDispatcherRevertChangesTool(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	runToolTestGit(t, root, "init")
	runToolTestGit(t, root, "config", "user.email", "test@example.com")
	runToolTestGit(t, root, "config", "user.name", "Test User")
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("original\n"), 0o644); err != nil {
		t.Fatalf("write notes file: %v", err)
	}
	runToolTestGit(t, root, "add", "notes.txt")
	runToolTestGit(t, root, "commit", "-m", "initial notes")
	if err := os.WriteFile(path, []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("modify notes file: %v", err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	call := agent.ToolCall{Name: "revert_changes", Args: map[string]string{"relPath": "notes.txt"}}

	blocked, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root})
	if err == nil || blocked.Risk != "high" || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected revert_changes approval block, got result=%#v err=%v", blocked, err)
	}
	reverted, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("revert_changes returned error: %v", err)
	}
	if !reverted.Mutated || !strings.Contains(reverted.Observation, "Prepared to restore notes.txt") || !strings.Contains(reverted.Observation, "Rollback:") {
		t.Fatalf("unexpected revert observation:\n%s", reverted.Observation)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original\n" {
		t.Fatalf("expected file to be restored, got %q", data)
	}
	workspace := workspaceSvc.New()
	rollbacks, err := workspace.ListRollbacks(root)
	if err != nil {
		t.Fatalf("ListRollbacks returned error: %v", err)
	}
	if len(rollbacks) != 1 {
		t.Fatalf("expected rollback record, got %#v", rollbacks)
	}

	untrackedPath := filepath.Join(root, "scratch.txt")
	if err := os.WriteFile(untrackedPath, []byte("draft\n"), 0o644); err != nil {
		t.Fatalf("write untracked file: %v", err)
	}
	untrackedBlocked, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "revert_changes", Args: map[string]string{"relPath": "scratch.txt"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err == nil || untrackedBlocked.Mutated || !strings.Contains(untrackedBlocked.Observation, "scope=untracked") {
		t.Fatalf("expected untracked scope rejection, got result=%#v err=%v", untrackedBlocked, err)
	}
	deleted, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "revert_changes", Args: map[string]string{"relPath": "scratch.txt", "scope": "untracked"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("revert_changes untracked returned error: %v", err)
	}
	if !deleted.Mutated || !strings.Contains(deleted.Observation, "delete untracked file scratch.txt") {
		t.Fatalf("unexpected untracked revert observation:\n%s", deleted.Observation)
	}
	if _, err := os.Stat(untrackedPath); !os.IsNotExist(err) {
		t.Fatalf("expected untracked file to be deleted, got err=%v", err)
	}
}

func TestDefaultDispatcherRevertStagedChangesTool(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	runToolTestGit(t, root, "init")
	runToolTestGit(t, root, "config", "user.email", "test@example.com")
	runToolTestGit(t, root, "config", "user.name", "Test User")
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("original\n"), 0o644); err != nil {
		t.Fatalf("write notes file: %v", err)
	}
	runToolTestGit(t, root, "add", "notes.txt")
	runToolTestGit(t, root, "commit", "-m", "initial notes")
	if err := os.WriteFile(path, []byte("staged\n"), 0o644); err != nil {
		t.Fatalf("modify notes file: %v", err)
	}
	runToolTestGit(t, root, "add", "notes.txt")
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	call := agent.ToolCall{Name: "revert_staged_changes", Args: map[string]string{"relPath": "notes.txt", "scope": "staged"}}

	blocked, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root})
	if err == nil || blocked.Risk != "high" || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected revert_staged_changes approval block, got result=%#v err=%v", blocked, err)
	}
	missingScope, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "revert_staged_changes", Args: map[string]string{"relPath": "notes.txt"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err == nil || missingScope.Mutated || !strings.Contains(missingScope.Observation, "scope=staged") {
		t.Fatalf("expected staged scope rejection, got result=%#v err=%v", missingScope, err)
	}
	reverted, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("revert_staged_changes returned error: %v", err)
	}
	if !reverted.Mutated || !strings.Contains(reverted.Observation, "Prepared to unstage and restore notes.txt") || !strings.Contains(reverted.Observation, "Discarded staged diff preview") || !strings.Contains(reverted.Observation, "Rollback:") {
		t.Fatalf("unexpected staged revert observation:\n%s", reverted.Observation)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original\n" {
		t.Fatalf("expected staged file to be restored, got %q", data)
	}
	if status := runToolTestGitOutput(t, root, "status", "--porcelain", "--", "notes.txt"); strings.TrimSpace(status) != "" {
		t.Fatalf("expected notes.txt to have no Git changes after staged revert, got %q", status)
	}

	addedPath := filepath.Join(root, "scratch.txt")
	if err := os.WriteFile(addedPath, []byte("draft\n"), 0o644); err != nil {
		t.Fatalf("write added file: %v", err)
	}
	runToolTestGit(t, root, "add", "scratch.txt")
	deleted, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "revert_staged_changes", Args: map[string]string{"relPath": "scratch.txt", "scope": "staged"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("revert_staged_changes staged add returned error: %v", err)
	}
	if !deleted.Mutated || !strings.Contains(deleted.Observation, "staged added file scratch.txt") {
		t.Fatalf("unexpected staged add revert observation:\n%s", deleted.Observation)
	}
	if _, err := os.Stat(addedPath); !os.IsNotExist(err) {
		t.Fatalf("expected staged added file to be deleted, got err=%v", err)
	}
}

func TestDefaultDispatcherRevertStagedChangesRejectsMixedEdits(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	runToolTestGit(t, root, "init")
	runToolTestGit(t, root, "config", "user.email", "test@example.com")
	runToolTestGit(t, root, "config", "user.name", "Test User")
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("original\n"), 0o644); err != nil {
		t.Fatalf("write notes file: %v", err)
	}
	runToolTestGit(t, root, "add", "notes.txt")
	runToolTestGit(t, root, "commit", "-m", "initial notes")
	if err := os.WriteFile(path, []byte("staged\n"), 0o644); err != nil {
		t.Fatalf("modify notes file: %v", err)
	}
	runToolTestGit(t, root, "add", "notes.txt")
	if err := os.WriteFile(path, []byte("unstaged too\n"), 0o644); err != nil {
		t.Fatalf("modify notes file again: %v", err)
	}

	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "revert_staged_changes", Args: map[string]string{"relPath": "notes.txt", "scope": "staged"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err == nil || result.Mutated || !strings.Contains(result.Observation, "unstaged edits") {
		t.Fatalf("expected mixed-edit rejection, got result=%#v err=%v", result, err)
	}
	if data, readErr := os.ReadFile(path); readErr != nil || string(data) != "unstaged too\n" {
		t.Fatalf("mixed-edit rejection changed worktree, data=%q err=%v", data, readErr)
	}
}

func TestDefaultDispatcherFormatFileTool(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "settings.json"), []byte(`{"name":"nexus","enabled":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	workspace := workspaceSvc.New()
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspace})
	call := agent.ToolCall{Name: "format_file", Args: map[string]string{"relPath": "settings.json"}}

	blocked, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root})
	if err == nil || blocked.Risk != "high" || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected format_file approval block, got result=%#v err=%v", blocked, err)
	}
	formatted, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("format_file returned error: %v", err)
	}
	if !formatted.Mutated || !strings.Contains(formatted.Observation, "Formatted workspace file.") || !strings.Contains(formatted.Observation, "Rollback:") || !strings.Contains(formatted.Observation, "Diff:") {
		t.Fatalf("unexpected format_file observation:\n%s", formatted.Observation)
	}
	data, err := os.ReadFile(filepath.Join(root, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"name\": \"nexus\",\n  \"enabled\": true\n}\n"
	if string(data) != want {
		t.Fatalf("expected formatted JSON, got %q", data)
	}
	rollbacks, err := workspace.ListRollbacks(root)
	if err != nil {
		t.Fatalf("ListRollbacks returned error: %v", err)
	}
	if len(rollbacks) != 1 {
		t.Fatalf("expected one rollback record, got %#v", rollbacks)
	}

	unchanged, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("format_file unchanged returned error: %v", err)
	}
	if unchanged.Mutated || !strings.Contains(unchanged.Observation, "already formatted") {
		t.Fatalf("expected unchanged format result, got %#v", unchanged)
	}
}

func TestDefaultDispatcherFormatFileRejectsUnsupportedAndUnsafeTargets(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "archive.bin"), []byte("bytes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "blob.txt"), []byte{'a', 0x00, 'b'}, 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})

	unsupported, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "format_file", Args: map[string]string{"relPath": "archive.bin"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err == nil || unsupported.Mutated || !strings.Contains(unsupported.Observation, "not available") {
		t.Fatalf("expected unsupported format rejection, got result=%#v err=%v", unsupported, err)
	}
	unsafe, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "format_file", Args: map[string]string{"relPath": "blob.txt"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err == nil || unsafe.Mutated || !strings.Contains(unsafe.Observation, "safe text") {
		t.Fatalf("expected unsafe text rejection, got result=%#v err=%v", unsafe, err)
	}
	badFormatter, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "format_file", Args: map[string]string{"relPath": "archive.bin", "formatter": "prettier"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err == nil || badFormatter.Mutated || !strings.Contains(badFormatter.Observation, "unsupported formatter") {
		t.Fatalf("expected formatter rejection, got result=%#v err=%v", badFormatter, err)
	}
}

func TestDefaultDispatcherLintFileTool(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "app.json"), []byte("{\n  \"name\": true,,\n  \"note\": \"TODO wire this\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	call := agent.ToolCall{Name: "lint_file", Args: map[string]string{"relPath": "config/app.json"}}

	blocked, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root})
	if err == nil || blocked.Risk != "medium" || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected lint_file approval block, got result=%#v err=%v", blocked, err)
	}
	approved := false
	result, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{
		WorkspaceRoot: root,
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
			approved = request.Name == "lint_file" && request.Risk == "medium"
			return approved
		},
	})
	if err != nil {
		t.Fatalf("lint_file returned error: %v", err)
	}
	for _, expected := range []string{"Native lint diagnostics.", "Diagnostics: 2", "error/json", "info/marker", "TODO wire this"} {
		if !strings.Contains(result.Observation, expected) {
			t.Fatalf("lint_file observation missing %q:\n%s", expected, result.Observation)
		}
	}
	if !approved || result.Mutated {
		t.Fatalf("unexpected lint_file approval/mutation state approved=%t result=%#v", approved, result)
	}

	cleanPath := filepath.Join(root, "config", "clean.json")
	if err := os.WriteFile(cleanPath, []byte("{\"name\": true}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	clean, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "lint_file", Args: map[string]string{"relPath": "config/clean.json"}}, agent.Request{
		WorkspaceRoot: root,
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
			return request.Name == "lint_file"
		},
	})
	if err != nil {
		t.Fatalf("clean lint_file returned error: %v", err)
	}
	if !strings.Contains(clean.Observation, "No lint diagnostics found.") || clean.Mutated {
		t.Fatalf("unexpected clean lint result: %#v", clean)
	}
}

func TestDefaultDispatcherLintFileRejectsUnsupportedLinterAndUnsafeTarget(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "blob.txt"), []byte{'a', 0x00, 'b'}, 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	approved := agent.Request{
		WorkspaceRoot: root,
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
			return request.Name == "lint_file"
		},
	}

	badLinter, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "lint_file", Args: map[string]string{"relPath": "blob.txt", "linter": "eslint"}}, approved)
	if err == nil || badLinter.Mutated || !strings.Contains(badLinter.Observation, "unsupported linter") {
		t.Fatalf("expected unsupported linter rejection, got result=%#v err=%v", badLinter, err)
	}
	unsafe, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "lint_file", Args: map[string]string{"relPath": "blob.txt"}}, approved)
	if err == nil || unsafe.Mutated || !strings.Contains(unsafe.Observation, "safe text") {
		t.Fatalf("expected unsafe lint target rejection, got result=%#v err=%v", unsafe, err)
	}
}

func TestDefaultDispatcherGotoDefinitionTool(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "package main\n\nfunc main() {\n  Start()\n}\n\nfunc Start() {}\n"
	if err := os.WriteFile(filepath.Join(root, "cmd", "main.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})

	local, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "goto_definition", Args: map[string]string{"relPath": "cmd/main.go", "line": "4", "column": "3"}}, agent.Request{WorkspaceRoot: root})
	if err != nil {
		t.Fatalf("goto_definition local returned error: %v", err)
	}
	for _, expected := range []string{"Native definition lookup.", "Query: Start", "Scope: local", "Resolved: true", "Path: cmd/main.go", "Line: 7", "Label: Start"} {
		if !strings.Contains(local.Observation, expected) {
			t.Fatalf("local definition observation missing %q:\n%s", expected, local.Observation)
		}
	}

	if err := os.MkdirAll(filepath.Join(root, "internal", "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "other.go"), []byte("package main\n\nfunc main() { Launch() }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "app", "app.go"), []byte("package app\n\nfunc Launch() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	workspaceResult, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "goto_definition", Args: map[string]string{"relPath": "cmd/other.go", "query": "Launch"}}, agent.Request{WorkspaceRoot: root})
	if err != nil {
		t.Fatalf("goto_definition workspace returned error: %v", err)
	}
	for _, expected := range []string{"Scope: workspace", "Resolved: true", "Path: internal/app/app.go", "Line: 3", "Label: Launch"} {
		if !strings.Contains(workspaceResult.Observation, expected) {
			t.Fatalf("workspace definition observation missing %q:\n%s", expected, workspaceResult.Observation)
		}
	}

	missing, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "goto_definition", Args: map[string]string{"relPath": "cmd/main.go", "query": "Missing"}}, agent.Request{WorkspaceRoot: root})
	if err != nil {
		t.Fatalf("goto_definition missing returned error: %v", err)
	}
	if !strings.Contains(missing.Observation, "Resolved: false") || !strings.Contains(missing.Observation, "No definition found") {
		t.Fatalf("unexpected missing definition observation:\n%s", missing.Observation)
	}
}

func TestDefaultDispatcherFindReferencesTool(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "main.go"), []byte("package main\n\nfunc main() {\n  Start()\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "start.go"), []byte("package main\n\nfunc Start() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})

	byQuery, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "find_references", Args: map[string]string{"query": "Start"}}, agent.Request{WorkspaceRoot: root})
	if err != nil {
		t.Fatalf("find_references query returned error: %v", err)
	}
	for _, expected := range []string{"Native reference lookup.", "Query: Start", "References: 2", "cmd/main.go:4", "cmd/start.go:3"} {
		if !strings.Contains(byQuery.Observation, expected) {
			t.Fatalf("reference observation missing %q:\n%s", expected, byQuery.Observation)
		}
	}

	byCursor, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "find_references", Args: map[string]string{"relPath": "cmd/main.go", "line": "4", "column": "3"}}, agent.Request{WorkspaceRoot: root})
	if err != nil {
		t.Fatalf("find_references cursor returned error: %v", err)
	}
	if !strings.Contains(byCursor.Observation, "Query: Start") || !strings.Contains(byCursor.Observation, "References: 2") {
		t.Fatalf("unexpected cursor references observation:\n%s", byCursor.Observation)
	}

	missingCursor, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "find_references", Args: map[string]string{"relPath": "cmd/main.go"}}, agent.Request{WorkspaceRoot: root})
	if err == nil || !strings.Contains(missingCursor.Observation, "line must be one or greater") {
		t.Fatalf("expected missing cursor rejection, got result=%#v err=%v", missingCursor, err)
	}
}

func TestDefaultDispatcherReadDependencyGraphTool(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "app.ts"), []byte("import { api } from './api'\nconst fs = require('fs')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "api.ts"), []byte("export const api = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})

	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "read_dependency_graph", Args: map[string]string{"relPath": "src", "maxFiles": "20", "maxEdges": "20"}}, agent.Request{WorkspaceRoot: root})
	if err != nil {
		t.Fatalf("read_dependency_graph returned error: %v", err)
	}
	for _, expected := range []string{"Native dependency graph.", "Scope: src", "scanned 2 file(s)", "src/app.ts:1 -> src/api.ts [js-import/resolved] ./api", "src/app.ts:2 -> external:fs [js-require/external] fs"} {
		if !strings.Contains(result.Observation, expected) {
			t.Fatalf("dependency graph observation missing %q:\n%s", expected, result.Observation)
		}
	}

	empty, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "read_dependency_graph", Args: map[string]string{"relPath": "src/api.ts"}}, agent.Request{WorkspaceRoot: root})
	if err != nil {
		t.Fatalf("read_dependency_graph focused file returned error: %v", err)
	}
	if !strings.Contains(empty.Observation, "No supported code dependency edges found") {
		t.Fatalf("expected empty graph observation, got:\n%s", empty.Observation)
	}
}

func TestDefaultDispatcherReadSymbolIndexTool(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "app.go"), []byte("package src\n\ntype Service struct{}\n\nfunc Start() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "view.ts"), []byte("export class Panel {}\nexport const render = () => {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# ignored\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})

	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "read_symbol_index", Args: map[string]string{"relPath": "src", "maxFiles": "20", "maxSymbols": "20"}}, agent.Request{WorkspaceRoot: root})
	if err != nil {
		t.Fatalf("read_symbol_index returned error: %v", err)
	}
	for _, expected := range []string{"Native symbol index.", "Scope: src", "Files scanned: 2", "Symbols: 4", "src/app.go:3 [type] Service", "src/app.go:5 [func] Start", "src/view.ts:1 [class] Panel", "src/view.ts:2 [func] render"} {
		if !strings.Contains(result.Observation, expected) {
			t.Fatalf("symbol index observation missing %q:\n%s", expected, result.Observation)
		}
	}

	filtered, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "read_symbol_index", Args: map[string]string{"query": "panel"}}, agent.Request{WorkspaceRoot: root})
	if err != nil {
		t.Fatalf("read_symbol_index filtered returned error: %v", err)
	}
	if !strings.Contains(filtered.Observation, "Symbols: 1") || !strings.Contains(filtered.Observation, "src/view.ts:1 [class] Panel") || strings.Contains(filtered.Observation, "Start") {
		t.Fatalf("unexpected filtered symbol index observation:\n%s", filtered.Observation)
	}
}

func TestDefaultDispatcherUpdateProjectMemoryTool(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})

	blocked, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "update_project_memory", Args: map[string]string{"key": "architecture.boundaries", "content": "Services stay framework-free."}}, agent.Request{WorkspaceRoot: root})
	if err == nil || blocked.Risk != "medium" || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected approval rejection, got result=%#v err=%v", blocked, err)
	}

	approved := false
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "update_project_memory", Args: map[string]string{"key": "architecture.boundaries", "content": "Services stay framework-free. API_KEY=secret", "sourceRelPaths": "[\"README.md\"]"}}, agent.Request{
		WorkspaceRoot: root,
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
			approved = request.Name == "update_project_memory" && request.Risk == "medium"
			return approved
		},
	})
	if err != nil {
		t.Fatalf("update_project_memory returned error: %v", err)
	}
	for _, expected := range []string{"Created project memory.", "Key: architecture.boundaries", "Total records: 1", "Sources: 1", "Services stay framework-free.", "API_KEY=[redacted]"} {
		if !strings.Contains(result.Observation, expected) {
			t.Fatalf("project memory observation missing %q:\n%s", expected, result.Observation)
		}
	}
	if !approved || !result.Mutated {
		t.Fatalf("expected approved mutated result, approved=%v result=%#v", approved, result)
	}
	stored, err := os.ReadFile(filepath.Join(root, ".nexusdesk", "project-memory", "memory.json"))
	if err != nil {
		t.Fatalf("expected project memory file: %v", err)
	}
	if !strings.Contains(string(stored), "API_KEY=[redacted]") || strings.Contains(string(stored), "secret") {
		t.Fatalf("expected stored memory to be redacted:\n%s", string(stored))
	}

	rejected, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "update_project_memory", Args: map[string]string{"key": "bad", "content": "No metadata", "sourceRelPaths": ".nexusdesk/project-memory/memory.json"}}, agent.Request{
		WorkspaceRoot: root,
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
			return request.Name == "update_project_memory"
		},
	})
	if err == nil || !strings.Contains(rejected.Observation, "metadata") {
		t.Fatalf("expected metadata source rejection, got result=%#v err=%v", rejected, err)
	}
}

func TestDefaultDispatcherArtifactLineageTool(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.WriteDocumentSetReport(artifactsSvc.DocumentSetReport{
		Title:       "Project Docs",
		Roots:       []string{"."},
		SourcePaths: []string{"README.md"},
		Content:     "# Project\n",
	}); err != nil {
		t.Fatalf("WriteDocumentSetReport returned error: %v", err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "read_artifact_lineage"}, agent.Request{WorkspaceRoot: root})
	if err != nil {
		t.Fatalf("read_artifact_lineage returned error: %v", err)
	}
	for _, expected := range []string{"lineage nodes", "artifact:", "source:README.md", "Relationship counts"} {
		if !strings.Contains(result.Observation, expected) {
			t.Fatalf("expected observation to contain %q:\n%s", expected, result.Observation)
		}
	}
}

func TestRegenerateArtifactRequiresApproval(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.WriteWorkspaceScanReport(artifactsSvc.WorkspaceScanReport{
		WorkspaceName: "repo",
		Included:      1,
		Message:       "Scanned 1 workspace entry, skipped 0.",
	})
	if err != nil {
		t.Fatalf("WriteWorkspaceScanReport returned error: %v", err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "regenerate_artifact", Args: map[string]string{"relPath": artifact.RelPath}}, agent.Request{WorkspaceRoot: root})
	if err == nil || result.Risk != "high" || !strings.Contains(result.Observation, "approval") {
		t.Fatalf("expected approval error, got result=%#v err=%v", result, err)
	}
}

func TestRegenerateArtifactDocumentExtractTool(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "guide.md"), []byte("# Guide\n\nUseful content.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	original, err := store.WriteDocumentExtractionReport(artifactsSvc.DocumentExtractionReport{
		Title:   "Old Guide",
		RelPath: "guide.md",
		Format:  "markdown",
		Content: "old content",
		Lines:   1,
		Words:   2,
	})
	if err != nil {
		t.Fatalf("WriteDocumentExtractionReport returned error: %v", err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "regenerate_artifact", Args: map[string]string{"relPath": original.RelPath}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("regenerate_artifact returned error: %v", err)
	}
	if !result.Mutated || !strings.Contains(result.Observation, "Regenerated document-extract artifact") || !strings.Contains(result.Observation, "guide.md") {
		t.Fatalf("unexpected regeneration result: %#v", result)
	}
	matches, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: "kind:document-extract"})
	if err != nil {
		t.Fatalf("ListArtifacts returned error: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected original and regenerated document artifacts, got %d", len(matches))
	}
	text, err := store.ReadArtifactText(matches[0].RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText returned error: %v", err)
	}
	if !strings.Contains(text, "Useful content.") {
		t.Fatalf("regenerated document did not use current source content:\n%s", text)
	}
}

func TestRegenerateArtifactDocumentReportTool(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Project\n\nFresh project notes.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	original, err := store.WriteDocumentSetReport(artifactsSvc.DocumentSetReport{
		Title:       "Old Docs",
		Roots:       []string{"README.md"},
		SourcePaths: []string{"README.md"},
		Content:     "old content",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport returned error: %v", err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "regenerate_artifact", Args: map[string]string{"relPath": original.RelPath}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("regenerate_artifact returned error: %v", err)
	}
	if !result.Mutated || !strings.Contains(result.Observation, "Regenerated document-report artifact") || !strings.Contains(result.Observation, "README.md") {
		t.Fatalf("unexpected regeneration result: %#v", result)
	}
	matches, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: "kind:document-report"})
	if err != nil {
		t.Fatalf("ListArtifacts returned error: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected original and regenerated document reports, got %d", len(matches))
	}
	foundFresh := false
	for _, match := range matches {
		text, err := store.ReadArtifactText(match.RelPath)
		if err != nil {
			t.Fatalf("ReadArtifactText returned error: %v", err)
		}
		if strings.Contains(text, "Fresh project notes.") {
			foundFresh = true
			break
		}
	}
	if !foundFresh {
		t.Fatalf("regenerated document report did not use current source content")
	}
}

func TestRegenerateArtifactComparisonTool(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	left, err := store.WriteDocumentSetReport(artifactsSvc.DocumentSetReport{
		Title:       "Left",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md"},
		Content:     "old",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport(left) returned error: %v", err)
	}
	right, err := store.WriteDocumentSetReport(artifactsSvc.DocumentSetReport{
		Title:       "Right",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md"},
		Content:     "new",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport(right) returned error: %v", err)
	}
	comparison, err := store.CompareArtifacts(left.RelPath, right.RelPath)
	if err != nil {
		t.Fatalf("CompareArtifacts returned error: %v", err)
	}
	original, err := store.WriteArtifactComparisonReport(comparison)
	if err != nil {
		t.Fatalf("WriteArtifactComparisonReport returned error: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "regenerate_artifact", Args: map[string]string{"relPath": original.RelPath}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("regenerate_artifact returned error: %v", err)
	}
	if !result.Mutated || !strings.Contains(result.Observation, "Regenerated artifact-comparison artifact") {
		t.Fatalf("unexpected comparison regeneration result: %#v", result)
	}
	matches, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: "kind:artifact-comparison"})
	if err != nil {
		t.Fatalf("ListArtifacts returned error: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected original and regenerated comparison artifacts, got %d", len(matches))
	}
}

func TestRegenerateArtifactPresentationPackageTool(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	outline, err := store.WritePresentationOutlineReport(artifactsSvc.PresentationOutlineReport{
		Title:       "Presentation Outline - Architecture Notes",
		SourcePath:  ".nexusdesk/artifacts/document-sets/report.md",
		SourceTitle: "Architecture Notes",
		SourceKind:  "document-report",
		SourcePaths: []string{"docs/a.md"},
		Content:     "### Slide 1: Goals\n\n- Keep shell native\n",
		SlideCount:  1,
	})
	if err != nil {
		t.Fatalf("WritePresentationOutlineReport returned error: %v", err)
	}
	original, err := store.WritePresentationPackageReport(artifactsSvc.BuildPresentationPackageReport("", outline.RelPath, outline.Title, outline.Kind, "### Slide 1: Old\n\n- Old content\n", outline.SourcePaths))
	if err != nil {
		t.Fatalf("WritePresentationPackageReport returned error: %v", err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "regenerate_artifact", Args: map[string]string{"relPath": original.RelPath}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("regenerate_artifact returned error: %v", err)
	}
	if !result.Mutated || !strings.Contains(result.Observation, "Regenerated presentation-package artifact") || !strings.Contains(result.Observation, outline.RelPath) {
		t.Fatalf("unexpected presentation package regeneration result: %#v", result)
	}
	matches, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: "kind:presentation-package"})
	if err != nil {
		t.Fatalf("ListArtifacts returned error: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected original and regenerated presentation packages, got %d", len(matches))
	}
}

func TestRegenerateArtifactDocumentBriefTool(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	source, err := store.WriteDocumentSetReport(artifactsSvc.DocumentSetReport{
		Title:       "Architecture Notes",
		Roots:       []string{"docs"},
		SourcePaths: []string{"docs/a.md"},
		Content:     "## Goals\n\n- Keep shell native\n- Missing release smoke is a blocker\n- Next action: verify diagnostics\n",
	})
	if err != nil {
		t.Fatalf("WriteDocumentSetReport returned error: %v", err)
	}
	original, err := store.WriteDocumentBriefReport(artifactsSvc.BuildDocumentBriefReport("", source.RelPath, source.Title, source.Kind, "old brief", source.SourcePaths))
	if err != nil {
		t.Fatalf("WriteDocumentBriefReport returned error: %v", err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "regenerate_artifact", Args: map[string]string{"relPath": original.RelPath}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("regenerate_artifact returned error: %v", err)
	}
	if !result.Mutated || !strings.Contains(result.Observation, "Regenerated document-brief artifact") || !strings.Contains(result.Observation, source.RelPath) {
		t.Fatalf("unexpected document brief regeneration result: %#v", result)
	}
	matches, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: "kind:document-brief"})
	if err != nil {
		t.Fatalf("ListArtifacts returned error: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected original and regenerated document briefs, got %d", len(matches))
	}
	foundFresh := false
	for _, match := range matches {
		text, err := store.ReadArtifactText(match.RelPath)
		if err != nil {
			t.Fatalf("ReadArtifactText returned error: %v", err)
		}
		if strings.Contains(text, "Keep shell native") && strings.Contains(text, "blocker") {
			foundFresh = true
			break
		}
	}
	if !foundFresh {
		t.Fatalf("regenerated document brief did not use current source artifact content")
	}
}

func TestRegenerateArtifactDocumentExportTool(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	brief, err := store.WriteDocumentBriefReport(artifactsSvc.DocumentBriefReport{
		Title:       "Document Brief - Architecture Notes",
		SourcePath:  ".nexusdesk/artifacts/document-sets/report.md",
		SourceKind:  "document-report",
		SourcePaths: []string{"docs/a.md"},
		Content:     "### Executive Summary\n\n- Keep shell native.\n\n### Risks And Gaps\n\n- Packaging smoke remains a blocker.\n",
	})
	if err != nil {
		t.Fatalf("WriteDocumentBriefReport returned error: %v", err)
	}
	original, err := store.WriteDocumentExportReport(artifactsSvc.BuildDocumentExportReport("", brief.RelPath, brief.Title, brief.Kind, "old export", brief.SourcePaths))
	if err != nil {
		t.Fatalf("WriteDocumentExportReport returned error: %v", err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "regenerate_artifact", Args: map[string]string{"relPath": original.RelPath}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("regenerate_artifact returned error: %v", err)
	}
	if !result.Mutated || !strings.Contains(result.Observation, "Regenerated document-export artifact") || !strings.Contains(result.Observation, brief.RelPath) {
		t.Fatalf("unexpected document export regeneration result: %#v", result)
	}
	matches, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: "kind:document-export"})
	if err != nil {
		t.Fatalf("ListArtifacts returned error: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected original and regenerated document exports, got %d", len(matches))
	}
}

func TestRegenerateArtifactPresentationDeckTool(t *testing.T) {
	root := t.TempDir()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	outline, err := store.WritePresentationOutlineReport(artifactsSvc.PresentationOutlineReport{
		Title:       "Presentation Outline - Architecture Notes",
		SourcePath:  ".nexusdesk/artifacts/document-sets/report.md",
		SourceTitle: "Architecture Notes",
		SourceKind:  "document-report",
		SourcePaths: []string{"docs/a.md"},
		Content:     "### Slide 1: Goals\n\n- Keep shell native\n",
		SlideCount:  1,
	})
	if err != nil {
		t.Fatalf("WritePresentationOutlineReport returned error: %v", err)
	}
	original, err := store.WritePresentationDeckReport(artifactsSvc.BuildPresentationDeckReport("", outline.RelPath, outline.Title, outline.Kind, "### Slide 1: Old\n\n- Old content\n", outline.SourcePaths))
	if err != nil {
		t.Fatalf("WritePresentationDeckReport returned error: %v", err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "regenerate_artifact", Args: map[string]string{"relPath": original.RelPath}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("regenerate_artifact returned error: %v", err)
	}
	if !result.Mutated || !strings.Contains(result.Observation, "Regenerated presentation-deck artifact") || !strings.Contains(result.Observation, outline.RelPath) {
		t.Fatalf("unexpected presentation deck regeneration result: %#v", result)
	}
	matches, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: "kind:presentation-deck"})
	if err != nil {
		t.Fatalf("ListArtifacts returned error: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected original and regenerated presentation decks, got %d", len(matches))
	}
}

func TestDefaultDispatcherWebFetchRequiresApproval(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "text/plain")
		_, _ = response.Write([]byte("hello from docs"))
	}))
	defer server.Close()
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	call := agent.ToolCall{Name: "web_fetch", Args: map[string]string{"url": server.URL, "allowLocal": "true"}}

	blocked, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{})
	if err == nil || blocked.Risk != "medium" || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected approval error, got result=%#v err=%v", blocked, err)
	}

	approved := false
	fetched, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
			approved = request.Name == "web_fetch" && request.Risk == "medium"
			return approved
		},
	})
	if err != nil {
		t.Fatalf("web_fetch returned error: %v", err)
	}
	if !approved || !strings.Contains(fetched.Observation, "hello from docs") || fetched.Risk != "medium" {
		t.Fatalf("unexpected web fetch result approved=%v result=%#v", approved, fetched)
	}
}

func TestRunTaskRequiresApproval(t *testing.T) {
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspaceSvc.New()})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "run_task", Args: map[string]string{"taskId": "go-test-root"}}, agent.Request{WorkspaceRoot: t.TempDir()})
	if err == nil || result.Risk != "high" || !strings.Contains(result.Observation, "approval") {
		t.Fatalf("expected approval error, got result=%#v err=%v", result, err)
	}
}

func runToolTestGit(t *testing.T, root string, args ...string) {
	t.Helper()
	_ = runToolTestGitOutput(t, root, args...)
}

func runToolTestGitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
	return string(output)
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

func TestDefaultDispatcherResolveConflictTool(t *testing.T) {
	root := t.TempDir()
	workspace := workspaceSvc.New()
	content := "before\n<<<<<<< HEAD\nours\n=======\ntheirs\n>>>>>>> branch\nafter\n"
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDefaultDispatcher(Dependencies{Workspace: workspace})
	call := agent.ToolCall{Name: "resolve_conflict", Args: map[string]string{"relPath": "notes.txt", "strategy": "theirs"}}

	blocked, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root})
	if err == nil || !strings.Contains(blocked.Observation, "approval") {
		t.Fatalf("expected approval block, got result=%#v err=%v", blocked, err)
	}
	resolved, err := dispatcher.ExecuteTool(context.Background(), call, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err != nil {
		t.Fatalf("resolve_conflict returned error: %v", err)
	}
	if !resolved.Mutated || !strings.Contains(resolved.Observation, "Resolved 1 conflict") || !strings.Contains(resolved.Observation, "Strategy: theirs") || !strings.Contains(resolved.Observation, "Rollback:") {
		t.Fatalf("unexpected resolve_conflict observation:\n%s", resolved.Observation)
	}
	data, err := os.ReadFile(filepath.Join(root, "notes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "before\ntheirs\nafter\n" {
		t.Fatalf("unexpected resolved content: %q", data)
	}
	rollbacks, err := workspace.ListRollbacks(root)
	if err != nil {
		t.Fatalf("ListRollbacks returned error: %v", err)
	}
	if len(rollbacks) != 1 {
		t.Fatalf("expected rollback record, got %#v", rollbacks)
	}

	rejected, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "resolve_conflict", Args: map[string]string{"relPath": "notes.txt", "strategy": "ours"}}, agent.Request{WorkspaceRoot: root, ApproveWrites: true})
	if err == nil || rejected.Mutated || !strings.Contains(rejected.Observation, "no conflict markers") {
		t.Fatalf("expected clean file rejection, got result=%#v err=%v", rejected, err)
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
