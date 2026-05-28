package metadata

import (
	"archive/zip"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jobssvc "nexusdesk/internal/services/jobs"
)

func TestEnsureCreatesSQLiteStoreAndManifest(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	status, err := store.Ensure()
	if err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}
	if status.SchemaVersion != schemaVersion || status.SchemaHash == "" {
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

func TestStoreReusesOpenDatabaseAfterEnsure(t *testing.T) {
	store := mustStore(t)
	first, err := store.open()
	if err != nil {
		t.Fatalf("first open returned error: %v", err)
	}
	second, err := store.open()
	if err != nil {
		t.Fatalf("second open returned error: %v", err)
	}
	if first != second {
		t.Fatal("expected metadata store to reuse the opened database handle")
	}
	schemaPath := filepath.Join(filepath.Dir(store.Path()), "schema.sql")
	before, err := os.Stat(schemaPath)
	if err != nil {
		t.Fatalf("stat schema before Ensure failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := store.Ensure(); err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}
	after, err := os.Stat(schemaPath)
	if err != nil {
		t.Fatalf("stat schema after Ensure failed: %v", err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("expected cached Ensure to leave schema file untouched")
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
	latest, ok, err := store.LatestTaskRunForJob("job-0001")
	if err != nil {
		t.Fatalf("LatestTaskRunForJob returned error: %v", err)
	}
	if !ok || latest.TaskID != "go-test-root" || latest.ArtifactPath == "" {
		t.Fatalf("unexpected latest task run: %#v ok=%v", latest, ok)
	}
	if _, ok, err := store.LatestTaskRunForJob("missing"); err != nil || ok {
		t.Fatalf("expected missing latest task run, ok=%v err=%v", ok, err)
	}
}

func TestDeleteJobsRemovesJobsAndTaskRuns(t *testing.T) {
	store := mustStore(t)
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	for _, id := range []string{"job-delete", "job-keep"} {
		if err := store.SaveJob(jobssvc.Job{
			ID:        id,
			Kind:      "task",
			Label:     id,
			Status:    jobssvc.StatusSuccess,
			Message:   "done",
			StartedAt: started,
		}); err != nil {
			t.Fatalf("SaveJob(%s) returned error: %v", id, err)
		}
		if err := store.SaveTaskRun(TaskRunRecord{
			JobID:       id,
			TaskID:      id + "-task",
			Kind:        "go-test",
			Label:       id,
			Command:     "go test ./...",
			Cwd:         ".",
			Status:      "success",
			StartedAt:   started,
			CompletedAt: started.Add(time.Second),
		}); err != nil {
			t.Fatalf("SaveTaskRun(%s) returned error: %v", id, err)
		}
	}
	if err := store.DeleteJobs([]string{"job-delete"}); err != nil {
		t.Fatalf("DeleteJobs returned error: %v", err)
	}
	jobs, err := store.ListJobs()
	if err != nil {
		t.Fatalf("ListJobs returned error: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != "job-keep" {
		t.Fatalf("unexpected jobs after delete: %#v", jobs)
	}
	if _, ok, err := store.LatestTaskRunForJob("job-delete"); err != nil || ok {
		t.Fatalf("expected deleted task run to be gone, ok=%v err=%v", ok, err)
	}
	if _, ok, err := store.LatestTaskRunForJob("job-keep"); err != nil || !ok {
		t.Fatalf("expected retained task run, ok=%v err=%v", ok, err)
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
	dependency, err := store.GetDatasetDependency(dependencies[0].ID)
	if err != nil {
		t.Fatalf("GetDatasetDependency returned error: %v", err)
	}
	if dependency.SourcePath != "data/sales.csv" {
		t.Fatalf("unexpected dependency lookup: %#v", dependency)
	}
	refreshed, err := store.UpdateDatasetDependencyArtifact(dependency.ID, ".nexusdesk/artifacts/data/refreshed.csv", map[string]string{"query": "channel=search"})
	if err != nil {
		t.Fatalf("UpdateDatasetDependencyArtifact returned error: %v", err)
	}
	if refreshed.DependentRef != ".nexusdesk/artifacts/data/refreshed.csv" || refreshed.Metadata["artifact"] == "" || refreshed.Metadata["query"] == "" {
		t.Fatalf("unexpected refreshed dependency: %#v", refreshed)
	}
}

func TestSaveAndListApprovalRecords(t *testing.T) {
	store := mustStore(t)
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	record := ApprovalRecord{
		Action:    "access.full_project.grant",
		Target:    ".",
		Risk:      "high",
		Decision:  "granted",
		Message:   "Full project access granted.",
		CreatedAt: started,
	}
	if err := store.SaveApprovalRecord(record); err != nil {
		t.Fatalf("SaveApprovalRecord returned error: %v", err)
	}
	records, err := store.ListApprovalRecords(10)
	if err != nil {
		t.Fatalf("ListApprovalRecords returned error: %v", err)
	}
	if len(records) != 1 || records[0].Action != "access.full_project.grant" || records[0].Risk != "high" {
		t.Fatalf("unexpected approval records: %#v", records)
	}
	if records[0].ID == "" || records[0].CreatedAt.IsZero() {
		t.Fatalf("expected normalized approval record: %#v", records[0])
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
		Role:           "assistant",
		Content:        "The native app gained chat persistence.",
		Model:          "qwen2.5-coder:14b",
		ContextRelPath: "context: tracker.md",
		SourcePaths:    []string{"tracker.md"},
		CreatedAt:      started.Add(time.Second),
	}); err != nil {
		t.Fatalf("SaveChatMessage assistant returned error: %v", err)
	}
	messages, err := store.ListChatMessages(10)
	if err != nil {
		t.Fatalf("ListChatMessages returned error: %v", err)
	}
	if len(messages) != 2 || messages[0].Role != "user" || messages[1].Model == "" || messages[1].ContextRelPath != "context: tracker.md" || len(messages[1].SourcePaths) != 1 {
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
	t.Cleanup(func() { _ = store.Close() })
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

func TestEnsureMigratesChatContextRelPathColumn(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, filepath.FromSlash(metadataDirRelPath), "nexusdesk.sqlite")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE chat_messages (
		id TEXT PRIMARY KEY,
		workspace_root TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		model TEXT,
		source_paths_json TEXT,
		created_at TEXT NOT NULL
	)`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.Ensure(); err != nil {
		t.Fatalf("Ensure migration returned error: %v", err)
	}
	if err := store.SaveChatMessage(ChatMessageRecord{
		Role:           "assistant",
		Content:        "Answer with context",
		ContextRelPath: "context: README.md",
		SourcePaths:    []string{"README.md"},
		CreatedAt:      time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveChatMessage after migration returned error: %v", err)
	}
	messages, err := store.ListChatMessages(10)
	if err != nil {
		t.Fatalf("ListChatMessages returned error: %v", err)
	}
	if len(messages) != 1 || messages[0].ContextRelPath != "context: README.md" {
		t.Fatalf("expected migrated context path, got %#v", messages)
	}
}

func TestEnsureRecoversCorruptSQLiteStore(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := os.MkdirAll(filepath.Dir(store.Path()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.Path(), []byte("not-a-sqlite-database"), 0o644); err != nil {
		t.Fatal(err)
	}
	status, err := store.Ensure()
	if err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}
	if !strings.Contains(strings.ToLower(status.Message), "recovered corrupt metadata database") {
		t.Fatalf("expected recovery message, got %q", status.Message)
	}
	recoveryDir := filepath.Join(filepath.Dir(store.Path()), "recovery")
	entries, err := os.ReadDir(recoveryDir)
	if err != nil {
		t.Fatalf("expected recovery directory to exist: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected archived corrupt metadata file")
	}
	if _, err := os.Stat(store.Path()); err != nil {
		t.Fatalf("expected recreated sqlite store to exist: %v", err)
	}
}

func TestExportBackupCreatesZipBundle(t *testing.T) {
	store := mustStore(t)
	if err := store.SaveJob(jobssvc.Job{
		ID:        "job-backup-1",
		Kind:      "task",
		Label:     "backup",
		Status:    jobssvc.StatusSuccess,
		Message:   "ok",
		StartedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveJob returned error: %v", err)
	}
	backup, err := store.ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup returned error: %v", err)
	}
	if backup.Path == "" || backup.SizeBytes <= 0 {
		t.Fatalf("unexpected backup result: %#v", backup)
	}
	if _, err := os.Stat(backup.Path); err != nil {
		t.Fatalf("expected backup zip file to exist: %v", err)
	}
	archive, err := zip.OpenReader(backup.Path)
	if err != nil {
		t.Fatalf("OpenReader failed: %v", err)
	}
	defer archive.Close()
	names := make([]string, 0, len(archive.File))
	for _, file := range archive.File {
		names = append(names, strings.TrimSpace(file.Name))
	}
	required := []string{"nexusdesk.sqlite", "schema.sql", "sqlite-manifest.json", "backup-summary.json"}
	for _, name := range required {
		if !containsStringCaseInsensitive(names, name) {
			t.Fatalf("backup zip missing %q in %#v", name, names)
		}
	}
}

func TestExportWorkspaceStateBackupCreatesZipBundle(t *testing.T) {
	store := mustStore(t)
	workspaceRoot := filepath.Dir(filepath.Dir(filepath.Dir(store.Path())))
	artifactPath := filepath.Join(workspaceRoot, filepath.FromSlash(".nexusdesk/artifacts/task-runs/report.md"))
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("# Task report\n"), 0o644); err != nil {
		t.Fatalf("WriteFile artifact failed: %v", err)
	}

	configDir := filepath.Join(t.TempDir(), "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll config failed: %v", err)
	}
	settingsPath := filepath.Join(configDir, "settings.json")
	profilesPath := filepath.Join(configDir, "connector-profiles.json")
	secretsPath := profilesPath + ".secrets"
	if err := os.WriteFile(settingsPath, []byte(`{"provider":"ollama"}`), 0o600); err != nil {
		t.Fatalf("WriteFile settings failed: %v", err)
	}
	if err := os.WriteFile(profilesPath, []byte(`[]`), 0o600); err != nil {
		t.Fatalf("WriteFile profiles failed: %v", err)
	}
	if err := os.WriteFile(secretsPath, []byte(`{"id":"encrypted"}`), 0o600); err != nil {
		t.Fatalf("WriteFile secrets failed: %v", err)
	}

	backup, err := store.ExportWorkspaceStateBackup(WorkspaceStateBackupOptions{
		SettingsPath:          settingsPath,
		ConnectorProfilesPath: profilesPath,
	})
	if err != nil {
		t.Fatalf("ExportWorkspaceStateBackup returned error: %v", err)
	}
	if backup.Path == "" || backup.SizeBytes <= 0 {
		t.Fatalf("unexpected workspace state backup result: %#v", backup)
	}
	if _, err := os.Stat(backup.Path); err != nil {
		t.Fatalf("expected workspace state backup zip file to exist: %v", err)
	}
	archive, err := zip.OpenReader(backup.Path)
	if err != nil {
		t.Fatalf("OpenReader failed: %v", err)
	}
	defer archive.Close()
	names := make([]string, 0, len(archive.File))
	for _, file := range archive.File {
		names = append(names, strings.TrimSpace(file.Name))
	}
	required := []string{
		".nexusdesk/metadata/nexusdesk.sqlite",
		".nexusdesk/metadata/schema.sql",
		".nexusdesk/metadata/sqlite-manifest.json",
		".nexusdesk/artifacts/task-runs/report.md",
		"app-config/settings.json",
		"app-config/connector-profiles.json",
		"app-config/connector-profiles.secrets.json",
		"workspace-state-summary.json",
	}
	for _, name := range required {
		if !containsStringCaseInsensitive(names, name) {
			t.Fatalf("workspace state backup zip missing %q in %#v", name, names)
		}
	}
}

func containsStringCaseInsensitive(values []string, candidate string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func mustStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	return store
}
