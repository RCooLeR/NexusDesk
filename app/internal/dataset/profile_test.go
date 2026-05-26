package dataset

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildPersistsCSVProfile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "data/report.csv", "name,value\nalpha,10\nbeta,20\n")

	profile, err := Build(root, "data/report.csv")
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if profile.Kind != "csv" {
		t.Fatalf("expected csv profile, got %s", profile.Kind)
	}
	if profile.Rows != 2 || profile.Columns != 2 {
		t.Fatalf("expected 2 rows and 2 columns, got %d/%d", profile.Rows, profile.Columns)
	}
	if len(profile.Profiles) != 2 {
		t.Fatalf("expected column profiles, got %d", len(profile.Profiles))
	}

	profiles, err := List(root)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(profiles) != 1 || profiles[0].RelPath != "data/report.csv" {
		t.Fatalf("expected persisted profile, got %#v", profiles)
	}
}

func TestBuildPersistsTSVProfile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "data/report.tsv", "name\tvalue\nalpha\t10\nbeta\t20\n")

	profile, err := Build(root, "data/report.tsv")
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if profile.Kind != "tsv" {
		t.Fatalf("expected tsv profile, got %s", profile.Kind)
	}
	if profile.Rows != 2 || profile.Columns != 2 {
		t.Fatalf("expected 2 rows and 2 columns, got %d/%d", profile.Rows, profile.Columns)
	}
	if len(profile.Profiles) != 2 || profile.Profiles[1].Type != "integer" {
		t.Fatalf("expected integer column profile, got %#v", profile.Profiles)
	}
}

func TestBuildPersistsJSONProfile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "data/report.json", `[{"name":"alpha","value":10},{"name":"beta","value":20}]`)

	profile, err := Build(root, "data/report.json")
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if profile.Kind != "json" {
		t.Fatalf("expected json profile, got %s", profile.Kind)
	}
	if profile.Rows != 2 || profile.Columns != 2 {
		t.Fatalf("expected 2 rows and 2 columns, got %d/%d", profile.Rows, profile.Columns)
	}
}

func TestBuildPersistsNDJSONProfile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "data/report.ndjson", "{\"name\":\"alpha\",\"value\":10}\n{\"name\":\"beta\",\"value\":20}\n")

	profile, err := Build(root, "data/report.ndjson")
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if profile.Kind != "ndjson" {
		t.Fatalf("expected ndjson profile, got %s", profile.Kind)
	}
	if profile.Rows != 2 || profile.Columns != 2 {
		t.Fatalf("expected 2 rows and 2 columns, got %d/%d", profile.Rows, profile.Columns)
	}
}

func TestBuildInspectsXLSXSheets(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "data", "workbook.xlsx")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	createXLSX(t, path)

	profile, err := Build(root, "data/workbook.xlsx")
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if profile.Kind != "xlsx" {
		t.Fatalf("expected xlsx profile, got %s", profile.Kind)
	}
	if len(profile.Sheets) != 2 || profile.Sheets[0] != "Summary" || profile.Sheets[1] != "Data" {
		t.Fatalf("unexpected sheets: %#v", profile.Sheets)
	}
}

func TestBuildRejectsTraversal(t *testing.T) {
	if _, err := Build(t.TempDir(), "../outside.csv"); err == nil {
		t.Fatal("expected traversal to be rejected")
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

func createXLSX(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()

	entry, err := writer.Create("xl/workbook.xml")
	if err != nil {
		t.Fatalf("Create workbook entry failed: %v", err)
	}
	_, err = entry.Write([]byte(`<workbook><sheets><sheet name="Summary"/><sheet name="Data"/></sheets></workbook>`))
	if err != nil {
		t.Fatalf("Write workbook entry failed: %v", err)
	}
}
