package workspace

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
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
	matchedRows := [][]string{}
	total := 0
	for _, record := range records[1:] {
		total++
		if !filter.matches(record) {
			continue
		}
		matchedRows = append(matchedRows, trimRecordWidth(record, csvPreviewMaxColumns))
	}

	if filter.orderIndex >= 0 {
		sort.SliceStable(matchedRows, func(i, j int) bool {
			left := valueAt(matchedRows[i], filter.orderIndex)
			right := valueAt(matchedRows[j], filter.orderIndex)
			result := compareDatasetValues(left, right)
			if filter.orderDesc {
				return result > 0
			}
			return result < 0
		})
	}

	displayLimit := datasetQueryMaxRows
	if filter.limit > 0 && filter.limit < displayLimit {
		displayLimit = filter.limit
	}
	rows := [][]string{}
	for _, row := range matchedRows {
		if len(rows) >= displayLimit {
			break
		}
		rows = append(rows, row)
	}

	matched := len(matchedRows)
	message := fmt.Sprintf("%d matching rows from %s.", matched, preview.RelPath)
	if filter.orderIndex >= 0 {
		message = fmt.Sprintf("%s Ordered by %s.", message, selectedColumnName(columns, filter.orderIndex))
	}
	if matched > len(rows) {
		message = fmt.Sprintf("%d matching rows from %s; showing %d.", matched, preview.RelPath, len(rows))
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
	operator    string
	orderIndex  int
	orderDesc   bool
	limit       int
}

func parseDatasetQueryFilter(query string, columns []string) datasetQueryFilter {
	query = strings.TrimSpace(query)
	filter := datasetQueryFilter{query: strings.ToLower(query), columnIndex: -1, orderIndex: -1}
	if query == "" {
		return filter
	}

	filter.limit, query = parseDatasetLimit(query)
	filter.orderIndex, filter.orderDesc, query = parseDatasetOrder(query, columns)
	filter.query = strings.ToLower(strings.TrimSpace(query))
	if strings.TrimSpace(query) == "" {
		return filter
	}

	if left, right, ok := strings.Cut(strings.ToLower(query), " contains "); ok {
		columnIndex := queryColumnIndexByName(columns, strings.TrimSpace(left))
		if columnIndex >= 0 {
			filter.columnIndex = columnIndex
			filter.operator = "contains"
			filter.query = strings.TrimSpace(right)
			return filter
		}
	}

	for _, operator := range []string{">=", "<=", "!=", ">", "<", "=", ":"} {
		left, right, ok := strings.Cut(query, operator)
		if !ok || strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
			continue
		}
		columnIndex := queryColumnIndexByName(columns, strings.TrimSpace(left))
		if columnIndex >= 0 {
			filter.columnIndex = columnIndex
			filter.operator = operator
			filter.query = strings.ToLower(strings.TrimSpace(right))
			return filter
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
		return matchDatasetCell(record[f.columnIndex], f.query, f.operator)
	}
	for _, value := range record {
		if strings.Contains(strings.ToLower(value), f.query) {
			return true
		}
	}
	return false
}

func parseDatasetLimit(query string) (int, string) {
	fields := strings.Fields(query)
	for index := 0; index < len(fields)-1; index++ {
		if !strings.EqualFold(fields[index], "limit") {
			continue
		}
		limit, err := strconv.Atoi(fields[index+1])
		if err != nil || limit <= 0 {
			continue
		}
		return limit, strings.Join(append(fields[:index], fields[index+2:]...), " ")
	}
	return 0, query
}

func parseDatasetOrder(query string, columns []string) (int, bool, string) {
	fields := strings.Fields(query)
	for index := 0; index < len(fields)-2; index++ {
		if !strings.EqualFold(fields[index], "order") || !strings.EqualFold(fields[index+1], "by") {
			continue
		}
		columnIndex := queryColumnIndexByName(columns, fields[index+2])
		if columnIndex < 0 {
			continue
		}
		desc := false
		removeEnd := index + 3
		if len(fields) > index+3 && (strings.EqualFold(fields[index+3], "desc") || strings.EqualFold(fields[index+3], "asc")) {
			desc = strings.EqualFold(fields[index+3], "desc")
			removeEnd = index + 4
		}
		return columnIndex, desc, strings.Join(append(fields[:index], fields[removeEnd:]...), " ")
	}
	return -1, false, query
}

func queryColumnIndexByName(columns []string, name string) int {
	for index, column := range columns {
		if strings.EqualFold(strings.TrimSpace(column), strings.TrimSpace(name)) {
			return index
		}
	}
	return -1
}

func matchDatasetCell(value string, query string, operator string) bool {
	value = strings.TrimSpace(value)
	query = strings.TrimSpace(query)
	switch operator {
	case ">", ">=", "<", "<=":
		left, leftErr := strconv.ParseFloat(value, 64)
		right, rightErr := strconv.ParseFloat(query, 64)
		if leftErr != nil || rightErr != nil {
			return false
		}
		switch operator {
		case ">":
			return left > right
		case ">=":
			return left >= right
		case "<":
			return left < right
		default:
			return left <= right
		}
	case "!=":
		return !strings.EqualFold(value, query)
	default:
		return strings.Contains(strings.ToLower(value), strings.ToLower(query))
	}
}

func compareDatasetValues(left string, right string) int {
	leftNumber, leftErr := strconv.ParseFloat(strings.TrimSpace(left), 64)
	rightNumber, rightErr := strconv.ParseFloat(strings.TrimSpace(right), 64)
	if leftErr == nil && rightErr == nil {
		if leftNumber < rightNumber {
			return -1
		}
		if leftNumber > rightNumber {
			return 1
		}
		return 0
	}
	return strings.Compare(strings.ToLower(left), strings.ToLower(right))
}

func valueAt(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}
	return record[index]
}

func fileExt(name string) string {
	index := strings.LastIndex(name, ".")
	if index < 0 {
		return ""
	}
	return name[index:]
}
