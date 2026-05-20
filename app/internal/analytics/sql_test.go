package analytics

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestQueryCSVSQLRunsReadOnlyProjection(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "data/campaigns.csv", "campaign,spend,channel\nA,10,Paid\nB,30,Organic\nC,20,Paid\n")

	result, err := QueryCSVSQL(root, SQLQueryRequest{
		RelPath: "data/campaigns.csv",
		SQL:     "select campaign,spend from campaigns where spend > 10 order by spend desc limit 1",
	})
	if err != nil {
		t.Fatalf("QueryCSVSQL returned error: %v", err)
	}

	if result.Engine != "duckdb-compatible-csv" {
		t.Fatalf("unexpected engine: %s", result.Engine)
	}
	if len(result.Rows) != 1 || result.Rows[0][0] != "B" || result.Columns[1] != "spend" {
		t.Fatalf("unexpected SQL rows: %#v", result)
	}
}

func TestQueryCSVSQLRejectsMutation(t *testing.T) {
	_, err := QueryCSVSQL(t.TempDir(), SQLQueryRequest{
		RelPath: "data/campaigns.csv",
		SQL:     "delete from campaigns",
	})
	if err == nil {
		t.Fatal("expected mutation to be rejected")
	}
}

func TestQueryCSVSQLRejectsMultiStatementSQL(t *testing.T) {
	_, err := QueryCSVSQL(t.TempDir(), SQLQueryRequest{
		RelPath: "data/campaigns.csv",
		SQL:     "select campaign from campaigns; select spend from campaigns limit 1",
	})
	if err == nil || !strings.Contains(err.Error(), "single SQL statement") {
		t.Fatalf("expected multi-statement query to fail, got %v", err)
	}
}

func TestQueryCSVSQLAllowsQuotedSemicolon(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "data/campaigns.csv", "campaign,note\nA,\"A;B\"\nB,B\n")
	result, err := QueryCSVSQL(root, SQLQueryRequest{
		RelPath: "data/campaigns.csv",
		SQL:     "select campaign from campaigns where note = 'A;B'",
	})
	if err != nil {
		t.Fatalf("QueryCSVSQL returned error: %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0][0] != "A" {
		t.Fatalf("unexpected query result: %#v", result)
	}
}

func TestQueryCSVSQLReturnsAllMatchCountWithCappedPreview(t *testing.T) {
	root := t.TempDir()
	lines := []string{"campaign,spend"}
	for index := 1; index <= 150; index++ {
		lines = append(lines, "campaign-"+strconv.Itoa(index)+","+strconv.Itoa(index*10))
	}
	writeFile(t, root, "data/campaigns.csv", strings.Join(lines, "\n"))

	result, err := QueryCSVSQL(root, SQLQueryRequest{
		RelPath: "data/campaigns.csv",
		SQL:     "select campaign, spend from campaigns where spend >= 10",
	})
	if err != nil {
		t.Fatalf("QueryCSVSQL returned error: %v", err)
	}
	if result.TotalRows != 150 {
		t.Fatalf("expected total rows to preserve full match count, got %d", result.TotalRows)
	}
	if result.MatchedRows != 150 {
		t.Fatalf("expected matched rows to preserve full match count, got %d", result.MatchedRows)
	}
	if len(result.Rows) != 50 {
		t.Fatalf("expected preview cap to limit returned rows to 50, got %d", len(result.Rows))
	}
	if !strings.Contains(result.Message, "showing 50") {
		t.Fatalf("expected message to include showing count, got %q", result.Message)
	}
}

func writeFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}
