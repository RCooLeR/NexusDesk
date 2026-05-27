package dbconnector

import (
	"database/sql"
	"fmt"
)

func rowScanners(columnCount int) []any {
	scanners := make([]any, columnCount)
	for index := range scanners {
		var value any
		scanners[index] = &value
	}
	return scanners
}

func scanRowAsStrings(rows *sql.Rows, scanners []any) ([]string, error) {
	if err := rows.Scan(scanners...); err != nil {
		return nil, err
	}
	row := make([]string, len(scanners))
	for index, scanner := range scanners {
		value := scanner.(*any)
		if value == nil || *value == nil {
			continue
		}
		row[index] = stringifyValue(*value)
	}
	return row, nil
}

func scanRowsAsStrings(rows *sql.Rows) ([][]string, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	scanners := rowScanners(len(columns))
	result := [][]string{}
	for rows.Next() {
		row, err := scanRowAsStrings(rows, scanners)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func stringifyValue(value any) string {
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(typed)
	}
}
