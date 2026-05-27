package datasets

import (
	"strconv"
	"strings"
)

type queryFilter struct {
	query       string
	columnIndex int
	operator    string
	orderIndex  int
	orderDesc   bool
	limit       int
}

func parseQueryFilter(query string, columns []string) queryFilter {
	query = strings.TrimSpace(query)
	filter := queryFilter{query: strings.ToLower(query), columnIndex: -1, orderIndex: -1}
	if query == "" {
		return filter
	}
	filter.limit, query = parseQueryLimit(query)
	filter.orderIndex, filter.orderDesc, query = parseQueryOrder(query, columns)
	filter.query = strings.ToLower(strings.TrimSpace(query))
	if strings.TrimSpace(query) == "" {
		return filter
	}
	if left, right, ok := strings.Cut(strings.ToLower(query), " contains "); ok {
		if columnIndex := columnIndexByName(columns, strings.TrimSpace(left)); columnIndex >= 0 {
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
		if columnIndex := columnIndexByName(columns, strings.TrimSpace(left)); columnIndex >= 0 {
			filter.columnIndex = columnIndex
			filter.operator = operator
			filter.query = strings.ToLower(strings.Trim(strings.TrimSpace(right), `"'`))
			return filter
		}
	}
	return filter
}

func (f queryFilter) matches(row []string) bool {
	if f.query == "" {
		return true
	}
	if f.columnIndex >= 0 {
		return matchCell(valueAt(row, f.columnIndex), f.query, f.operator)
	}
	for _, value := range row {
		if strings.Contains(strings.ToLower(value), f.query) {
			return true
		}
	}
	return false
}

func parseQueryLimit(query string) (int, string) {
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

func parseQueryOrder(query string, columns []string) (int, bool, string) {
	fields := strings.Fields(query)
	for index := 0; index < len(fields)-2; index++ {
		if !strings.EqualFold(fields[index], "order") || !strings.EqualFold(fields[index+1], "by") {
			continue
		}
		columnIndex := columnIndexByName(columns, strings.Trim(fields[index+2], `"'`))
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

func matchCell(value string, query string, operator string) bool {
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
