package analytics

import (
	"encoding/csv"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"NexusDesk/internal/workspace"
)

const maxSQLRows = 100

type SQLQueryRequest struct {
	RelPath string `json:"relPath"`
	SQL     string `json:"sql"`
}

type SQLQueryResult struct {
	RelPath     string     `json:"relPath"`
	SQL         string     `json:"sql"`
	Engine      string     `json:"engine"`
	Columns     []string   `json:"columns"`
	Rows        [][]string `json:"rows"`
	TotalRows   int        `json:"totalRows"`
	MatchedRows int        `json:"matchedRows"`
	Message     string     `json:"message"`
}

type parsedSelect struct {
	columns   []string
	where     string
	orderBy   string
	orderDesc bool
	limit     int
}

func QueryCSVSQL(root string, request SQLQueryRequest) (SQLQueryResult, error) {
	sql := strings.TrimSpace(request.SQL)
	if sql == "" {
		return SQLQueryResult{}, errors.New("enter a read-only SELECT query")
	}
	parsed, err := parseSelectSQL(sql)
	if err != nil {
		return SQLQueryResult{}, err
	}
	filterQuery := parsed.where
	if parsed.orderBy != "" {
		filterQuery = strings.TrimSpace(filterQuery + " order by " + parsed.orderBy + orderSuffix(parsed.orderDesc))
	}
	if parsed.limit > 0 {
		filterQuery = strings.TrimSpace(filterQuery + " limit " + strconv.Itoa(parsed.limit))
	}

	result, err := workspace.QueryCSV(root, request.RelPath, filterQuery)
	if err != nil {
		return SQLQueryResult{}, err
	}
	columns, rows, err := projectColumns(result.Columns, result.Rows, parsed.columns)
	if err != nil {
		return SQLQueryResult{}, err
	}
	if len(rows) > maxSQLRows {
		rows = rows[:maxSQLRows]
	}

	return SQLQueryResult{
		RelPath:     result.RelPath,
		SQL:         sql,
		Engine:      "duckdb-compatible-csv",
		Columns:     columns,
		Rows:        rows,
		TotalRows:   result.TotalRows,
		MatchedRows: result.MatchedRows,
		Message:     fmt.Sprintf("DuckDB-compatible read-only query returned %d rows from %s.", len(rows), result.RelPath),
	}, nil
}

func parseSelectSQL(sql string) (parsedSelect, error) {
	normalized := strings.TrimSpace(strings.TrimSuffix(sql, ";"))
	lower := strings.ToLower(normalized)
	if !strings.HasPrefix(lower, "select ") {
		return parsedSelect{}, errors.New("only read-only SELECT queries are supported")
	}
	if containsBlockedSQL(lower) {
		return parsedSelect{}, errors.New("query contains a blocked SQL statement")
	}

	fromIndex := strings.Index(lower, " from ")
	if fromIndex < 0 {
		return parsedSelect{}, errors.New("SELECT query must include FROM")
	}
	columnText := strings.TrimSpace(normalized[len("select "):fromIndex])
	if columnText == "" {
		return parsedSelect{}, errors.New("SELECT query must include columns")
	}
	columns := splitCSVList(columnText)
	remainder := strings.TrimSpace(normalized[fromIndex+len(" from "):])
	lowerRemainder := strings.ToLower(remainder)
	if lowerRemainder == "" {
		return parsedSelect{}, errors.New("SELECT query must include a source")
	}
	sourceEnd := len(remainder)
	for _, marker := range []string{" where ", " order by ", " limit "} {
		if index := strings.Index(lowerRemainder, marker); index >= 0 && index < sourceEnd {
			sourceEnd = index
		}
	}
	source := strings.TrimSpace(remainder[:sourceEnd])
	if source == "" {
		return parsedSelect{}, errors.New("SELECT query must include a source")
	}

	tail := strings.TrimSpace(remainder[sourceEnd:])
	where, tail := cutSQLClause(tail, "where", []string{" order by ", " limit "})
	orderBy, tail := cutSQLClause(tail, "order by", []string{" limit "})
	limitText, _ := cutSQLClause(tail, "limit", nil)
	orderColumn, orderDesc := parseOrder(orderBy)
	limit := 0
	if strings.TrimSpace(limitText) != "" {
		parsedLimit, err := strconv.Atoi(strings.Fields(limitText)[0])
		if err != nil || parsedLimit <= 0 {
			return parsedSelect{}, errors.New("LIMIT must be a positive integer")
		}
		limit = parsedLimit
	}

	return parsedSelect{
		columns:   columns,
		where:     sqlWhereToDatasetQuery(where),
		orderBy:   orderColumn,
		orderDesc: orderDesc,
		limit:     limit,
	}, nil
}

func containsBlockedSQL(lower string) bool {
	for _, blocked := range []string{" insert ", " update ", " delete ", " drop ", " alter ", " truncate ", " create ", " attach ", " copy "} {
		if strings.Contains(" "+lower+" ", blocked) {
			return true
		}
	}
	return false
}

func splitCSVList(value string) []string {
	reader := csv.NewReader(strings.NewReader(value))
	reader.TrimLeadingSpace = true
	fields, err := reader.Read()
	if err != nil {
		return []string{strings.TrimSpace(value)}
	}
	for index, field := range fields {
		fields[index] = strings.Trim(strings.TrimSpace(field), "\"`")
	}
	return fields
}

func cutSQLClause(input string, keyword string, nextMarkers []string) (string, string) {
	trimmed := strings.TrimSpace(input)
	lower := strings.ToLower(trimmed)
	prefix := keyword + " "
	if !strings.HasPrefix(lower, prefix) {
		return "", trimmed
	}
	body := strings.TrimSpace(trimmed[len(prefix):])
	lowerBody := strings.ToLower(body)
	end := len(body)
	for _, marker := range nextMarkers {
		if index := strings.Index(lowerBody, marker); index >= 0 && index < end {
			end = index
		}
	}
	return strings.TrimSpace(body[:end]), strings.TrimSpace(body[end:])
}

func parseOrder(value string) (string, bool) {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "", false
	}
	desc := len(fields) > 1 && strings.EqualFold(fields[1], "desc")
	return strings.Trim(fields[0], "\"`"), desc
}

func sqlWhereToDatasetQuery(where string) string {
	where = strings.TrimSpace(where)
	if where == "" {
		return ""
	}
	where = strings.ReplaceAll(where, " AND ", " and ")
	if strings.Contains(strings.ToLower(where), " and ") {
		where = strings.SplitN(where, " and ", 2)[0]
	}
	where = strings.ReplaceAll(where, "'", "")
	where = strings.ReplaceAll(where, "\"", "")
	where = strings.ReplaceAll(where, " like ", " contains ")
	where = strings.ReplaceAll(where, " LIKE ", " contains ")
	return strings.TrimSpace(where)
}

func projectColumns(columns []string, rows [][]string, selected []string) ([]string, [][]string, error) {
	if len(selected) == 1 && selected[0] == "*" {
		return columns, rows, nil
	}
	indexes := []int{}
	nextColumns := []string{}
	for _, column := range selected {
		column = strings.TrimSpace(strings.TrimSuffix(filepath.Base(column), ".csv"))
		index := -1
		for candidateIndex, candidate := range columns {
			if strings.EqualFold(candidate, column) {
				index = candidateIndex
				break
			}
		}
		if index < 0 {
			return nil, nil, fmt.Errorf("selected column %q is not in the dataset", column)
		}
		indexes = append(indexes, index)
		nextColumns = append(nextColumns, columns[index])
	}
	nextRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		nextRow := make([]string, 0, len(indexes))
		for _, index := range indexes {
			if index < len(row) {
				nextRow = append(nextRow, row[index])
			} else {
				nextRow = append(nextRow, "")
			}
		}
		nextRows = append(nextRows, nextRow)
	}
	return nextColumns, nextRows, nil
}

func orderSuffix(desc bool) string {
	if desc {
		return " desc"
	}
	return ""
}
