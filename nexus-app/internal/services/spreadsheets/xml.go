package spreadsheets

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type workbookXML struct {
	Sheets workbookSheetsXML `xml:"sheets"`
}

type workbookSheetsXML struct {
	Items []workbookSheetXML `xml:"sheet"`
}

type workbookSheetXML struct {
	Name  string `xml:"name,attr"`
	RelID string `xml:"id,attr"`
}

type relationshipsXML struct {
	Items []relationshipXML `xml:"Relationship"`
}

type relationshipXML struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
}

type worksheetRows struct {
	Rows      [][]string
	Truncated bool
}

func parseWorkbookXML(file *zip.File) (workbookXML, error) {
	body, err := readZipFile(file, xlsxMaxMetadataXMLBytes)
	if err != nil {
		return workbookXML{}, err
	}
	var parsed workbookXML
	err = xml.Unmarshal(body, &parsed)
	return parsed, err
}

func parseRelationships(file *zip.File, baseDir string) (map[string]string, error) {
	if file == nil {
		return map[string]string{}, nil
	}
	body, err := readZipFile(file, xlsxMaxMetadataXMLBytes)
	if err != nil {
		return nil, err
	}
	var parsed relationshipsXML
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	mapped := map[string]string{}
	for _, rel := range parsed.Items {
		if rel.ID == "" || rel.Target == "" {
			continue
		}
		mapped[rel.ID] = normalizeTarget(baseDir, rel.Target)
	}
	return mapped, nil
}

func normalizeTarget(baseDir string, target string) string {
	target = strings.TrimSpace(target)
	target = strings.TrimPrefix(target, "/")
	if strings.HasPrefix(target, baseDir+"/") {
		return filepath.ToSlash(target)
	}
	return filepath.ToSlash(path.Clean(path.Join(baseDir, target)))
}

func parseSharedStrings(file *zip.File) ([]string, error) {
	if file == nil {
		return nil, nil
	}
	body, err := readZipFile(file, xlsxMaxMetadataXMLBytes)
	if err != nil {
		return nil, err
	}

	decoder := xml.NewDecoder(bytes.NewReader(body))
	values := []string{}
	var builder strings.Builder
	inString := false
	inText := false
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return values, nil
			}
			return nil, err
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "si" {
				inString = true
				builder.Reset()
			}
			if inString && value.Name.Local == "t" {
				inText = true
			}
		case xml.EndElement:
			if value.Name.Local == "t" {
				inText = false
			}
			if value.Name.Local == "si" {
				inString = false
				values = append(values, builder.String())
			}
		case xml.CharData:
			if inText {
				builder.Write([]byte(value))
			}
		}
	}
}

func parseWorksheet(file *zip.File, sharedStrings []string, options Options) (worksheetRows, error) {
	body, err := readZipFile(file, xlsxMaxWorksheetXMLBytes)
	if err != nil {
		return worksheetRows{}, err
	}

	decoder := xml.NewDecoder(bytes.NewReader(body))
	rows := [][]string{}
	var currentRow []string
	var currentCell cellState
	inCell := false
	inValue := false
	inInlineText := false
	truncated := false
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return worksheetRows{Rows: rows, Truncated: truncated}, nil
			}
			return worksheetRows{}, err
		}
		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "row":
				currentRow = []string{}
			case "c":
				inCell = true
				currentCell = newCellState(value)
			case "v":
				if inCell {
					inValue = true
				}
			case "t":
				if inCell && currentCell.Kind == "inlineStr" {
					inInlineText = true
				}
			}
		case xml.EndElement:
			switch value.Name.Local {
			case "v":
				inValue = false
			case "t":
				inInlineText = false
			case "c":
				inCell = false
				if len(rows) < options.MaxRows {
					currentRow, truncated = appendCell(currentRow, currentCell, sharedStrings, options.MaxColumns, truncated)
				}
			case "row":
				if len(rows) < options.MaxRows {
					rows = append(rows, trimTrailingEmpty(currentRow))
				} else {
					truncated = true
				}
				currentRow = nil
			}
		case xml.CharData:
			if inValue {
				currentCell.Value += string(value)
			}
			if inInlineText {
				currentCell.Inline += string(value)
			}
		}
	}
}

type cellState struct {
	Ref    string
	Kind   string
	Value  string
	Inline string
}

func newCellState(start xml.StartElement) cellState {
	state := cellState{}
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "r":
			state.Ref = attr.Value
		case "t":
			state.Kind = attr.Value
		}
	}
	return state
}

func appendCell(row []string, cell cellState, sharedStrings []string, maxColumns int, truncated bool) ([]string, bool) {
	column := cellColumnIndex(cell.Ref)
	if column < 0 {
		column = len(row)
	}
	if column >= maxColumns {
		return row, true
	}
	for len(row) <= column {
		row = append(row, "")
	}
	row[column] = cellText(cell, sharedStrings)
	return row, truncated
}

func cellText(cell cellState, sharedStrings []string) string {
	switch cell.Kind {
	case "s":
		index, err := strconv.Atoi(strings.TrimSpace(cell.Value))
		if err == nil && index >= 0 && index < len(sharedStrings) {
			return sharedStrings[index]
		}
	case "inlineStr":
		return cell.Inline
	case "b":
		if strings.TrimSpace(cell.Value) == "1" {
			return "true"
		}
		if strings.TrimSpace(cell.Value) == "0" {
			return "false"
		}
	}
	return strings.TrimSpace(cell.Value)
}

func cellColumnIndex(ref string) int {
	index := 0
	seen := false
	for _, char := range ref {
		if char >= 'A' && char <= 'Z' {
			index = index*26 + int(char-'A'+1)
			seen = true
			continue
		}
		if char >= 'a' && char <= 'z' {
			index = index*26 + int(char-'a'+1)
			seen = true
			continue
		}
		break
	}
	if !seen {
		return -1
	}
	return index - 1
}

func trimTrailingEmpty(row []string) []string {
	for len(row) > 0 && strings.TrimSpace(row[len(row)-1]) == "" {
		row = row[:len(row)-1]
	}
	return row
}
