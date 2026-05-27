package datasets

import (
	"fmt"
	"strconv"
	"strings"
)

func compareValues(left string, right string) int {
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

func normalizeColumns(headers []string) []string {
	columns := make([]string, len(headers))
	for index, header := range headers {
		columns[index] = columnName(headers, index)
		if strings.TrimSpace(header) == "" {
			columns[index] = fmt.Sprintf("column_%d", index+1)
		}
	}
	return columns
}

func columnIndexByName(columns []string, name string) int {
	for index, column := range columns {
		if strings.EqualFold(strings.TrimSpace(column), strings.TrimSpace(name)) {
			return index
		}
	}
	return -1
}

func trimRowWidth(row []string, width int) []string {
	if len(row) > width {
		row = row[:width]
	}
	trimmed := make([]string, width)
	copy(trimmed, row)
	return trimmed
}

func valueAt(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}

func queryMessage(relPath string, total int, matched int, shown int, filter queryFilter, columns []string, sourceTruncated bool) string {
	message := fmt.Sprintf("%d matching rows from %s; showing %d of %d loaded rows.", matched, relPath, shown, total)
	if filter.orderIndex >= 0 {
		message += " Ordered by " + columns[filter.orderIndex] + "."
	}
	if sourceTruncated {
		message += " Source preview was truncated before query."
	}
	return message
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
