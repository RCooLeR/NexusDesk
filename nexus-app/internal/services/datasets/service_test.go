package datasets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfileCSVReturnsColumnSummaries(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "people.csv", "name,age,active\nAda,37,true\nLinus,55,false\nGrace,,true\n")

	profile, err := New(nil).Profile(root, "people.csv")
	if err != nil {
		t.Fatalf("Profile returned error: %v", err)
	}
	if profile.Format != "CSV" || profile.Rows != 3 || len(profile.Columns) != 3 {
		t.Fatalf("unexpected profile summary: %#v", profile)
	}
	if profile.Columns[1].Name != "age" || profile.Columns[1].Type != "integer" || profile.Columns[1].Empty != 1 {
		t.Fatalf("unexpected age profile: %#v", profile.Columns[1])
	}
	if profile.Columns[2].Type != "boolean" {
		t.Fatalf("unexpected active type: %#v", profile.Columns[2])
	}
}

func TestProfileTSVDetectsFormat(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "metrics.tsv", "day\tcount\n2026-05-27\t10\n")

	profile, err := New(nil).Profile(root, "metrics.tsv")
	if err != nil {
		t.Fatalf("Profile returned error: %v", err)
	}
	if profile.Format != "TSV" || profile.Columns[0].Type != "date" {
		t.Fatalf("unexpected TSV profile: %#v", profile)
	}
}

func TestProfileJSONProfilesArrayObjects(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "events.json", `[{"channel":"search","spend":12.5},{"channel":"email","spend":4}]`)

	profile, err := New(nil).Profile(root, "events.json")
	if err != nil {
		t.Fatalf("Profile returned error: %v", err)
	}
	if profile.Format != "JSON" || profile.JSONProfile == nil || profile.JSONProfile.TopLevel != "array" || profile.Rows != 2 {
		t.Fatalf("unexpected JSON summary: %#v", profile)
	}
	if len(profile.Columns) != 2 || profile.Columns[0].Name != "channel" || profile.Columns[1].Type != "number" {
		t.Fatalf("unexpected JSON columns: %#v", profile.Columns)
	}
}

func TestProfileRejectsUnsupportedFile(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "notes.txt", "hello")

	if _, err := New(nil).Profile(root, "notes.txt"); err == nil {
		t.Fatal("expected unsupported file error")
	}
}

func TestQueryCSVFiltersOrdersAndLimitsRows(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "sales.csv", "channel,spend\nsearch,12.5\nemail,4\nsearch,20\nsocial,8\n")

	result, err := New(nil).Query(root, "sales.csv", "channel=search order by spend desc limit 1")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if result.Format != "CSV" || result.TotalRows != 4 || result.MatchedRows != 2 || len(result.Rows) != 1 {
		t.Fatalf("unexpected query summary: %#v", result)
	}
	if result.Rows[0][0] != "search" || result.Rows[0][1] != "20" {
		t.Fatalf("expected highest search spend first, got %#v", result.Rows)
	}
}

func TestQueryTSVSupportsNumericComparisons(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "metrics.tsv", "name\tcount\nsmall\t2\nlarge\t10\n")

	result, err := New(nil).Query(root, "metrics.tsv", "count>=10")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if result.Format != "TSV" || result.MatchedRows != 1 || result.Rows[0][0] != "large" {
		t.Fatalf("unexpected TSV query result: %#v", result)
	}
}

func TestQueryJSONArrayObjects(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "events.json", `[{"channel":"search","spend":12.5},{"channel":"email","spend":4}]`)

	result, err := New(nil).Query(root, "events.json", "spend>5")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if result.Format != "JSON" || result.MatchedRows != 1 || result.Rows[0][0] != "search" {
		t.Fatalf("unexpected JSON query result: %#v", result)
	}
}

func TestQueryGlobalSearch(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "people.csv", "name,role\nAda,engineer\nGrace,admiral\n")

	result, err := New(nil).Query(root, "people.csv", "adm")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if result.MatchedRows != 1 || result.Rows[0][0] != "Grace" {
		t.Fatalf("unexpected global search result: %#v", result)
	}
}

func writeTestFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
