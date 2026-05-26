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

func TestBuildInspectsXLSXWorkbookMetadata(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "data", "workbook.xlsx")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	createRichXLSX(t, path)

	profile, err := Build(root, "data/workbook.xlsx")
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if profile.Rows != 13 || profile.Columns != 3 {
		t.Fatalf("expected workbook row/column summary 13/3, got %d/%d", profile.Rows, profile.Columns)
	}
	if profile.Workbook.FormulaCount != 2 {
		t.Fatalf("expected 2 formulas, got %d", profile.Workbook.FormulaCount)
	}
	if len(profile.Workbook.NamedRanges) != 1 || profile.Workbook.NamedRanges[0] != "RevenueRange=Data!$B$2:$B$10" {
		t.Fatalf("unexpected named ranges: %#v", profile.Workbook.NamedRanges)
	}
	if len(profile.Workbook.TableRanges) != 1 || profile.Workbook.TableRanges[0].Name != "Campaigns" || profile.Workbook.TableRanges[0].Sheet != "Data" || profile.Workbook.TableRanges[0].Ref != "A1:C10" {
		t.Fatalf("unexpected table ranges: %#v", profile.Workbook.TableRanges)
	}
	if len(profile.Workbook.PivotTables) != 1 || profile.Workbook.PivotTables[0] != "PivotTable1" {
		t.Fatalf("unexpected pivot tables: %#v", profile.Workbook.PivotTables)
	}
	dataSheet := profile.Workbook.Sheets[1]
	if dataSheet.Name != "Data" || dataSheet.Dimension != "A1:C10" || dataSheet.Rows != 10 || dataSheet.Columns != 3 || dataSheet.FormulaCount != 2 || dataSheet.TableCount != 1 {
		t.Fatalf("unexpected data sheet info: %#v", dataSheet)
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

func createRichXLSX(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()

	writeZipEntry(t, writer, "xl/workbook.xml", `<workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Summary" r:id="rId1"/><sheet name="Data" r:id="rId2"/></sheets><definedNames><definedName name="RevenueRange">Data!$B$2:$B$10</definedName></definedNames></workbook>`)
	writeZipEntry(t, writer, "xl/_rels/workbook.xml.rels", `<Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Target="worksheets/sheet2.xml"/></Relationships>`)
	writeZipEntry(t, writer, "xl/worksheets/sheet1.xml", `<worksheet><dimension ref="A1:B3"/><sheetData><row><c r="A1"/><c r="B1"/></row><row><c r="A2"/></row><row><c r="B3"/></row></sheetData></worksheet>`)
	writeZipEntry(t, writer, "xl/worksheets/sheet2.xml", `<worksheet><dimension ref="A1:C10"/><sheetData><row><c r="A1"/><c r="B1"/><c r="C1"><f>SUM(B2:B9)</f></c></row><row><c r="A2"/><c r="B2"/><c r="C2"><f>B2*2</f></c></row></sheetData><tableParts count="1"><tablePart r:id="rTable1"/></tableParts></worksheet>`)
	writeZipEntry(t, writer, "xl/worksheets/_rels/sheet2.xml.rels", `<Relationships><Relationship Id="rTable1" Target="../tables/table1.xml"/></Relationships>`)
	writeZipEntry(t, writer, "xl/tables/table1.xml", `<table name="Table1" displayName="Campaigns" ref="A1:C10"/>`)
	writeZipEntry(t, writer, "xl/pivotTables/pivotTable1.xml", `<pivotTableDefinition name="PivotTable1"/>`)
}

func writeZipEntry(t *testing.T, writer *zip.Writer, name string, content string) {
	t.Helper()
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatalf("Create %s entry failed: %v", name, err)
	}
	if _, err := entry.Write([]byte(content)); err != nil {
		t.Fatalf("Write %s entry failed: %v", name, err)
	}
}
