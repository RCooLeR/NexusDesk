package datasets

import (
	"archive/zip"
	"bytes"
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

func TestProfileNDJSONProfilesLineObjects(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "events.ndjson", "{\"channel\":\"search\",\"spend\":12.5}\n{\"channel\":\"email\",\"spend\":4}\n")

	profile, err := New(nil).Profile(root, "events.ndjson")
	if err != nil {
		t.Fatalf("Profile returned error: %v", err)
	}
	if profile.Format != "NDJSON" || profile.JSONProfile == nil || profile.Rows != 2 || len(profile.Columns) != 2 {
		t.Fatalf("unexpected NDJSON profile: %#v", profile)
	}
	if profile.Columns[1].Name != "spend" || profile.Columns[1].Type != "number" {
		t.Fatalf("unexpected NDJSON spend profile: %#v", profile.Columns)
	}
}

func TestProfileLogDetectsLevels(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "app.log", "2026-05-27 INFO booted\n2026-05-27 ERROR failed\n")

	profile, err := New(nil).Profile(root, "app.log")
	if err != nil {
		t.Fatalf("Profile returned error: %v", err)
	}
	if profile.Format != "LOG" || profile.Rows != 2 || len(profile.Columns) != 3 {
		t.Fatalf("unexpected log profile: %#v", profile)
	}
	if profile.Columns[1].Name != "level" || len(profile.Notes) == 0 {
		t.Fatalf("expected log level notes: %#v", profile)
	}
}

func TestProfileParquetReadsFooterMetadata(t *testing.T) {
	root := t.TempDir()
	writeTestBytes(t, root, "data/sample.parquet", makeParquetStub(12))

	profile, err := New(nil).Profile(root, "data/sample.parquet")
	if err != nil {
		t.Fatalf("Profile returned error: %v", err)
	}
	if profile.Format != "PARQUET" || profile.Size == 0 || len(profile.Notes) == 0 {
		t.Fatalf("unexpected parquet profile: %#v", profile)
	}
}

func TestProfileXLSXReturnsFirstSheetSummary(t *testing.T) {
	root := t.TempDir()
	writeTestBytes(t, root, "data/campaigns.xlsx", makeDatasetXLSX(t))

	profile, err := New(nil).Profile(root, "data/campaigns.xlsx")
	if err != nil {
		t.Fatalf("Profile returned error: %v", err)
	}
	if profile.Format != "XLSX" || profile.Sheet != "Campaigns" || profile.Rows != 2 || len(profile.Columns) != 2 {
		t.Fatalf("unexpected XLSX profile: %#v", profile)
	}
	if profile.Columns[1].Name != "spend" || profile.Columns[1].Type != "number" {
		t.Fatalf("unexpected XLSX spend profile: %#v", profile.Columns[1])
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

func TestQueryNDJSONObjects(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "events.jsonl", "{\"channel\":\"search\",\"spend\":12.5}\n{\"channel\":\"email\",\"spend\":4}\n")

	result, err := New(nil).Query(root, "events.jsonl", "spend>5")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if result.Format != "NDJSON" || result.MatchedRows != 1 || result.Rows[0][0] != "search" {
		t.Fatalf("unexpected NDJSON query result: %#v", result)
	}
}

func TestQueryLogRows(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "app.log", "INFO ready\nERROR failed\n")

	result, err := New(nil).Query(root, "app.log", "level=error")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if result.Format != "LOG" || result.MatchedRows != 1 || result.Rows[0][1] != "error" {
		t.Fatalf("unexpected log query result: %#v", result)
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

func TestQueryXLSXUsesFirstSheetRows(t *testing.T) {
	root := t.TempDir()
	writeTestBytes(t, root, "data/campaigns.xlsx", makeDatasetXLSX(t))

	result, err := New(nil).Query(root, "data/campaigns.xlsx", "channel=search")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if result.Format != "XLSX" || result.MatchedRows != 1 || result.Rows[0][0] != "search" {
		t.Fatalf("unexpected XLSX query result: %#v", result)
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

func writeTestBytes(t *testing.T, root string, relPath string, content []byte) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func makeDatasetXLSX(t *testing.T) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	writeDatasetZipEntry(t, writer, "xl/workbook.xml", `<workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Campaigns" r:id="rId1"/></sheets></workbook>`)
	writeDatasetZipEntry(t, writer, "xl/_rels/workbook.xml.rels", `<Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>`)
	writeDatasetZipEntry(t, writer, "xl/sharedStrings.xml", `<sst><si><t>channel</t></si><si><t>spend</t></si><si><t>search</t></si><si><t>email</t></si></sst>`)
	writeDatasetZipEntry(t, writer, "xl/worksheets/sheet1.xml", `<worksheet><sheetData><row><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row><row><c r="A2" t="s"><v>2</v></c><c r="B2"><v>12.5</v></c></row><row><c r="A3" t="s"><v>3</v></c><c r="B3"><v>4</v></c></row></sheetData></worksheet>`)
	if err := writer.Close(); err != nil {
		t.Fatalf("close xlsx zip: %v", err)
	}
	return output.Bytes()
}

func writeDatasetZipEntry(t *testing.T, writer *zip.Writer, name string, content string) {
	t.Helper()
	file, err := writer.Create(name)
	if err != nil {
		t.Fatalf("create zip entry %s: %v", name, err)
	}
	if _, err := file.Write([]byte(content)); err != nil {
		t.Fatalf("write zip entry %s: %v", name, err)
	}
}

func makeParquetStub(footerLength uint32) []byte {
	content := []byte("PAR1stub-footer")
	content = append(content, byte(footerLength), byte(footerLength>>8), byte(footerLength>>16), byte(footerLength>>24))
	content = append(content, []byte("PAR1")...)
	return content
}
