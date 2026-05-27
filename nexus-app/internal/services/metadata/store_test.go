package metadata

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	jobssvc "nexusdesk/internal/services/jobs"
)

func TestEnsureCreatesSQLiteStoreAndManifest(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	status, err := store.Ensure()
	if err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}
	if status.SchemaVersion != 6 || status.SchemaHash == "" {
		t.Fatalf("unexpected status: %#v", status)
	}
	if _, err := os.Stat(status.Path); err != nil {
		t.Fatalf("expected sqlite db: %v", err)
	}
	if _, err := os.Stat(status.SchemaPath); err != nil {
		t.Fatalf("expected schema file: %v", err)
	}
	if len(status.Tables) < 5 {
		t.Fatalf("expected core tables, got %#v", status.Tables)
	}
}

func TestSaveAndListJobs(t *testing.T) {
	store := mustStore(t)
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	completed := started.Add(time.Second)
	err := store.SaveJob(jobssvc.Job{
		ID:          "job-0001",
		Kind:        "task",
		Label:       "go test ./...",
		Status:      jobssvc.StatusSuccess,
		Message:     "done",
		LogTail:     []string{"line 1", "line 2"},
		StartedAt:   started,
		CompletedAt: completed,
	})
	if err != nil {
		t.Fatalf("SaveJob returned error: %v", err)
	}
	jobs, err := store.ListJobs()
	if err != nil {
		t.Fatalf("ListJobs returned error: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != "job-0001" || len(jobs[0].LogTail) != 2 {
		t.Fatalf("unexpected jobs: %#v", jobs)
	}
}

func TestSaveAndListTaskRuns(t *testing.T) {
	store := mustStore(t)
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	record := TaskRunRecord{
		JobID:        "job-0001",
		TaskID:       "go-test-root",
		Kind:         "go-test",
		Label:        "go test ./...",
		Command:      "go test ./...",
		Cwd:          ".",
		Status:       "success",
		ExitCode:     0,
		Stdout:       "ok\n",
		Message:      "done",
		ArtifactPath: ".nexusdesk/artifacts/task-runs/run.md",
		StartedAt:    started,
		CompletedAt:  started.Add(time.Second),
		DurationMs:   1000,
	}
	if err := store.SaveTaskRun(record); err != nil {
		t.Fatalf("SaveTaskRun returned error: %v", err)
	}
	runs, err := store.ListTaskRuns(10)
	if err != nil {
		t.Fatalf("ListTaskRuns returned error: %v", err)
	}
	if len(runs) != 1 || runs[0].TaskID != "go-test-root" || runs[0].Stdout != "ok\n" || runs[0].ArtifactPath == "" {
		t.Fatalf("unexpected task runs: %#v", runs)
	}
}

func TestSaveListAndDeleteArtifacts(t *testing.T) {
	store := mustStore(t)
	generated := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	record := ArtifactRecord{
		Kind:         "document-report",
		Title:        "Project report",
		RelPath:      ".nexusdesk/artifacts/document-sets/report.md",
		MetadataPath: ".nexusdesk/artifacts/document-sets/report.md.json",
		Size:         512,
		Source:       "docs",
		SourcePaths:  []string{"docs/a.md", "docs/b.md"},
		GeneratedAt:  generated,
		CreatedAt:    generated,
	}
	if err := store.SaveArtifact(record); err != nil {
		t.Fatalf("SaveArtifact returned error: %v", err)
	}
	records, err := store.ListArtifacts("project", false, 10)
	if err != nil {
		t.Fatalf("ListArtifacts returned error: %v", err)
	}
	if len(records) != 1 || records[0].Title != "Project report" || len(records[0].SourcePaths) != 2 {
		t.Fatalf("unexpected artifact records: %#v", records)
	}
	record.Archived = true
	if err := store.SaveArtifact(record); err != nil {
		t.Fatalf("SaveArtifact archive returned error: %v", err)
	}
	active, err := store.ListArtifacts("", false, 10)
	if err != nil {
		t.Fatalf("ListArtifacts active returned error: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("archived artifact should be hidden by default: %#v", active)
	}
	all, err := store.ListArtifacts("", true, 10)
	if err != nil {
		t.Fatalf("ListArtifacts all returned error: %v", err)
	}
	if len(all) != 1 || !all[0].Archived {
		t.Fatalf("expected archived artifact record: %#v", all)
	}
	if err := store.DeleteArtifact(record.RelPath); err != nil {
		t.Fatalf("DeleteArtifact returned error: %v", err)
	}
	all, err = store.ListArtifacts("", true, 10)
	if err != nil {
		t.Fatalf("ListArtifacts after delete returned error: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected artifact record deletion, got %#v", all)
	}
}

func TestSaveSQLRunAndDatasetDependency(t *testing.T) {
	store := mustStore(t)
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	sqlRun := SQLRunRecord{
		RelPath:     "data/sales.csv",
		SQL:         "select * from dataset",
		Engine:      "native-dataset-sql",
		Status:      "success",
		RowCount:    4,
		MatchedRows: 2,
		ShownRows:   2,
		Message:     "ok",
		StartedAt:   started,
		CompletedAt: started.Add(time.Second),
		DurationMs:  1000,
	}
	sqlRun = store.NormalizeSQLRunRecord(sqlRun)
	if err := store.SaveSQLRun(sqlRun); err != nil {
		t.Fatalf("SaveSQLRun returned error: %v", err)
	}
	if err := store.SaveDatasetDependency(DatasetDependencyRecord{
		SourcePath:    "data/sales.csv",
		DependentKind: "sql-run",
		DependentRef:  sqlRun.ID,
		Relation:      "reads",
		Metadata:      map[string]string{"engine": "native-dataset-sql"},
		CreatedAt:     started,
		UpdatedAt:     started,
	}); err != nil {
		t.Fatalf("SaveDatasetDependency returned error: %v", err)
	}
	runs, err := store.ListSQLRuns(10)
	if err != nil {
		t.Fatalf("ListSQLRuns returned error: %v", err)
	}
	if len(runs) != 1 || runs[0].RelPath != "data/sales.csv" || runs[0].ShownRows != 2 {
		t.Fatalf("unexpected sql runs: %#v", runs)
	}
	dependencies, err := store.ListDatasetDependencies("data/sales.csv", 10)
	if err != nil {
		t.Fatalf("ListDatasetDependencies returned error: %v", err)
	}
	if len(dependencies) != 1 || dependencies[0].DependentRef != sqlRun.ID || dependencies[0].Metadata["engine"] == "" {
		t.Fatalf("unexpected dependencies: %#v", dependencies)
	}
}

func TestSaveAndListChatMessages(t *testing.T) {
	store := mustStore(t)
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	if err := store.SaveChatMessage(ChatMessageRecord{
		Role:      "user",
		Content:   "What changed?",
		CreatedAt: started,
	}); err != nil {
		t.Fatalf("SaveChatMessage user returned error: %v", err)
	}
	if err := store.SaveChatMessage(ChatMessageRecord{
		Role:        "assistant",
		Content:     "The native app gained chat persistence.",
		Model:       "qwen2.5-coder:14b",
		SourcePaths: []string{"tracker.md"},
		CreatedAt:   started.Add(time.Second),
	}); err != nil {
		t.Fatalf("SaveChatMessage assistant returned error: %v", err)
	}
	messages, err := store.ListChatMessages(10)
	if err != nil {
		t.Fatalf("ListChatMessages returned error: %v", err)
	}
	if len(messages) != 2 || messages[0].Role != "user" || messages[1].Model == "" || len(messages[1].SourcePaths) != 1 {
		t.Fatalf("unexpected chat messages: %#v", messages)
	}

	found, err := store.SearchChatMessages("native app", 10)
	if err != nil {
		t.Fatalf("SearchChatMessages returned error: %v", err)
	}
	if len(found) != 1 || found[0].Role != "assistant" {
		t.Fatalf("unexpected chat search results: %#v", found)
	}

	recent, err := store.SearchChatMessages("", 10)
	if err != nil {
		t.Fatalf("SearchChatMessages recent returned error: %v", err)
	}
	if len(recent) != 2 || recent[0].Role != "assistant" {
		t.Fatalf("expected newest message first, got %#v", recent)
	}
}

func TestSaveAndListAgentRunsAndToolRuns(t *testing.T) {
	store := mustStore(t)
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	agentRun := AgentRunRecord{
		JobID:       "job-0002",
		Prompt:      "Review project",
		Status:      "success",
		Message:     "Done",
		Iterations:  2,
		Plan:        []AgentPlanStep{{Step: "Inspect", Status: "completed"}},
		SourcePaths: []string{"README.md"},
		StartedAt:   started,
		CompletedAt: started.Add(2 * time.Second),
		DurationMs:  2000,
	}
	agentRun = store.NormalizeAgentRunRecord(agentRun)
	if err := store.SaveAgentRun(agentRun); err != nil {
		t.Fatalf("SaveAgentRun returned error: %v", err)
	}
	if err := store.SaveToolRun(ToolRunRecord{
		AgentRunID:  agentRun.ID,
		JobID:       agentRun.JobID,
		Sequence:    1,
		ToolName:    "read_context",
		Risk:        "low",
		Args:        map[string]string{"relPath": "README.md"},
		Observation: "README content",
		StartedAt:   started,
		CompletedAt: started.Add(time.Second),
	}); err != nil {
		t.Fatalf("SaveToolRun returned error: %v", err)
	}

	runs, err := store.ListAgentRuns(10)
	if err != nil {
		t.Fatalf("ListAgentRuns returned error: %v", err)
	}
	if len(runs) != 1 || runs[0].Prompt != "Review project" || len(runs[0].Plan) != 1 || len(runs[0].SourcePaths) != 1 {
		t.Fatalf("unexpected agent runs: %#v", runs)
	}
	tools, err := store.ListToolRuns(agentRun.ID)
	if err != nil {
		t.Fatalf("ListToolRuns returned error: %v", err)
	}
	if len(tools) != 1 || tools[0].ToolName != "read_context" || tools[0].Args["relPath"] != "README.md" {
		t.Fatalf("unexpected tool runs: %#v", tools)
	}
}

func TestEnsureMigratesTaskRunsArtifactPathColumn(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, filepath.FromSlash(metadataDirRelPath), "nexusdesk.sqlite")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE task_runs (
		id TEXT PRIMARY KEY,
		workspace_root TEXT NOT NULL,
		job_id TEXT,
		task_id TEXT NOT NULL,
		kind TEXT NOT NULL,
		label TEXT NOT NULL,
		command TEXT NOT NULL,
		cwd TEXT NOT NULL,
		source TEXT,
		status TEXT NOT NULL,
		exit_code INTEGER,
		stdout TEXT,
		stderr TEXT,
		message TEXT,
		started_at TEXT NOT NULL,
		completed_at TEXT,
		duration_ms INTEGER
	)`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Ensure(); err != nil {
		t.Fatalf("Ensure migration returned error: %v", err)
	}
	if err := store.SaveTaskRun(TaskRunRecord{
		JobID:        "job-0001",
		TaskID:       "go-test-root",
		Kind:         "go-test",
		Label:        "go test ./...",
		Command:      "go test ./...",
		Cwd:          ".",
		Status:       "success",
		ArtifactPath: ".nexusdesk/artifacts/task-runs/run.md",
		StartedAt:    time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveTaskRun after migration returned error: %v", err)
	}
}

func mustStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	return store
}
