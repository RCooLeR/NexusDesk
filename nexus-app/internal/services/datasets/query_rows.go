package datasets

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"nexusdesk/internal/domain"
)

func queryableRowsFromPreview(preview domain.FilePreview) ([]string, [][]string, string, bool, error) {
	if strings.EqualFold(filepath.Ext(preview.RelPath), ".xlsx") {
		if preview.Table == nil {
			return nil, nil, "", false, fmt.Errorf("xlsx table preview is unavailable")
		}
		return normalizeColumns(preview.Table.Headers), preview.Table.Rows, "XLSX", preview.Table.Truncated, nil
	}
	return queryableRows(preview.RelPath, preview.Text)
}

func queryableRows(relPath string, text string) ([]string, [][]string, string, bool, error) {
	switch strings.ToLower(filepath.Ext(relPath)) {
	case ".csv":
		columns, rows, err := parseDelimitedRows(text, ',')
		return columns, rows, "CSV", false, err
	case ".tsv":
		columns, rows, err := parseDelimitedRows(text, '\t')
		return columns, rows, "TSV", false, err
	case ".json":
		columns, rows, err := parseJSONRows(text)
		return columns, rows, "JSON", false, err
	case ".ndjson", ".jsonl":
		columns, rows, _, err := parseNDJSONRows(text)
		return columns, rows, "NDJSON", false, err
	case ".log":
		columns, rows := parseLogRows(text)
		return columns, rows, "LOG", false, nil
	default:
		return nil, nil, "", false, fmt.Errorf("dataset query supports CSV, TSV, JSON, NDJSON, XLSX, and LOG files, not %q", filepath.Ext(relPath))
	}
}

func parseDelimitedRows(text string, delimiter rune) ([]string, [][]string, error) {
	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(records) == 0 {
		return nil, nil, nil
	}
	columns := normalizeColumns(records[0])
	rows := make([][]string, 0, len(records)-1)
	for _, record := range records[1:] {
		rows = append(rows, trimRowWidth(record, len(columns)))
	}
	return columns, rows, nil
}

func parseJSONRows(text string) ([]string, [][]string, error) {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, nil, err
	}
	switch typed := value.(type) {
	case []any:
		columns, rows := rowsFromJSONArray(typed)
		return columns, rows, nil
	case map[string]any:
		keys := sortedKeys(typed)
		row := make([]string, len(keys))
		for index, key := range keys {
			row[index] = scalarSummary(typed[key])
		}
		return keys, [][]string{row}, nil
	default:
		return []string{"value"}, [][]string{{scalarSummary(typed)}}, nil
	}
}

func rowsFromJSONArray(values []any) ([]string, [][]string) {
	keySet := map[string]struct{}{}
	allObjects := true
	for _, value := range values {
		object, ok := value.(map[string]any)
		if !ok {
			allObjects = false
			break
		}
		for key := range object {
			keySet[key] = struct{}{}
		}
	}
	if !allObjects {
		rows := make([][]string, 0, len(values))
		for _, value := range values {
			rows = append(rows, []string{scalarSummary(value)})
		}
		return []string{"value"}, rows
	}
	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make([][]string, 0, len(values))
	for _, value := range values {
		object := value.(map[string]any)
		row := make([]string, len(keys))
		for index, key := range keys {
			row[index] = scalarSummary(object[key])
		}
		rows = append(rows, row)
	}
	return keys, rows
}
