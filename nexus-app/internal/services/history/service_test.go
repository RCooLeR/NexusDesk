package history

import (
	"strings"
	"testing"
	"time"

	artifactsSvc "nexusdesk/internal/services/artifacts"
	jobsSvc "nexusdesk/internal/services/jobs"
	metadataSvc "nexusdesk/internal/services/metadata"
)

func TestListMergesWorkspaceHistorySources(t *testing.T) {
	root := t.TempDir()
	metadataStore, err := metadataSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = metadataStore.Close() })
	if _, err := metadataStore.Ensure(); err != nil {
		t.Fatal(err)
	}
	artifactStore, err := artifactsSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	if err := metadataStore.SaveChatMessage(metadataSvc.ChatMessageRecord{Role: "user", Content: "Review data quality", CreatedAt: base}); err != nil {
		t.Fatalf("SaveChatMessage returned error: %v", err)
	}
	if err := metadataStore.SaveJob(jobsSvc.Job{ID: "job-0001", Kind: "task", Label: "go test", Status: jobsSvc.StatusSuccess, Message: "done", StartedAt: base.Add(time.Minute)}); err != nil {
		t.Fatalf("SaveJob returned error: %v", err)
	}
	run := metadataSvc.AgentRunRecord{Prompt: "Analyze", Status: "success", Message: "Finished", StartedAt: base.Add(2 * time.Minute)}
	run = metadataStore.NormalizeAgentRunRecord(run)
	if err := metadataStore.SaveAgentRun(run); err != nil {
		t.Fatalf("SaveAgentRun returned error: %v", err)
	}
	sqlRun := metadataStore.NormalizeSQLRunRecord(metadataSvc.SQLRunRecord{
		RelPath:     "data/sales.csv",
		SQL:         "select * from dataset limit 5",
		Engine:      "native-dataset-sql",
		Status:      "success",
		MatchedRows: 5,
		ShownRows:   5,
		StartedAt:   base.Add(3 * time.Minute),
		CompletedAt: base.Add(3 * time.Minute),
	})
	if err := metadataStore.SaveSQLRun(sqlRun); err != nil {
		t.Fatalf("SaveSQLRun returned error: %v", err)
	}
	if err := metadataStore.SaveDatasetDependency(metadataSvc.DatasetDependencyRecord{
		SourcePath:    "data/sales.csv",
		DependentKind: "sql-run",
		DependentRef:  sqlRun.ID,
		Relation:      "reads",
		CreatedAt:     base.Add(3 * time.Minute),
		UpdatedAt:     base.Add(3 * time.Minute),
	}); err != nil {
		t.Fatalf("SaveDatasetDependency returned error: %v", err)
	}
	artifact, err := artifactStore.WriteTaskRunReport(artifactsSvc.TaskRunReport{ID: "report", Label: "Task report", Command: "go test", Cwd: ".", Status: "success", StartedAt: base})
	if err != nil {
		t.Fatalf("WriteTaskRunReport returned error: %v", err)
	}
	if err := metadataStore.SaveArtifact(metadataSvc.ArtifactRecord{
		Kind:         artifact.Kind,
		Title:        artifact.Title,
		RelPath:      artifact.RelPath,
		MetadataPath: artifact.MetadataPath,
		Size:         artifact.Size,
		JobID:        artifact.JobID,
		TaskID:       artifact.TaskID,
		Source:       artifact.Source,
		SourcePaths:  artifact.SourcePaths,
		CreatedAt:    artifact.CreatedAt,
		GeneratedAt:  artifact.GeneratedAt,
	}); err != nil {
		t.Fatalf("SaveArtifact returned error: %v", err)
	}

	items, err := New(metadataStore, artifactStore).List(Options{Limit: 20})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	kinds := map[Kind]bool{}
	for _, item := range items {
		kinds[item.Kind] = true
	}
	for _, kind := range []Kind{KindChat, KindArtifact, KindJob, KindAgent, KindData} {
		if !kinds[kind] {
			t.Fatalf("missing history kind %q in %#v", kind, items)
		}
	}
	if items[0].Kind != KindArtifact && items[0].Kind != KindAgent {
		t.Fatalf("expected newest records first, got %#v", items)
	}
}

func TestListIncludesSQLRunsAndDependenciesAsDataHistory(t *testing.T) {
	root := t.TempDir()
	metadataStore, err := metadataSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = metadataStore.Close() })
	if _, err := metadataStore.Ensure(); err != nil {
		t.Fatal(err)
	}
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	run := metadataStore.NormalizeSQLRunRecord(metadataSvc.SQLRunRecord{
		RelPath:     "data/sales.csv",
		SQL:         "select channel from dataset",
		Status:      "success",
		MatchedRows: 2,
		ShownRows:   2,
		StartedAt:   started,
		CompletedAt: started.Add(time.Second),
	})
	if err := metadataStore.SaveSQLRun(run); err != nil {
		t.Fatalf("SaveSQLRun returned error: %v", err)
	}
	if err := metadataStore.SaveDatasetDependency(metadataSvc.DatasetDependencyRecord{
		SourcePath:    "data/sales.csv",
		DependentKind: "chart",
		DependentRef:  "artifacts/charts/sales.svg",
		Relation:      "visualizes",
		Metadata:      map[string]string{"mode": "line"},
		CreatedAt:     started,
		UpdatedAt:     started.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("SaveDatasetDependency returned error: %v", err)
	}

	items, err := New(metadataStore, nil).List(Options{Kind: KindData, Query: "sales", Limit: 10})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected sql run and dependency items, got %#v", items)
	}
	foundSQL := false
	foundDependency := false
	for _, item := range items {
		if item.Kind != KindData || len(item.SourcePaths) != 1 || item.SourcePaths[0] != "data/sales.csv" {
			t.Fatalf("unexpected data history item: %#v", item)
		}
		foundSQL = foundSQL || strings.Contains(item.Detail, "data/sql-run") && strings.Contains(item.Detail, "select channel")
		foundDependency = foundDependency || strings.Contains(item.Detail, "data/dependency") && strings.Contains(item.Detail, "visualizes")
	}
	if !foundSQL || !foundDependency {
		t.Fatalf("missing SQL/dependency detail in %#v", items)
	}
}

func TestListFiltersByKindAndQuery(t *testing.T) {
	root := t.TempDir()
	metadataStore, err := metadataSvc.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = metadataStore.Close() })
	if _, err := metadataStore.Ensure(); err != nil {
		t.Fatal(err)
	}
	if err := metadataStore.SaveChatMessage(metadataSvc.ChatMessageRecord{Role: "assistant", Content: "SQLite metadata is active.", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("SaveChatMessage returned error: %v", err)
	}
	if err := metadataStore.SaveJob(jobsSvc.Job{ID: "job-0001", Kind: "task", Label: "go test", Status: jobsSvc.StatusSuccess, Message: "done", StartedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("SaveJob returned error: %v", err)
	}

	items, err := New(metadataStore, nil).List(Options{Kind: KindChat, Query: "sqlite", Limit: 10})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 || items[0].Kind != KindChat || !strings.Contains(strings.ToLower(items[0].Summary), "sqlite") {
		t.Fatalf("unexpected filtered items: %#v", items)
	}
}
