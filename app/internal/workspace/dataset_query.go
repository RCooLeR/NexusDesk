package workspace

import (
	"errors"
	"fmt"
	"strings"
)

const datasetQueryMaxRows = 50

type DatasetQueryResult struct {
	RelPath     string     `json:"relPath"`
	Query       string     `json:"query"`
	Columns     []string   `json:"columns"`
	Rows        [][]string `json:"rows"`
	TotalRows   int        `json:"totalRows"`
	MatchedRows int        `json:"matchedRows"`
	Message     string     `json:"message"`
}

func QueryCSV(root string, relPath string, query string) (DatasetQueryResult, error) {
	preview, err := Preview(root, relPath, PreviewOptions{MaxBytes: csvProfileMaxBytes})
	if err != nil {
		return DatasetQueryResult{}, err
	}
	if preview.Table == nil || !strings.EqualFold(fileExt(preview.Name), ".csv") {
		return DatasetQueryResult{}, errors.New("dataset query currently supports CSV files")
	}

	records, err := readCSVRecords(preview.Content, 0)
	if err != nil {
		return DatasetQueryResult{}, err
	}
	if len(records) == 0 {
		return DatasetQueryResult{}, errors.New("CSV dataset is empty")
	}

	columns := buildCSVColumns(records, csvPreviewMaxColumns)
	filter := parseDatasetQueryFilter(query, columns)
	rows := [][]string{}
	matched := 0
	total := 0
	for _, record := range records[1:] {
		total++
		if !filter.matches(record) {
			continue
		}
		matched++
		if len(rows) < datasetQueryMaxRows {
			rows = append(rows, trimRecordWidth(record, csvPreviewMaxColumns))
		}
	}

	message := fmt.Sprintf("%d matching rows from %s.", matched, preview.RelPath)
	if matched > len(rows) {
		message = fmt.Sprintf("%d matching rows from %s; showing first %d.", matched, preview.RelPath, len(rows))
	}
	return DatasetQueryResult{
		RelPath:     preview.RelPath,
		Query:       strings.TrimSpace(query),
		Columns:     columns,
		Rows:        rows,
		TotalRows:   total,
		MatchedRows: matched,
		Message:     message,
	}, nil
}

type datasetQueryFilter struct {
	query       string
	columnIndex int
}

func parseDatasetQueryFilter(query string, columns []string) datasetQueryFilter {
	query = strings.TrimSpace(query)
	filter := datasetQueryFilter{query: strings.ToLower(query), columnIndex: -1}
	if query == "" {
		return filter
	}

	for _, separator := range []string{"=", ":"} {
		left, right, ok := strings.Cut(query, separator)
		if !ok {
			continue
		}
		left = strings.TrimSpace(left)
		right = strings.TrimSpace(right)
		if left == "" || right == "" {
			continue
		}
		for index, column := range columns {
			if strings.EqualFold(strings.TrimSpace(column), left) {
				return datasetQueryFilter{query: strings.ToLower(right), columnIndex: index}
			}
		}
	}

	return filter
}

func (f datasetQueryFilter) matches(record []string) bool {
	if f.query == "" {
		return true
	}
	if f.columnIndex >= 0 {
		if f.columnIndex >= len(record) {
			return false
		}
		return strings.Contains(strings.ToLower(record[f.columnIndex]), f.query)
	}
	for _, value := range record {
		if strings.Contains(strings.ToLower(value), f.query) {
			return true
		}
	}
	return false
}

func fileExt(name string) string {
	index := strings.LastIndex(name, ".")
	if index < 0 {
		return ""
	}
	return name[index:]
}
