package artifacts

import (
	"strings"
	"testing"
)

func TestWriteDatasetQueryAndSQLArtifacts(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	queryArtifact, err := store.WriteDatasetQueryCSVArtifact(DatasetQueryReport{
		SourcePath:  "data/sales.csv",
		Query:       "channel=search",
		Format:      "CSV",
		Columns:     []string{"channel", "spend"},
		Rows:        [][]string{{"search", "42"}},
		TotalRows:   2,
		MatchedRows: 1,
		Message:     "1 row matched.",
	})
	if err != nil {
		t.Fatalf("WriteDatasetQueryCSVArtifact returned error: %v", err)
	}
	if queryArtifact.Kind != "dataset-query-csv" || !strings.Contains(queryArtifact.Source, "channel=search") {
		t.Fatalf("unexpected dataset query artifact: %#v", queryArtifact)
	}
	queryText, err := store.ReadArtifactText(queryArtifact.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText query returned error: %v", err)
	}
	if !strings.Contains(queryText, "channel,spend") || !strings.Contains(queryText, "search,42") {
		t.Fatalf("unexpected query CSV:\n%s", queryText)
	}

	sqlArtifact, err := store.WriteDatasetSQLMarkdownArtifact(DatasetSQLReport{
		SourcePath:  "data/sales.csv",
		SQL:         "select channel, spend from dataset where channel = 'search'",
		Engine:      "native-dataset-sql",
		Columns:     []string{"channel", "spend"},
		Rows:        [][]string{{"search", "42"}},
		TotalRows:   2,
		MatchedRows: 1,
		ShownRows:   1,
		Plan:        []string{"Read selected dataset."},
		Message:     "Native SQL completed.",
	})
	if err != nil {
		t.Fatalf("WriteDatasetSQLMarkdownArtifact returned error: %v", err)
	}
	if sqlArtifact.Kind != "dataset-sql-report" || !strings.Contains(sqlArtifact.Source, "native-dataset-sql") {
		t.Fatalf("unexpected dataset SQL artifact: %#v", sqlArtifact)
	}
	sqlText, err := store.ReadArtifactText(sqlArtifact.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText SQL returned error: %v", err)
	}
	for _, expected := range []string{"# Dataset SQL Report", "```sql", "Read selected dataset.", "| channel | spend |"} {
		if !strings.Contains(sqlText, expected) {
			t.Fatalf("expected SQL artifact to contain %q:\n%s", expected, sqlText)
		}
	}
}

func TestDatasetQueryCSVRequiresColumns(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.WriteDatasetQueryCSVArtifact(DatasetQueryReport{SourcePath: "data.csv"}); err == nil {
		t.Fatal("expected dataset query CSV without columns to fail")
	}
}

func TestWriteDatasetSummaryMarkdownArtifact(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.WriteDatasetSummaryMarkdownArtifact(DatasetSummaryReport{
		SourcePath: "data/sales.csv",
		Format:     "CSV",
		MediaType:  "text/csv",
		Size:       128,
		Rows:       2,
		Columns: []DatasetSummaryColumnReport{
			{Name: "channel", Type: "text", NonEmpty: 2, Samples: []string{"search", "email"}},
			{Name: "spend", Type: "integer", NonEmpty: 2, Samples: []string{"42", "7"}},
		},
		Truncated: true,
		Notes:     []string{"Profile uses bounded preview rows."},
	})
	if err != nil {
		t.Fatalf("WriteDatasetSummaryMarkdownArtifact returned error: %v", err)
	}
	if artifact.Kind != "dataset-summary" || !strings.Contains(artifact.RelPath, "/dataset-summaries/") {
		t.Fatalf("unexpected dataset summary artifact: %#v", artifact)
	}
	text, err := store.ReadArtifactText(artifact.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText summary returned error: %v", err)
	}
	for _, expected := range []string{"# Dataset Summary - sales.csv", "Format:** CSV", "| channel | text |", "| spend | integer |", "Which segments explain the largest values", "bounded preview"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected summary artifact to contain %q:\n%s", expected, text)
		}
	}
	metadata, _ := store.readMetadata(artifact.RelPath)
	if metadata.Kind != "dataset-summary" || metadata.ContextRelPath != "data/sales.csv" || len(metadata.SourcePaths) != 1 {
		t.Fatalf("unexpected dataset summary metadata: %#v", metadata)
	}
}

func TestDatasetSummaryRequiresColumns(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.WriteDatasetSummaryMarkdownArtifact(DatasetSummaryReport{SourcePath: "data.csv"}); err == nil {
		t.Fatal("expected dataset summary without columns to fail")
	}
}
