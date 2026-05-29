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
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
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
