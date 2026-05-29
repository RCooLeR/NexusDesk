package spreadsheets

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestParseXLSXReadsSharedStringsInlineStringsAndSheets(t *testing.T) {
	content := buildXLSX(t, map[string]string{
		"xl/workbook.xml":            `<workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Summary" r:id="rId1"/><sheet name="Data" r:id="rId2"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Target="worksheets/sheet2.xml"/></Relationships>`,
		"xl/sharedStrings.xml":       `<sst><si><t>Name</t></si><si><t>Spend</t></si><si><t>Search</t></si></sst>`,
		"xl/worksheets/sheet1.xml":   `<worksheet><sheetData><row><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row><row><c r="A2" t="s"><v>2</v></c><c r="B2"><v>12.5</v></c></row></sheetData></worksheet>`,
		"xl/worksheets/sheet2.xml":   `<worksheet><sheetData><row><c r="A1" t="inlineStr"><is><t>Channel</t></is></c></row><row><c r="A2" t="inlineStr"><is><t>Paid</t></is></c></row></sheetData></worksheet>`,
		"docProps/app.xml":           `<Properties/>`,
	})

	workbook, err := ParseXLSX(content, Options{MaxRows: 10, MaxColumns: 10})
	if err != nil {
		t.Fatalf("ParseXLSX() error = %v", err)
	}
	if len(workbook.Sheets) != 2 {
		t.Fatalf("sheets = %d, want 2", len(workbook.Sheets))
	}
	if got := workbook.Sheets[0].Rows[1][0]; got != "Search" {
		t.Fatalf("shared string cell = %q, want Search", got)
	}
	if got := workbook.Sheets[1].Rows[0][0]; got != "Channel" {
		t.Fatalf("inline string cell = %q, want Channel", got)
	}
}

func TestParseXLSXMarksTruncatedRows(t *testing.T) {
	content := buildXLSX(t, map[string]string{
		"xl/workbook.xml":            `<workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Data" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/worksheets/sheet1.xml":   `<worksheet><sheetData><row><c r="A1"><v>1</v></c></row><row><c r="A2"><v>2</v></c></row></sheetData></worksheet>`,
	})
	workbook, err := ParseXLSX(content, Options{MaxRows: 1, MaxColumns: 10})
	if err != nil {
		t.Fatalf("ParseXLSX() error = %v", err)
	}
	if !workbook.Sheets[0].Truncated {
		t.Fatal("expected truncated sheet")
	}
}

func TestParseXLSXRejectsOversizedSharedStrings(t *testing.T) {
	content := buildXLSX(t, map[string]string{
		"xl/workbook.xml":            `<workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Data" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/sharedStrings.xml":       `<sst><si><t>` + strings.Repeat("a", int(xlsxMaxMetadataXMLBytes)+1) + `</t></si></sst>`,
		"xl/worksheets/sheet1.xml":   `<worksheet><sheetData><row><c r="A1" t="s"><v>0</v></c></row></sheetData></worksheet>`,
	})

	_, err := ParseXLSX(content, Options{MaxRows: 10, MaxColumns: 10})
	if err == nil || !strings.Contains(err.Error(), "sharedStrings.xml") {
		t.Fatalf("expected shared string safety cap error, got %v", err)
	}
}

func buildXLSX(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("Create(%s) error = %v", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("Write(%s) error = %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return buffer.Bytes()
}
