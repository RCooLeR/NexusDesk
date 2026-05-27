package datasets

import (
	"strings"
	"testing"
)

func TestQuerySQLFiltersOrdersLimitsAndProjects(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "sales.csv", "channel,spend,active\nsearch,12.5,true\nemail,4,false\nsearch,20,true\n")

	result, err := New(nil).QuerySQL(root, "sales.csv", "select channel, spend from dataset where channel = 'search' order by spend desc limit 1")
	if err != nil {
		t.Fatalf("QuerySQL returned error: %v", err)
	}
	if result.Engine != nativeDatasetSQLEngine || result.Format != "CSV" || result.SQL == "" || len(result.Plan) == 0 {
		t.Fatalf("unexpected SQL metadata: %#v", result)
	}
	if len(result.Columns) != 2 || result.Columns[0] != "channel" || result.Columns[1] != "spend" {
		t.Fatalf("unexpected projection: %#v", result.Columns)
	}
	if len(result.Rows) != 1 || result.Rows[0][0] != "search" || result.Rows[0][1] != "20" {
		t.Fatalf("unexpected rows: %#v", result.Rows)
	}
}

func TestQuerySQLRejectsMutationKeywords(t *testing.T) {
	if _, err := New(nil).QuerySQL(t.TempDir(), "sales.csv", "delete from dataset"); err == nil || !strings.Contains(err.Error(), "SELECT") {
		t.Fatalf("expected mutation rejection, got %v", err)
	}
}

func TestQuerySQLRequiresSelectedSource(t *testing.T) {
	if _, err := New(nil).QuerySQL(t.TempDir(), "sales.csv", "select * from other"); err == nil || !strings.Contains(err.Error(), "selected dataset") {
		t.Fatalf("expected selected source rejection, got %v", err)
	}
}

func TestQuerySQLRejectsCompoundWhereUntilNotebookEngine(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "sales.csv", "channel,spend,active\nsearch,12.5,true\nemail,4,false\nsearch,20,true\n")

	_, err := New(nil).QuerySQL(root, "sales.csv", "select * from dataset where channel = 'search' and spend > 10")
	if err == nil || !strings.Contains(err.Error(), "one WHERE predicate") {
		t.Fatalf("expected compound WHERE rejection, got %v", err)
	}
}

func TestQuerySQLRejectsUnknownProjectionColumn(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "sales.csv", "channel,spend\nsearch,12.5\n")

	_, err := New(nil).QuerySQL(root, "sales.csv", "select missing from dataset")
	if err == nil || !strings.Contains(err.Error(), "projection column") {
		t.Fatalf("expected projection column rejection, got %v", err)
	}
}
