package workspace

import (
	"encoding/csv"
	"io"
	"path/filepath"
	"strings"

	"nexusdesk/internal/domain"
	"nexusdesk/internal/services/spreadsheets"
)

const (
	tablePreviewMaxRows    = 50
	tablePreviewMaxColumns = 30
)

func decodeTable(content []byte, relPath string) (string, string, *domain.TablePreview, error) {
	if strings.EqualFold(filepath.Ext(relPath), ".xlsx") {
		return decodeXLSXTable(content)
	}
	text, encoding, err := decodeText(content)
	if err != nil {
		return "", "", nil, err
	}
	delimiter := tableDelimiter(relPath)
	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	records, truncated, err := readTableRecords(reader)
	if err != nil {
		return "", "", nil, err
	}
	table := &domain.TablePreview{
		Headers:   tableHeaders(records),
		Rows:      tableRows(records),
		Delimiter: string(delimiter),
		Truncated: truncated,
	}
	return text, encoding, table, nil
}

func decodeXLSXTable(content []byte) (string, string, *domain.TablePreview, error) {
	workbook, err := spreadsheets.ParseXLSX(content, spreadsheets.DefaultOptions())
	if err != nil {
		return "", "", nil, err
	}
	sheet := firstSheetWithRows(workbook)
	text := workbookPreviewText(workbook)
	table := &domain.TablePreview{
		Headers:   tableHeaders(sheet.Rows),
		Rows:      tableRows(sheet.Rows),
		Delimiter: "\t",
		Sheet:     sheet.Name,
		Sheets:    workbookSheetNames(workbook),
		Truncated: sheet.Truncated,
	}
	return text, "xlsx", table, nil
}

func readTableRecords(reader *csv.Reader) ([][]string, bool, error) {
	records := [][]string{}
	maxRecords := tablePreviewMaxRows + 1
	for len(records) < maxRecords {
		record, err := reader.Read()
		if err == io.EOF {
			return records, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		records = append(records, trimRecord(record))
	}
	if _, err := reader.Read(); err == io.EOF {
		return records, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return records, true, nil
}

func trimRecord(record []string) []string {
	if len(record) > tablePreviewMaxColumns {
		record = record[:tablePreviewMaxColumns]
	}
	values := make([]string, len(record))
	copy(values, record)
	return values
}

func tableHeaders(records [][]string) []string {
	if len(records) == 0 {
		return nil
	}
	return records[0]
}

func tableRows(records [][]string) [][]string {
	if len(records) <= 1 {
		return nil
	}
	return records[1:]
}

func tableDelimiter(relPath string) rune {
	if strings.EqualFold(filepath.Ext(relPath), ".tsv") {
		return '\t'
	}
	return ','
}

func firstSheetWithRows(workbook spreadsheets.Workbook) spreadsheets.Sheet {
	for _, sheet := range workbook.Sheets {
		if len(sheet.Rows) > 0 {
			return sheet
		}
	}
	if len(workbook.Sheets) > 0 {
		return workbook.Sheets[0]
	}
	return spreadsheets.Sheet{}
}

func workbookSheetNames(workbook spreadsheets.Workbook) []string {
	names := make([]string, 0, len(workbook.Sheets))
	for _, sheet := range workbook.Sheets {
		names = append(names, sheet.Name)
	}
	return names
}

func workbookPreviewText(workbook spreadsheets.Workbook) string {
	var builder strings.Builder
	builder.WriteString("Workbook sheets: ")
	builder.WriteString(strings.Join(workbookSheetNames(workbook), ", "))
	for _, sheet := range workbook.Sheets {
		builder.WriteString("\n\n## ")
		builder.WriteString(sheet.Name)
		if sheet.Truncated {
			builder.WriteString(" (truncated)")
		}
		for _, row := range sheet.Rows {
			builder.WriteString("\n")
			builder.WriteString(strings.Join(row, "\t"))
		}
	}
	return strings.TrimSpace(builder.String())
}
