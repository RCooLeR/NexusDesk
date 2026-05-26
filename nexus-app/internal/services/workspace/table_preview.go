package workspace

import (
	"encoding/csv"
	"io"
	"path/filepath"
	"strings"

	"nexusdesk/internal/domain"
)

const (
	tablePreviewMaxRows    = 50
	tablePreviewMaxColumns = 30
)

func decodeTable(content []byte, relPath string) (string, string, *domain.TablePreview, error) {
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
